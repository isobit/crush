package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SetFieldByPath sets a field on a struct using a dotted json-tag path.
// It walks nested structs/pointers (allocating nil pointers as needed)
// and sets the leaf value. Supported leaf types: string, *string, bool,
// *bool, *int, int, []string.
func SetFieldByPath(target any, path string, value string) error {
	segments := strings.Split(path, ".")
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}
	v = v.Elem()

	for i, seg := range segments {
		isLast := i == len(segments)-1

		fv, ok := fieldByJSONTag(v, seg)
		if !ok {
			return fmt.Errorf("unknown field %q in path %q", seg, path)
		}

		if isLast {
			return setLeafValue(fv, value, path)
		}

		// Descend into nested struct or pointer-to-struct.
		switch fv.Kind() {
		case reflect.Pointer:
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			fv = fv.Elem()
			if fv.Kind() != reflect.Struct {
				return fmt.Errorf("path segment %q is not a struct in %q", seg, path)
			}
		case reflect.Struct:
			// ok, continue
		default:
			return fmt.Errorf("path segment %q is not a struct in %q", seg, path)
		}
		v = fv
	}
	return nil
}

// fieldByJSONTag finds a struct field by its json tag name.
func fieldByJSONTag(v reflect.Value, tag string) (reflect.Value, bool) {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name, _, _ := strings.Cut(jsonTag, ",")
		if name == tag {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// setLeafValue assigns a string value to a reflected field, coercing
// types as needed.
func setLeafValue(fv reflect.Value, value, path string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool for %q: %w", path, err)
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int for %q: %w", path, err)
		}
		fv.SetInt(n)
	case reflect.Pointer:
		return setPointerLeaf(fv, value, path)
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(value, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			fv.Set(reflect.ValueOf(parts))
		} else {
			return fmt.Errorf("unsupported slice type for %q", path)
		}
	default:
		return fmt.Errorf("unsupported field type %s for %q", fv.Kind(), path)
	}
	return nil
}

// setPointerLeaf handles *string, *bool, *int pointer fields.
func setPointerLeaf(fv reflect.Value, value, path string) error {
	elem := fv.Type().Elem()
	switch elem.Kind() {
	case reflect.String:
		fv.Set(reflect.ValueOf(&value))
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool for %q: %w", path, err)
		}
		fv.Set(reflect.ValueOf(&b))
	case reflect.Int:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid int for %q: %w", path, err)
		}
		fv.Set(reflect.ValueOf(&n))
	default:
		return fmt.Errorf("unsupported pointer type *%s for %q", elem.Kind(), path)
	}
	return nil
}

// ApplyCLIOverrides applies --set key=value overrides to the Options struct.
// Keys use dotted paths matching json tags (e.g. "tui.compact_mode",
// "debug").
func (s *ConfigStore) ApplyCLIOverrides(overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}
	if s.config.Options == nil {
		s.config.Options = &Options{}
	}
	for key, val := range overrides {
		if err := SetFieldByPath(s.config.Options, key, val); err != nil {
			return fmt.Errorf("--set %s=%s: %w", key, val, err)
		}
	}
	return nil
}

// ParseCLIOverrides parses a slice of "key=value" strings into a map.
// It returns an error if any entry is missing the "=" separator.
func ParseCLIOverrides(args []string) (map[string]string, error) {
	result := make(map[string]string, len(args))
	for _, arg := range args {
		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --set value %q: must be key=value", arg)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid --set value %q: key cannot be empty", arg)
		}
		result[key] = val
	}
	return result, nil
}
