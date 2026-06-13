package util

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestNormalizeCategoryName(t *testing.T) {
	t.Run("WhitespaceAndSeparators", func(t *testing.T) {
		for _, value := range []string{"E-Bike", "E Bike", "e-bike"} {
			if got := NormalizeCategoryName(value); got != "e bike" {
				t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", value, got, "e bike")
			}
		}

		if got := NormalizeCategoryName("  Mountain  Biking "); got != "mountain biking" {
			t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", "  Mountain  Biking ", got, "mountain biking")
		}
	})

	t.Run("AccentFolding", func(t *testing.T) {
		for _, value := range []string{"Canoë", "Canoe"} {
			if got := NormalizeCategoryName(value); got != "canoe" {
				t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", value, got, "canoe")
			}
		}
	})

	t.Run("SeparatorsNotRemoved", func(t *testing.T) {
		if got := NormalizeCategoryName("EBike"); got != "ebike" {
			t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", "EBike", got, "ebike")
		}

		if NormalizeCategoryName("EBike") == NormalizeCategoryName("E Bike") {
			t.Fatalf("expected %q and %q to normalize differently", "EBike", "E Bike")
		}
	})
}

func TestParseCategoryTranslations(t *testing.T) {
	tests := []struct {
		name    string
		input   types.JSONRaw
		wantErr bool
		want    map[string]CategoryTranslation
	}{
		{
			name:  "valid base locale",
			input: types.JSONRaw(`{"de": {"name": "Wandern", "short_name": "WAND"}}`),
			want:  map[string]CategoryTranslation{"de": {Name: "Wandern", ShortName: "WAND"}},
		},
		{
			name:  "null is nil without error",
			input: types.JSONRaw(`null`),
			want:  nil,
		},
		{
			name:    "region locale rejected",
			input:   types.JSONRaw(`{"pt-BR": {"name": "Caminhada"}}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCategoryTranslations(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseCategoryTranslations() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCategoryTranslations() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseCategoryTranslations() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDefaultCategoryTranslationsAreSupportedAndNonEmpty(t *testing.T) {
	for _, category := range DefaultCategoryNames() {
		translations, ok := defaultCategoryTranslations[category]
		if !ok {
			t.Fatalf("defaultCategoryTranslations missing category %q", category)
		}

		for locale, name := range translations {
			if _, ok := supportedCategoryLocales[locale]; !ok {
				t.Fatalf("defaultCategoryTranslations[%q] contains unsupported locale %q", category, locale)
			}

			if name == "" {
				t.Fatalf("defaultCategoryTranslations[%q][%q] is empty", category, locale)
			}
		}
	}
}

func TestDefaultCategoryIconsAreNonEmpty(t *testing.T) {
	for _, category := range DefaultCategoryNames() {
		icon, ok := defaultCategoryIcons[category]
		if !ok {
			t.Fatalf("defaultCategoryIcons missing category %q", category)
		}

		if strings.TrimSpace(icon) == "" {
			t.Fatalf("defaultCategoryIcons[%q] is empty", category)
		}
	}
}

func TestMergeDefaultCategoryTranslations(t *testing.T) {
	merged, changed := mergeDefaultCategoryTranslations(
		map[string]string{
			"de": "Wandern",
			"en": "Hiking",
			"fr": "",
		},
		map[string]CategoryTranslation{
			"de": {
				Name:      "Custom Wandern",
				ShortName: "WAND",
			},
		},
	)

	if !changed {
		t.Fatal("mergeDefaultCategoryTranslations changed = false, want true")
	}

	want := map[string]CategoryTranslation{
		"de": {
			Name:      "Custom Wandern",
			ShortName: "WAND",
		},
		"en": {
			Name: "Hiking",
		},
	}
	if !reflect.DeepEqual(merged, want) {
		t.Fatalf("mergeDefaultCategoryTranslations() = %#v, want %#v", merged, want)
	}
}

func TestMergeDefaultSubcategoryTranslations(t *testing.T) {
	merged, changed := mergeDefaultSubcategoryTranslations(
		map[string]CategoryTranslation{
			"de": {
				Name:      "Rennrad",
				ShortName: "ROAD",
			},
			"en": {
				Name:      "Road",
				ShortName: "ROAD",
			},
		},
		map[string]CategoryTranslation{
			"de": {
				Name:      "Custom Road",
				ShortName: "ROADX",
			},
			"en": {
				Name: "Road",
			},
		},
	)

	if !changed {
		t.Fatal("mergeDefaultSubcategoryTranslations changed = false, want true")
	}

	want := map[string]CategoryTranslation{
		"de": {
			Name:      "Custom Road",
			ShortName: "ROADX",
		},
		"en": {
			Name:      "Road",
			ShortName: "ROAD",
		},
	}
	if !reflect.DeepEqual(merged, want) {
		t.Fatalf("mergeDefaultSubcategoryTranslations() = %#v, want %#v", merged, want)
	}
}

func TestPrepopulateDefaultCategoryIcons(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	hiking := createTestCategory(t, app, "Hiking")
	walking := createTestCategory(t, app, "Walking")
	canoeing := createTestCategory(t, app, "Canoeing")
	walking.Set("icon", "custom-icon")
	if err := app.Save(walking); err != nil {
		t.Fatal(err)
	}
	canoeing.Set("icon", "ship")
	if err := app.Save(canoeing); err != nil {
		t.Fatal(err)
	}

	if err := PrepopulateDefaultCategoryIcons(app); err != nil {
		t.Fatalf("PrepopulateDefaultCategoryIcons() error = %v", err)
	}

	hiking, err := app.FindRecordById("categories", hiking.Id)
	if err != nil {
		t.Fatal(err)
	}
	walking, err = app.FindRecordById("categories", walking.Id)
	if err != nil {
		t.Fatal(err)
	}
	canoeing, err = app.FindRecordById("categories", canoeing.Id)
	if err != nil {
		t.Fatal(err)
	}

	if got := hiking.GetString("icon"); got != defaultCategoryIcons["Hiking"] {
		t.Fatalf("hiking icon = %q, want %q", got, defaultCategoryIcons["Hiking"])
	}
	if got := walking.GetString("icon"); got != "custom-icon" {
		t.Fatalf("walking icon = %q, want custom icon", got)
	}
	if got := canoeing.GetString("icon"); got != defaultCategoryIcons["Canoeing"] {
		t.Fatalf("canoeing icon = %q, want %q", got, defaultCategoryIcons["Canoeing"])
	}
}

func TestDefaultSubcategories(t *testing.T) {
	seen := map[string]struct{}{}
	wantDefaults := map[string]struct{}{
		"Hiking/Winter":        {},
		"Hiking/Alpine":        {},
		"Hiking/Long-distance": {},
		"Hiking/Snowshoeing":   {},
		"Hiking/Family":        {},
		"Hiking/Pilgrimage":    {},
		"Running/Trail":        {},
		"Running/Road":         {},
		"Skiing/Cross-country": {},
		"Skiing/Skating":       {},
		"Skiing/Backcountry":   {},
	}
	wantBadgeIcons := map[string]string{
		"Biking/MTB":         "mountain",
		"Biking/Road":        "grip-lines-vertical",
		"Biking/E-Bike":      "bolt",
		"Hiking/Winter":      "snowflake",
		"Hiking/Alpine":      "mountain",
		"Hiking/Snowshoeing": "snowflake",
		"Hiking/Family":      "child",
		"Hiking/Pilgrimage":  "cross",
		"Running/Road":       "grip-lines-vertical",
	}
	for _, subcategory := range defaultSubcategories {
		if subcategory.parentCategory == "" {
			t.Fatalf("default subcategory %q parent is empty", subcategory.name)
		}
		if subcategory.name == "" {
			t.Fatal("default subcategory name is empty")
		}
		if subcategory.shortName == "" {
			t.Fatalf("default subcategory %q shortName is empty", subcategory.name)
		}

		normalizedName := NormalizeCategoryName(subcategory.name)
		key := subcategory.parentCategory + "/" + normalizedName
		if _, ok := seen[key]; ok {
			t.Fatalf("default subcategory %q duplicates normalized key %q", subcategory.name, key)
		}
		seen[key] = struct{}{}

		if _, ok := wantDefaults[subcategory.parentCategory+"/"+subcategory.name]; ok {
			delete(wantDefaults, subcategory.parentCategory+"/"+subcategory.name)
		}
		if wantBadgeIcon, ok := wantBadgeIcons[subcategory.parentCategory+"/"+subcategory.name]; ok {
			if subcategory.badgeIcon != wantBadgeIcon {
				t.Fatalf("%s/%s badgeIcon = %q, want %q", subcategory.parentCategory, subcategory.name, subcategory.badgeIcon, wantBadgeIcon)
			}
		}
		if subcategory.parentCategory == "Hiking" && subcategory.name == "Winter" {
			if subcategory.translations["de"].Name != "Winterwandern" {
				t.Fatalf("Winter de translation = %q, want Winterwandern", subcategory.translations["de"].Name)
			}
		}
		if subcategory.parentCategory == "Hiking" && subcategory.name == "Pilgrimage" {
			if subcategory.translations["de"].Name != "Pilgern" {
				t.Fatalf("Pilgrimage de translation = %q, want Pilgern", subcategory.translations["de"].Name)
			}
		}
	}

	if len(wantDefaults) > 0 {
		t.Fatalf("default subcategories missing entries: %#v", wantDefaults)
	}
}

func TestValidateSubcategoryRecord(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	hiking := createTestCategory(t, app, "Hiking")
	createTestSubcategory(t, app, biking.Id, "MTB", "MTB")

	subcategoriesCollection := mustFindTestCollection(t, app, "subcategories")

	tests := []struct {
		name      string
		category  string
		record    string
		wantError bool
	}{
		{
			name:      "requires parent category",
			category:  "",
			record:    "Road",
			wantError: true,
		},
		{
			name:      "rejects normalized collision in same parent",
			category:  biking.Id,
			record:    "mtb",
			wantError: true,
		},
		{
			name:     "allows same normalized name in different parent",
			category: hiking.Id,
			record:   "mtb",
		},
		{
			name:     "allows unique name in same parent",
			category: biking.Id,
			record:   "Road",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := core.NewRecord(subcategoriesCollection)
			record.Set("category", tt.category)
			record.Set("name", tt.record)

			err := ValidateSubcategoryRecord(app, record)
			if (err != nil) != tt.wantError {
				t.Fatalf("ValidateSubcategoryRecord() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateTrailSubcategoryRecord(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	hiking := createTestCategory(t, app, "Hiking")
	mtb := createTestSubcategory(t, app, biking.Id, "MTB", "MTB")

	trailsCollection := mustFindTestCollection(t, app, "trails")
	missingSubcategoryID := "missing1234567"

	tests := []struct {
		name                string
		category            string
		subcategory         string
		subcategoryExplicit bool
		wantSubcategory     string
		wantError           bool
	}{
		{
			name: "allows empty subcategory",
		},
		{
			name:                "requires category when subcategory is explicit",
			subcategory:         mtb.Id,
			subcategoryExplicit: true,
			wantSubcategory:     mtb.Id,
			wantError:           true,
		},
		{
			name:            "clears subcategory without category when subcategory is implicit",
			subcategory:     mtb.Id,
			wantSubcategory: "",
		},
		{
			name:                "allows matching parent category",
			category:            biking.Id,
			subcategory:         mtb.Id,
			subcategoryExplicit: true,
			wantSubcategory:     mtb.Id,
		},
		{
			name:                "rejects mismatched explicit subcategory",
			category:            hiking.Id,
			subcategory:         mtb.Id,
			subcategoryExplicit: true,
			wantSubcategory:     mtb.Id,
			wantError:           true,
		},
		{
			name:            "clears mismatched implicit subcategory",
			category:        hiking.Id,
			subcategory:     mtb.Id,
			wantSubcategory: "",
		},
		{
			name:                "rejects missing explicit subcategory",
			category:            biking.Id,
			subcategory:         missingSubcategoryID,
			subcategoryExplicit: true,
			wantSubcategory:     missingSubcategoryID,
			wantError:           true,
		},
		{
			name:            "clears missing implicit subcategory",
			category:        biking.Id,
			subcategory:     missingSubcategoryID,
			wantSubcategory: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := core.NewRecord(trailsCollection)
			record.Set("category", tt.category)
			record.Set("subcategory", tt.subcategory)

			err := ValidateTrailSubcategoryRecord(app, record, tt.subcategoryExplicit)
			if (err != nil) != tt.wantError {
				t.Fatalf("ValidateTrailSubcategoryRecord() error = %v, wantError %v", err, tt.wantError)
			}

			if got := record.GetString("subcategory"); got != tt.wantSubcategory {
				t.Fatalf("record subcategory = %q, want %q", got, tt.wantSubcategory)
			}
		})
	}
}

func TestSeedDefaultSubcategoriesSkipsNormalizedExisting(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	mtb := createTestSubcategory(t, app, biking.Id, "mtb", "CUSTOM")

	if err := SeedDefaultSubcategories(app); err != nil {
		t.Fatalf("SeedDefaultSubcategories() error = %v", err)
	}

	records, err := app.FindRecordsByFilter(
		"subcategories",
		"category = {:category}",
		"",
		0,
		0,
		map[string]any{"category": biking.Id},
	)
	if err != nil {
		t.Fatal(err)
	}

	var normalizedMTBCount int
	for _, record := range records {
		if NormalizeCategoryName(record.GetString("name")) == NormalizeCategoryName("MTB") {
			normalizedMTBCount++
		}
	}

	if normalizedMTBCount != 1 {
		t.Fatalf("normalized MTB subcategory count = %d, want 1", normalizedMTBCount)
	}

	updatedMTB, err := app.FindRecordById("subcategories", mtb.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedMTB.GetString("name"); got != "mtb" {
		t.Fatalf("existing MTB subcategory name = %q, want mtb", got)
	}
	if got := updatedMTB.GetString("short_name"); got != "CUSTOM" {
		t.Fatalf("existing MTB short_name = %q, want CUSTOM", got)
	}

	defaultBikingSubcategories := 0
	for _, subcategory := range defaultSubcategories {
		if subcategory.parentCategory == "Biking" {
			defaultBikingSubcategories++
		}
	}

	if len(records) != defaultBikingSubcategories {
		t.Fatalf("seeded Biking subcategory count = %d, want %d", len(records), defaultBikingSubcategories)
	}
}

func TestResolveCategoryAndSubcategoryByNormalizedNames(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	mtb := createTestSubcategory(t, app, biking.Id, "E-Bike", "EBIKE")
	hiking := createTestCategory(t, app, "Hiking")
	createTestSubcategory(t, app, hiking.Id, "E-Bike", "EBIKE")

	category, subcategory, err := ResolveCategoryAndSubcategoryByNormalizedNames(app, " biking ", "e bike")
	if err != nil {
		t.Fatalf("ResolveCategoryAndSubcategoryByNormalizedNames() error = %v", err)
	}
	if category == nil || category.Id != biking.Id {
		t.Fatalf("category = %#v, want Biking", category)
	}
	if subcategory == nil || subcategory.Id != mtb.Id {
		t.Fatalf("subcategory = %#v, want Biking/E-Bike", subcategory)
	}

	category, subcategory, err = ResolveCategoryAndSubcategoryByNormalizedNames(app, "Biking", "Bikepacking")
	if err != nil {
		t.Fatalf("ResolveCategoryAndSubcategoryByNormalizedNames() unknown subcategory error = %v", err)
	}
	if category == nil || category.Id != biking.Id {
		t.Fatalf("unknown subcategory category = %#v, want Biking", category)
	}
	if subcategory != nil {
		t.Fatalf("unknown subcategory = %#v, want nil", subcategory)
	}

	category, subcategory, err = ResolveCategoryAndSubcategoryByNormalizedNames(app, "Skydiving", "Wingsuit")
	if err != nil {
		t.Fatalf("ResolveCategoryAndSubcategoryByNormalizedNames() unknown category error = %v", err)
	}
	if category != nil || subcategory != nil {
		t.Fatalf("unknown category resolved category=%#v subcategory=%#v, want nil/nil", category, subcategory)
	}
}

func TestBackfillRemoteTrailCategoryBackfillsTargetCategoryOnly(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	gravel := createTestSubcategory(t, app, biking.Id, "Gravel", "GRVL")
	matchingTrail := createTestTrail(t, app, "", "", "biking", "gravel")
	otherTrail := createTestTrail(t, app, "", "", "Hiking", "gravel")

	if err := BackfillRemoteTrailCategory(app, biking); err != nil {
		t.Fatalf("BackfillRemoteTrailCategory() error = %v", err)
	}

	updatedMatchingTrail, err := app.FindRecordById("trails", matchingTrail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedMatchingTrail.GetString("category"); got != biking.Id {
		t.Fatalf("matching trail category = %q, want %q", got, biking.Id)
	}
	if got := updatedMatchingTrail.GetString("subcategory"); got != gravel.Id {
		t.Fatalf("matching trail subcategory = %q, want %q", got, gravel.Id)
	}

	updatedOtherTrail, err := app.FindRecordById("trails", otherTrail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedOtherTrail.GetString("category"); got != "" {
		t.Fatalf("other trail category = %q, want empty", got)
	}
	if got := updatedOtherTrail.GetString("subcategory"); got != "" {
		t.Fatalf("other trail subcategory = %q, want empty", got)
	}
}

func TestBackfillRemoteTrailSubcategoryBackfillsWithinParentScope(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	hiking := createTestCategory(t, app, "Hiking")
	bikingGravel := createTestSubcategory(t, app, biking.Id, "Gravel", "GRVL")
	createTestSubcategory(t, app, hiking.Id, "Gravel", "GRVL")

	existingParentTrail := createTestTrail(t, app, biking.Id, "", "Biking", "gravel")
	emptyParentTrail := createTestTrail(t, app, "", "", "biking", "gravel")
	wrongParentTrail := createTestTrail(t, app, hiking.Id, "", "Hiking", "gravel")

	if err := BackfillRemoteTrailSubcategory(app, bikingGravel); err != nil {
		t.Fatalf("BackfillRemoteTrailSubcategory() error = %v", err)
	}

	updatedExistingParentTrail, err := app.FindRecordById("trails", existingParentTrail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedExistingParentTrail.GetString("category"); got != biking.Id {
		t.Fatalf("existing parent trail category = %q, want %q", got, biking.Id)
	}
	if got := updatedExistingParentTrail.GetString("subcategory"); got != bikingGravel.Id {
		t.Fatalf("existing parent trail subcategory = %q, want %q", got, bikingGravel.Id)
	}

	updatedEmptyParentTrail, err := app.FindRecordById("trails", emptyParentTrail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedEmptyParentTrail.GetString("category"); got != biking.Id {
		t.Fatalf("empty parent trail category = %q, want %q", got, biking.Id)
	}
	if got := updatedEmptyParentTrail.GetString("subcategory"); got != bikingGravel.Id {
		t.Fatalf("empty parent trail subcategory = %q, want %q", got, bikingGravel.Id)
	}

	updatedWrongParentTrail, err := app.FindRecordById("trails", wrongParentTrail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedWrongParentTrail.GetString("category"); got != hiking.Id {
		t.Fatalf("wrong parent trail category = %q, want %q", got, hiking.Id)
	}
	if got := updatedWrongParentTrail.GetString("subcategory"); got != "" {
		t.Fatalf("wrong parent trail subcategory = %q, want empty", got)
	}
}

func TestSeedDefaultSubcategoriesRenamesLegacyDefault(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	hiking := createTestCategory(t, app, "Hiking")
	winter := createTestSubcategory(t, app, hiking.Id, "Winter Hiking", "WINT")

	if err := SeedDefaultSubcategories(app); err != nil {
		t.Fatalf("SeedDefaultSubcategories() error = %v", err)
	}
	if err := SeedDefaultSubcategories(app); err != nil {
		t.Fatalf("second SeedDefaultSubcategories() error = %v", err)
	}

	records, err := app.FindRecordsByFilter(
		"subcategories",
		"category = {:category} && short_name = 'WINT'",
		"",
		0,
		0,
		map[string]any{"category": hiking.Id},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 1 {
		t.Fatalf("Winter subcategory count = %d, want 1", len(records))
	}
	if got := records[0].GetString("name"); got != "Winter" {
		t.Fatalf("Winter subcategory name = %q, want Winter", got)
	}
	if records[0].Id != winter.Id {
		t.Fatalf("Winter subcategory id = %q, want %q", records[0].Id, winter.Id)
	}
	if got := records[0].GetString("badge_icon"); got != "snowflake" {
		t.Fatalf("Winter badge_icon = %q, want snowflake", got)
	}
}

func TestSeedDefaultSubcategoriesDoesNotOverwriteBadgeIcon(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	hiking := createTestCategory(t, app, "Hiking")
	winter := createTestSubcategory(t, app, hiking.Id, "Winter", "WINT")
	winter.Set("badge_icon", "custom")
	if err := app.Save(winter); err != nil {
		t.Fatal(err)
	}

	if err := SeedDefaultSubcategories(app); err != nil {
		t.Fatalf("SeedDefaultSubcategories() error = %v", err)
	}

	updatedWinter, err := app.FindRecordById("subcategories", winter.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedWinter.GetString("badge_icon"); got != "custom" {
		t.Fatalf("Winter badge_icon = %q, want custom", got)
	}
}

func TestSeedDefaultSubcategoriesDoesNotRenameAliasWhenCanonicalExists(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	hiking := createTestCategory(t, app, "Hiking")
	winter := createTestSubcategory(t, app, hiking.Id, "Winter", "WINT")
	legacy := createTestSubcategory(t, app, hiking.Id, "Winter Hiking", "WHIKE")

	if err := SeedDefaultSubcategories(app); err != nil {
		t.Fatalf("SeedDefaultSubcategories() error = %v", err)
	}

	updatedWinter, err := app.FindRecordById("subcategories", winter.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedWinter.GetString("name"); got != "Winter" {
		t.Fatalf("canonical subcategory name = %q, want Winter", got)
	}

	updatedLegacy, err := app.FindRecordById("subcategories", legacy.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedLegacy.GetString("name"); got != "Winter Hiking" {
		t.Fatalf("legacy subcategory name = %q, want Winter Hiking", got)
	}

	records, err := app.FindRecordsByFilter(
		"subcategories",
		"category = {:category} && name = 'Winter'",
		"",
		0,
		0,
		map[string]any{"category": hiking.Id},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("canonical Winter subcategory count = %d, want 1", len(records))
	}
}

func TestValidateUserCategoryPreferenceRequest(t *testing.T) {
	tests := []struct {
		name             string
		priorityExplicit bool
		wantError        bool
	}{
		{
			name: "allows requests without priority",
		},
		{
			name:             "rejects explicit priority",
			priorityExplicit: true,
			wantError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserCategoryPreferenceRequest(tt.priorityExplicit)
			if (err != nil) != tt.wantError {
				t.Fatalf("ValidateUserCategoryPreferenceRequest() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestReorderUserCategoryPreferencesValidation(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		categoryIDs func(categories []*core.Record) []string
		wantError   bool
	}{
		{
			name:   "requires authenticated user",
			userID: "",
			categoryIDs: func(categories []*core.Record) []string {
				return []string{categories[0].Id, categories[1].Id}
			},
			wantError: true,
		},
		{
			name:   "rejects incomplete list",
			userID: "user12345678901",
			categoryIDs: func(categories []*core.Record) []string {
				return []string{categories[0].Id}
			},
			wantError: true,
		},
		{
			name:   "rejects duplicate category",
			userID: "user12345678901",
			categoryIDs: func(categories []*core.Record) []string {
				return []string{categories[0].Id, categories[0].Id}
			},
			wantError: true,
		},
		{
			name:   "rejects unknown category",
			userID: "user12345678901",
			categoryIDs: func(categories []*core.Record) []string {
				return []string{categories[0].Id, "unknown12345678"}
			},
			wantError: true,
		},
		{
			name:   "allows complete category list",
			userID: "user12345678901",
			categoryIDs: func(categories []*core.Record) []string {
				return []string{categories[1].Id, categories[0].Id}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupCategoryValidationTestApp(t)
			defer app.Cleanup()

			categories := []*core.Record{
				createTestCategory(t, app, "Biking"),
				createTestCategory(t, app, "Hiking"),
			}

			err := ReorderUserCategoryPreferences(app, tt.userID, tt.categoryIDs(categories))
			if (err != nil) != tt.wantError {
				t.Fatalf("ReorderUserCategoryPreferences() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestReorderUserCategoryPreferencesUpsertsAndRenumbers(t *testing.T) {
	app := setupCategoryValidationTestApp(t)
	defer app.Cleanup()

	biking := createTestCategory(t, app, "Biking")
	hiking := createTestCategory(t, app, "Hiking")
	walking := createTestCategory(t, app, "Walking")
	userID := "user12345678901"

	existing := core.NewRecord(mustFindTestCollection(t, app, "user_category_preferences"))
	existing.Set("user", userID)
	existing.Set("category", biking.Id)
	existing.Set("exclude_search", true)
	existing.Set("priority", 99)
	if err := app.SaveNoValidate(existing); err != nil {
		t.Fatal(err)
	}

	if err := ReorderUserCategoryPreferences(app, userID, []string{walking.Id, biking.Id, hiking.Id}); err != nil {
		t.Fatalf("ReorderUserCategoryPreferences() error = %v", err)
	}

	records, err := app.FindRecordsByFilter(
		"user_category_preferences",
		"user = {:user}",
		"priority",
		0,
		0,
		map[string]any{"user": userID},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 3 {
		t.Fatalf("preference count = %d, want 3", len(records))
	}

	wantPriorityByCategory := map[string]int{
		walking.Id: 1,
		biking.Id:  2,
		hiking.Id:  3,
	}
	for _, record := range records {
		categoryID := record.GetString("category")
		if got := record.GetInt("priority"); got != wantPriorityByCategory[categoryID] {
			t.Fatalf("priority for category %q = %d, want %d", categoryID, got, wantPriorityByCategory[categoryID])
		}
		if categoryID == biking.Id {
			if record.Id != existing.Id {
				t.Fatalf("biking preference id = %q, want existing id %q", record.Id, existing.Id)
			}
			if !record.GetBool("exclude_search") {
				t.Fatal("existing preference exclude_search = false, want true")
			}
		}
	}
}

func setupCategoryValidationTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	categories := core.NewBaseCollection("categories")
	categories.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "icon"},
		&core.JSONField{Name: "translations"},
	)
	if err := app.Save(categories); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	subcategories := core.NewBaseCollection("subcategories")
	subcategories.Fields.Add(
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1, Required: true},
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "icon"},
		&core.TextField{Name: "badge_icon"},
		&core.JSONField{Name: "translations"},
	)
	if err := app.Save(subcategories); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1},
		&core.RelationField{Name: "subcategory", CollectionId: subcategories.Id, MaxSelect: 1},
		&core.TextField{Name: "remote_category"},
		&core.TextField{Name: "remote_subcategory"},
	)
	if err := app.Save(trails); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	preferences := core.NewBaseCollection("user_category_preferences")
	preferences.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: "_pb_users_auth_", MaxSelect: 1, Required: true},
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1, Required: true},
		&core.BoolField{Name: "exclude_search"},
		&core.BoolField{Name: "hide_design"},
		&core.BoolField{Name: "exclude_federated"},
		&core.NumberField{Name: "priority", OnlyInt: true},
	)
	if err := app.Save(preferences); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	return app
}

func mustFindTestCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}

	return collection
}

func createTestCategory(t *testing.T, app core.App, name string) *core.Record {
	t.Helper()

	record := core.NewRecord(mustFindTestCollection(t, app, "categories"))
	record.Set("name", name)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	return record
}

func createTestSubcategory(t *testing.T, app core.App, categoryID string, name string, shortName string) *core.Record {
	t.Helper()

	record := core.NewRecord(mustFindTestCollection(t, app, "subcategories"))
	record.Set("category", categoryID)
	record.Set("name", name)
	record.Set("short_name", shortName)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	return record
}

func createTestTrail(t *testing.T, app core.App, categoryID string, subcategoryID string, remoteCategory string, remoteSubcategory string) *core.Record {
	t.Helper()

	record := core.NewRecord(mustFindTestCollection(t, app, "trails"))
	record.Set("category", categoryID)
	record.Set("subcategory", subcategoryID)
	record.Set("remote_category", remoteCategory)
	record.Set("remote_subcategory", remoteSubcategory)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	return record
}
