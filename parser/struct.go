package parser

import (
	"github.com/erpc-go/jce2go/lex"
	"github.com/erpc-go/jce2go/utils"
)

// StructMember member struct.
type StructMember struct {
	CommentType string
	Tag         int32
	Require     bool
	Type        *VarType
	Key         string // after the uppercase converted key
	OriginKey   string // original key
	Default     string
	DefType     lex.TokenType
	Comment     string
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
	Comment             string
	commentTagNum       int
	DependModule        map[string]bool
	DependModuleWithJce map[string]string
}

// 1. struct Rename
// struct Name { 1 require Mb type}
func (st *StructInfo) Rename() {
	st.Name = utils.UpperFirstLetter(st.Name)

	for i := range st.Member {
		st.Member[i].OriginKey = st.Member[i].Key
		st.Member[i].Key = utils.UpperFirstLetter(st.Member[i].Key)
	}
}
