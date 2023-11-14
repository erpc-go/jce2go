package lex

// SemInfo 结构用于存储与词法分析器相关的语义信息，例如整数、浮点数和字符串值。
// SemInfo is struct.
type SemInfo struct {
	I int64
	F float64
	S string
}

// Token 结构表示词法分析器中的一个标记。它包含一个 TK 类型的字段 T，一个指向 SemInfo 结构的指针 S 和一个整数类型的字段 Line，用于表示标记所在的行号。
// Token record token information.
type Token struct {
	T    TK
	S    *SemInfo
	Line int
}
