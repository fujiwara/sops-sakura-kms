package ssk

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Env struct {
	// KMSKeyID is read from SAKURA_KMS_KEY_ID. SAKURACLOUD_KMS_KEY_ID is a legacy name kept for backward compatibility.
	KMSKeyID   string `env:"SAKURA_KMS_KEY_ID,SAKURACLOUD_KMS_KEY_ID"`
	ServerOnly bool   `env:"SSK_SERVER_ONLY" default:"false"`
	// ServerAddr is the listen address. If empty, the wrapper listens on an
	// ephemeral port (or DefaultServerAddr in server-only mode).
	ServerAddr string `env:"SSK_SERVER_ADDR"`
	Command    string `env:"SSK_COMMAND" default:"sops"`
}

// LoadEnv loads environment variables into an Env struct based on struct tags.
// It reads the "env" tag for the environment variable name,
// "default" tag for default values, and "required" tag for required fields.
func LoadEnv() (*Env, error) {
	env := &Env{}
	v := reflect.ValueOf(env).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Support comma-separated environment variable names (first match wins)
		envNames := strings.Split(envTag, ",")
		var value string
		for _, envName := range envNames {
			envName = strings.TrimSpace(envName)
			if v := os.Getenv(envName); v != "" {
				value = v
				break
			}
		}
		if value == "" {
			value = field.Tag.Get("default")
		}

		_, required := field.Tag.Lookup("required")
		if required && value == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", strings.Join(envNames, " or "))
		}

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(value)
		case reflect.Bool:
			if value != "" {
				b, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("invalid boolean value for %s: %w", envTag, err)
				}
				fieldValue.SetBool(b)
			}
		default:
			return nil, fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
		}
	}

	return env, nil
}

// listenAddr returns the address for the wrapper to listen on.
// If ServerAddr is empty, it returns an ephemeral port address, or
// DefaultServerAddr in server-only mode.
func (e *Env) listenAddr() string {
	switch {
	case e.ServerAddr != "":
		return e.ServerAddr
	case e.ServerOnly:
		return DefaultServerAddr
	default:
		return ephemeralServerAddr
	}
}
