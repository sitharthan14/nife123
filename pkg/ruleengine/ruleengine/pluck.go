package ruleengine

import (
	"strings"
)

// pluck will pull out the value from the props given a path delimited by '.'
func pluck(props map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	for i := 0; i < len(parts)-1; i++ {
		var ok bool
		props, ok = props[parts[i]].(map[string]interface{})
		if !ok {
			return nil
		}
	}
	return props[parts[len(parts)-1]]
}

// pluck will pull out the value from the props given a path delimited by '.'
func GetKeyValue(props map[string]interface{}, keyName string) interface{} {
	keyValue, ok := props[keyName]
	if !ok {
		return nil
	}
	return keyValue
}
