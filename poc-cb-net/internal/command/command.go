package cmd

import (
	"fmt"

	"github.com/tidwall/gjson"
)

const (
	Resume            = "resume"
	Suspend           = "suspend"
	CheckConnectivity = "check-connectivity"
)

var placeHolder = `{"controlCommand": "%s", "controlCommandOption": "%s"}`

func BuildMessageBody(controlCommand string, controlCommandOption string) string {
	return fmt.Sprintf(placeHolder, controlCommand, controlCommandOption)
}

func ParseMessageBody(message string) (string, string) {
	cmd := gjson.Get(message, "controlCommand").String()
	cmdOption := gjson.Get(message, "controlCommandOption").String()
	return cmd, cmdOption
}
