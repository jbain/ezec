package config

import "strings"

type StringArgs struct {
	String string
}

func (s StringArgs) Args() []string {
	strs := strings.Split(s.String, " ")
	return strs
}
