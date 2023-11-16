package utils

import (
	"bytes"
	"encoding/json"
)

func FormatJOSN(j interface{}) string {
	b, err := json.Marshal(j)
	if err != nil {
		return ""
	}
	var str bytes.Buffer
	err = json.Indent(&str, b, "", "    ")
	if err != nil {
		return ""
	}
	return str.String()
}
