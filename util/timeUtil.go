package util

import (
	"time"
)

const (
	MILI_IN_YEAR = 1000 * 60 * 60 * 24 * 365
)

func GetMili() int64 {
	return int64(time.Now().Unix())
}

func GetDayFromMili(mili int64) int64 {
	return mili / (MILI_IN_YEAR)
}
