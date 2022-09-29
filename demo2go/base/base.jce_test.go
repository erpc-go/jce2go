package base

import (
	"bytes"
	"fmt"
	"testing"
)

func TestRequest(t *testing.T) {
	req := Request{
		B: 12,
	}

	b := bytes.NewBuffer(make([]byte, 0))
	_, err := req.WriteTo(b)
	if err != nil {
		panic(err)
	}

	rsp := &Request{}
	_, err = rsp.ReadFrom(b)
	if err != nil {
		panic(err)
	}

	fmt.Println(rsp)
}
