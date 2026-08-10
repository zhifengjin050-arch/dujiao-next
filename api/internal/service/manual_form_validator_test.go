package service

import (
	"errors"
	"testing"

	"github.com/dujiao-next/internal/models"
)

func TestValidateAndNormalizeManualFormRequiredMissing(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "name",
				"type":     "text",
				"required": true,
			},
		},
	}
	submission := models.JSON{}
	_, _, err := validateAndNormalizeManualForm(schema, submission)
	if !errors.Is(err, ErrManualFormRequiredMissing) {
		t.Fatalf("expected ErrManualFormRequiredMissing, got %v", err)
	}
}

func TestValidateAndNormalizeManualFormTypeInvalid(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "age",
				"type":     "number",
				"required": true,
			},
		},
	}
	submission := models.JSON{"age": map[string]interface{}{"value": 18}}
	_, _, err := validateAndNormalizeManualForm(schema, submission)
	if !errors.Is(err, ErrManualFormTypeInvalid) {
		t.Fatalf("expected ErrManualFormTypeInvalid, got %v", err)
	}
}

func TestValidateAndNormalizeManualFormOptionInvalid(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "province",
				"type":     "select",
				"required": true,
				"options":  []interface{}{"GD", "BJ"},
			},
		},
	}
	submission := models.JSON{"province": "SH"}
	_, _, err := validateAndNormalizeManualForm(schema, submission)
	if !errors.Is(err, ErrManualFormOptionInvalid) {
		t.Fatalf("expected ErrManualFormOptionInvalid, got %v", err)
	}
}

func TestValidateAndNormalizeManualFormSuccess(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "name",
				"type":     "text",
				"required": true,
				"max_len":  float64(20),
			},
			map[string]interface{}{
				"key":      "phone",
				"type":     "text",
				"required": true,
				"regex":    "^1[0-9]{10}$",
			},
			map[string]interface{}{
				"key":      "city",
				"type":     "select",
				"required": false,
				"options":  []interface{}{"shanghai", "beijing"},
			},
			map[string]interface{}{
				"key":      "count",
				"type":     "number",
				"required": true,
				"min":      float64(1),
				"max":      float64(99),
			},
		},
	}
	submission := models.JSON{
		"name":  " Alice ",
		"phone": "13800138000",
		"city":  "beijing",
		"count": float64(2),
	}

	normalizedSchema, normalizedSubmission, err := validateAndNormalizeManualForm(schema, submission)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := normalizedSchema["fields"]; !ok {
		t.Fatalf("normalized schema missing fields")
	}
	if normalizedSubmission["name"] != "Alice" {
		t.Fatalf("expected trimmed name Alice, got %v", normalizedSubmission["name"])
	}
	if normalizedSubmission["phone"] != "13800138000" {
		t.Fatalf("unexpected phone value: %v", normalizedSubmission["phone"])
	}
	if normalizedSubmission["city"] != "beijing" {
		t.Fatalf("unexpected city value: %v", normalizedSubmission["city"])
	}
	if normalizedSubmission["count"] != float64(2) {
		t.Fatalf("unexpected count value: %v", normalizedSubmission["count"])
	}
}

func TestParseManualFormSchemaInvalidRegex(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "phone",
				"type":     "text",
				"required": true,
				"regex":    "[",
			},
		},
	}
	_, _, err := parseManualFormSchema(schema)
	if !errors.Is(err, ErrManualFormSchemaInvalid) {
		t.Fatalf("expected ErrManualFormSchemaInvalid, got %v", err)
	}
}

func TestValidateAndNormalizeManualFormEmailPhoneAndCheckbox(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "contact_phone",
				"type":     "phone",
				"required": true,
			},
			map[string]interface{}{
				"key":      "contact_email",
				"type":     "email",
				"required": true,
			},
			map[string]interface{}{
				"key":      "tags",
				"type":     "checkbox",
				"required": false,
				"options":  []interface{}{"A", "B", "C"},
			},
		},
	}
	submission := models.JSON{
		"contact_phone": "+86 13800138000",
		"contact_email": "test@example.com",
		"tags":          []interface{}{"B", "A", "A"},
	}

	_, normalizedSubmission, err := validateAndNormalizeManualForm(schema, submission)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	tags, ok := normalizedSubmission["tags"].([]string)
	if !ok {
		t.Fatalf("expected []string tags, got %T", normalizedSubmission["tags"])
	}
	if len(tags) != 2 || tags[0] != "A" || tags[1] != "B" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestValidateAndNormalizeManualFormSanitizeText(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "memo",
				"type":     "textarea",
				"required": true,
			},
		},
	}
	submission := models.JSON{
		"memo": "<script>alert(1)</script>",
	}

	_, normalizedSubmission, err := validateAndNormalizeManualForm(schema, submission)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalizedSubmission["memo"] != "&lt;script&gt;alert(1)&lt;/script&gt;" {
		t.Fatalf("unexpected sanitized memo: %v", normalizedSubmission["memo"])
	}
}

func TestParseManualFormSchemaKeepI18nMeta(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":  "receiver_name",
				"type": "text",
				"label": map[string]interface{}{
					"zh-CN": "???",
					"en-US": "Receiver",
				},
				"placeholder": map[string]interface{}{
					"zh-CN": "??????",
				},
				"required": true,
			},
		},
	}

	normalizedSchema, _, err := validateAndNormalizeManualForm(schema, models.JSON{"receiver_name": "Alice"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	fields, ok := normalizedSchema["fields"].([]models.JSON)
	if !ok || len(fields) != 1 {
		t.Fatalf("unexpected normalized fields: %#v", normalizedSchema["fields"])
	}
	label, ok := fields[0]["label"].(models.JSON)
	if !ok || label["zh-CN"] != "???" {
		t.Fatalf("expected zh-CN label kept, got %#v", fields[0]["label"])
	}
}

func TestParseManualFormSchemaInvalidKeyFormat(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "Receiver-Name",
				"type":     "text",
				"required": true,
			},
		},
	}

	_, _, err := parseManualFormSchema(schema)
	if !errors.Is(err, ErrManualFormSchemaInvalid) {
		t.Fatalf("expected ErrManualFormSchemaInvalid, got %v", err)
	}
}

func TestValidateAndNormalizeManualFormRegexLiteralPhone(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "contact_phone",
				"type":     "phone",
				"required": true,
				"regex":    "/^1[0-9]{10}$/",
			},
		},
	}
	submission := models.JSON{
		"contact_phone": "13277745648",
	}

	_, normalizedSubmission, err := validateAndNormalizeManualForm(schema, submission)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalizedSubmission["contact_phone"] != "13277745648" {
		t.Fatalf("unexpected phone value: %v", normalizedSubmission["contact_phone"])
	}
}

func TestValidateAndNormalizeManualFormRegexLiteralIgnoreCase(t *testing.T) {
	schema := models.JSON{
		"fields": []interface{}{
			map[string]interface{}{
				"key":      "code",
				"type":     "text",
				"required": true,
				"regex":    "/^abc$/i",
			},
		},
	}
	submission := models.JSON{
		"code": "ABC",
	}

	_, normalizedSubmission, err := validateAndNormalizeManualForm(schema, submission)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalizedSubmission["code"] != "ABC" {
		t.Fatalf("unexpected code value: %v", normalizedSubmission["code"])
	}
}
