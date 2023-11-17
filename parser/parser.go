package parser

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/erpc-go/jce2go/lex"
	"github.com/erpc-go/jce2go/log"
	"github.com/erpc-go/jce2go/utils"
)

// 语法分析器
// Parser record information of parse file.
type Parser struct {
	Filepath string // 源文件路径

	Module        string // 包名
	ModuleComment string

	Includes       []string // 依赖的其他 jce 文件
	IncludeComment string

	Enums   []EnumInfo   // 枚举信息列表
	Consts  []ConstInfo  // 常量信息列表
	Structs []StructInfo // 结构体信息列表

	comments []lex.Token // 临时存储的注释

	// have parsed include file
	IncParse []*Parser // 已解析的包含文件

	lex       *lex.LexState // 词法分析器状态
	token     *lex.Token    // 当前处理的 token
	lastToken *lex.Token    // 上一个处理的 token

	// jce include chain
	IncChain []string // jce 包含链

	// proto file name(not include .jce)
	ProtoName string // 协议名（不包括 .jce 扩展名）

	fileNames map[string]bool // 一个存储文件名的映射
}

// newParse 函数接受一个文件路径 s、一个字节切片 b 和一个包含链 incChain 作为参数，用于初始化并返回一个新的 Parser 结构。
// 在初始化过程中，它会检查包含链中是否存在循环引用，并将当前文件路径添加到包含链中。
// 然后，它创建一个新的 LexState 结构，用于在词法分析过程中存储状态。
func newParse(filepath string, source []byte, incChain []string) *Parser {
	p := &Parser{
		Filepath:  filepath,
		ProtoName: utils.Path2PackageName(filepath, ".jce"),
	}

	for _, v := range incChain {
		if filepath == v {
			panic("jce circular reference: " + filepath)
		}
	}

	incChain = append(incChain, filepath)
	p.IncChain = incChain
	p.lex = lex.NewLexState(filepath, source)
	p.fileNames = map[string]bool{}

	return p
}

// ParseFile parse a file,return grammar tree.
// ParseFile 函数接受一个文件路径 filePath 和一个包含链 incChain 作为参数，
// 用于解析文件并返回一个语法树。它首先使用 ioutil.ReadFile 函数读取文件内容，
// 然后调用 newParse 函数创建一个新的 Parser 结构。最后，它调用 parse 方法解析文件，并返回解析后的 Parser 结构。
func ParseFile(filePath string, incChain []string) *Parser {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic("file read error: " + filePath + ". " + err.Error())
	}

	p := newParse(filePath, b, incChain)
	p.parse()
	log.Debug("end parseFile,%+v", filePath)

	return p
}

// parse 方法是 Parser 结构的一个成员方法，用于执行语法分析。它遍历由词法分析器生成的 token，并根据 token 的类型调用相应的处理方法。以下是方法的主要步骤：

// 使用 for 循环遍历 token。在每次迭代中，调用 p.next() 方法获取下一个 token，并将其存储在局部变量 t 中。
// 使用 switch 语句检查 token 的类型。各种情况如下：
// 如果 token 类型为 lex.TkEos（表示输入结束），则跳出循环。
// 如果 token 类型为 lex.TkInclude，则调用 parseInclude 方法处理包含指令。
// 如果 token 类型为 lex.TkModule，则调用 parseModule 方法处理模块声明。
// 对于其他 token 类型，引发一个错误，指出期望的 token 类型。
// 在循环结束后，调用 analyzeDepend 方法分析文件的依赖关系。
// 这个方法在语法分析过程中被调用，以处理 token 并生成相应的语法树。

func (p *Parser) parse() {
OUT:
	for {
		p.next()
		t := p.token
		switch t.Type {
		case lex.TkEos:
			break OUT
		case lex.TkInclude:
			p.parseInclude()
		case lex.TkModule:
			p.parseModule()
		case lex.TkComment: // 注释暂存
			p.comments = append(p.comments, *p.token)
		default:
			p.parseErr("Expect include or module.")
		}
	}
	log.Debug("end parse")
	p.analyzeDepend()
}

// 获取下一个 token
func (p *Parser) next() {
	p.lastToken = p.token
	p.token = p.lex.NextToken()
}

// 该方法返回下一个 token，而不实际前进到下一个 token
func (p *Parser) peek() (t *lex.Token) {
	return p.lex.PeekToken()
}

// expect 方法接受一个 lex.TokenType 类型的参数 t。
// 它首先调用 next 方法获取下一个 token，然后检查 token 的类型是否与期望的类型相匹配。如果不匹配，将引发一个解析错误。
func (p *Parser) expect(t lex.TokenType) {
	p.next()
	if p.token.Type != t {
		p.parseErr("expect " + lex.TokenMap[t])
	}
}

// parseErr 方法接受一个错误字符串 err 作为参数，并引发一个 panic，其中包含发生错误的文件路径和行号。这个方法在解析过程中遇到错误时被调用。
func (p *Parser) parseErr(err string) {
	line := "0"
	if p.token != nil {
		line = strconv.Itoa(p.token.Line)
	}

	panic("[" + p.Filepath + ":" + line + "]" + err)
}

// parseInclude 方法用于处理包含指令。它首先调用 expect 方法，期望下一个 token 是一个字符串。然后，将该字符串添加到 Includes 字段中。
func (p *Parser) parseInclude() {
	p.expect(lex.TkString)
	p.Includes = append(p.Includes, p.token.Value.String)
	p.IncludeComment = p.getPreComments()
}

// parseModule 方法用于处理模块声明。
func (p *Parser) parseModule() {
	// 它首先调用 expect 方法，期望下一个 token 是一个名称。
	p.expect(lex.TkName)

	// 然后，检查 Module 字段是否已经设置。
	// 将模块名存储在 Module 字段中，并调用 parseModuleSegment 方法处理模块内的内容。
	if p.Module == "" {
		p.Module = p.token.Value.String
		p.ModuleComment = p.getPreComments()
		p.parseModuleSegment()
		return
	}

	// 如果已设置，表示一个 jce 文件中定义了多个模块，
	// 需要创建一个新的 Parser 结构来处理这个新模块。
	// 解决一个jce文件中定义多个module
	name := p.ProtoName + "_" + p.token.Value.String + ".jce"
	newp := newParse(name, nil, nil)
	newp.IncChain = p.IncChain
	newp.lex = p.lex
	newp.Includes = p.Includes
	newp.IncParse = p.IncParse
	cowp := *p
	newp.IncParse = append(newp.IncParse, &cowp)
	newp.Module = p.token.Value.String
	newp.parseModuleSegment()
	newp.analyzeDepend()
	if p.fileNames[name] {
		// merge
		for _, incParse := range p.IncParse {
			if incParse.ProtoName == newp.ProtoName {
				incParse.Structs = append(incParse.Structs, newp.Structs...)
				incParse.Enums = append(incParse.Enums, newp.Enums...)
				incParse.Consts = append(incParse.Consts, newp.Consts...)
				break
			}
		}
	} else {
		// 增加已经解析的module
		p.IncParse = append(p.IncParse, newp)
		p.fileNames[name] = true
	}
	p.lex = newp.lex
}

// parseModuleSegment 方法是 Parser 结构的一个成员方法，用于解析模块内的内容。
func (p *Parser) parseModuleSegment() {
	log.Debug("begin parseModuleSegment")
	// 使用 expect 方法检查下一个 token 是否为左大括号（lex.TkBraceLeft）。
	p.expect(lex.TkBraceLeft)

	// 使用 for 循环遍历 token。在每次迭代中，调用 p.next() 方法获取下一个 token
	for {
		p.next()
		// fmt.Println(lex.TokenMap[p.token.Type])
		switch p.token.Type {
		case lex.TkBraceRight: //  如果 token 类型为 lex.TkBraceRight（表示模块声明的结束
			p.expect(lex.TkSemi) // 则调用 expect 方法检查下一个 token 是否为分号（lex.TkSemi），然后返回
			log.Debug("end parseModuleSegment")
			return
		case lex.TkConst: // 如果 token 类型为 lex.TkConst，则调用 parseConst 方法处理常量声明
			p.parseConst()
		case lex.TkEnum: // 如果 token 类型为 lex.TkEnum，则调用 parseEnum 方法处理枚举声明。
			p.parseEnum()
		case lex.TkStruct: //  如果 token 类型为 lex.TkStruct，则调用 parseStruct 方法处理结构体声明
			p.parseStruct()
		case lex.TkComment: // 注释暂存
			p.comments = append(p.comments, *p.token)
		default: // 对于其他 token 类型，引发一个解析错误，指出不期望的 token 类型
			j, _ := json.Marshal(p.token.Value)
			p.parseErr("not except " + lex.TokenMap[p.token.Type] + " type, value: " + string(j))
		}
	}
}

func (p *Parser) getPreComments() (comment string) {
	for _, c := range p.comments {
		comment += c.Value.String
		comment += "\n"
	}
	p.comments = []lex.Token{}
	return
}

// parseConst 方法是 Parser 结构的一个成员方法，用于解析常量声明。
// 它遍历由词法分析器生成的 token，并根据 token 的类型执行相应的操作。以下是方法的主要步骤：
func (p *Parser) parseConst() {
	// 创建一个名为 consts 的新 ConstInfo 结构，用于存储常量信息。
	consts := ConstInfo{}
	consts.PreComment = p.getPreComments()

	// 调用 next 方法获取下一个 token，并检查其类型。

	p.next()
	switch p.token.Type {
	case lex.TkTVector, lex.TkTMap: // 如果 token 类型为 lex.TkTVector 或 lex.TkTMap，则引发一个错误，因为常量不支持向量或映射类型。
		p.parseErr("const no supports type vector or map.")
	case lex.TkTBool, lex.TkTByte, lex.TkTShort, // 对于其他支持的类型，调用 parseType 方法解析类型并将其存储在 m.Type 中
		lex.TkTInt, lex.TkTLong, lex.TkTFloat,
		lex.TkTDouble, lex.TkTString, lex.TkUnsigned:
		consts.Type = p.parseType()
	default:
		p.parseErr("expect type.")
	}

	// 使用 expect 方法检查下一个 token 是否为名称，并将其存储在 m.Name 中。
	p.expect(lex.TkName)
	consts.Name = p.token.Value.String

	// 使用 expect 方法检查下一个 token 是否为等号（lex.TkEq）。
	p.expect(lex.TkEq)

	// 调用 next 方法获取下一个 token
	// 根据 token 的类型和常量的类型设置默认值。
	// 如果 token 类型与常量类型不匹配，将引发一个错误。将默认值存储在 m.Value 中。
	p.next()
	switch p.token.Type {
	case lex.TkInteger, lex.TkFloat:
		if !lex.IsNumberType(consts.Type.Type) {
			p.parseErr("type does not accept number")
		}
		consts.Value = p.token.Value.String
	case lex.TkString:
		if lex.IsNumberType(consts.Type.Type) {
			p.parseErr("type does not accept string")
		}
		consts.Value = `"` + p.token.Value.String + `"`
	case lex.TkTrue:
		if consts.Type.Type != lex.TkTBool {
			p.parseErr("default value format error")
		}
		consts.Value = "true"
	case lex.TkFalse:
		if consts.Type.Type != lex.TkTBool {
			p.parseErr("default value format error")
		}
		consts.Value = "false"
	default:
		p.parseErr("default value format error")
	}

	// 使用 expect 方法检查下一个 token 是否为分号（lex.TkSemi）。
	p.expect(lex.TkSemi)

	// 后面同一行的注释
	t := p.peek()
	if t.Type == lex.TkComment && t.Line == p.token.Line {
		consts.Comment = t.Value.String
		p.next()
	}

	// 将常量信息结构 m 追加到 Consts 字段中。
	p.Consts = append(p.Consts, consts)
}

// parseEnum 方法是 Parser 结构的一个成员方法，用于解析枚举声明。它遍历由词法分析器生成的 token，并根据 token 的类型执行相应的操作。以下是方法的主要步骤：
func (p *Parser) parseEnum() {
	// 创建一个名为 enum 的新 EnumInfo 结构，用于存储枚举信息。
	enum := EnumInfo{}
	enum.TypeComment = p.getPreComments()
	// 使用 expect 方法检查下一个 token 是否为名称，并将其存储在 enum.Name 中。
	p.expect(lex.TkName)
	enum.Name = p.token.Value.String
	// 遍历已解析的枚举列表，检查是否有与当前枚举名称相同的枚举。
	for _, v := range p.Enums {
		// 如果有重复的枚举名称，引发一个解析错误。
		if v.Name == enum.Name {
			p.parseErr(enum.Name + " Redefine.")
		}
	}

	// { 前的注释
	for {
		t := p.peek()
		if t.Type == lex.TkComment {
			if enum.Comment != "" {
				enum.Comment += "\n"
			}
			enum.Comment += t.Value.String
			p.next()
			continue
		}
		break
	}

	// 使用 expect 方法检查下一个 token 是否为左大括号（lex.TkBraceLeft）。
	p.expect(lex.TkBraceLeft)

	// 使用 for 循环遍历 token。在每次迭代中，根据 token 的类型处理枚举成员。各种情况如下：
LFOR:
	for {
		p.next()
		switch p.token.Type {
		case lex.TkBraceRight: // 如果 token 类型为 lex.TkBraceRight（表示枚举声明的结束），则跳出循环。
			break LFOR
		case lex.TkName: // 如果 token 类型为 lex.TkName，则获取成员名称，并根据下一个 token 的类型设置成员值。成员值可以是整数、名称或未指定。
			k := p.token.Value.String
			p.next()
			switch p.token.Type {
			case lex.TkComma: // ,
				m := EnumMember{Key: k, Type: EnumTypeEqual}
				t := p.peek()
				if t.Type == lex.TkComment { // 枚举支持一个注释
					m.Comment = t.Value.String
					p.next()
				}
				enum.Member = append(enum.Member, m)
			case lex.TkBraceRight: // }
				m := EnumMember{Key: k, Type: EnumTypeEqual}
				enum.Member = append(enum.Member, m)
				break LFOR
			case lex.TkEq: // =
				p.next()
				var m EnumMember
				switch p.token.Type {
				case lex.TkInteger: // int
					m = EnumMember{Key: k, Value: int32(p.token.Value.Int)}
				case lex.TkName: // name
					m = EnumMember{Key: k, Type: EnumTypeName, Name: p.token.Value.String}
				default:
					p.parseErr("not expect " + lex.TokenMap[p.token.Type])
				}
				p.next()
				if p.token.Type == lex.TkBraceRight { // }
					enum.Member = append(enum.Member, m)
					break LFOR
				} else if p.token.Type == lex.TkComma { // ,
					t := p.peek()
					if t.Type == lex.TkComment { // 枚举支持一个注释
						m.Comment = t.Value.String
						p.next()
					}
					enum.Member = append(enum.Member, m)
				} else {
					p.parseErr("expect , or }")
				}
			}
		case lex.TkComment:
			m := EnumMember{Type: 3, Comment: p.token.Value.String}
			enum.Member = append(enum.Member, m)

		default:
			// 对于其他 token 类型，引发一个解析错误，指出不期望的 token 类型。

		}

	}

	// 使用 expect 方法检查下一个 token 是否为分号（lex.TkSemi）。
	p.expect(lex.TkSemi)
	// 将枚举信息结构 enum 追加到 Enums 字段中。
	p.Enums = append(p.Enums, enum)
}

// parseStruct 方法是 Parser 结构的一个成员方法，用于解析结构体声明。它遍历由词法分析器生成的 token，并根据 token 的类型执行相应的操作。以下是方法的主要步骤：
func (p *Parser) parseStruct() {
	log.Debug("begin parseStruct")

	st := StructInfo{}
	st.commentTagNum = -9999
	// 使用 getPreComments 方法获取结构体前的注释并存储在 st.Comment 中。
	st.Comment = p.getPreComments()

	// 使用 expect 方法检查下一个 token 是否为名称，并将其存储在 st.Name 中。
	p.expect(lex.TkName)
	st.Name = p.token.Value.String

	// 遍历已解析的结构体列表，检查是否有与当前结构体名称相同的结构体。如果有重复的结构体名称，引发一个解析错误。
	for _, v := range p.Structs {
		if v.Name == st.Name {
			p.parseErr(st.Name + " Redefine.")
		}
	}

	// 注释
	for {
		t := p.peek()
		if t.Type == lex.TkComment {
			st.Comment += t.Value.String
			st.Comment += "\n"
			continue
		}
		break
	}

	// 使用 expect 方法检查下一个 token 是否为左大括号（lex.TkBraceLeft）。
	p.expect(lex.TkBraceLeft)

	// 使用 for 循环遍历 token，解析结构体成员。调用 parseStructMember 方法解析结构体成员，并将其添加到 st.Member 列表中。循环直到 parseStructMember 返回 nil。
	for {
		m := p.parseStructMember()
		if m == nil {
			break
		}
		if m.CommentType != "" {
			st.commentTagNum++
			m.Tag = int32(st.commentTagNum)
		}
		log.Debug("%+v", m)
		st.Member = append(st.Member, *m)
	}
	// 使用 expect 方法检查下一个 token 是否为分号（lex.TkSemi）。
	p.expect(lex.TkSemi) // semicolon at the end of the struct.

	// 调用 checkTag 和 sortTag 方法处理结构体成员的标签。
	// p.sortTag(&st)
	p.checkTag(&st)

	// 将结构体信息结构 st 追加到 Structs 字段中。
	p.Structs = append(p.Structs, st)
}

// parseStructMember 方法是 Parser 结构的一个成员方法，用于解析结构体成员。它遍历由词法分析器生成的 token，并根据 token 的类型执行相应的操作。以下是方法的主要步骤：
func (p *Parser) parseStructMember() *StructMember {
	log.Debug("begin parseStructMember")
	// tag or end
	// 获取下一个 token。
	// 如果 token 类型为 lex.TkBraceRight（表示结构体声明的结束），则返回 nil。
	p.next()
	if p.token.Type == lex.TkBraceRight { // }
		return nil
	}

	log.Debug("1")

	// 是注释
	if p.token.Type == lex.TkComment {
		m := &StructMember{}
		m.CommentType = p.token.Value.String
		return m
	}

	// 否则，检查 token 是否为整数（表示成员标签）。
	if p.token.Type != lex.TkInteger {
		p.parseErr("expect tags.")
	}
	m := &StructMember{}
	m.Tag = int32(p.token.Value.Int)

	log.Debug("2")

	// 获取下一个 token。
	// 如果 token 类型为 lex.TkRequire 或 lex.TkOptional，则设置成员的 Require 属性。否则，引发一个解析错误。
	// require or optional
	p.next()
	if p.token.Type == lex.TkRequire {
		m.Require = true
	} else if p.token.Type == lex.TkOptional {
		m.Require = false
	} else {
		p.parseErr("expect require or optional")
	}
	log.Debug("5")

	// 获取下一个 token。如果 token 是一个有效的类型或名称，调用 parseType 方法解析类型并将其存储在 m.Type 中。否则，引发一个解析错误。
	// type
	p.next()
	if !lex.IsType(p.token.Type) && p.token.Type != lex.TkName && p.token.Type != lex.TkUnsigned {
		p.parseErr("expect type")
	} else {
		m.Type = p.parseType()
	}

	// 使用 expect 方法检查下一个 token 是否为名称，并将其存储在 m.Key 中。
	// key
	p.expect(lex.TkName)
	m.Key = p.token.Value.String

	log.Debug("8")
	// 获取下一个 token。根据 token 的类型，处理成员的默认值、数组类型或其他情况。如果遇到不符合预期的 token 类型，引发一个解析错误。
	p.next()
	if p.token.Type == lex.TkSemi { // ;
		t := p.peek()
		if t.Type == lex.TkComment && t.Line == p.token.Line {
			m.Comment = t.Value.String
			p.next()
		}
		return m
	}
	if p.token.Type == lex.TkSquareLeft { // [
		p.expect(lex.TkInteger)
		m.Type = &VarType{Type: lex.TkTArray, TypeK: m.Type, TypeL: p.token.Value.Int}
		p.expect(lex.TkSquarerRight)
		p.expect(lex.TkSemi)
		return m
	}
	if p.token.Type != lex.TkEq {
		p.parseErr("expect ; or =")
	}
	if p.token.Type == lex.TkTMap || p.token.Type == lex.TkTVector || p.token.Type == lex.TkName {
		p.parseErr("map, vector, custom type cannot set default value")
	}

	log.Debug("9")
	// default
	// 使用 expect 方法检查下一个 token 是否为分号（lex.TkSemi）。
	p.next()
	p.parseStructMemberDefault(m)
	p.expect(lex.TkSemi) // ;

	log.Debug("11")
	// 后面同一行的注释
	t := p.peek()
	// log.Debug("a: %+v %+v %v", lex.TokenMap[t.Type], *t.Value, t.Line)
	// log.Debug("b: %+v, %+v, %+v", lex.TokenMap[p.token.Type], p.token.Line, p.token.Value == nil)
	if t.Type == lex.TkComment && t.Line == p.token.Line {
		m.Comment = t.Value.String
		p.next()
	}

	return m
}

func (p *Parser) makeUnsigned(utype *VarType) {
	switch utype.Type {
	case lex.TkTInt, lex.TkTShort, lex.TkTByte:
		utype.Unsigned = true
	default:
		p.parseErr("type " + lex.TokenMap[utype.Type] + " unsigned decoration is not supported")
	}
}

func (p *Parser) parseType() *VarType {
	vtype := &VarType{Type: p.token.Type}

	switch vtype.Type {
	case lex.TkName:
		vtype.TypeSt = p.token.Value.String
	case lex.TkTInt, lex.TkTBool, lex.TkTShort, lex.TkTLong, lex.TkTByte, lex.TkTFloat, lex.TkTDouble, lex.TkTString:
		// no nothing
	case lex.TkTVector:
		p.expect(lex.TkShl)
		p.next()
		vtype.TypeK = p.parseType()
		p.expect(lex.TkShr)
	case lex.TkTMap:
		p.expect(lex.TkShl)
		p.next()
		vtype.TypeK = p.parseType()
		p.expect(lex.TkComma)
		p.next()
		vtype.TypeV = p.parseType()
		p.expect(lex.TkShr)
	case lex.TkUnsigned:
		p.next()
		utype := p.parseType()
		p.makeUnsigned(utype)
		return utype
	default:
		p.parseErr("expert type")
	}
	return vtype
}

func (p *Parser) parseStructMemberDefault(m *StructMember) {
	m.DefType = p.token.Type
	switch p.token.Type {
	case lex.TkInteger:
		if !lex.IsNumberType(m.Type.Type) && m.Type.Type != lex.TkName {
			// enum auto defined type ,default value is number.
			p.parseErr("type does not accept number")
		}
		m.Default = p.token.Value.String
	case lex.TkFloat:
		if !lex.IsNumberType(m.Type.Type) {
			p.parseErr("type does not accept number")
		}
		m.Default = p.token.Value.String
	case lex.TkString:
		if lex.IsNumberType(m.Type.Type) {
			p.parseErr("type does not accept string")
		}
		m.Default = `"` + p.token.Value.String + `"`
	case lex.TkTrue:
		if m.Type.Type != lex.TkTBool {
			p.parseErr("default value format error")
		}
		m.Default = "true"
	case lex.TkFalse:
		if m.Type.Type != lex.TkTBool {
			p.parseErr("default value format error")
		}
		m.Default = "false"
	case lex.TkName:
		m.Default = p.token.Value.String
	default:
		p.parseErr("default value format error")
	}
}

func (p *Parser) checkTag(st *StructInfo) {
	log.Debug("begin check Tag")
	set := make(map[int32]bool)

	for _, v := range st.Member {
		if set[v.Tag] {
			p.parseErr("tag = " + strconv.Itoa(int(v.Tag)) + ". have duplicates")
		}
		set[v.Tag] = true
	}
}

func (p *Parser) sortTag(st *StructInfo) {
	log.Debug("begin sort Tag")
	sort.Sort(StructMemberSorter(st.Member))
}

// Looking for the true type of user-defined identifier
func (p *Parser) findTNameType(tname string) (lex.TokenType, string, string) {
	log.Debug("begin findTNameType")
	for _, v := range p.Structs {
		if p.Module+"::"+v.Name == tname {
			return lex.TkStruct, p.Module, p.ProtoName
		}
	}

	for _, v := range p.Enums {
		if p.Module+"::"+v.Name == tname {
			return lex.TkEnum, p.Module, p.ProtoName
		}
	}

	for _, pInc := range p.IncParse {
		ret, mod, protoName := pInc.findTNameType(tname)
		if ret != lex.TkName {
			return ret, mod, protoName
		}
	}
	// not find
	return lex.TkName, p.Module, p.ProtoName
}

func (p *Parser) findEnumName(ename string) (*EnumMember, *EnumInfo) {
	if strings.Contains(ename, "::") {
		vec := strings.Split(ename, "::")
		if len(vec) >= 2 {
			ename = vec[1]
		}
	}
	var cmb *EnumMember
	var cenum *EnumInfo
	for ek, enum := range p.Enums {
		for mk, mb := range enum.Member {
			if mb.Key != ename {
				continue
			}
			if cmb == nil {
				cmb = &enum.Member[mk]
				cenum = &p.Enums[ek]
			} else {
				p.parseErr(ename + " name conflict [" + cenum.Name + "::" + cmb.Key + " or " + enum.Name + "::" + mb.Key)
				return nil, nil
			}
		}
	}
	for _, pInc := range p.IncParse {
		if cmb == nil {
			cmb, cenum = pInc.findEnumName(ename)
		} else {
			break
		}
	}
	if cenum != nil && cenum.Module == "" {
		cenum.Module = p.Module
	}
	return cmb, cenum
}

func addToSet(m *map[string]bool, module string) {
	if *m == nil {
		*m = make(map[string]bool)
	}
	(*m)[module] = true
}

func (p *Parser) checkDepTName(ty *VarType, dm *map[string]bool, dmj *map[string]string) {
	log.Debug("being checkDepTName, %+v, %+v, %+v", ty, dm, dmj)
	if ty == nil {
		return
	}
	if ty.Type == lex.TkName {
		name := ty.TypeSt
		if strings.Count(name, "::") == 0 {
			name = p.Module + "::" + name
		}

		mod := ""
		ty.CType, mod, _ = p.findTNameType(name)

		if ty.CType == lex.TkName {
			p.parseErr(ty.TypeSt + " not find define")
		}
		if mod != p.Module {
			addToSet(dm, mod)
		} else {
			// the same Module ,do not add self.
			ty.TypeSt = strings.Replace(ty.TypeSt, mod+"::", "", 1)
		}

	} else if ty.Type == lex.TkTVector {
		p.checkDepTName(ty.TypeK, dm, dmj)
	} else if ty.Type == lex.TkTMap {
		p.checkDepTName(ty.TypeK, dm, dmj)
		p.checkDepTName(ty.TypeV, dm, dmj)
	}

	log.Debug("end checkDepTName")
}

// analysis custom type，whether have definition
func (p *Parser) analyzeTName() {
	for i, v := range p.Structs {
		for _, v := range v.Member {
			ty := v.Type
			p.checkDepTName(ty, &p.Structs[i].DependModule, &p.Structs[i].DependModuleWithJce)
		}
	}

	log.Debug("end analyzeTName")
}

func (p *Parser) analyzeDefault() {
	for _, v := range p.Structs {
		for i, r := range v.Member {
			if r.Default != "" && r.DefType == lex.TkName {
				mb, enum := p.findEnumName(r.Default)

				if mb == nil || enum == nil {
					p.parseErr("can not find default value" + r.Default)
				}

				defValue := enum.Name + "_" + utils.UpperFirstLetter(mb.Key)

				var currModule string
				currModule = p.Module

				if len(enum.Module) > 0 && currModule != enum.Module {
					defValue = enum.Module + "." + defValue
				}
				v.Member[i].Default = defValue
			}
		}
	}
	log.Debug("end analyzeDefault")
}

// 分析文件的依赖关系
func (p *Parser) analyzeDepend() {
	for _, v := range p.Includes {
		relativePath := path.Dir(p.Filepath)
		dependFile := relativePath + "/" + v
		pInc := ParseFile(dependFile, p.IncChain)
		p.IncParse = append(p.IncParse, pInc)
	}

	p.analyzeDefault()
	p.analyzeTName()
	log.Debug("end analyzeDepend")
}
