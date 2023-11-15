package parser

import (
	"github.com/erpc-go/jce2go/utils"
)

const (
	EnumTypeValue   = 0
	EnumTypeName    = 1
	EnumTypeEqual   = 2
	EnumTypeComment = 3
)

// EnumMember record member information.
type EnumMember struct {
	Key     string
	Type    int
	Value   int32  // type 0
	Name    string // type 1
	Comment string
}

// EnumInfo record EnumMember information include name.
type EnumInfo struct {
	Module      string
	Name        string
	TypeComment string
	Comment     string
	Member      []EnumMember
}

// enum 变量重命名，即把首字母都大写
func (en *EnumInfo) Rename() {
	en.Name = utils.UpperFirstLetter(en.Name)
	for i := range en.Member {
		en.Member[i].Key = utils.UpperFirstLetter(en.Member[i].Key)
	}
}
