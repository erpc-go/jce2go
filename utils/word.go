package utils

import "strings"

// 首字母大写
func UpperFirstLetter(s string) string {
	if len(s) == 0 {
		return ""
	}

	if len(s) == 1 {
		return strings.ToUpper(string(s[0]))
	}

	return strings.ToUpper(string(s[0])) + s[1:]
}
