package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

type toStringTestSpec struct {
	input    interface{}
	expected string
}

var toStringTests = []toStringTestSpec{
	{bson.ObjectIdHex("ffffffffffffffffffffffff"), "ffffffffffffffffffffffff"},
	{"string", "string"},
	{nil, "<nil>"},
}

func TestToString(t *testing.T) {
	for _, testCase := range toStringTests {
		actual := toString(testCase.input)
		assert.Equal(t, testCase.expected, actual, "toString(%v) failed", testCase.input)
	}
}

type safeToLowerTestSpec struct {
	input    interface{}
	expected string
}

var safeToLowerTests = []safeToLowerTestSpec{
	{"ALL CAPS", "all caps"},
	{nil, ""},
	{"lower", "lower"},
	{[]string{"test"}, ""},
}

func TestSafeToLower(t *testing.T) {
	for _, testCase := range safeToLowerTests {
		actual := safeToLower(testCase.input)
		assert.Equal(t, testCase.expected, actual, "safeToLower(%v) failed", testCase.input)
	}
}

type applyTemplateTestSpec struct {
	name           string
	templateString string
	payload        map[string]interface{}
	expected       string
	expectedError  bool
}

var applyTemplateTests = []applyTemplateTestSpec{
	{
		name:           "empty input",
		templateString: "",
		payload:        map[string]interface{}{},
		expected:       "",
	},
	{
		name:           "simple field sub",
		templateString: "{{.field}}",
		payload:        map[string]interface{}{"field": "value"},
		expected:       "value",
	},
	{
		name:           "field sub and text",
		templateString: "text:{{.field}}",
		payload:        map[string]interface{}{"field": "value"},
		expected:       "text:value",
	},
	{
		name:           "function calling",
		templateString: "{{toLower .field}}",
		payload:        map[string]interface{}{"field": "VALUE"},
		expected:       "value",
	},
	{
		name:           "invalid template string",
		templateString: "{{()}}",
		payload:        map[string]interface{}{},
		expectedError:  true,
	},
	{
		name:           "invalid function in template",
		templateString: "{{nonExistentFunc}}",
		payload:        map[string]interface{}{},
		expectedError:  true,
	},
}

func TestApplyTemplate(t *testing.T) {
	for _, testCase := range applyTemplateTests {
		actual, err := ApplyTemplate(testCase.templateString, testCase.payload)
		if !testCase.expectedError {
			assert.Nil(t, err)
			assert.Equal(t, testCase.expected, actual, "failed applyTemplate test: %s", testCase.name)
		} else {
			assert.Error(t, err, "wanted error, but returned %s", actual)
		}
	}
}

func TestObjectIds(t *testing.T) {
	in := map[string]interface{}{
		"nil": nil,
		"id1": "ffffffffffffffffffffffff",
		"nested": map[string]interface{}{
			"id2": "111111111111111111111111",
			"int": 5,
			"str": "test",
		},
	}

	expected := map[string]interface{}{
		"nil": nil,
		"id1": bson.ObjectIdHex("ffffffffffffffffffffffff"),
		"nested": map[string]interface{}{
			"id2": bson.ObjectIdHex("111111111111111111111111"),
			"int": 5,
			"str": "test",
		},
	}

	setObjectIds(in)

	assert.Equal(t, expected, in)
}

type parseQueryTestSpec struct {
	name          string
	queryString   string
	payload       map[string]interface{}
	expected      map[string]interface{}
	expectedError bool
}

var parseQueryTests = []parseQueryTestSpec{
	{
		name:        "simple query with ObjectId substitution",
		queryString: `{"_id": "{{.id}}"}`,
		payload:     map[string]interface{}{"id": "111111111111111111111111"},
		expected:    map[string]interface{}{"_id": bson.ObjectIdHex("111111111111111111111111")},
	},
	{
		name:        "simple substitution and mongo operator",
		queryString: `{"{{.field}}": {"$exists": true}}`,
		payload:     map[string]interface{}{"field": "somefield"},
		expected:    map[string]interface{}{"somefield": map[string]interface{}{"$exists": true}},
	},
	{
		name:          "invalid json (missing quotes)",
		queryString:   `{id: 5}`,
		payload:       map[string]interface{}{},
		expectedError: true,
	},
	{
		name:          "invalid template",
		queryString:   `{"field": {{()}}}`,
		payload:       map[string]interface{}{},
		expectedError: true,
	},
}

func TestParseQuery(t *testing.T) {
	for _, testCase := range parseQueryTests {
		actual, err := ParseQuery(testCase.queryString,
			testCase.payload)
		if !testCase.expectedError {
			assert.Nil(t, err)
			assert.Equal(t, testCase.expected, actual)
		} else {
			assert.Error(t, err, "expected error, but got %s", actual)
		}
	}
}
