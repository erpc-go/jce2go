package main

import (
	"io/ioutil"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Parse record information of parse file.
type Parse struct {
	Source string // 源文件

	Module   string   // 包名
	Includes []string // 依赖的其他 jce 文件

	Structs []StructInfo // struct list
	Enums   []EnumInfo   // enum list
	Consts  []ConstInfo  // const list

	// have parsed include file
	IncParse []*Parse

	lex   *LexState
	t     *Token
	lastT *Token

	// jce include chain
	IncChain []string

	// proto file name(not include .jce)
	ProtoName string

	fileNames map[string]bool
}

func newParse(s string, b []byte, incChain []string) *Parse {
	p := &Parse{
		Source:    s,
		ProtoName: path2ProtoName(s),
	}

	for _, v := range incChain {
		if s == v {
			panic("jce circular reference: " + s)
		}
	}

	incChain = append(incChain, s)
	p.IncChain = incChain
	p.lex = NewLexState(s, b)
	p.fileNames = map[string]bool{}

	return p
}

func (p *Parse) parse() {
OUT:
	for {
		p.next()
		t := p.t
		switch t.T {
		case tkEos:
			break OUT
		case tkInclude:
			p.parseInclude()
		case tkModule:
			p.parseModule()
		default:
			p.parseErr("Expect include or module.")
		}
	}
	p.analyzeDepend()
}

// VarType contains variable type(token)
type VarType struct {
	Type     TK       // basic type
	Unsigned bool     // whether unsigned
	TypeSt   string   // custom type name, such as an enumerated struct,at this time Type=tkName
	CType    TK       // make sure which type of custom type is,tkEnum, tkStruct
	TypeK    *VarType // vector's member variable,the key of map
	TypeV    *VarType // the value of map
	TypeL    int64    // length of array
}

// StructMember member struct.
type StructMember struct {
	Tag       int32
	Require   bool
	Type      *VarType
	Key       string // after the uppercase converted key
	OriginKey string // original key
	Default   string
	DefType   TK
}

// StructMemberSorter When serializing, make sure the tags are ordered.
type StructMemberSorter []StructMember

func (a StructMemberSorter) Len() int           { return len(a) }
func (a StructMemberSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StructMemberSorter) Less(i, j int) bool { return a[i].Tag < a[j].Tag }

// StructInfo record struct information.
type StructInfo struct {
	Name                string
	Member              []StructMember
	DependModule        map[string]bool
	DependModuleWithJce map[string]string
}

// 1. struct rename
// struct Name { 1 require Mb type}
func (st *StructInfo) rename() {
	st.Name = upperFirstLetter(st.Name)

	for i := range st.Member {
		st.Member[i].OriginKey = st.Member[i].Key
		st.Member[i].Key = upperFirstLetter(st.Member[i].Key)
	}
}

// EnumMember record member information.
type EnumMember struct {
	Key   string
	Type  int
	Value int32  //type 0  // TODO: 这个 type 是啥？name 又是啥？
	Name  string //type 1
}

// EnumInfo record EnumMember information include name.
type EnumInfo struct {
	Module string
	Name   string
	Member []EnumMember
}

// enum 变量重命名，即把首字母都大写
func (en *EnumInfo) rename() {
	en.Name = upperFirstLetter(en.Name)
	for i := range en.Member {
		en.Member[i].Key = upperFirstLetter(en.Member[i].Key)
	}
}

// ConstInfo record const information.
type ConstInfo struct {
	Type  *VarType
	Name  string
	Value string
}

func (cst *ConstInfo) rename() {
	cst.Name = upperFirstLetter(cst.Name)
}

// HashKeyInfo record hash key information.
type HashKeyInfo struct {
	Name   string
	Member []string
}

func (p *Parse) parseErr(err string) {
	line := "0"
	if p.t != nil {
		line = strconv.Itoa(p.t.Line)
	}

	panic(p.Source + ": " + line + ". " + err)
}

func (p *Parse) next() {
	p.lastT = p.t
	p.t = p.lex.NextToken()
}

func (p *Parse) expect(t TK) {
	p.next()
	if p.t.T != t {
		p.parseErr("expect " + TokenMap[t])
	}
}

func (p *Parse) makeUnsigned(utype *VarType) {
	switch utype.Type {
	case tkTInt, tkTShort, tkTByte:
		utype.Unsigned = true
	default:
		p.parseErr("type " + TokenMap[utype.Type] + " unsigned decoration is not supported")
	}
}

func (p *Parse) parseType() *VarType {
	vtype := &VarType{Type: p.t.T}

	switch vtype.Type {
	case tkName:
		vtype.TypeSt = p.t.S.S
	case tkTInt, tkTBool, tkTShort, tkTLong, tkTByte, tkTFloat, tkTDouble, tkTString:
		// no nothing
	case tkTVector:
		p.expect(tkShl)
		p.next()
		vtype.TypeK = p.parseType()
		p.expect(tkShr)
	case tkTMap:
		p.expect(tkShl)
		p.next()
		vtype.TypeK = p.parseType()
		p.expect(tkComma)
		p.next()
		vtype.TypeV = p.parseType()
		p.expect(tkShr)
	case tkUnsigned:
		p.next()
		utype := p.parseType()
		p.makeUnsigned(utype)
		return utype
	default:
		p.parseErr("expert type")
	}
	return vtype
}

func (p *Parse) parseEnum() {
	enum := EnumInfo{}
	p.expect(tkName)
	enum.Name = p.t.S.S
	for _, v := range p.Enums {
		if v.Name == enum.Name {
			p.parseErr(enum.Name + " Redefine.")
		}
	}
	p.expect(tkBraceLeft)

LFOR:
	for {
		p.next()
		switch p.t.T {
		case tkBraceRight:
			break LFOR
		case tkName:
			k := p.t.S.S
			p.next()
			switch p.t.T {
			case tkComma:
				m := EnumMember{Key: k, Type: 2}
				enum.Member = append(enum.Member, m)
			case tkBraceRight:
				m := EnumMember{Key: k, Type: 2}
				enum.Member = append(enum.Member, m)
				break LFOR
			case tkEq:
				p.next()
				switch p.t.T {
				case tkInteger:
					m := EnumMember{Key: k, Value: int32(p.t.S.I)}
					enum.Member = append(enum.Member, m)
				case tkName:
					m := EnumMember{Key: k, Type: 1, Name: p.t.S.S}
					enum.Member = append(enum.Member, m)
				default:
					p.parseErr("not expect " + TokenMap[p.t.T])
				}
				p.next()
				if p.t.T == tkBraceRight {
					break LFOR
				} else if p.t.T == tkComma {
				} else {
					p.parseErr("expect , or }")
				}
			}
		}
	}
	p.expect(tkSemi)
	p.Enums = append(p.Enums, enum)
}

func (p *Parse) parseStructMemberDefault(m *StructMember) {
	m.DefType = p.t.T
	switch p.t.T {
	case tkInteger:
		if !isNumberType(m.Type.Type) && m.Type.Type != tkName {
			// enum auto defined type ,default value is number.
			p.parseErr("type does not accept number")
		}
		m.Default = p.t.S.S
	case tkFloat:
		if !isNumberType(m.Type.Type) {
			p.parseErr("type does not accept number")
		}
		m.Default = p.t.S.S
	case tkString:
		if isNumberType(m.Type.Type) {
			p.parseErr("type does not accept string")
		}
		m.Default = `"` + p.t.S.S + `"`
	case tkTrue:
		if m.Type.Type != tkTBool {
			p.parseErr("default value format error")
		}
		m.Default = "true"
	case tkFalse:
		if m.Type.Type != tkTBool {
			p.parseErr("default value format error")
		}
		m.Default = "false"
	case tkName:
		m.Default = p.t.S.S
	default:
		p.parseErr("default value format error")
	}
}

func (p *Parse) parseStructMember() *StructMember {
	// tag or end
	p.next()
	if p.t.T == tkBraceRight {
		return nil
	}
	if p.t.T != tkInteger {
		p.parseErr("expect tags.")
	}
	m := &StructMember{}
	m.Tag = int32(p.t.S.I)

	// require or optional
	p.next()
	if p.t.T == tkRequire {
		m.Require = true
	} else if p.t.T == tkOptional {
		m.Require = false
	} else {
		p.parseErr("expect require or optional")
	}

	// type
	p.next()
	if !isType(p.t.T) && p.t.T != tkName && p.t.T != tkUnsigned {
		p.parseErr("expect type")
	} else {
		m.Type = p.parseType()
	}

	// key
	p.expect(tkName)
	m.Key = p.t.S.S

	p.next()
	if p.t.T == tkSemi {
		return m
	}
	if p.t.T == tkSquareLeft {
		p.expect(tkInteger)
		m.Type = &VarType{Type: tkTArray, TypeK: m.Type, TypeL: p.t.S.I}
		p.expect(tkSquarerRight)
		p.expect(tkSemi)
		return m
	}
	if p.t.T != tkEq {
		p.parseErr("expect ; or =")
	}
	if p.t.T == tkTMap || p.t.T == tkTVector || p.t.T == tkName {
		p.parseErr("map, vector, custom type cannot set default value")
	}

	// default
	p.next()
	p.parseStructMemberDefault(m)
	p.expect(tkSemi)

	return m
}

func (p *Parse) checkTag(st *StructInfo) {
	set := make(map[int32]bool)

	for _, v := range st.Member {
		if set[v.Tag] {
			p.parseErr("tag = " + strconv.Itoa(int(v.Tag)) + ". have duplicates")
		}
		set[v.Tag] = true
	}
}

func (p *Parse) sortTag(st *StructInfo) {
	sort.Sort(StructMemberSorter(st.Member))
}

func (p *Parse) parseStruct() {
	st := StructInfo{}
	p.expect(tkName)
	st.Name = p.t.S.S
	for _, v := range p.Structs {
		if v.Name == st.Name {
			p.parseErr(st.Name + " Redefine.")
		}
	}
	p.expect(tkBraceLeft)

	for {
		m := p.parseStructMember()
		if m == nil {
			break
		}
		st.Member = append(st.Member, *m)
	}
	p.expect(tkSemi) //semicolon at the end of the struct.

	p.checkTag(&st)
	p.sortTag(&st)

	p.Structs = append(p.Structs, st)
}

func (p *Parse) parseConst() {
	m := ConstInfo{}

	// type
	p.next()
	switch p.t.T {
	case tkTVector, tkTMap:
		p.parseErr("const no supports type vector or map.")
	case tkTBool, tkTByte, tkTShort,
		tkTInt, tkTLong, tkTFloat,
		tkTDouble, tkTString, tkUnsigned:
		m.Type = p.parseType()
	default:
		p.parseErr("expect type.")
	}

	p.expect(tkName)
	m.Name = p.t.S.S

	p.expect(tkEq)

	// default
	p.next()
	switch p.t.T {
	case tkInteger, tkFloat:
		if !isNumberType(m.Type.Type) {
			p.parseErr("type does not accept number")
		}
		m.Value = p.t.S.S
	case tkString:
		if isNumberType(m.Type.Type) {
			p.parseErr("type does not accept string")
		}
		m.Value = `"` + p.t.S.S + `"`
	case tkTrue:
		if m.Type.Type != tkTBool {
			p.parseErr("default value format error")
		}
		m.Value = "true"
	case tkFalse:
		if m.Type.Type != tkTBool {
			p.parseErr("default value format error")
		}
		m.Value = "false"
	default:
		p.parseErr("default value format error")
	}
	p.expect(tkSemi)

	p.Consts = append(p.Consts, m)
}

func (p *Parse) parseHashKey() {
	hashKey := HashKeyInfo{}
	p.expect(tkSquareLeft)
	p.expect(tkName)
	hashKey.Name = p.t.S.S
	p.expect(tkComma)
	for {
		p.expect(tkName)
		hashKey.Member = append(hashKey.Member, p.t.S.S)
		p.next()
		t := p.t
		switch t.T {
		case tkSquarerRight:
			p.expect(tkSemi)
			return
		case tkComma:
		default:
			p.parseErr("expect ] or ,")
		}
	}
}

func (p *Parse) parseModuleSegment() {
	p.expect(tkBraceLeft)

	for {
		p.next()
		t := p.t
		switch t.T {
		case tkBraceRight:
			p.expect(tkSemi)
			return
		case tkConst:
			p.parseConst()
		case tkEnum:
			p.parseEnum()
		case tkStruct:
			p.parseStruct()
		case tkKey:
			p.parseHashKey()
		default:
			p.parseErr("not except " + TokenMap[t.T])
		}
	}
}

func (p *Parse) parseModule() {
	p.expect(tkName)

	if p.Module != "" {
		// 解决一个tars文件中定义多个module
		name := p.ProtoName + "_" + p.t.S.S + ".tars"
		newp := newParse(name, nil, nil)
		newp.IncChain = p.IncChain
		newp.lex = p.lex
		newp.Includes = p.Includes
		newp.IncParse = p.IncParse
		cowp := *p
		newp.IncParse = append(newp.IncParse, &cowp)
		newp.Module = p.t.S.S
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
	} else {
		p.Module = p.t.S.S
		p.parseModuleSegment()
	}
}

func (p *Parse) parseInclude() {
	p.expect(tkString)
	p.Includes = append(p.Includes, p.t.S.S)
}

// Looking for the true type of user-defined identifier
func (p *Parse) findTNameType(tname string) (TK, string, string) {
	for _, v := range p.Structs {
		if p.Module+"::"+v.Name == tname {
			return tkStruct, p.Module, p.ProtoName
		}
	}

	for _, v := range p.Enums {
		if p.Module+"::"+v.Name == tname {
			return tkEnum, p.Module, p.ProtoName
		}
	}

	for _, pInc := range p.IncParse {
		ret, mod, protoName := pInc.findTNameType(tname)
		if ret != tkName {
			return ret, mod, protoName
		}
	}
	// not find
	return tkName, p.Module, p.ProtoName
}

func (p *Parse) findEnumName(ename string) (*EnumMember, *EnumInfo) {
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

func (p *Parse) checkDepTName(ty *VarType, dm *map[string]bool, dmj *map[string]string) {
	if ty.Type == tkName {
		name := ty.TypeSt
		if strings.Count(name, "::") == 0 {
			name = p.Module + "::" + name
		}

		mod := ""
		ty.CType, mod, _ = p.findTNameType(name)

		if ty.CType == tkName {
			p.parseErr(ty.TypeSt + " not find define")
		}
		if mod != p.Module {
			addToSet(dm, mod)
		} else {
			// the same Module ,do not add self.
			ty.TypeSt = strings.Replace(ty.TypeSt, mod+"::", "", 1)
		}

	} else if ty.Type == tkTVector {
		p.checkDepTName(ty.TypeK, dm, dmj)
	} else if ty.Type == tkTMap {
		p.checkDepTName(ty.TypeK, dm, dmj)
		p.checkDepTName(ty.TypeV, dm, dmj)
	}
}

// analysis custom type，whether have definition
func (p *Parse) analyzeTName() {
	for i, v := range p.Structs {
		for _, v := range v.Member {
			ty := v.Type
			p.checkDepTName(ty, &p.Structs[i].DependModule, &p.Structs[i].DependModuleWithJce)
		}
	}
}

func (p *Parse) analyzeDefault() {
	for _, v := range p.Structs {
		for i, r := range v.Member {
			if r.Default != "" && r.DefType == tkName {
				mb, enum := p.findEnumName(r.Default)

				if mb == nil || enum == nil {
					p.parseErr("can not find default value" + r.Default)
				}

				defValue := enum.Name + "_" + upperFirstLetter(mb.Key)

				var currModule string
				currModule = p.Module

				if len(enum.Module) > 0 && currModule != enum.Module {
					defValue = enum.Module + "." + defValue
				}
				v.Member[i].Default = defValue
			}
		}
	}
}

// TODO analysis key[]，have quoted the correct struct and member name.
func (p *Parse) analyzeHashKey() {

}

func (p *Parse) analyzeDepend() {
	for _, v := range p.Includes {
		relativePath := path.Dir(p.Source)
		dependFile := relativePath + "/" + v
		pInc := ParseFile(dependFile, p.IncChain)
		p.IncParse = append(p.IncParse, pInc)
	}

	p.analyzeDefault()
	p.analyzeTName()
	p.analyzeHashKey()
}

func path2ProtoName(path string) string {
	iBegin := strings.LastIndex(path, "/")
	if iBegin == -1 || iBegin >= len(path)-1 {
		iBegin = 0
	} else {
		iBegin++
	}
	iEnd := strings.LastIndex(path, ".jce")
	if iEnd == -1 {
		iEnd = len(path)
	}

	return path[iBegin:iEnd]
}

// ParseFile parse a file,return grammar tree.
func ParseFile(filePath string, incChain []string) *Parse {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic("file read error: " + filePath + ". " + err.Error())
	}

	p := newParse(filePath, b, incChain)
	p.parse()

	return p
}
