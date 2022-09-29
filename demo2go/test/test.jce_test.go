package test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/edte/jce2go/demo2go/base"
)

func TestRequestPacket(t *testing.T) {
	req := &RequestPacket{
		B:       1,
		S:       2,
		I:       3,
		L:       4,
		F:       5,
		D:       6,
		S1:      "hello",
		S2:      "test",
		I2:      99,
		Buffer1: []int8{1, 2, 3},
		Buffer2: []uint8{8, 8, 2},
		Arr1:    []string{"a", "b", "c"},
		Arr2: [][]int32{
			{23, 23}, {2, 1, 8},
		},
		M1: map[string]string{
			"a": "b",
			"c": "ad",
		},
		Arr4: []map[int32]string{
			{
				1: "2", 23: "88",
			},
		},
		Arr3: []base.Request{
			{
				B: 2,
			},
		},
		M2: map[string]base.Request{
			"a": {
				B: 88,
			},
		},
	}

	b := bytes.NewBuffer(make([]byte, 0))
	_, err := req.WriteTo(b)
	if err != nil {
		panic(err)
	}

	rsp := &RequestPacket{}
	_, err = rsp.ReadFrom(b)
	if err != nil {
		panic(err)
	}

	fmt.Println(rsp)
}
