package moredis

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type flagEnvOrDefaultTestSpec struct {
	name       string
	env        map[string]string
	envVar     string
	flagVal    string
	defaultVal string
	expected   string
}

var payloadOrEnvTests = []flagEnvOrDefaultTestSpec{
	{
		name:       "val not in flags or env",
		env:        map[string]string{},
		envVar:     "TEST",
		flagVal:    "",
		defaultVal: "expected",
		expected:   "expected",
	},
	{
		name:       "val in env only",
		env:        map[string]string{"TEST": "expected"},
		envVar:     "TEST",
		flagVal:    "",
		defaultVal: "you dun goofed",
		expected:   "expected",
	},
	{
		name:       "val in env and flags",
		env:        map[string]string{"TEST": "don't return me"},
		envVar:     "TEST",
		flagVal:    "expected",
		defaultVal: "nope",
		expected:   "expected",
	},
	{
		name:       "val in flags only",
		env:        map[string]string{},
		envVar:     "TEST",
		flagVal:    "expected",
		defaultVal: "nope",
		expected:   "expected",
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
		actual := FlagEnvOrDefault(testCase.flagVal, testCase.envVar, testCase.defaultVal)
		assert.Equal(t, testCase.expected, actual, "payloadOrEnv [%s]", testCase.name)
	}
}
