package env

import "os"

func Get(env string, fallback string) string {
	value := os.Getenv(env)
	if value == "" {
		return fallback
	}

	return value
}