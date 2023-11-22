package timeutil

import "time"

const DateFormat = "2006-01-02T15:04:05.999999-07:00"

func FormatNow() string {
	return time.Now().Format(DateFormat)
}
