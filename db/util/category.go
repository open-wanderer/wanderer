package util

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

type CategoryTranslation struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
}

func NormalizeCategoryName(name string) string {
	decomposed := norm.NFD.String(name)

	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}

	folded := cases.Fold().String(b.String())

	b.Reset()
	b.Grow(len(folded))
	lastWasSeparator := false
	for _, r := range folded {
		if unicode.IsSpace(r) || r == '-' || r == '_' {
			if !lastWasSeparator {
				b.WriteByte(' ')
				lastWasSeparator = true
			}
			continue
		}

		b.WriteRune(r)
		lastWasSeparator = false
	}

	return strings.TrimSpace(b.String())
}

func ParseCategoryTranslations(raw any) (map[string]CategoryTranslation, error) {
	if raw == nil {
		return nil, nil
	}

	switch value := raw.(type) {
	case map[string]CategoryTranslation:
		return value, nil
	case map[string]any:
		return normalizeCategoryTranslations(value)
	case types.JSONRaw:
		if len(value) == 0 {
			return nil, nil
		}

		var decoded map[string]any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("translations must be valid JSON: %w", err)
		}

		return normalizeCategoryTranslations(decoded)
	case []byte:
		if len(value) == 0 {
			return nil, nil
		}

		var decoded map[string]any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("translations must be valid JSON: %w", err)
		}

		return normalizeCategoryTranslations(decoded)
	case string:
		if strings.TrimSpace(value) == "" {
			return nil, nil
		}

		var decoded map[string]any
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return nil, fmt.Errorf("translations must be valid JSON: %w", err)
		}

		return normalizeCategoryTranslations(decoded)
	default:
		return nil, fmt.Errorf("translations must be a JSON object")
	}
}

func ValidateCategoryRecord(app core.App, record *core.Record) error {
	name := record.GetString("name")
	normalizedName := NormalizeCategoryName(name)

	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	for _, existing := range allCategories {
		if existing.Id == record.Id {
			continue
		}

		if NormalizeCategoryName(existing.GetString("name")) == normalizedName {
			return fmt.Errorf("category name %q collides with existing category %q after normalization", name, existing.GetString("name"))
		}
	}

	if _, err := ParseCategoryTranslations(record.Get("translations")); err != nil {
		return err
	}

	return nil
}

func FindCategoryByNormalizedName(app core.App, name string) (*core.Record, error) {
	normalizedName := NormalizeCategoryName(name)
	if normalizedName == "" {
		return nil, nil
	}

	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return nil, err
	}

	for _, category := range allCategories {
		if NormalizeCategoryName(category.GetString("name")) == normalizedName {
			return category, nil
		}
	}

	return nil, nil
}

func ValidateCategoryCollectionState(app core.App) error {
	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	seen := map[string]string{}
	for _, category := range allCategories {
		name := category.GetString("name")
		normalizedName := NormalizeCategoryName(name)
		if other, ok := seen[normalizedName]; ok {
			return fmt.Errorf("category normalization collision: %q conflicts with %q", name, other)
		}
		seen[normalizedName] = name

		if _, err := ParseCategoryTranslations(category.Get("translations")); err != nil {
			return fmt.Errorf("invalid translations for category %q: %w", name, err)
		}
	}

	return nil
}

func normalizeCategoryTranslations(raw map[string]any) (map[string]CategoryTranslation, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	translations := make(map[string]CategoryTranslation, len(raw))
	for locale, entry := range raw {
		tag, err := language.Parse(locale)
		if err != nil {
			return nil, fmt.Errorf("translations locale %q is invalid", locale)
		}
		base, _ := tag.Base()
		if locale != base.String() {
			return nil, fmt.Errorf("translations locale %q must use the base locale %q", locale, base.String())
		}
		if _, ok := supportedCategoryLocales[base.String()]; !ok {
			return nil, fmt.Errorf("translations locale %q is not supported", locale)
		}

		entryMap, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("translations[%s] must be an object", locale)
		}

		translation := CategoryTranslation{}
		if name, ok := entryMap["name"]; ok {
			nameString, ok := name.(string)
			if !ok {
				return nil, fmt.Errorf("translations[%s].name must be a string", locale)
			}
			translation.Name = nameString
		}

		if shortName, ok := entryMap["short_name"]; ok {
			shortNameString, ok := shortName.(string)
			if !ok {
				return nil, fmt.Errorf("translations[%s].short_name must be a string", locale)
			}
			translation.ShortName = shortNameString
		}

		translations[locale] = translation
	}

	return translations, nil
}
