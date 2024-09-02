package utils

import "strconv"

// StringToInt64 converts the given string to integer of base64
func StringToInt64(value string) int64 {
	integerValue, _ := strconv.ParseInt(value, 10, 64)
	return integerValue
}
