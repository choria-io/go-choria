package lifecycle

import "time"

var mockTime int64

func timeStamp() int64 {
	if mockTime != 0 {
		return mockTime
	}

	return time.Now().UTC().Unix()
}
