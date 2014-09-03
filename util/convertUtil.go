package util

import (
	"strconv"
)

//讲64位整数变成字符串
func I64ToStr(i64 int64) string {
	return strconv.FormatInt(i64, 10)
}

//将字符串变成64位整数
func StrToI64(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}
