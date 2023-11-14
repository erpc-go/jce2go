package parser

import "github.com/erpc-go/jce2go/utils"

// ConstInfo record const information.
type ConstInfo struct {
	Type    *VarType
	Name    string
	Value   string
	Comment string
}

func (cst *ConstInfo) Rename() {
	cst.Name = utils.UpperFirstLetter(cst.Name)
}

func (cst *ConstInfo) String() string {
	return cst.Name + cst.Value + cst.Comment
}
