package main

import "os"

// FlagEnvOrDefault takes in the value returned from flag parsing, and environment variable to check, and a default value.
// If the flag was not set, it tries to retrieve the value from environment variables.  If it is not found there it returns
// the default value.
func FlagEnvOrDefault(flagVal, envVar, defaultValue string) string {
	if flagVal != "" {
		return flagVal
	}
	if fromEnv := os.Getenv(envVar); fromEnv != "" {
		return fromEnv
	}
	return defaultValue
}
