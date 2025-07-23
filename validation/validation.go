package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type ValidateFunc func(rule string, fieldName string, fieldValue any) error

var rulesMutex sync.RWMutex
var knownRules map[*regexp.Regexp]ValidateFunc = make(map[*regexp.Regexp]ValidateFunc, 0)

func init() {
	// Rule "required"
	RegisterRule(`^required$`, func(rule, fieldName string, fieldValue any) error {
		err := fmt.Errorf("%s is required", fieldName)

		switch v := fieldValue.(type) {
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
		default:
			{
				// Support for more complex types
				v := reflect.ValueOf(fieldValue)
				switch v.Kind() {
				case reflect.Slice, reflect.Array, reflect.Map:
					{
						if v.Len() == 0 {
							return err
						}
					}
				default:
					{
						return fmt.Errorf("unknown type %T", fieldValue)
					}
				}
			}
		}

		return nil
	})

	// Rule "max"
	RegisterRule(`^max:\d+`, func(rule, fieldName string, fieldValue any) error {
		parts := strings.Split(rule, ":")
		rawMaxSize := parts[1]
		maxSize, err := strconv.Atoi(rawMaxSize)
		if err != nil {
			return err
		}

		switch value := fieldValue.(type) {
		case string:
			{
				if len(value) > maxSize {
					return fmt.Errorf("%s exceeds max size of %d", fieldName, maxSize)
				}
			}
		case int:
			{
				if value > maxSize {
					return fmt.Errorf("%s exceeds max size of %d", fieldName, maxSize)
				}
			}
		default:
			{
				v := reflect.ValueOf(fieldValue)
				switch v.Kind() {
				case reflect.Slice, reflect.Array, reflect.Map:
					{
						if v.Len() > maxSize {
							return fmt.Errorf("%s exceeds max size of %d", fieldName, maxSize)
						}
					}
				default:
					{
						return fmt.Errorf("unknown type %T", fieldValue)
					}
				}
			}
		}

		return nil
	})

	// Rule "min"
	RegisterRule(`^min:\d+`, func(rule, fieldName string, fieldValue any) error {
		parts := strings.Split(rule, ":")
		rawMinSize := parts[1]
		minSize, err := strconv.Atoi(rawMinSize)
		if err != nil {
			return err
		}

		switch value := fieldValue.(type) {
		case string:
			{
				if len(value) < minSize {
					return fmt.Errorf("%s subceeds min size of %d", fieldName, minSize)
				}
			}
		case int:
			{
				if value < minSize {
					return fmt.Errorf("%s subceeds min size of %d", fieldName, minSize)
				}
			}
		default:
			{

				v := reflect.ValueOf(fieldValue)
				switch v.Kind() {
				case reflect.Slice, reflect.Array, reflect.Map:
					{
						if v.Len() < minSize {
							return fmt.Errorf("%s subceeds min size of %d", fieldName, minSize)
						}
					}
				default:
					{
						return fmt.Errorf("unknown type %T", fieldValue)
					}
				}
			}
		}

		return nil
	})
}

func RegisterRule(regExpSyntax string, validateFunc ValidateFunc) {
	rulesMutex.Lock()
	defer rulesMutex.Unlock()
	ruleRegexp, err := regexp.Compile(regExpSyntax)
	if err != nil {
		panic(err)
	}

	knownRules[ruleRegexp] = validateFunc
}

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

	for fieldName, fieldValue := range data {
		fieldRules, ruleExistsForField := rules[fieldName]
		if !ruleExistsForField {
			violations.Errors[fieldName] = append(violations.Errors[fieldName], fmt.Errorf("validation: no rules found :: %s", fieldName))
			continue
		}

		var errorCollection []error

	nextRule:
		for _, fieldRule := range fieldRules {
			for knownRuleRegexp, validateRuleFunc := range knownRules {
				if !knownRuleRegexp.MatchString(fieldRule) {
					continue
				}

				if err := validateRuleFunc(fieldRule, fieldName, fieldValue); err != nil {
					errorCollection = append(errorCollection, err)
				}

				continue nextRule
			}

			panic(fmt.Sprintf("validation: invalid rule found :: %s", fieldRule))
		}

		if len(errorCollection) != 0 {
			violations.Errors[fieldName] = errorCollection
		}
	}

	return violations
}
