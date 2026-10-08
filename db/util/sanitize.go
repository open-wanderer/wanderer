package util

import (
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

// SanitizedHTMLChanges returns changed rich-text fields without truncating them.
// PocketBase validates the complete sanitized serialization on ordinary writes.
func SanitizedHTMLChanges(record *core.Record) map[string]string {
	return sanitizedHTMLChanges(record, false)
}

// SanitizeHTMLFieldsWithLimits explicitly bounds imported or derived rich text
// before saving it. TextField.Max=0 means PocketBase's 5,000-character default.
// Ordinary user writes use SanitizeHTML instead and fail validation if too long.
func SanitizeHTMLFieldsWithLimits(record *core.Record) {
	for field, value := range sanitizedHTMLChanges(record, true) {
		record.Set(field, value)
	}
}

func sanitizedHTMLChanges(record *core.Record, bounded bool) map[string]string {
	changes := map[string]string{}
	for _, name := range htmlFields[record.Collection().Name] {
		field, ok := record.Collection().Fields.GetByName(name).(*core.TextField)
		if !ok {
			continue
		}
		before := record.GetString(name)
		var after string
		if bounded {
			limit := field.Max
			if limit == 0 {
				limit = 5000
			}
			after = SanitizeHTMLTextWithLimit(before, limit)
		} else {
			after = SanitizeHTMLText(before)
		}
		if after != before {
			changes[name] = after
		}
	}
	return changes
}

// SanitizeHTML runs before PocketBase field validation on every model save,
// including provider imports, federation, merges and direct collection writes.
// It never silently shortens a value to satisfy the field's length constraint.
func SanitizeHTML() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		for field, value := range SanitizedHTMLChanges(e.Record) {
			e.Record.Set(field, value)
		}
		return e.Next()
	}
}

// SanitizeHTMLText sanitizes a complete value without shortening its safe text.
func SanitizeHTMLText(value string) string {
	return serializeSanitizedHTML(value, 0)
}

// SanitizeHTMLTextWithLimit explicitly limits the sanitized serialization.
// Complete entities and tags are retained; closing tags also consume the budget.
func SanitizeHTMLTextWithLimit(value string, maxRunes int) string {
	return serializeSanitizedHTML(value, maxRunes)
}

func serializeSanitizedHTML(value string, maxRunes int) string {
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
			// Quotes are ordinary text here. Attribute values are escaped by
			// htmlparser.Token.String below, where their quoting matters.
			encoded := string(r)
			switch r {
			case '&':
				encoded = "&amp;"
			case '<':
				encoded = "&lt;"
			case '>':
				encoded = "&gt;"
			}
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
