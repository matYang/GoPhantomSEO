package util

import (
	"strconv"
)

func I64ToStr(i64 int64) string {
	return strconv.FormatInt(i64, 10)
}

func StrToI64(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}
