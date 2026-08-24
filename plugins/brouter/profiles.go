package main

import (
	"embed"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/open-wanderer/wanderer/plugins/brouter/core"
)

const (
	errUnsupportedProfile  = "unsupported_profile"
	brouterProfileMaxBytes = 64 * 1024
)

//go:embed profiles/*.brf
var brouterProfilePresets embed.FS

type codedError struct {
	code              string
	message           string
	retryAfterSeconds *int
}

func (e codedError) Error() string {
	return e.message
}

func brouterTemplateKeyForProfile(profileKey string) string {
	switch profileKey {
	case "hiking-mountain":
		return core.TemplateHike
	case "trekking":
		return core.TemplateTrekking
	case "fastbike":
		return core.TemplateFastbike
	case "gravel":
		return core.TemplateGravel
	case "mtb":
		return core.TemplateMTB
	case "car-vario":
		return core.TemplateCar
	default:
		return ""
	}
}

func validateBRouterProfileContent(content []byte) error {
	if len(content) == 0 || len(content) > brouterProfileMaxBytes {
		return codedError{code: "invalid_request", message: "BRouter profile content is empty or too large"}
	}
	if !utf8.Valid(content) || strings.ContainsRune(string(content), '\x00') {
		return codedError{code: "invalid_request", message: "BRouter profile content must be UTF-8 text"}
	}
	for i, line := range strings.Split(string(content), "\n") {
		if len(line) > 4096 {
			return codedError{code: "invalid_request", message: fmt.Sprintf("BRouter profile line %d is too long", i+1)}
		}
	}
	return nil
}

type brouterTemplateParameter struct {
	Key       string
	Min       float64
	Max       float64
	Default   any
	ValueType string
}

type brouterProfileTemplate struct {
	Base       string
	Parameters []brouterTemplateParameter
}

func brouterTemplateDefinitions() map[string]brouterProfileTemplate {
	return map[string]brouterProfileTemplate{
		core.TemplateHike: {
			Base: "hiking-mountain.brf",
			Parameters: []brouterTemplateParameter{
				{Key: "consider_elevation", ValueType: "boolean", Default: false},
				{Key: "uphillcostvalue", ValueType: "number", Min: 0, Max: 30, Default: 7.0},
				{Key: "downhillcostvalue", ValueType: "number", Min: 0, Max: 30, Default: 7.0},
				{Key: "SAC_scale_limit", ValueType: "number", Min: 0, Max: 6, Default: 3.0},
				{Key: "SAC_scale_preferred", ValueType: "number", Min: 0, Max: 6, Default: 1.0},
				{Key: "hiking_routes_preference", ValueType: "number", Min: 0, Max: 2, Default: 0.2},
				{Key: "path_preference", ValueType: "number", Min: 0, Max: 20, Default: 0.0},
				{Key: "allow_steps", ValueType: "boolean", Default: true},
				{Key: "iswet", ValueType: "boolean", Default: false},
			},
		},
		core.TemplateTrekking: {
			Base: "trekking.brf",
			Parameters: []brouterTemplateParameter{
				{Key: "avoid_unsafe", ValueType: "boolean", Default: false},
				{Key: "consider_traffic", ValueType: "boolean", Default: false},
				{Key: "consider_elevation", ValueType: "boolean", Default: true},
				{Key: "uphillcost", ValueType: "number", Min: 0, Max: 120, Default: 0.0},
				{Key: "bikerPower", ValueType: "number", Min: 50, Max: 300, Default: 100.0},
				{Key: "unpavedPenalty", ValueType: "number", Min: 0, Max: 5, Default: 1.0},
				{Key: "allow_steps", ValueType: "boolean", Default: true},
				{Key: "stick_to_cycleroutes", ValueType: "boolean", Default: false},
			},
		},
		core.TemplateFastbike: {
			Base: "fastbike.brf",
			Parameters: []brouterTemplateParameter{
				{Key: "consider_traffic", ValueType: "number", Min: 0, Max: 1, Default: 0.1},
				{Key: "consider_elevation", ValueType: "boolean", Default: true},
				{Key: "uphillcost", ValueType: "number", Min: 0, Max: 120, Default: 0.0},
				{Key: "bikerPower", ValueType: "number", Min: 50, Max: 300, Default: 100.0},
				{Key: "unpavedPenalty", ValueType: "number", Min: 0, Max: 5, Default: 1.0},
				{Key: "allow_motorways", ValueType: "boolean", Default: false},
			},
		},
		core.TemplateGravel: {
			Base: "gravel.brf",
			Parameters: []brouterTemplateParameter{
				{Key: "consider_elevation", ValueType: "boolean", Default: false},
				{Key: "avoid_steep_inclines", ValueType: "boolean", Default: false},
				{Key: "prefer_unpaved_paths", ValueType: "boolean", Default: false},
				{Key: "assume_wet_conditions", ValueType: "boolean", Default: false},
				{Key: "consider_traffic_estimate", ValueType: "boolean", Default: false},
				{Key: "bikerPower", ValueType: "number", Min: 50, Max: 300, Default: 150.0},
				{Key: "prefer_cycle_routes", ValueType: "boolean", Default: false},
				{Key: "prefer_forests", ValueType: "boolean", Default: false},
			},
		},
		core.TemplateMTB: {
			Base: "mtb.brf",
			Parameters: []brouterTemplateParameter{
				{Key: "hills", ValueType: "number", Min: 0, Max: 5, Default: 0.0},
				{Key: "avoid_unsafe", ValueType: "boolean", Default: false},
				{Key: "MTB_factor", ValueType: "number", Min: -3, Max: 3, Default: 0.0},
				{Key: "smallpaved_factor", ValueType: "number", Min: -3, Max: 3, Default: -0.5},
				{Key: "path_preference", ValueType: "number", Min: 0, Max: 40, Default: 20.0},
			},
		},
		core.TemplateCar: {
			Base: "car-vario.brf",
			Parameters: []brouterTemplateParameter{
				{Key: "vmax", ValueType: "number", Min: core.MinCarSpeed, Max: 300, Default: 90.0},
				{Key: "avoid_unpaved", ValueType: "boolean", Default: false},
				{Key: "avoid_toll", ValueType: "boolean", Default: false},
				{Key: "avoid_motorways", ValueType: "boolean", Default: false},
			},
		},
	}
}

// brouterTemplateValuesEqual compares parameter values across the JSON and
// profile representations, where the same value may arrive as a bool or a number.
func brouterTemplateValuesEqual(left any, right any) bool {
	if leftBool, ok := left.(bool); ok {
		if rightBool, ok := right.(bool); ok {
			return leftBool == rightBool
		}
		rightNumber, ok := core.ProfileNumericValue(right)
		return ok && leftBool == (rightNumber != 0)
	}
	leftNumber, leftOK := core.ProfileNumericValue(left)
	rightNumber, rightOK := core.ProfileNumericValue(right)
	if leftOK && rightOK {
		return leftNumber == rightNumber
	}
	if rightBool, ok := right.(bool); ok && leftOK {
		return (leftNumber != 0) == rightBool
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func renderBRouterGeneratedProfile(metadata map[string]any, nativeConfig map[string]any) (string, error) {
	templateKey := stringValue(metadata["templateKey"])
	template, ok := brouterTemplateDefinitions()[templateKey]
	if !ok {
		return "", codedError{code: errUnsupportedProfile, message: "unknown BRouter profile template"}
	}
	values := core.MapValue(nativeConfig["parameters"])
	contentBytes, err := brouterProfilePresets.ReadFile("profiles/" + template.Base)
	if err != nil {
		return "", err
	}
	content := string(contentBytes)
	for _, parameter := range template.Parameters {
		value, ok := values[parameter.Key]
		if !ok || value == nil {
			// Leave the assignment alone: the base file's own value is the
			// upstream default, and rewriting it with ours would silently turn
			// an unmodified profile into a different one.
			continue
		}
		renderedValue, err := renderBRouterTemplateParameterValue(templateKey, parameter, value)
		if err != nil {
			return "", err
		}
		content, err = replaceBRouterAssign(content, parameter.Key, renderedValue)
		if err != nil {
			return "", err
		}
	}
	return content, nil
}

func renderBRouterCustomProfile(content string, nativeConfig map[string]any) (string, error) {
	values := core.MapValue(nativeConfig["parameters"])
	for _, parameter := range parseBRouterProfileParameters(content) {
		value, ok := values[parameter.Key]
		if !ok {
			continue
		}
		renderedValue, err := renderBRouterProfileParameterValue(parameter, value)
		if err != nil {
			return "", err
		}
		content, err = replaceBRouterAssign(content, parameter.Key, renderedValue)
		if err != nil {
			return "", err
		}
	}
	return content, nil
}

func renderBRouterProfileParameterValue(parameter brouterProfileParameter, raw any) (string, error) {
	if parameter.ValueType == "boolean" {
		if value, ok := raw.(bool); ok {
			return formatBRouterValue(value), nil
		}
		// A knob one base exposes as a scale is a flag in another, so a scale
		// value crossing the midpoint counts as enabling the flag.
		if value, ok := core.ProfileNumericValue(raw); ok {
			return formatBRouterValue(value >= 0.5), nil
		}
		return "", codedError{code: "invalid_request", message: "BRouter profile parameter must be boolean"}
	}
	value, ok := core.ProfileNumericValue(raw)
	if !ok {
		// A base may expose a knob as a number that another exposes as a flag.
		if flag, isBool := raw.(bool); isBool {
			value, ok = 0, true
			if flag {
				value = 1
			}
		}
	}
	if !ok {
		return "", codedError{code: "invalid_request", message: "BRouter profile parameter must be numeric"}
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", codedError{code: "invalid_request", message: "BRouter profile parameter must be a finite number"}
	}
	if parameter.Bounded && (value < parameter.Min || value > parameter.Max) {
		return "", codedError{
			code:    "invalid_request",
			message: fmt.Sprintf("BRouter profile parameter %s is outside allowed bounds", parameter.Key),
		}
	}
	return formatBRouterValue(value), nil
}

func renderBRouterTemplateParameterValue(templateKey string, parameter brouterTemplateParameter, raw any) (string, error) {
	if parameter.ValueType == "boolean" {
		value, ok := raw.(bool)
		if !ok {
			return "", codedError{code: "invalid_request", message: "BRouter template parameter must be boolean"}
		}
		return formatBRouterValue(value), nil
	}
	value, ok := core.ProfileNumericValue(raw)
	if !ok {
		return "", codedError{code: "invalid_request", message: "BRouter template parameter must be numeric"}
	}
	if value < parameter.Min || value > parameter.Max {
		return "", codedError{
			code:    "invalid_request",
			message: fmt.Sprintf("BRouter template %s parameter %s is outside allowed bounds", templateKey, parameter.Key),
		}
	}
	return formatBRouterValue(value), nil
}

// replaceBRouterAssign rewrites an annotated assignment in place. Appending a
// new assignment instead would land in whatever context the file ends with,
// where BRouter either ignores it or fails, so a missing annotation is an error.

// replaceBRouterAssign rewrites an annotated assignment in place. Appending a
// new assignment instead would land in whatever context the file ends with,
// where BRouter either ignores it or fails, so a missing annotation is an error.
func replaceBRouterAssign(content string, key string, value string) (string, error) {
	pattern := regexp.MustCompile(`(?m)^(\s*assign\s+` + regexp.QuoteMeta(key) + `\s*=?\s*)([^\s#]+)(\s*#\s*%` + regexp.QuoteMeta(key) + `%.*)$`)
	if !pattern.MatchString(content) {
		return "", codedError{
			code:    errUnsupportedProfile,
			message: fmt.Sprintf("BRouter profile does not expose parameter %s", key),
		}
	}
	return pattern.ReplaceAllString(content, "${1}"+value+"${3}"), nil
}

func formatBRouterValue(value any) string {
	switch value := value.(type) {
	case bool:
		if value {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", value)
	case int64:
		return fmt.Sprintf("%d", value)
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
	case float32:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
	case string:
		if !isSafeBRouterLiteral(value) {
			return ""
		}
		return value
	default:
		return fmt.Sprintf("%v", value)
	}
}

func isSafeBRouterLiteral(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return true
}

type brouterProfileParameter struct {
	Key         string
	Label       string
	Description string
	ValueType   string
	// Bounded reports whether the annotation actually declares a range. The
	// BRouter annotation syntax carries none for plain numbers, and inventing
	// one rejects perfectly valid values: Poutnik's factors are negative and a
	// car's mass is in the thousands.
	Bounded bool
	Min     float64
	Max     float64
	Step    float64
	Default any
	Options []map[string]string
}

// brouterProfileParameterTypes maps every annotated parameter of a profile to
// its declared value type, which is what identifies the upstream base.
func brouterProfileParameterTypes(content string) map[string]string {
	parameters := parseBRouterProfileParameters(content)
	types := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		types[parameter.Key] = parameter.ValueType
	}
	return types
}

func parseBRouterProfileParameters(content string) []brouterProfileParameter {
	pattern := regexp.MustCompile(`^\s*assign\s+([A-Za-z_][A-Za-z0-9_]*)\s*=?\s*([^\s#]+)\s*#\s*%([^%]+)%\s*\|\s*([^|]*?)\s*\|\s*(.*?)\s*$`)
	parameters := []brouterProfileParameter{}
	for _, line := range strings.Split(content, "\n") {
		match := pattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		key := strings.TrimSpace(match[1])
		defaultValue := strings.TrimSpace(match[2])
		label := strings.TrimSpace(match[3])
		description := strings.TrimSpace(match[4])
		rawType := strings.TrimSpace(match[5])
		parameter := brouterProfileParameter{
			Key:         key,
			Label:       fallbackString(label, key),
			Description: description,
			ValueType:   "number",
			Step:        1,
			Default:     parseFloat(defaultValue),
		}
		switch {
		case strings.EqualFold(rawType, "boolean"):
			parameter.ValueType = "boolean"
			parameter.Default = strings.EqualFold(defaultValue, "true") || defaultValue == "1"
		case strings.HasPrefix(rawType, "[") && strings.HasSuffix(rawType, "]"):
			options, min, max := parseBRouterEnumOptions(rawType)
			parameter.Options = options
			parameter.Bounded = true
			parameter.Min = min
			parameter.Max = max
		}
		parameters = append(parameters, parameter)
	}
	return parameters
}

func parseBRouterEnumOptions(raw string) ([]map[string]string, float64, float64) {
	body := strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
	options := []map[string]string{}
	min := 0.0
	max := 0.0
	first := true
	for _, part := range strings.Split(body, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		value := strings.TrimSpace(pair[0])
		label := value
		if len(pair) == 2 {
			label = strings.TrimSpace(pair[1])
		}
		number := parseFloat(value)
		if first || number < min {
			min = number
		}
		if first || number > max {
			max = number
		}
		first = false
		options = append(options, map[string]string{"value": value, "label": label})
	}
	return options, min, max
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return parsed
}

func stringValue(value any) string {
	if value, ok := value.(string); ok {
		return value
	}
	return ""
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
