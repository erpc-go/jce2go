package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/erpc-go/jce2go/log"
)

// 全局 map 避免重复生成
var (
	fileMap = make(map[string]bool, 0)
)

// Generate record go code information.
// 代码生成
type Generate struct {
	I         []string     // imports with path
	code      bytes.Buffer // 最终生成的代码
	vc        int          // var count. Used to generate unique variable names
	filepath  string       // 当前解析的 jce 文件
	codecPath string       // 生成后的代码依赖的基础 codec 代码
	module    string       // 包名
	prefix    string       // 最终的生成目录
	p         *Parse       // 当前文件生成的语法分析树
}

// NewGenerate build up a new path
func NewGenerate(path string, module string, outdir string) *Generate {
	if outdir != "" {
		b := []byte(outdir)
		last := b[len(b)-1:]
		if string(last) != "/" {
			outdir += "/"
		}
	}

	return &Generate{
		I:        []string{},
		code:     bytes.Buffer{},
		vc:       0,
		filepath: path,

		codecPath: "github.com/edte/erpc/codec/jce",
		module:    module,
		prefix:    outdir,
		p:         &Parse{},
	}
}

// Gen to parse file.
func (gen *Generate) Gen() {
	// recover  panic
	defer func() {
		if err := recover(); err != nil {
			log.Raw(err)
			os.Exit(1)
		}
	}()

	// 解析文件
	gen.p = ParseFile(gen.filepath, make([]string, 0))

	b, _ := json.Marshal(gen.p)
	log.Debugf(string(b))

	// 开始代码生成
	gen.genAll()
}

func (gen *Generate) genAll() {
	if fileMap[gen.filepath] {
		return
	}

	if len(gen.p.Enums) == 0 && len(gen.p.Consts) == 0 && len(gen.p.Structs) == 0 {
		return
	}

	gen.genIncludeFiles()
	gen.genFileComment()
	gen.genPackage()
	gen.genEnums()
	gen.genConst()
	gen.genStructs()
	gen.saveFiles()

	fileMap[gen.filepath] = true
}

// 先生成依赖的其他文件，即 include 的其他文件
func (gen *Generate) genIncludeFiles() {
	for _, v := range gen.p.IncParse {
		NewGenerate(v.Source, gen.module, gen.prefix).genAll()
	}
}

// genFileComment 写文件注释
func (gen *Generate) genFileComment() {
	gen.writeString(`// DO NOT EDIT IT.` + ` 
// code generated by jce2go ` + VERSION + `. 
// source: ` + filepath.Base(gen.filepath) + `
`)
}

// genPackage 写包名、导第三方包
func (gen *Generate) genPackage() {
	// [setp 1] 包名
	gen.writeString("package " + gen.p.Module + "\n\n")

	// [step 2] 导基础包
	gen.writeString(`
import (
	"fmt"
    "io"

`)

	// [step 3] 包依赖的第三方包
	gen.genImports()

	// [step 4] 写占位
	gen.writeString(`)

// 占位使用，避免导入的这些包没有被使用
var _ = fmt.Errorf
var _ = io.ReadFull
var _ = jce.BYTE

`)

}

// 导第三方包
func (gen *Generate) genImports() {
	// [step 1] 导 jce 编码包
	gen.writeString("\"" + gen.codecPath + "\"\n")

	// [step 2] 导 struct 依赖的包
	for _, st := range gen.p.Structs {
		for k := range st.DependModule {
			gen.genStructImport(k)
		}
	}
}

// 导 struct 依赖的包
func (gen *Generate) genStructImport(module string) {
	moduleStr := module

	for _, p := range gen.I {
		if strings.HasSuffix(p, "/"+moduleStr) {
			gen.writeString(`"` + moduleStr + `"` + "\n")
			return
		}
	}

	if gen.module == "" {
		return
	}

	mf := filepath.Clean(filepath.Join(gen.module, gen.prefix))

	if runtime.GOOS == "windows" {
		mf = strings.ReplaceAll(mf, string(os.PathSeparator), string('/'))
	}

	moduleStr = fmt.Sprintf("%s/%s", mf, moduleStr)

	gen.writeString(`"` + moduleStr + `"` + "\n")

	return
}

// 写枚举
func (gen *Generate) genEnums() {
	for _, v := range gen.p.Enums {
		gen.genEnum(&v)
	}
}

func (gen *Generate) genEnum(en *EnumInfo) {
	if len(en.Member) == 0 {
		return
	}

	en.rename()

	gen.writeString("// enum " + en.Name + " implement\n")
	gen.writeString("type " + en.Name + " int32\n")
	gen.writeString("const (\n")

	var it int32

	for _, v := range en.Member {
		if v.Type == 0 {
			//use value
			gen.writeString(gen.makeEnumName(en, &v) + " " + en.Name + ` = ` + strconv.Itoa(int(v.Value)) + "\n")
			it = v.Value + 1
			continue
		}

		if v.Type == 1 {
			// use name
			find := false

			for _, ref := range en.Member {
				if ref.Key == v.Name {
					find = true
					gen.writeString(gen.makeEnumName(en, &v) + " " + en.Name + ` = ` + gen.makeEnumName(en, &ref) + "\n")
					it = ref.Value + 1
					break
				}

				if ref.Key == v.Key {
					break
				}
			}

			if !find {
				panic(v.Name + " not define before use.")
			}
			continue
		}

		// use auto add
		gen.writeString(gen.makeEnumName(en, &v) + " " + en.Name + ` = ` + strconv.Itoa(int(it)) + "\n")
		it++
	}

	gen.writeString(")\n\n")
}

// typeName + constName
// 首字母大写
func (gen *Generate) makeEnumName(en *EnumInfo, mb *EnumMember) string {
	return upperFirstLetter(en.Name) + upperFirstLetter(mb.Key)
}

// 生成 const
func (gen *Generate) genConst() {
	if len(gen.p.Consts) == 0 {
		return
	}

	gen.writeString("// const implement")
	gen.writeString("\nconst (\n")

	for _, v := range gen.p.Consts {
		v.rename()
		gen.writeString(v.Name + " " + gen.genType(v.Type) + " = " + v.Value + "\n")
	}

	gen.writeString(")\n")
}

// 生成 struct
func (gen *Generate) genStructs() {
	for _, v := range gen.p.Structs {
		gen.genStruct(&v)
	}
}

func (gen *Generate) genStruct(st *StructInfo) {
	gen.vc = 0
	st.rename()

	gen.genStructDefine(st)
	gen.genFunResetDefault(st)

	gen.genFunReadFrom(st)
	gen.genFunWriteTo(st)
}

// 生成 struct 的定义
// 默认生成 json、tag
func (gen *Generate) genStructDefine(st *StructInfo) {
	gen.writeString("// " + st.Name + " struct implement\n")
	gen.writeString("type " + st.Name + " struct {\n")

	for _, v := range st.Member {
		if jsonOmitEmpty {
			gen.writeString("\t" + v.Key + " " + gen.genType(v.Type) + " `json:\"" + v.OriginKey + ",omitempty\"`\n")
			continue
		}

		gen.writeString("\t" + v.Key + " " + gen.genType(v.Type) + " `json:\"" + v.OriginKey + `"` + ` tag:"` + strconv.Itoa(int(v.Tag)) + "\"`\n")
	}

	gen.writeString("}\n")
}

// 生成 struct optional 成员的默认赋值
func (gen *Generate) genFunResetDefault(st *StructInfo) {
	gen.writeString("\nfunc (st *" + st.Name + ") resetDefault() {\n")

	for _, v := range st.Member {
		if v.Type.CType == tkStruct {
			gen.writeString("st." + v.Key + ".resetDefault()\n")
		}
		if v.Default == "" {
			continue
		}
		gen.writeString("st." + v.Key + " = " + v.Default + "\n")
	}

	gen.writeString("}\n")
}

// 实现反序列化
func (gen *Generate) genFunReadFrom(st *StructInfo) {
	gen.writeString("\n" + `// ReadFrom reads from io.Reader and put into struct.
func (st *` + st.Name + `) ReadFrom(r io.Reader) (n int64, err error) {
	var (
		have bool
		ty jce.JceEncodeType
	)

    decoder := jce.NewDecoder(r)
	st.resetDefault()

`)

	for _, v := range st.Member {
		gen.genReadVar(&v, "st.")
	}

	gen.code.WriteString(`
	_ = err
	_ = have
	_ = ty
	return 
}
`)
}

// 生成 struct 的成员
func (gen *Generate) genReadVar(v *StructMember, prefix string) {
	gen.writeString("    // [step " + strconv.Itoa(int(v.Tag)) + "] read " + v.Key)

	require := "false"
	if v.Require {
		require = "true"
	}

	switch v.Type.Type {
	case tkTVector:
		gen.genReadVector(v, prefix)
	case tkTArray:
		gen.genReadVector(v, prefix)
	case tkTMap:
		gen.genReadMap(v, prefix)
	case tkName: //TODO: 这是啥？
		if v.Type.CType == tkEnum {
			require := "false"
			if v.Require {
				require = "true"
			}

			gen.writeString(`
    if err = decoder.ReadInt32((*int32)(&` + prefix + v.Key + `),` + strconv.Itoa(int(v.Tag)) + `, ` + require + `); err !=nil {
        return
    }
`)
			return

		}

		gen.writeString(`
    if _, err = ` + prefix + v.Key + `.ReadFrom(decoder.Reader()); err !=nil {
        return
    }
`)

	default: // 默认基础类型，即非 list、map、
		gen.writeString(`
    if err = decoder.Read` + upperFirstLetter(gen.genType(v.Type)) + `(&` + prefix + v.Key + `, ` + strconv.Itoa(int(v.Tag)) + `, ` + require + `); err != nil {
        return         
    }
`)
	}
}

// 序列化 vector
func (gen *Generate) genReadVector(mb *StructMember, prefix string) {
	tag := strconv.Itoa(int(mb.Tag))
	vc := strconv.Itoa(gen.vc)
	gen.vc++

	require := "false"
	if mb.Require {
		require = "true"
	}

	// SimpleList
	if mb.Type.TypeK.Type == tkTByte {
		if mb.Type.TypeK.Unsigned {
			gen.writeString(`
    if err = decoder.ReadSliceUint8(&` + gen.genVariableName(prefix, mb.Key) + `,` + tag + `,` + require + `); err != nil {
        return
    }
`)
			return
		}

		gen.writeString(`
    if err = decoder.ReadSliceInt8(&` + gen.genVariableName(prefix, mb.Key) + `,` + tag + `,` + require + `); err != nil {
        return
    }
`)
		return
	}

	// LIST
	gen.writeString(`
    var length` + vc + ` uint32

    // [step ` + strconv.Itoa(int(mb.Tag)) + `.1] read type、tag        
    if ty, have, err = decoder.ReadHead(` + tag + `,` + require + ` );err != nil || !have {
        return
    } 
    // [step ` + strconv.Itoa(int(mb.Tag)) + `.2] read list length        
    if length` + vc + `, err = decoder.ReadLength(); err !=nil {
        return
    }
    // [step ` + strconv.Itoa(int(mb.Tag)) + `.3] read data        
    ` + gen.genVariableName(prefix, mb.Key) + ` = make(` + gen.genType(mb.Type) + `, length` + vc + `)`)

	gen.writeString(`  
    for i` + vc + `:= uint32(0); i` + vc + `< length` + vc + `; i` + vc + `++ {
`)

	dummy := &StructMember{
		Type: mb.Type.TypeK,
		Key:  mb.Key + "[i" + vc + "]",
	}

	gen.genReadVar(dummy, prefix)

	gen.writeString(`
	}
	`)
}

// 反序列化 map
func (gen *Generate) genReadMap(mb *StructMember, prefix string) {
	require := "false"
	if mb.Require {
		require = "true"
	}
	vc := strconv.Itoa(gen.vc)
	gen.vc++

	gen.writeString(`
    var length` + vc + ` uint32

    // [step ` + strconv.Itoa(int(mb.Tag)) + `.1] read type、tag
    if ty, have, err = decoder.ReadHead(` + strconv.Itoa(int(mb.Tag)) + "," + require + `); err != nil {
        return
    }
    // [step ` + strconv.Itoa(int(mb.Tag)) + `.2] read length
    if length` + vc + `, err = decoder.ReadLength(); err != nil {
        return
    }        
    // [step ` + strconv.Itoa(int(mb.Tag)) + `.3] read data
    ` + gen.genVariableName(prefix, mb.Key) + ` = make(` + gen.genType(mb.Type) + `, 0)` + `
	var k` + vc + ` ` + gen.genType(mb.Type.TypeK) + `
	var v` + vc + ` ` + gen.genType(mb.Type.TypeV) + `        
    for i := uint32(0);i < length` + vc + `; i++ {
`)

	dummy := &StructMember{
		Type: mb.Type.TypeK,
		Key:  "k" + vc,
	}
	gen.genReadVar(dummy, "")

	dummy = &StructMember{
		Type: mb.Type.TypeV,
		Key:  "v" + vc,
		Tag:  1,
	}
	gen.genReadVar(dummy, "")

	gen.writeString(`
	` + prefix + mb.Key + `[k` + vc + `] = v` + vc + `
}
`)
}

// 生成序列化函数
// tips: 这里注意，如果 optional 有默认值，那么序列化的时候写不写默认值呢？
// 1. 写：优点方便维护，缺点增大带宽
// 2. 不写：优点节约带宽，缺点维护不方便
// 最后综合考虑，其实带宽开销并不大，而维护更加重要，故默认写
func (gen *Generate) genFunWriteTo(st *StructInfo) {
	gen.writeString(`// WriteTo encode struct to io.Writer 
func (st *` + st.Name + `) WriteTo(w io.Writer) (n int64, err error) {
    encoder := jce.NewEncoder(w)
	st.resetDefault()

`)

	for _, v := range st.Member {
		gen.genWriteVar(&v, "st.", false)
	}

	gen.writeString(`
// flush to io.Writer        
    err = encoder.Flush()
    return
}
`)
}

// 序列化 struct 成员
func (gen *Generate) genWriteVar(v *StructMember, prefix string, hasRet bool) {
	gen.writeString("// [step " + strconv.Itoa(int(v.Tag)) + "] write " + v.Key)

	switch v.Type.Type {
	case tkTVector:
		gen.genWriteVector(v, prefix, hasRet)
	case tkTArray:
		gen.genWriteVector(v, prefix, hasRet)
	case tkTMap:
		gen.genWriteMap(v, prefix, hasRet)
	case tkName: // TODO: ?
		if v.Type.CType == tkEnum {
			gen.writeString(`
            if err = encoder.WriteInt32(int32(` + gen.genVariableName(prefix, v.Key) + `),` + strconv.Itoa(int(v.Tag)) + `); err !=nil {
                return
            }
`)
			return
		}

		gen.writeString(`
        if _, err = ` + prefix + v.Key + `.WriteTo(encoder.Writer()); err != nil {
            return
        }
`)

	default: // 默认基础类型
		gen.writeString(`
if err = encoder.Write` + upperFirstLetter(gen.genType(v.Type)) + `(` + gen.genVariableName(prefix, v.Key) + `, ` + strconv.Itoa(int(v.Tag)) + `); err != nil {
    return
} 
`)
	}
}

// 序列化数组
func (gen *Generate) genWriteVector(mb *StructMember, prefix string, hasRet bool) {
	vc := strconv.Itoa(gen.vc)
	gen.vc++

	// SimpleList
	if mb.Type.TypeK.Type == tkTByte {
		tag := strconv.Itoa(int(mb.Tag))

		if mb.Type.TypeK.Unsigned {
			gen.writeString(`
if err = encoder.WriteSliceUint8(` + gen.genVariableName(prefix, mb.Key) + `,` + tag + `,` + `); err != nil {
    return
}
`)
			return
		}

		gen.writeString(`
if err = encoder.WriteSliceInt8(` + gen.genVariableName(prefix, mb.Key) + `,` + tag + `,` + `); err != nil {
    return
}
`)
		return
	}

	// LIST
	// ---------------------------------------------
	// | head(type、tag) |   length  |     data    |
	// |    1 or 2 B     |     4B    |       ?     |
	// ---------------------------------------------
	// data 可以是任何 type

	gen.writeString(`
// [step ` + strconv.Itoa(int(mb.Tag)) + `.1] write type、tag
if err = encoder.WriteHead(jce.LIST, ` + strconv.Itoa(int(mb.Tag)) + `); err != nil {
    return
}
// [step ` + strconv.Itoa(int(mb.Tag)) + `.2] write list length
if err = encoder.WriteLength(uint32(len(` + gen.genVariableName(prefix, mb.Key) + `))); err != nil {
    return
}
// [step ` + strconv.Itoa(int(mb.Tag)) + `.3] write data 
    for _, v` + vc + ` := range ` + gen.genVariableName(prefix, mb.Key) + ` {
`)

	dummy := &StructMember{
		Type: mb.Type.TypeK,
		Key:  "v" + vc,
	}

	gen.genWriteVar(dummy, "", hasRet)

	gen.writeString("}\n")
}

// 序列化 map
func (gen *Generate) genWriteMap(mb *StructMember, prefix string, hasRet bool) {
	vc := strconv.Itoa(gen.vc)
	gen.vc++

	gen.writeString(`
// [step ` + strconv.Itoa(int(mb.Tag)) + `.1] write type、tag
if err = encoder.WriteHead(jce.MAP, ` + strconv.Itoa(int(mb.Tag)) + `); err != nil {
    return
}
// [step ` + strconv.Itoa(int(mb.Tag)) + `.2] write length
if err = encoder.WriteLength(uint32(len(` + gen.genVariableName(prefix, mb.Key) + `))); err != nil {
    return
}
// [step ` + strconv.Itoa(int(mb.Tag)) + `.3] write data
for k` + vc + `, v` + vc + ` := range ` + gen.genVariableName(prefix, mb.Key) + ` {
`)

	// write key
	dummy := &StructMember{
		Type: mb.Type.TypeK,
		Key:  "k" + vc,
	}
	gen.genWriteVar(dummy, "", hasRet)

	// write value
	dummy = &StructMember{
		Type: mb.Type.TypeV,
		Key:  "v" + vc,
		Tag:  1,
	}
	gen.genWriteVar(dummy, "", hasRet)

	gen.writeString("}\n")
}

// 保存文件
func (gen *Generate) saveFiles() {
	log.Debugf(gen.code.String())

	filename := gen.p.ProtoName + ".jce.go"

	beauty, err := format.Source(gen.code.Bytes())
	if err != nil {
		panic("go fmt fail. " + filename + " " + err.Error())
	}

	mkPath := gen.prefix + gen.p.Module

	if err = os.MkdirAll(mkPath, 0766); err != nil {
		panic(err.Error())
	}

	if err = ioutil.WriteFile(mkPath+"/"+filename, beauty, 0666); err != nil {
		panic(err.Error())
	}
}

// 生成变量名
func (gen *Generate) genVariableName(prefix, name string) string {
	if prefix != "" {
		return prefix + name
	} else {
		return strings.Trim(name, "()")
	}
}

// 生成对应的类型字符串
func (gen *Generate) genType(ty *VarType) string {
	ret := ""

	switch ty.Type {
	case tkTBool:
		ret = "bool"
	case tkTInt:
		if ty.Unsigned {
			ret = "uint32"
		} else {
			ret = "int32"
		}
	case tkTShort:
		if ty.Unsigned {
			ret = "uint16"
		} else {
			ret = "int16"
		}
	case tkTByte:
		if ty.Unsigned {
			ret = "uint8"
		} else {
			ret = "int8"
		}
	case tkTLong:
		if ty.Unsigned {
			ret = "uint64"
		} else {
			ret = "int64"
		}
	case tkTFloat:
		ret = "float32"
	case tkTDouble:
		ret = "float64"
	case tkTString:
		ret = "string"
	case tkTVector:
		ret = "[]" + gen.genType(ty.TypeK)
	case tkTMap:
		ret = "map[" + gen.genType(ty.TypeK) + "]" + gen.genType(ty.TypeV)
	case tkName:
		ret = strings.Replace(ty.TypeSt, "::", ".", -1)
		vec := strings.Split(ty.TypeSt, "::")
		for i := range vec {
			if i == (len(vec) - 1) {
				vec[i] = upperFirstLetter(vec[i])
			}
		}
		ret = strings.Join(vec, ".")
	case tkTArray:
		ret = "[" + fmt.Sprintf("%v", ty.TypeL) + "]" + gen.genType(ty.TypeK)
	default:
		panic("Unknown Type " + TokenMap[ty.Type])
	}

	return ret
}

// 首字母大写
func upperFirstLetter(s string) string {
	if len(s) == 0 {
		return ""
	}

	if len(s) == 1 {
		return strings.ToUpper(string(s[0]))
	}

	return strings.ToUpper(string(s[0])) + s[1:]
}

func (gen *Generate) writeString(s string) (err error) {
	_, err = gen.code.WriteString(s)
	return
}
