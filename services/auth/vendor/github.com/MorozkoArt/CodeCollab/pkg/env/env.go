package env

import (
	"os"
	"strconv"
	"time"
)

func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func GetIntEnv(key string, defaultVal int) int {
	if value, exists := os.LookupEnv(key); exists {
		val, err := strconv.ParseInt(value, 10, 64)

		if err != nil {
			return defaultVal
		}

		return int(val)
	}

	return defaultVal
}

func GetBoolEnv(key string, defaultVal bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultVal
	}

	parsedVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultVal
	}

	return parsedVal
}

func GetDurationEnv(key string, defaultVal time.Duration) time.Duration {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultVal
	}

	parsedVal, err := time.ParseDuration(value)
	if err != nil {
		return defaultVal
	}
	return parsedVal
}
