package config

import "strings"

type StringArgs struct {
	String string
}

func (s StringArgs) Args() []string {
	return strings.Fields(s.String)
}
