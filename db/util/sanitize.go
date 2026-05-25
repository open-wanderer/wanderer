package util

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
)

var targetBlankPattern = regexp.MustCompile(`^_blank$`)

func SanitizeHTML() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		fieldsToSanitize := map[string][]string{
			"lists":       {"description"},
			"settings":    {"bio"},
			"summit_logs": {"text"},
			"trails":      {"description"},
			"comments":    {"text"},
			"waypoints":   {"description"},
		}
		collection := e.Collection.Name
		fields, ok := fieldsToSanitize[collection]
		if !ok {
			return e.Next()
		}

		p := bluemonday.NewPolicy()
		p.AllowStandardAttributes()
		p.AllowStandardURLs()
		p.AllowLists()
		p.AllowElements("br", "div", "hr", "p", "span", "wbr")
		p.AllowElements("b", "strong", "em", "u", "blockquote", "a")
		p.AllowAttrs("href").OnElements("a")
		// AllowStandardURLs (above) already limits href schemes to
		// http/https/mailto, so javascript: URLs are stripped. Constrain target
		// to _blank only, instead of allowing arbitrary frame targets.
		p.AllowAttrs("target").Matching(targetBlankPattern).OnElements("a")
		p.AllowAttrs("class").OnElements("a")

		for _, field := range fields {
			if val, ok := e.Record.Get(field).(string); ok {
				sanitizedValue := p.Sanitize(val)
				e.Record.Set(field, sanitizedValue)
			}
		}

		return e.Next()
	}
}
