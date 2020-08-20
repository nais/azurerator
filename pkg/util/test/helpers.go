package test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertContainsKeysWithNonEmptyValues(t *testing.T, a interface{}, keys []string) {
	for _, key := range keys {
		assert.Containsf(t, a, key, "should contain key '%s'", key)
	}
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Map {
		for _, val := range v.MapKeys() {
			assert.NotEmpty(t, v.MapIndex(val).String())
		}
	}
	assert.Lenf(t, a, len(keys), "should contain %v keys", len(keys))
}

func AssertAllNotEmpty(t *testing.T, values []interface{}) {
	for _, val := range values {
		assert.NotEmpty(t, val)
	}
}
