package riqu

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

func bindValues(target any, tagName string, values TValues) error {
	targetValue := reflect.ValueOf(target)

	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return fmt.Errorf("riqu: binding target must be a non-nil pointer to a struct")
	}
	targetValue = targetValue.Elem()

	if targetValue.Kind() != reflect.Struct {
		return fmt.Errorf("riqu: binding target must point to a struct")
	}
	targetType := targetValue.Type()

	for i := 0; i < targetValue.NumField(); i++ {
		field := targetValue.Field(i)
		structField := targetType.Field(i)

		if !field.CanSet() {
			continue
		}
		name := structField.Tag.Get(tagName)

		if name == "-" {
			continue
		}
		if name == "" {
			name = structField.Name
		}
		if tagName == "header" {
			name = http.CanonicalHeaderKey(name)
		}
		fieldValues := values[name]

		if len(fieldValues) == 0 {
			continue
		}
		if err := setBoundValue(field, fieldValues); err != nil {
			return fmt.Errorf("riqu: bind %s field %s: %w", tagName, structField.Name, err)
		}
	}
	return nil
}
func setBoundValue(field reflect.Value, values []string) error {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setBoundValue(field.Elem(), values)
	}
	if field.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(field.Type(), len(values), len(values))

		for i, value := range values {
			if err := setBoundScalar(slice.Index(i), value); err != nil {
				return err
			}
		}
		field.Set(slice)
		return nil
	}
	return setBoundScalar(field, values[0])
}
func setBoundScalar(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)

		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, field.Type().Bits())

		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, field.Type().Bits())

		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	default:
		return fmt.Errorf("riqu: unsupported target type %s", field.Type())
	}
	return nil
}
