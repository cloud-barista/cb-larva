package cmdtype

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	// Up is a constant variable for command "UP"
	Up string = "UP"

	// Down is a constant variable for command "DOWN"
	Down string = "DOWN"

	// EnableEncryption is a constant variable for command "ENABLE_ENCRYPTION"
	EnableEncryption string = "ENABLE_ENCRYPTION"

	// DisableEncryption is a constant variable for command "DISABLE_ENCRYPTION"
	DisableEncryption string = "DISABLE_ENCRYPTION"
)

var placeHolder = `{"commandType": "%s"}`

// BuildCommandMessage represents a function to build a message with a command.
func BuildCommandMessage(commandType string) string {
	return fmt.Sprintf(placeHolder, commandType)
}

// ParseCommandMessage represents a function to parse a command from a message.
func ParseCommandMessage(message string) string {
	cmd := gjson.Get(message, "commandType").String()
	return cmd
}
