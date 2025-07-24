package json

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrUnexpectedValueTypeError = errors.New("unexpected value type encountered")
	ErrUnexpectedTokenError     = errors.New("unexpected json token encountered")
)

type Unmarshaler interface {
	Unmarshal([]byte) error
}

func Unmarshal(data []byte, value any) error {
	rv := reflect.ValueOf(value)

	if rv.IsNil() {
		return errors.New("gravel: Unmarshal() with null value is not supported")
	}

	if rv.Kind() != reflect.Pointer {
		return errors.New("gravel: Unmarshal() requires pointer to value")
	}

	scanner := NewScanner(data)
	return unmarshal(rv, &scanner)
}

func unmarshal(rv reflect.Value, scanner *Scanner) error {
	switch rv.Kind() {
	case reflect.Bool:
		{
			token, err := scanner.Next()
			if err != nil {
				return err
			}

			switch token.Type {
			case TOKEN_TYPE_FALSE:
				{
					rv.SetBool(false)
					return nil
				}
			case TOKEN_TYPE_TRUE:
				{
					rv.SetBool(true)
					return nil
				}
			default:
				{
					return ErrUnexpectedTokenError
				}
			}

		}
	case reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uint:
		{
			for {
				token, err := scanner.Next()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_PARTIAL_NUMBER {
					continue
				}

				if token.Type != TOKEN_TYPE_NUMBER {
					return ErrUnexpectedTokenError
				}

				v, err := strconv.ParseUint(string(token.Value), 10, 64)
				if err != nil {
					return err
				}

				rv.SetUint(v)
				return nil
			}
		}
	case reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Int:
		{
			for {
				token, err := scanner.Next()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_PARTIAL_NUMBER {
					continue
				}

				if token.Type != TOKEN_TYPE_NUMBER {
					return ErrUnexpectedTokenError
				}

				v, err := strconv.ParseInt(string(token.Value), 10, 64)
				if err != nil {
					return err
				}

				rv.SetInt(v)
				return nil
			}
		}
	case reflect.Float32,
		reflect.Float64:
		{
			for {
				token, err := scanner.Next()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_PARTIAL_NUMBER {
					continue
				}

				if token.Type != TOKEN_TYPE_NUMBER {
					return ErrUnexpectedTokenError
				}

				v, err := strconv.ParseFloat(string(token.Value), 64)
				if err != nil {
					return err
				}

				rv.SetFloat(v)
				return nil
			}

		}
	case reflect.String:
		{
			for {
				token, err := scanner.Next()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_PARTIAL_STRING {
					continue
				}

				if token.Type != TOKEN_TYPE_STRING {
					return ErrUnexpectedTokenError
				}

				rv.SetString(string(token.Value))
				return nil
			}

		}
	case reflect.Pointer:
		{
			// Check if we need to create a new value for nil pointer
			if rv.IsNil() {
				// Create new instance of the pointed-to type
				newVal := reflect.New(rv.Type().Elem())
				rv.Set(newVal)
			}
			return unmarshal(rv.Elem(), scanner)
		}
	case reflect.Slice:
		{
			// u, rv := indirect(rv)
			// if u != nil {
			// 	return unmarshalByInterface(u, scanner)
			// }

			token, err := scanner.Next()
			if err != nil {
				return err
			}

			if token.Type != TOKEN_TYPE_ARRAY_BEGIN {
				return ErrUnexpectedTokenError
			}

			// Capped at max items
			for i := range 1000 {
				token, err := scanner.Peek()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_ARRAY_END {
					_, err := scanner.Next()
					return err
				}

				// Expand slice
				if i >= rv.Cap() {
					rv.Grow(1)
				}
				if i >= rv.Len() {
					rv.SetLen(i + 1)
				}

				if i < rv.Len() {
					// Decode into element.
					if err := unmarshal(rv.Index(i), scanner); err != nil {
						return errors.Join(fmt.Errorf("%s [%d]", rv.String(), i), err)
					}
				} else {
					// Ran out of fixed array: skip.
					if err := unmarshal(reflect.Value{}, scanner); err != nil {
						return err
					}
				}
			}

			return nil
		}
	case reflect.Struct:
		{
			u, rv := indirect(rv)
			if u != nil {
				v, err := scanner.ExtractValue()
				if err != nil {
					return err
				}

				return u.Unmarshal(v)
			}

			token, err := scanner.Next()
			if err != nil {
				return err
			}

			if token.Type != TOKEN_TYPE_OBJECT_BEGIN {
				return ErrUnexpectedTokenError
			}

			rt := rv.Type()
			numFields := rv.NumField()
			fieldsSeen := make([]bool, numFields)

		fieldsLoop:
			for {
				token, err := scanner.Next()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_OBJECT_END {
					break
				}

				if token.Type != TOKEN_TYPE_STRING {
					return ErrUnexpectedTokenError
				}

				for i := range numFields {
					field := rt.Field(i)
					tag := field.Tag.Get("json")
					if tag == "" {
						return fmt.Errorf("gravel: Marshal missing 'json' tag for field %s in struct %s", field.Type, rv.Type().Name())
					}

					// Parse JSON tag - extract field name before comma
					fieldName := tag
					if commaIndex := strings.Index(tag, ","); commaIndex != -1 {
						fieldName = tag[:commaIndex]
					}

					if fieldName != string(token.Value) {
						continue
					}

					if fieldsSeen[i] {
						return fmt.Errorf("duplicate field in data %s", tag)
					}

					if err := unmarshal(rv.Field(i), scanner); err != nil {
						return err
					}

					fieldsSeen[i] = true
					continue fieldsLoop
				}

				return fmt.Errorf("unknown field %s", token.Value)
			}

			for i, seen := range fieldsSeen {
				if seen {
					continue
				}

				fieldName := rv.Field(i).Type().Name()
				if fieldName[:6] == "Option" {
					continue
				}

				return errors.New("field not set")
			}

			return nil
		}
	case reflect.Interface:
		{
			// Handle empty interface{} by detecting JSON type and creating appropriate value
			if rv.Type() == reflect.TypeOf((*interface{})(nil)).Elem() {
				token, err := scanner.Peek()
				if err != nil {
					return err
				}

				switch token.Type {
				case TOKEN_TYPE_STRING:
					var s string
					stringValue := reflect.ValueOf(&s).Elem()
					if err := unmarshal(stringValue, scanner); err != nil {
						return err
					}
					rv.Set(reflect.ValueOf(s))
					return nil
				case TOKEN_TYPE_NUMBER:
					var f float64
					floatValue := reflect.ValueOf(&f).Elem()
					if err := unmarshal(floatValue, scanner); err != nil {
						return err
					}
					rv.Set(reflect.ValueOf(f))
					return nil
				case TOKEN_TYPE_TRUE, TOKEN_TYPE_FALSE:
					var b bool
					boolValue := reflect.ValueOf(&b).Elem()
					if err := unmarshal(boolValue, scanner); err != nil {
						return err
					}
					rv.Set(reflect.ValueOf(b))
					return nil
				case TOKEN_TYPE_ARRAY_BEGIN:
					var arr []interface{}
					arrValue := reflect.ValueOf(&arr).Elem()
					if err := unmarshal(arrValue, scanner); err != nil {
						return err
					}
					rv.Set(reflect.ValueOf(arr))
					return nil
				case TOKEN_TYPE_OBJECT_BEGIN:
					var obj map[string]interface{}
					objValue := reflect.ValueOf(&obj).Elem()
					if err := unmarshal(objValue, scanner); err != nil {
						return err
					}
					rv.Set(reflect.ValueOf(obj))
					return nil
				}
			}
			return unmarshal(rv.Elem(), scanner)
		}
	case reflect.Map:
		{
			token, err := scanner.Next()
			if err != nil {
				return err
			}

			if token.Type != TOKEN_TYPE_OBJECT_BEGIN {
				return ErrUnexpectedTokenError
			}

			// Initialize map if nil
			if rv.IsNil() {
				rv.Set(reflect.MakeMap(rv.Type()))
			}

			keyType := rv.Type().Key()
			valueType := rv.Type().Elem()

			for {
				token, err := scanner.Next()
				if err != nil {
					return err
				}

				if token.Type == TOKEN_TYPE_OBJECT_END {
					break
				}

				if token.Type != TOKEN_TYPE_STRING {
					return ErrUnexpectedTokenError
				}

				// Create key (only string keys supported for now)
				if keyType.Kind() != reflect.String {
					return fmt.Errorf("unsupported map key type %s", keyType.Kind())
				}
				keyValue := reflect.ValueOf(string(token.Value))

				// Create value and unmarshal into it
				valueValue := reflect.New(valueType).Elem()
				if err := unmarshal(valueValue, scanner); err != nil {
					return err
				}

				// Set the key-value pair in the map
				rv.SetMapIndex(keyValue, valueValue)
			}

			return nil
		}
	default:
		{
			return fmt.Errorf("gravel: unmarshal with unsupported type %s", rv.Kind().String())
		}
	}
}

func indirect(v reflect.Value) (Unmarshaler, reflect.Value) {
	decodingNull := false
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Pointer && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Pointer) {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Pointer {
			break
		}

		if decodingNull && v.CanSet() {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v any
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem().Equal(v) {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 && v.CanInterface() {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, reflect.Value{}
			}
			// if !decodingNull {
			// 	if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
			// 		return nil, u, reflect.Value{}
			// 	}
			// }
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}
	return nil, v
}
