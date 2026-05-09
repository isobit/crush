package config

import (
	"log/slog"
	"os"
	"reflect"
	"strings"
)

// applyEnvOverrides applies CRUSH_<UPPER_SNAKE> environment variable
// overrides to the Options struct. The env var name maps to dotted json-tag
// paths with underscores (e.g. CRUSH_TUI_COMPACT_MODE -> tui.compact_mode,
// CRUSH_DEBUG -> debug). Flat fields retain backward compatibility.
func applyEnvOverrides(opts *Options) {
	if opts == nil {
		return
	}

	// Collect all settable paths from the struct tree.
	paths := collectPaths(reflect.TypeOf(*opts), "")
	for _, path := range paths {
		envKey := "CRUSH_" + strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
		val, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}
		if err := SetFieldByPath(opts, path, val); err != nil {
			slog.Warn("Invalid value for env var", "key", envKey, "value", val, "error", err)
		}
	}
}

// collectPaths recursively collects all settable json-tag paths from a
// struct type. It descends into struct and *struct fields.
func collectPaths(t reflect.Type, prefix string) []string {
	var paths []string
	for i := range t.NumField() {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name, _, _ := strings.Cut(jsonTag, ",")
		if name == "" {
			continue
		}

		fullPath := name
		if prefix != "" {
			fullPath = prefix + "." + name
		}

		ft := field.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Struct && ft != reflect.TypeOf(Attribution{}) {
			// Descend into nested structs (except opaque ones like
			// Attribution which have complex sub-fields we don't want
			// to expose as flat env vars).
			paths = append(paths, collectPaths(ft, fullPath)...)
		} else {
			paths = append(paths, fullPath)
		}
	}
	return paths
}
