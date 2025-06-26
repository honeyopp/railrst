package utils

import "strings"

func JoinIDs(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, "|")
}
