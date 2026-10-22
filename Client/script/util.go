package script

import (
	"strconv"
	"strings"
)

var run = true
var start = false

func EqualValue(a, b string) bool {

	// thử parse thành số
	f1, err1 := strconv.ParseFloat(a, 64)
	f2, err2 := strconv.ParseFloat(b, 64)

	if err1 == nil && err2 == nil {
		return f1 == f2
	}

	// nếu không phải số → so sánh string
	return strings.EqualFold(a, b)
}
