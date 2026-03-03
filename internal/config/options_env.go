package config

import (
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// applyEnvOverrides applies CRUSH_<UPPER_SNAKE> environment variable overrides
// to the Options struct. The env var name is derived from each field's JSON tag
// (e.g. json:"hashline_edit" -> CRUSH_HASHLINE_EDIT). Supported field types are
// string, bool, *bool, and []string (comma-separated).
func applyEnvOverrides(opts *Options) {
	if opts == nil {
		return
	}

	v := reflect.ValueOf(opts).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract the JSON field name (before any comma).
		name, _, _ := strings.Cut(jsonTag, ",")
		if name == "" {
			continue
		}

		envKey := "CRUSH_" + strings.ToUpper(name)
		str, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}

		fv := v.Field(i)
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(str)
		case reflect.Bool:
			b, err := strconv.ParseBool(str)
			if err != nil {
				slog.Warn("Invalid boolean value for env var", "key", envKey, "value", str)
				continue
			}
			fv.SetBool(b)
		case reflect.Pointer:
			if fv.Type().Elem().Kind() == reflect.Bool {
				b, err := strconv.ParseBool(str)
				if err != nil {
					slog.Warn("Invalid boolean value for env var", "key", envKey, "value", str)
					continue
				}
				fv.Set(reflect.ValueOf(&b))
			}
		case reflect.Slice:
			if fv.Type().Elem().Kind() == reflect.String {
				parts := strings.Split(str, ",")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				fv.Set(reflect.ValueOf(parts))
			}
		}
	}
}
