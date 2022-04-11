package cmd

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	// Up is a constant variable for command "up"
	Up = "up"

	// Down is a constant variable for command "down"
	Down = "down"

	// CheckConnectivity is a constant variable for command "check-connectivity"
	CheckConnectivity = "check-connectivity"

	// EnableEncryption is a constant variable for command "enable-encryption"
	EnableEncryption = "enable-encryption"

	// DisableEncryption is a constant variable for command "disable-encryption"
	DisableEncryption = "disable-encryption"
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
