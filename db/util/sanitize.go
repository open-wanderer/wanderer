package util

import (
	"html"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
	htmlparser "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var htmlFields = map[string][]string{
	"activitypub_actors": {"summary"},
	"comments":           {"text"},
	"lists":              {"description"},
	"settings":           {"bio"},
	"summit_logs":        {"text"},
	"trails":             {"description"},
	"waypoints":          {"description"},
}

var richTextPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowStandardAttributes()
	p.AllowStandardURLs()
	p.AllowLists()
	p.AllowElements("br", "div", "hr", "p", "span", "wbr")
	p.AllowElements("b", "strong", "em", "u", "blockquote", "a")
	p.AllowAttrs("href", "target", "class").OnElements("a")
	return p
}()

// SanitizedHTMLChanges returns only changed rich-text fields, using the same
// serialized-rune limits as PocketBase. TextField.Max=0 means 5,000, not unlimited.
func SanitizedHTMLChanges(record *core.Record) map[string]string {
	changes := map[string]string{}
	for _, name := range htmlFields[record.Collection().Name] {
		field, ok := record.Collection().Fields.GetByName(name).(*core.TextField)
		if !ok {
			continue
		}
		limit := field.Max
		if limit == 0 {
			limit = 5000
		}
		before := record.GetString(name)
		if after := SanitizeHTMLText(before, limit); after != before {
			changes[name] = after
		}
	}
	return changes
}

// SanitizeHTML runs before PocketBase field validation on every model save,
// including provider imports, federation, merges and direct collection writes.
func SanitizeHTML() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		for field, value := range SanitizedHTMLChanges(e.Record) {
			e.Record.Set(field, value)
		}
		return e.Next()
	}
}

// SanitizeHTMLText limits the sanitized serialization, rather than the input.
// Complete entities and tags are retained; closing tags also consume the budget.
func SanitizeHTMLText(value string, maxRunes int) string {
	safe := richTextPolicy.Sanitize(value)
	nodes, err := htmlparser.ParseFragment(strings.NewReader(safe), &htmlparser.Node{
		Type: htmlparser.ElementNode, Data: "div", DataAtom: atom.Div,
	})
	if err != nil {
		return ""
	}
	output := boundedHTML{limit: maxRunes}
	for _, node := range nodes {
		output.appendNode(node)
	}
	return output.text.String()
}

type boundedHTML struct {
	text     strings.Builder
	limit    int
	used     int
	reserved int
	stopped  bool
}

func (b *boundedHTML) fits(cost int) bool {
	return b.limit <= 0 || b.used+b.reserved+cost <= b.limit
}

func (b *boundedHTML) appendNode(node *htmlparser.Node) {
	if b.stopped {
		return
	}
	switch node.Type {
	case htmlparser.TextNode:
		for _, r := range node.Data {
			encoded := html.EscapeString(string(r))
			cost := utf8.RuneCountInString(encoded)
			if !b.fits(cost) {
				b.stopped = true
				return
			}
			b.text.WriteString(encoded)
			b.used += cost
		}
	case htmlparser.ElementNode:
		start := (htmlparser.Token{Type: htmlparser.StartTagToken, Data: node.Data, Attr: node.Attr}).String()
		end := "</" + node.Data + ">"
		if node.Data == "br" || node.Data == "hr" || node.Data == "wbr" {
			end = ""
		}
		startCost, endCost := utf8.RuneCountInString(start), utf8.RuneCountInString(end)
		if !b.fits(startCost + endCost) {
			// A long URL or wrapper must not consume the whole text budget.
			// Its already-sanitized children remain useful without the wrapper.
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				b.appendNode(child)
			}
			return
		}
		b.text.WriteString(start)
		b.used += startCost
		b.reserved += endCost
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			b.appendNode(child)
		}
		b.reserved -= endCost
		b.text.WriteString(end)
		b.used += endCost
	}
}
