package lex

import (
	"bytes"
	"strconv"
	"strings"
)

// 词法分析器的状态
// LexState record lexical state.
type LexState struct {
	current    byte // 当前正在处理的字节
	lineNumber int  // 当前处理的行号

	tokenBuff bytes.Buffer // 存储标记的缓冲区

	filename string        // 处理的文件名
	source   *bytes.Buffer // 存储输入源代码的缓冲区
}

// NewLexState to update LexState struct.
func NewLexState(filename string, source []byte) *LexState {
	return &LexState{
		current:    ' ',
		lineNumber: 1,
		filename:   filename,
		source:     bytes.NewBuffer(source),
	}
}

// NextToken return token after lexical analysis.
// NextToken 方法是词法分析器的公共接口，用于返回经过词法分析后的下一个标记。
// 它创建一个名为 tk 的新 Token 结构，并调用 llex 方法执行词法分析。
// llex 方法返回一个 TK 值和一个指向 SemInfo 结构的指针，这两个值分别被赋给 tk.T 和 tk.S。然后，将当前行号赋给 tk.Line。
// 最后，NextToken 方法返回指向填充的 Token 结构的指针。这个方法可以在词法分析过程中多次调用，以依次获取源代码中的所有标记。
func (ls *LexState) NextToken() *Token {
	tk := &Token{}
	tk.Type, tk.Value = ls.llex()
	tk.Line = ls.lineNumber
	return tk
}

// lexErr 方法接受一个错误字符串作为参数，然后将其与当前行号和源代码文件名组合，
// 生成一个详细的错误消息并引发一个 panic。这个方法在词法分析过程中遇到错误时被调用。
func (ls *LexState) lexErr(err string) {
	line := strconv.Itoa(ls.lineNumber)
	panic(ls.filename + ": " + line + ".    " + err)
}

// incLineNumber 方法用于在遇到换行符时递增行号。
// 它首先跳过换行符（'\n' 或 '\r'），
// 然后检查下一个字符是否为另一个换行符（'\r\n' 或 '\n\r'）。
// 如果是，它再次跳过该字符。最后，行号递增。
func (ls *LexState) incLineNumber() {
	old := ls.current
	ls.next() /* skip '\n' or '\r' */
	if isNewLine(ls.current) && ls.current != old {
		ls.next() /* skip '\n\r' or '\r\n' */
	}
	ls.lineNumber++
}

// readNumber 方法用于从输入缓冲区读取一个数字（整数或浮点数）。
// 它首先检查当前字符是否为数字、点（'.'）或十六进制数字（'x'、'X' 或其他十六进制数字）。
// 然后，它将字符添加到 tokenBuff 并获取下一个字符。当读取完整个数字后，它将尝试将其解析为浮点数或整数，并将结果存储在 SemInfo 结构中。
// 如果解析过程中出现错误，它将调用 lexErr 方法引发一个 panic。
func (ls *LexState) readNumber() (TokenType, *TokenValue) {
	hasDot := false
	isHex := false
	sem := &TokenValue{}
	for isNumber(ls.current) || ls.current == '.' || ls.current == 'x' || ls.current == 'X' ||
		(isHex && isHexNumber(ls.current)) {

		if ls.current == '.' {
			hasDot = true
		} else if ls.current == 'x' || ls.current == 'X' {
			isHex = true
		}
		ls.tokenBuff.WriteByte(ls.current)
		ls.next()
	}
	sem.S = ls.tokenBuff.String()
	if hasDot {
		f, err := strconv.ParseFloat(sem.S, 64)
		if err != nil {
			ls.lexErr(err.Error())
		}
		sem.F = f
		return TkFloat, sem
	}
	i, err := strconv.ParseInt(sem.S, 0, 64)
	if err != nil {
		ls.lexErr(err.Error())
	}
	sem.I = i
	return TkInteger, sem
}

// readIdent 方法用于从输入缓冲区读取一个标识符（变量名、关键字或类型）。
// 它首先检查当前字符是否为字母、数字或冒号（':'）。然后，它将字符添加到 tokenBuff 并获取下一个字符。
// 当读取完整个标识符后，它将尝试将其与已知的关键字和类型进行匹配。
// 如果标识符包含冒号（':'），则表示它可能包含命名空间限定符（'::'）。
// 在这种情况下，代码会检查标识符的格式是否合法（即是否只包含一个 '::'），并在必要时修剪命名空间前缀。
// 接下来，代码遍历已知的关键字和类型，检查标识符是否与它们之一匹配。如果找到匹配项，它将返回对应的 TK 值。
// 如果标识符不是关键字或类型，那么它被视为普通的变量名，返回 tkName 和包含标识符字符串的 SemInfo 结构。
// 这个方法在词法分析过程中被调用，以读取标识符并将其分类为关键字、类型或变量名。
func (ls *LexState) readIdent() (TokenType, *TokenValue) {
	sem := &TokenValue{}
	var last byte

	// :: Point number processing namespace
	for isLetter(ls.current) || isNumber(ls.current) || ls.current == ':' {
		if isNumber(ls.current) && last == ':' {
			ls.lexErr("the identification is illegal.")
		}
		last = ls.current
		ls.tokenBuff.WriteByte(ls.current)
		ls.next()
	}
	sem.S = ls.tokenBuff.String()
	if strings.Count(sem.S, ":") > 0 {
		if strings.Count(sem.S, "::") == 2 && strings.Count(sem.S, ":") == 4 {
			sem.S = sem.S[strings.Index(sem.S, "::")+2:]
		}
		if strings.Count(sem.S, "::") != 1 || strings.Count(sem.S, ":") != 2 {
			ls.lexErr("namespace qualifier::is illegal")
		}
	}

	for i := TkDummyKeywordBegin + 1; i < TkDummyKeywordEnd; i++ {
		if TokenMap[i] == sem.S {
			return i, nil
		}
	}
	for i := TkDummyTypeBegin + 1; i < TkDummyTypeEnd; i++ {
		if TokenMap[i] == sem.S {
			return i, nil
		}
	}

	return TkName, sem
}

// readSharp 方法用于处理以 # 开头的预处理指令。
// 它首先跳过 # 字符，然后读取后面的字母。如果读取到的字符串不是 "include"，则引发一个错误，
// 因为这里仅处理 #include 指令。如果是 "include"，则返回 tkInclude 标记。
func (ls *LexState) readSharp() (TokenType, *TokenValue) {
	ls.next()
	for isLetter(ls.current) {
		ls.tokenBuff.WriteByte(ls.current)
		ls.next()
	}
	if ls.tokenBuff.String() != "include" {
		ls.lexErr("not #include")
	}

	return TkInclude, nil
}

// readString 方法用于从输入缓冲区读取一个字符串。
// 它首先跳过开始的双引号（"），然后读取后面的字符，直到遇到另一个双引号或输入结束（EOS）。
// 如果在读取字符串时遇到输入结束，将引发一个错误。否则，将读取到的字符串存储在 SemInfo 结构中，并返回 tkString 标记。
func (ls *LexState) readString() (TokenType, *TokenValue) {
	sem := &TokenValue{}
	ls.next()
	for {
		if ls.current == EOS {
			ls.lexErr(`no match "`)
		} else if ls.current == '"' {
			ls.next()
			break
		} else {
			ls.tokenBuff.WriteByte(ls.current)
			ls.next()
		}
	}
	sem.S = ls.tokenBuff.String()

	return TkString, sem
}

func (ls *LexState) readComment() (TokenType, *TokenValue) {
	comment := &TokenValue{}
	ls.next()
	if ls.current == '/' {
		ls.tokenBuff.WriteByte('/')
		for !isNewLine(ls.current) && ls.current != EOS {
			ls.tokenBuff.WriteByte(ls.current)
			ls.next()
		}
	} else if ls.current == '*' {
		ls.tokenBuff.WriteByte('/')
		ls.tokenBuff.WriteByte('*')
		ls.next()
		ls.readLongComment()
	} else {
		ls.lexErr("lexical error，/")
	}
	comment.S = ls.tokenBuff.String()
	return TkComment, comment
}

// readLongComment 方法用于处理长注释。
// 它遍历输入缓冲区的字符，直到遇到 */（表示注释结束）或输入结束（EOS）。
// 在遍历过程中，如果遇到换行符，它会调用 incLine 方法递增行号。如果在注释内遇到输入结束，将引发一个错误。
func (ls *LexState) readLongComment() {
	for {
		switch ls.current {
		case EOS:
			ls.lexErr("respect */")
			return
		case '\n', '\r':
			ls.tokenBuff.WriteByte(ls.current)
			ls.incLineNumber()
		case '*':
			ls.tokenBuff.WriteByte(ls.current)
			ls.next()
			if ls.current == EOS {
				return
			} else if ls.current == '/' {
				ls.tokenBuff.WriteByte(ls.current)
				ls.next()
				return
			}
		default:
			ls.tokenBuff.WriteByte(ls.current)
			ls.next()
		}
	}
}

// next 方法用于从输入缓冲区读取下一个字符。
// 它尝试从 buff 中读取一个字节，如果读取成功，则将其存储在 current 字段中。
// 如果读取过程中出现错误（例如，已到达输入结束），则将 current 设置为 EOS。
func (ls *LexState) next() {
	var err error
	ls.current, err = ls.source.ReadByte()
	if err != nil {
		ls.current = EOS
	}
}

// llexDefault 方法用于处理词法分析器中的默认情况。
// 它首先检查当前字符是否为数字或字母。
// 如果是数字，则调用 readNumber 方法读取数字。
// 如果是字母，则调用 readIdent 方法读取标识符。
// 如果当前字符既不是数字也不是字母，则引发一个错误，指出无法识别的字符。
func (ls *LexState) llexDefault() (TokenType, *TokenValue) {
	switch {
	case isNumber(ls.current):
		return ls.readNumber()
	case isLetter(ls.current):
		return ls.readIdent()
	default:
		ls.lexErr("unrecognized characters, " + string(ls.current))
		return '0', nil
	}
}

// Do lexical analysis.
// llex 方法是词法分析器的主要方法，用于执行词法分析。它遍历输入缓冲区的字符，并根据当前字符的类型调用相应的处理方法。以下是方法的主要步骤：
// 清空 tokenBuff 以存储新的标记。
// 使用 switch 语句检查当前字符。各种情况如下：
// 如果输入结束（EOS），返回 tkEos 标记。
// 如果是空白字符（空格、制表符等），跳过它并读取下一个字符。
// 如果是换行符，调用 incLineNumber 方法递增行号。
// 如果是注释开头（/），根据注释类型（单行或长注释）调用相应的处理方法。
// 对于其他特殊字符（如括号、分号、等号等），返回相应的 TK 值。
// 如果是双引号，调用 readString 方法读取字符串。
// 如果是 #，调用 readSharp 方法处理预处理指令（如 #include）。
// 对于其他字符，调用 llexDefault 方法处理默认情况（例如，读取数字和标识符）。
// 当所有字符都被处理后，方法返回相应的 TK 值和 SemInfo 结构（如果适用）。
// 这个方法在词法分析过程中被调用，以处理输入缓冲区中的各种字符并生成相应的词法标记。
func (ls *LexState) llex() (TokenType, *TokenValue) {
	for {
		ls.tokenBuff.Reset()
		switch ls.current {
		case EOS:
			return TkEos, nil
		case ' ', '\t', '\f', '\v':
			ls.next()
		case '\n', '\r':
			ls.incLineNumber()
		case '/':
			return ls.readComment()
		case '{':
			ls.next()
			return TkBraceLeft, nil
		case '}':
			ls.next()
			return TkBraceRight, nil
		case ';':
			ls.next()
			return TkSemi, nil
		case '=':
			ls.next()
			return TkEq, nil
		case '<':
			ls.next()
			return TkShl, nil
		case '>':
			ls.next()
			return TkShr, nil
		case ',':
			ls.next()
			return TkComma, nil
		case '(':
			ls.next()
			return TkPtl, nil
		case ')':
			ls.next()
			return TkPtr, nil
		case '[':
			ls.next()
			return TkSquareLeft, nil
		case ']':
			ls.next()
			return TkSquarerRight, nil
		case '"':
			return ls.readString()
		case '#':
			return ls.readSharp()
		default:
			return ls.llexDefault()

		}
	}
}
