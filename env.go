package ssk

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

type Env struct {
	KMSKeyID   string `env:"SAKURACLOUD_KMS_KEY_ID" required:""`
	ServerOnly bool   `env:"SSK_SERVER_ONLY" default:"false"`
	ServerAddr string `env:"SSK_SERVER_ADDR" default:"127.0.0.1:8200"`
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

		envName := field.Tag.Get("env")
		if envName == "" {
			continue
		}

		value := os.Getenv(envName)
		if value == "" {
			value = field.Tag.Get("default")
		}

		_, required := field.Tag.Lookup("required")
		if required && value == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", envName)
		}

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(value)
		case reflect.Bool:
			if value != "" {
				b, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("invalid boolean value for %s: %w", envName, err)
				}
				fieldValue.SetBool(b)
			}
		default:
			return nil, fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
		}
	}

	return env, nil
}
