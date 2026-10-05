package util

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestSanitizeHTMLTextWithLimitKeepsCompleteMarkupAndEntities(t *testing.T) {
	for _, limit := range []int{5000, 10000} {
		for _, value := range []string{"'", `"`, "&", "山", "🚲"} {
			t.Run(value+"/"+strconv.Itoa(limit), func(t *testing.T) {
				input := "<p><strong>" + strings.Repeat(value, limit) + "</strong></p>"
				got := SanitizeHTMLTextWithLimit(input, limit)
				if !utf8.ValidString(got) || utf8.RuneCountInString(got) > limit {
					t.Fatalf("invalid serialized length %d (limit %d)", utf8.RuneCountInString(got), limit)
				}
				if !strings.HasPrefix(got, "<p><strong>") || !strings.HasSuffix(got, "</strong></p>") {
					t.Fatalf("truncated markup is not balanced: %.50s", got)
				}
				if again := SanitizeHTMLTextWithLimit(got, limit); again != got {
					t.Fatal("sanitization or entity truncation is not idempotent")
				}
			})
		}
	}
	if got := SanitizeHTMLTextWithLimit("<p>1234&amp;Z</p>", 15); got != "<p>1234</p>" {
		t.Fatalf("entity was partially serialized: %q", got)
	}
	if got := SanitizeHTMLTextWithLimit(`<a href="https://example.test/`+strings.Repeat("a", 100)+`">Useful text</a>`, 20); got != "Useful text" {
		t.Fatalf("oversized link lost its safe text: %q", got)
	}
}

func TestSanitizeHTMLTextRetainsSafeContent(t *testing.T) {
	input := `<p>Grüezi <strong>山歩き 🚲</strong></p><ul><li>One</li><li>Two</li></ul><a class="mention" href="https://example.test/profile">@friend</a>`
	got := SanitizeHTMLText(input)
	for _, want := range []string{"Grüezi", "<strong>山歩き 🚲</strong>", "<ul><li>One</li><li>Two</li></ul>", `class="mention"`, `href="https://example.test/profile"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("safe content %q was lost: %q", want, got)
		}
	}
	if again := SanitizeHTMLText(got); again != got {
		t.Fatalf("safe HTML changed on repeated save: %q", again)
	}
}

func TestSanitizeHTMLTextDoesNotTruncateOrEscapeTextQuotes(t *testing.T) {
	for _, input := range []string{
		strings.Repeat("x", 6000),
		strings.Repeat(`'"`, 2028) + "x", // 4,057 visible and serialized characters.
	} {
		if got := SanitizeHTMLText(input); got != input {
			t.Fatalf("plain text changed: input length %d, output length %d", utf8.RuneCountInString(input), utf8.RuneCountInString(got))
		}
	}
	input := strings.Repeat("x", 4997) + "&&"
	want := strings.Repeat("x", 4997) + "&amp;&amp;"
	if got := SanitizeHTMLText(input); got != want {
		t.Fatalf("required text escaping lost content: input length %d, output length %d", len(input), len(got))
	}
	quotes := strings.Repeat(`'"`, 2028) + "x"
	if got := SanitizeHTMLTextWithLimit(quotes, 5000); got != quotes {
		t.Fatal("the bounded import sanitizer needlessly escaped or truncated text quotes")
	}
}

func TestSanitizeHTMLCoversModelCreateUpdateAndValidation(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	app.OnRecordValidate().BindFunc(SanitizeHTML())
	fields := map[string]string{
		"activitypub_actors": "summary", "comments": "text", "lists": "description", "settings": "bio",
		"summit_logs": "text", "trails": "description", "waypoints": "description",
	}
	for collectionName, field := range fields {
		t.Run(collectionName, func(t *testing.T) {
			collection := core.NewBaseCollection(collectionName)
			limit := 0 // PocketBase's default 5,000-character constraint.
			if collectionName == "trails" || collectionName == "settings" {
				limit = 10000
			}
			collection.Fields.Add(&core.TextField{Name: field, Max: limit}, &core.TextField{Name: "name"})
			if err := app.Save(collection); err != nil {
				t.Fatal(err)
			}
			record := core.NewRecord(collection)
			record.Set("name", "unchanged")
			for _, input := range []string{
				`<p onclick="blocked()">Safe</p><img src=x onerror="blocked()"><script>blocked()</script>`,
				`<a href="javascript:blocked()">Link</a><p>Updated safe text</p>`,
				strings.Repeat(`'"`, 2028) + "x",
			} {
				record.Set(field, input)
				if err := app.Save(record); err != nil {
					t.Fatalf("model save failed before/after field validation: %v", err)
				}
				stored, err := app.FindRecordById(collection, record.Id)
				if err != nil {
					t.Fatal(err)
				}
				got := stored.GetString(field)
				if strings.Contains(got, "blocked()") || strings.Contains(got, "javascript:") {
					t.Fatalf("unsafe model write: %.80s", got)
				}
				if stored.GetString("name") != "unchanged" || got == "" {
					t.Fatal("sanitization changed an unrelated field or removed safe text")
				}
				if err := app.Validate(stored); err != nil {
					t.Fatalf("sanitized text exceeds its actual field constraint: %v", err)
				}
				if !strings.Contains(input, "blocked()") && got != input {
					t.Fatal("valid text quotes changed while saving")
				}
			}
			maximum := limit
			if maximum == 0 {
				maximum = 5000
			}
			for _, input := range []string{
				strings.Repeat("x", maximum+1000),
				strings.Repeat("x", maximum-3) + "&&", // Fits before escaping, exceeds the field limit after it.
			} {
				created := core.NewRecord(collection)
				created.Set(field, input)
				if err := app.Save(created); err == nil {
					t.Fatal("overlong model create succeeded through silent truncation")
				}
				if records, err := app.FindAllRecords(collection); err != nil || len(records) != 1 {
					t.Fatalf("failed create changed the database: records=%d, error=%v", len(records), err)
				}
				before, err := app.FindRecordById(collection, record.Id)
				if err != nil {
					t.Fatal(err)
				}
				record.Set(field, input)
				if err := app.Save(record); err == nil {
					t.Fatal("overlong model update succeeded through silent truncation")
				}
				stored, err := app.FindRecordById(collection, record.Id)
				if err != nil {
					t.Fatal(err)
				}
				if stored.GetString(field) != before.GetString(field) || stored.GetString("name") != "unchanged" {
					t.Fatal("failed update changed previously stored data")
				}
			}
		})
	}
}

func TestSanitizeHTMLTextParserVariants(t *testing.T) {
	inputs := []string{
		`<a href="&#x6a;avascript:blocked()">Text</a>`,
		`<svg><g onload="blocked()">Text</g></svg>`,
		`<math><mtext><table><mglyph><style><!--</style><img title="--><img src=x onerror=blocked()>">`,
		`<p title='quote" &amp;'>Text</p>`,
		`<p><b>First<p>Second</b>Third`,
		`<a href="https://example.test/?a=1&amp;b=2" target="_blank">Text</a>`,
	}
	for _, input := range inputs {
		got := SanitizeHTMLText(input)
		for _, unsafe := range []string{"<svg", "<math", "<script", "<style", "onload=", "onerror=", "javascript:"} {
			if strings.Contains(got, unsafe) {
				t.Fatalf("unsafe parser output: %q", got)
			}
		}
		if again := SanitizeHTMLText(got); again != got {
			t.Fatalf("parser output is not idempotent: %q -> %q", got, again)
		}
	}
}
