package lex

// EOS is byte stream terminator
const EOS = 0

// TK is a byte type.
// 这段代码定义了一个名为 TK 的类型，它是一个字节类型。TK 类型用于表示词法分析器的各种标记（tokens）。这些标记包括括号、分号、等号、关键字、类型等。
type TK byte

// 代码中的 const 部分定义了一系列 TK 类型的常量，它们表示各种可能的标记。这些常量被分为几个部分，例如关键字、类型和值。
const (
	TkEos          TK = iota
	TkBraceLeft       //({)
	TkBraceRight      //}
	TkSemi            //;
	TkEq              //=
	TkShl             //<
	TkShr             //>
	TkComma           //,
	TkPtl             //(
	TkPtr             //)
	TkSquareLeft      //[
	TkSquarerRight    //]
	TkInclude         //#include

	TkDummyKeywordBegin
	// keyword
	TkModule
	TkEnum
	TkStruct
	TkInterface
	TkRequire
	TkOptional
	TkConst
	TkUnsigned
	TkVoid
	TkOut
	TkTrue
	TkFalse
	TkDummyKeywordEnd

	TkDummyTypeBegin
	// type
	TkTInt
	TkTBool
	TkTShort
	TkTByte
	TkTLong
	TkTFloat
	TkTDouble
	TkTString
	TkTVector
	TkTMap
	TkTArray
	TkDummyTypeEnd

	TkName // variable name
	// value
	TkString
	TkInteger
	TkFloat
)

// TokenMap record token  value.
// TokenMap 数组将 TK 类型的值映射到它们对应的字符串表示。这个映射有助于在调试或输出错误消息时更容易地理解和显示这些标记。
var TokenMap = [...]string{
	TkEos: "<eos>",

	TkBraceLeft:    "{",
	TkBraceRight:   "}",
	TkSemi:         ";",
	TkEq:           "=",
	TkShl:          "<",
	TkShr:          ">",
	TkComma:        ",",
	TkPtl:          "(",
	TkPtr:          ")",
	TkSquareLeft:   "[",
	TkSquarerRight: "]",
	TkInclude:      "#include",

	// keyword
	TkModule:    "module",
	TkEnum:      "enum",
	TkStruct:    "struct",
	TkInterface: "interface",
	TkRequire:   "require",
	TkOptional:  "optional",
	TkConst:     "const",
	TkUnsigned:  "unsigned",
	TkVoid:      "void",
	TkOut:       "out",
	TkTrue:      "true",
	TkFalse:     "false",

	// type
	TkTInt:    "int",
	TkTBool:   "bool",
	TkTShort:  "short",
	TkTByte:   "byte",
	TkTLong:   "long",
	TkTFloat:  "float",
	TkTDouble: "double",
	TkTString: "string",
	TkTVector: "vector",
	TkTMap:    "map",
	TkTArray:  "array",

	TkName: "<name>",
	// value
	TkString:  "<string>",
	TkInteger: "<INTEGER>",
	TkFloat:   "<FLOAT>",
}

// isNewLine 函数接受一个字节参数 b，如果它是换行符（'\r' 或 '\n'），则返回 true。
func isNewLine(b byte) bool {
	return b == '\r' || b == '\n'
}

// isNumber 函数接受一个字节参数 b，如果它是数字字符（'0' 到 '9'）或负号（'-'），则返回 true。
func isNumber(b byte) bool {
	return (b >= '0' && b <= '9') || b == '-'
}

// isHexNumber 函数接受一个字节参数 b，如果它是十六进制数字字符（'a' 到 'f' 或 'A' 到 'F'），则返回 true。
func isHexNumber(b byte) bool {
	return (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// isLetter 函数接受一个字节参数 b，如果它是字母字符（'a' 到 'z' 或 'A' 到 'Z'）或下划线（'_'），则返回 true。
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

// IsType 函数接受一个 TK 类型参数 t，如果它表示一个有效的类型（在 tkDummyTypeBegin 和 tkDummyTypeEnd 之间），则返回 true。
func IsType(t TK) bool {
	return t > TkDummyTypeBegin && t < TkDummyTypeEnd
}

// IsNumberType 函数接受一个 TK 类型参数 t，如果它表示一个数字类型（例如 tkTInt、tkTBool、tkTShort 等），则返回 true。
func IsNumberType(t TK) bool {
	switch t {
	case TkTInt, TkTBool, TkTShort, TkTByte, TkTLong, TkTFloat, TkTDouble:
		return true
	default:
		return false
	}
}
