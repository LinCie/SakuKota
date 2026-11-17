package places

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// validateAttributes checks the attributes against the category's fields and returns clean JSON.
// Missing required fields and unknown keys are rejected, and blank optional text is dropped.
func validateAttributes(fields []CategoryField, raw json.RawMessage) (json.RawMessage, error) {
	var values map[string]any
	if len(raw) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		// Keep numbers as json.Number so they are stored exactly as sent.
		decoder.UseNumber()
		if err := decoder.Decode(&values); err != nil {
			return nil, fmt.Errorf("%w: must be a JSON object", ErrInvalidAttributes)
		}
	}

	clean := make(map[string]any, len(fields))
	for _, f := range fields {
		value := values[f.Key]
		// Whatever is left in values after this loop is not a field of the category.
		delete(values, f.Key)

		if text, ok := value.(string); ok && f.DataType == FieldTypeText && strings.TrimSpace(text) == "" {
			value = nil
		}
		if value == nil {
			if f.Required {
				return nil, fmt.Errorf("%w: %q is required", ErrInvalidAttributes, f.Key)
			}
			continue
		}

		if err := validateAttribute(f, value); err != nil {
			return nil, err
		}
		clean[f.Key] = value
	}

	for key := range values {
		return nil, fmt.Errorf("%w: unknown field %q", ErrInvalidAttributes, key)
	}

	return json.Marshal(clean)
}

// validateAttribute checks that one value has the right type and format for its field.
// The database allows only these data types, so there is no case for others.
func validateAttribute(f CategoryField, value any) error {
	switch f.DataType {
	case FieldTypeText:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: %q must be text", ErrInvalidAttributes, f.Key)
		}
	case FieldTypeNumber:
		if _, ok := value.(json.Number); !ok {
			return fmt.Errorf("%w: %q must be a number", ErrInvalidAttributes, f.Key)
		}
	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: %q must be a boolean", ErrInvalidAttributes, f.Key)
		}
	case FieldTypeSelect:
		option, ok := value.(string)
		if !ok || !slices.Contains(f.Options, option) {
			return fmt.Errorf("%w: %q must be one of: %s",
				ErrInvalidAttributes, f.Key, strings.Join(f.Options, ", "))
		}
	case FieldTypeRange:
		if !validRange(value) {
			return fmt.Errorf(`%w: %q must be {"min": number, "max": number} with min <= max`,
				ErrInvalidAttributes, f.Key)
		}
	case FieldTypeDate:
		date, ok := value.(string)
		if _, err := time.Parse(time.DateOnly, date); !ok || err != nil {
			return fmt.Errorf("%w: %q must be a date in YYYY-MM-DD format", ErrInvalidAttributes, f.Key)
		}
	}

	return nil
}

// validRange reports whether value is {"min": number, "max": number} with min <= max.
func validRange(value any) bool {
	bounds, ok := value.(map[string]any)
	if !ok || len(bounds) != 2 {
		return false
	}

	// A missing or non-number bound is the empty json.Number, which fails Float64.
	minNumber, _ := bounds["min"].(json.Number)
	maxNumber, _ := bounds["max"].(json.Number)
	low, lowErr := minNumber.Float64()
	high, highErr := maxNumber.Float64()

	return lowErr == nil && highErr == nil && low <= high
}
