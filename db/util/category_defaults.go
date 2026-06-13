package util

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

var supportedCategoryLocales = map[string]struct{}{
	"cs": {},
	"de": {},
	"en": {},
	"es": {},
	"eu": {},
	"fr": {},
	"hu": {},
	"it": {},
	"nl": {},
	"no": {},
	"pl": {},
	"pt": {},
	"ru": {},
	"zh": {},
}

var defaultCategoryNames = []string{"Hiking", "Walking", "Running", "Climbing", "Skiing", "Canoeing", "Biking"}

func DefaultCategoryNames() []string {
	return append([]string(nil), defaultCategoryNames...)
}

var defaultCategoryTranslations = map[string]map[string]string{
	"Biking": {
		"cs": "Cyklistika",
		"de": "Radfahren",
		"en": "Biking",
		"es": "Ciclismo",
		"eu": "Bizikleta",
		"fr": "Vélo",
		"hu": "Biking",
		"it": "Ciclismo",
		"nl": "Fietsen",
		"no": "Sykling",
		"pl": "Rower",
		"pt": "Ciclismo",
		"ru": "Велоспорт",
		"zh": "骑行",
	},
	"Canoeing": {
		"cs": "Kanoistika",
		"de": "Kanufahren",
		"en": "Canoeing",
		"es": "Remo",
		"eu": "Kanoa",
		"fr": "Canoë",
		"hu": "Canoeing",
		"it": "Canoa",
		"nl": "Kanoën",
		"no": "Padling",
		"pl": "Kajak",
		"pt": "Canoagem",
		"ru": "Каякинг",
		"zh": "划艇",
	},
	"Climbing": {
		"cs": "Horolezectví",
		"de": "Klettern",
		"en": "Climbing",
		"es": "Escalada",
		"eu": "Eskalada",
		"fr": "Escalade",
		"hu": "Climbing",
		"it": "Arrampicata",
		"nl": "Klimmen",
		"no": "Klatring",
		"pl": "Wspinaczka",
		"pt": "Escalada",
		"ru": "Скалолазание",
		"zh": "攀岩",
	},
	"Hiking": {
		"cs": "Turistika",
		"de": "Wandern",
		"en": "Hiking",
		"es": "Senderismo",
		"eu": "Mendi-ibilaldia",
		"fr": "Randonnée",
		"hu": "Hiking",
		"it": "Escursionismo",
		"nl": "Hiken",
		"no": "Vandring",
		"pl": "Wędrówka",
		"pt": "Montanhismo",
		"ru": "Пеший туризм",
		"zh": "徒步",
	},
	"Running": {
		"cs": "Běh",
		"de": "Laufen",
		"en": "Running",
		"es": "Carrera",
		"eu": "Korrika",
		"fr": "Course à pied",
		"hu": "Futás",
		"it": "Corsa",
		"nl": "Hardlopen",
		"no": "Løping",
		"pl": "Bieganie",
		"pt": "Corrida",
		"ru": "Бег",
		"zh": "跑步",
	},
	"Skiing": {
		"de": "Skifahren",
		"no": "Skisport",
	},
	"Walking": {
		"cs": "Chůze",
		"de": "Spazieren",
		"en": "Walking",
		"es": "Paseo",
		"eu": "Oinez",
		"fr": "Marche",
		"hu": "Walking",
		"it": "Camminare",
		"nl": "Wandelen",
		"no": "Gåtur",
		"pl": "Spacer",
		"pt": "Caminhada",
		"ru": "Прогулка",
		"zh": "步行",
	},
}

var defaultCategoryIcons = map[string]string{
	"Biking":   "person-biking",
	"Canoeing": "sailboat",
	"Climbing": "mountain",
	"Hiking":   "person-hiking",
	"Running":  "person-running",
	"Skiing":   "person-skiing-nordic",
	"Walking":  "person-walking",
}

var deprecatedDefaultCategoryIcons = map[string]map[string]struct{}{
	"Canoeing": {
		"ship": {},
	},
	"Climbing": {
		"mountain-sun": {},
	},
	"Skiing": {
		"person-skiing": {},
	},
}

func PrepopulateDefaultCategoryTranslations(app core.App) error {
	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	for _, category := range allCategories {
		staticTranslations, ok := defaultCategoryTranslations[category.GetString("name")]
		if !ok {
			continue
		}

		currentTranslations, err := ParseCategoryTranslations(category.Get("translations"))
		if err != nil {
			return fmt.Errorf("invalid existing translations for category %q: %w", category.GetString("name"), err)
		}
		mergedTranslations, changed := mergeDefaultCategoryTranslations(staticTranslations, currentTranslations)
		if !changed {
			continue
		}

		category.Set("translations", mergedTranslations)
		if err := app.Save(category); err != nil {
			return fmt.Errorf("failed to prepopulate translations for category %q: %w", category.GetString("name"), err)
		}
	}

	return nil
}

func PrepopulateDefaultCategoryIcons(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		return err
	}
	if collection.Fields.GetByName("icon") == nil {
		return nil
	}

	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	for _, category := range allCategories {
		categoryName := category.GetString("name")
		defaultIcon, ok := defaultCategoryIcons[categoryName]
		if !ok {
			continue
		}

		currentIcon := strings.TrimSpace(category.GetString("icon"))
		if currentIcon == defaultIcon {
			continue
		}
		if currentIcon != "" {
			deprecatedIcons := deprecatedDefaultCategoryIcons[categoryName]
			if _, ok := deprecatedIcons[currentIcon]; !ok {
				continue
			}
		}

		category.Set("icon", defaultIcon)
		if err := app.Save(category); err != nil {
			return fmt.Errorf("failed to prepopulate icon for category %q: %w", category.GetString("name"), err)
		}
	}

	return nil
}

func mergeDefaultCategoryTranslations(staticTranslations map[string]string, currentTranslations map[string]CategoryTranslation) (map[string]CategoryTranslation, bool) {
	if currentTranslations == nil {
		currentTranslations = map[string]CategoryTranslation{}
	}

	changed := false
	for locale, name := range staticTranslations {
		if name == "" {
			continue
		}

		translation := currentTranslations[locale]
		if translation.Name != "" {
			continue
		}

		translation.Name = name
		currentTranslations[locale] = translation
		changed = true
	}

	return currentTranslations, changed
}
