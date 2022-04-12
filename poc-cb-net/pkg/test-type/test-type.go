package testtype

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	// Connectivity is a constant variable for test "CONNECTIVITY"
	Connectivity = "CONNECTIVITY"
)

var placeHolder = `{"testType": "%s", "testSpec": %s}`

// BuildTestMessage represents a function to build a message with a test type and its test specifiction.
func BuildTestMessage(testType string, testSpec string) string {
	return fmt.Sprintf(placeHolder, testType, testSpec)
}

// ParseTestMessage represents a function to parse a test type and its test specification from a message.
func ParseTestMessage(message string) (string, string) {
	testType := gjson.Get(message, "testType").String()
	testSpec := gjson.Get(message, "testSpec").String()
	return testType, testSpec
}
