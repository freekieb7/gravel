package validation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// type Validator interface {
// 	Validate(request http.Request) (Violations, error)
// }

// type validator struct {
// 	rules map[string]string
// }

type Violations struct {
	Errors map[string][]error
}

func (violations Violations) MarshalJSON() ([]byte, error) {
	errors := make(map[string][]string)
	for fieldName, fieldErrors := range violations.Errors {
		errors[fieldName] = make([]string, len(fieldErrors))
		for index, fieldError := range fieldErrors {
			errors[fieldName][index] = fieldError.Error()
		}
	}

	return json.Marshal(map[string]map[string][]string{
		"errors": errors,
	})
}

func (violations Violations) IsEmpty() bool {
	return len(violations.Errors) == 0
}

func ValidateMap(data map[string]any, rules map[string][]string) Violations {
	var violations Violations
	violations.Errors = make(map[string][]error)

	for attributeName, attributeValue := range data {
		attributeRules, attributeRulesExists := rules[attributeName]
		if !attributeRulesExists {
			violations.Errors[attributeName] = append(violations.Errors[attributeName], fmt.Errorf("validation: no rules found :: %s", attributeName))
			continue
		}

		var errorCollection []error
		for _, attributeRule := range attributeRules {
			if err := validate(attributeRule, attributeName, attributeValue); err != nil {
				errorCollection = append(errorCollection, err)
			}
		}

		if len(errorCollection) != 0 {
			violations.Errors[attributeName] = errorCollection
		}
	}

	return violations
}

func validate(rule string, name string, value any) error {
	switch rule {
	case "required":
		{
			err := fmt.Errorf("%s is required", name)

			switch v := value.(type) {
			case nil:
				{
					return err
				}
			case string:
				{
					if v == "" {
						return err
					}
				}
			case []any:
				{
					if len(v) == 0 {
						return err
					}
				}
			}
		}
	default:
		{
			return fmt.Errorf("invalid validation rule :: %s", rule)
		}
	}

	return nil
}

// Numberic operations
func ValidateInteger(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func ValidateGreaterThen(value string, size int) bool {
	valueAsInt, err := strconv.Atoi(value)
	if err != nil {
		return false
	}

	return valueAsInt > size
}

func ValidateGreaterThenOrEqual(value string, size int) bool {
	valueAsInt, err := strconv.Atoi(value)
	if err != nil {
		return false
	}

	return valueAsInt >= size
}

func ValidateLesserThen(value string, size int) bool {
	valueAsInt, err := strconv.Atoi(value)
	if err != nil {
		return false
	}

	return valueAsInt < size
}

func ValidateLesserThenOrEqual(value string, size int) bool {
	valueAsInt, err := strconv.Atoi(value)
	if err != nil {
		return false
	}

	return valueAsInt <= size
}

// Boolean operations
func ValidateBoolean(value string) bool {
	return ValidateTrue(value) || ValidateFalse(value)
}

func ValidateTrue(value string) bool {
	return value == "1" || value == "true"
}

func ValidateFalse(value string) bool {
	return value == "0" || value == "false"
}

// string operations
func ValidateContains(value string, needle string) bool {
	return strings.Contains(value, needle)
}

// Time operations
func ValidateBefore(value string, format string, timestamp time.Time) bool {
	valueAsTime, err := time.Parse(format, value)
	if err != nil {
		return false
	}

	return timestamp.Before(valueAsTime)
}

func ValidateAfter(value string, format string, timestamp time.Time) bool {
	valueAsTime, err := time.Parse(format, value)
	if err != nil {
		return false
	}

	return timestamp.After(valueAsTime)
}
