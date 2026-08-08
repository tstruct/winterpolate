package winterpolate

import (
	"runtime"
	"strings"
)

// Env provides access to variables by name.
//
// The source of the variables is intentionally left to the caller.
// It may be the process environment, application configuration,
// runtime context, or any other source.
type Env interface {
	Get(key string) (string, bool)
}

// NewSliceEnv creates an Env from environment variables in the form "key=value".
//
// This can be used directly with os.Environ().
func NewSliceEnv(env []string) Env {
	envMap := mapEnv{}

	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envMap[normalizeKeyName(parts[0])] = parts[1]
	}

	return envMap
}

// NewMapEnv creates an Env from a map of variables.
func NewMapEnv(env map[string]string) Env {
	envMap := mapEnv{}

	for key, value := range env {
		envMap[normalizeKeyName(key)] = value
	}

	return envMap
}

type mapEnv map[string]string

func (m mapEnv) Get(key string) (string, bool) {
	if m == nil {
		return "", false
	}

	value, ok := m[normalizeKeyName(key)]
	return value, ok
}

// Windows environment variable names are case-insensitive.
func normalizeKeyName(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}

	return key
}
