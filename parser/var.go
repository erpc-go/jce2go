package parser

import "github.com/erpc-go/jce2go/lex"

// VarType contains variable type(token)
type VarType struct {
	Type     lex.TokenType // basic type
	Unsigned bool          // whether unsigned
	TypeSt   string        // custom type name, such as an enumerated struct,at this time Type=lex.TkName
	CType    lex.TokenType // make sure which type of custom type is,lex.TkEnum, lex.TkStruct
	TypeK    *VarType      // vector's member variable,the key of map
	TypeV    *VarType      // the value of map
	TypeL    int64         // length of array
}
