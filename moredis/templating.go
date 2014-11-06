package moredis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/mgo.v2/bson"
)

// funcMap defines functions that are exported to the templates used
// by the config yaml.
var funcMap = template.FuncMap{
	"toLower":  safeToLower,
	"toString": toString,
	"toSet":    toSet,
}

// toString is a function that is exported to templates to allow
// converting non-string objects to strings.  Normally we would
// let types do this themselves by implementing Stringer interface
// (which is what Sprint will do) but for ObjectId's, it's more
// useful to get the Hex value.
func toString(toConvert interface{}) string {
	switch toConvert := toConvert.(type) {
	case bson.ObjectId:
		return toConvert.Hex()
	default:
		return fmt.Sprint(toConvert)
	}
}

// safeToLower is a function that is exported to templates as 'toLower'
// to allow converting strings to lowercase.  The reason for not
// just exporting strings.ToLower is we need to be able to handle
// non-strings in a consistent way.
func safeToLower(toConvert interface{}) string {
	switch toConvert := toConvert.(type) {
	case string:
		return strings.ToLower(toConvert)
	default:
		return ""
	}
}

// toSet is a function that is exported to templates as 'toSet'
// to allow printing sets of the bson.M{'key1': true, 'key2': true, ...}
// as stringified sets [key1, key2, ...].  For non-sets, this will
// return the empty string.
func toSet(toConvert interface{}) string {
	switch toConvert := toConvert.(type) {
	case bson.M:
		set := make([]string, 0, len(toConvert))
		for key, val := range toConvert {
			switch val := val.(type) {
			case bool:
				if !val {
					return ""
				}
			case string:
				valBool, err := strconv.ParseBool(val)
				if err != nil || !valBool {
					return ""
				}
			default:
				return ""
			}
			set = append(set, key)
		}
		// sort here so that the order is determinate, otherwise testing is more
		// difficult
		sort.Strings(set)
		return "[" + strings.Join(set, ",") + "]"
	}

	return ""
}

// ParseTemplates takes a collection config and parses the templates
// for all of the contained maps.
func ParseTemplates(collection *CollectionConfig) error {
	for ix, rmap := range collection.Maps {
		keyTmpl, err := template.New(rmap.HashKey + ":key").Funcs(funcMap).Parse(rmap.Key)
		if err != nil {
			return err
		}
		collection.Maps[ix].KeyTemplate = keyTmpl

		valTmpl, err := template.New(rmap.HashKey + ":val").Funcs(funcMap).Parse(rmap.Value)
		if err != nil {
			return err
		}
		collection.Maps[ix].ValueTemplate = valTmpl
	}
	return nil
}

// ApplyTemplate takes a template string and a map of values to use
// for evaluating the template.  Returns the evaluated template as
// a string or an error.
func ApplyTemplate(templateString string, payload bson.M) (string, error) {
	tmpl, err := template.New("").Funcs(funcMap).Parse(templateString)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, payload); err != nil {
		return "", err
	}
	return b.String(), nil
}

// ParseTemplatedJSON takes a template string for a json object and
// a map of values for substitution and returns a map that can be used in
// mgo queries.
// The template string must evaluate to a valid JSON object.  This means
// that all keys must be strings.  For mongo operators, you should
// encase them in quotes, for example "$or".  For ObjectIds, you should
// use the hex string in place of the ObjectId.
func ParseTemplatedJSON(query string, params Params) (map[string]interface{}, error) {
	parsed, err := ApplyTemplate(query, params.Bson())
	if err != nil {
		return map[string]interface{}{}, err
	}
	var queryObject map[string]interface{}
	if err := json.Unmarshal([]byte(parsed), &queryObject); err != nil {
		return map[string]interface{}{}, err
	}
	setObjectIds(queryObject)
	return queryObject, nil
}

// setObjectIds recursively searches a map for string values that can
// be converted to mongo ObjectIds.  The map is mutated in place.
func setObjectIds(part map[string]interface{}) {
	for key, val := range part {
		switch val := val.(type) {
		case string:
			if bson.IsObjectIdHex(val) {
				part[key] = bson.ObjectIdHex(val)
			}
		case map[string]interface{}:
			setObjectIds(val)
		}
	}
}
