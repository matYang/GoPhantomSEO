package util

import (
	"time"
)

const (
	MILI_IN_DAY = 1000 * 60 * 60 * 24
)

//得到当前的毫秒值
func GetMili() int64 {
	return int64(time.Now().UnixNano() / 1000000)
}

//从毫秒值得到当前的天数
func GetDayFromMili(mili int64) int64 {
	return mili / (MILI_IN_DAY)
}
