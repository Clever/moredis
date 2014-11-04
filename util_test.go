package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type payloadOrEnvTestSpec struct {
	name     string
	env      map[string]string
	payload  map[string]interface{}
	val      string
	fallback string
	expected string
}

var payloadOrEnvTests = []payloadOrEnvTestSpec{
	{
		name:     "val not in payload or env",
		env:      map[string]string{},
		payload:  map[string]interface{}{},
		val:      "test",
		fallback: "expected",
		expected: "expected",
	},
	{
		name:     "val in env only (uppercase)",
		env:      map[string]string{"TEST": "expected"},
		payload:  map[string]interface{}{},
		val:      "test",
		fallback: "you dun goofed",
		expected: "expected",
	},
	{
		name:     "val in env and payload",
		env:      map[string]string{"TEST": "don't return me"},
		payload:  map[string]interface{}{"test": "expected"},
		val:      "test",
		fallback: "nope",
		expected: "expected",
	},
	{
		name:     "val in payload only",
		env:      map[string]string{},
		payload:  map[string]interface{}{"test": "expected"},
		val:      "test",
		fallback: "nope",
		expected: "expected",
	},
}

func TestPayloadOrEnv(t *testing.T) {
	var err error
	for _, testCase := range payloadOrEnvTests {
		os.Clearenv()
		for key, val := range testCase.env {
			err = os.Setenv(key, val)
			assert.Nil(t, err)
		}
		actual := PayloadOrEnv(testCase.payload, testCase.val, testCase.fallback)
		assert.Equal(t, testCase.expected, actual, "payloadOrEnv [%s]", testCase.name)
	}
}
