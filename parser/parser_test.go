package parser

import (
	"fmt"
	"testing"
)

func Test_newParse(t *testing.T) {
	filename := "../demo/base.jce"
	p := ParseFile(filename, make([]string, 0))
	fmt.Printf("%+v", p.Consts)
}
