package main

import (
	"os"
	"strings"
)

// PayloadOrEnv looks for a value in the payload.  If it's not there, it will look for the value in
// the processes environment (with the value uppercased).  If that fails, it will return the fallback value.
func PayloadOrEnv(payload map[string]interface{}, val string, fallback string) string {
	if fromPayload, ok := payload[val]; ok {
		return fromPayload.(string)
	}
	if fromEnvUpper := os.Getenv(strings.ToUpper(val)); fromEnvUpper != "" {
		return fromEnvUpper
	}
	return fallback
}
