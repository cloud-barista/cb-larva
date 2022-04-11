package cmd

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	// Up is a constant variable for command "UP"
	Up = "UP"

	// Down is a constant variable for command "DOWN"
	Down = "DOWN"

	// CheckConnectivity is a constant variable for command "CHECK_CONNECTIVITY"
	CheckConnectivity = "CHECK_CONNECTIVITY"

	// EnableEncryption is a constant variable for command "ENABLE_ENCRYPTION"
	EnableEncryption = "ENABLE_ENCRYPTION"

	// DisableEncryption is a constant variable for command "DISABLE_ENCRYPTION"
	DisableEncryption = "DISABLE_ENCRYPTION"
)

var placeHolder = `{"controlCommand": "%s", "controlCommandOption": "%s"}`

// BuildCommandMessage represents a function to build a message with a command and its option.
func BuildCommandMessage(controlCommand string, controlCommandOption string) string {
	return fmt.Sprintf(placeHolder, controlCommand, controlCommandOption)
}

// ParseCommandMessage represents a function to parse a command and its option from a message.
func ParseCommandMessage(message string) (string, string) {
	cmd := gjson.Get(message, "controlCommand").String()
	cmdOption := gjson.Get(message, "controlCommandOption").String()
	return cmd, cmdOption
}
