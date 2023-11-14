// DO NOT EDIT IT.
// code generated by jce2go v1.0.
// source: base.jce
package base

import (
	"fmt"
	"io"

	"github.com/erpc-go/jce-codec"
)

// 占位使用，避免导入的这些包没有被使用
var _ = fmt.Errorf
var _ = io.ReadFull
var _ = jce.Int1

// enum EMsgSendType implement
type EMsgSendType int32

const (
	EMsgSendTypeESendTypeOnline  EMsgSendType = 1
	EMsgSendTypeESendTypeOffline EMsgSendType = 2
)

// const implement
const (
	ERPC_VERSION int16  = 0x01  // hhhh
	TUP_VERSION  int32  = 0x03  // lll
	Jj           string = "tet" // owd
)
