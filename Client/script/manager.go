package script

var Fmt func(...any)
var msgbox func(string)

func Manager(f func(...any), m func(string)) {
	Fmt = f
	msgbox = m
}
