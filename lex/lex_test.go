package lex

import (
	"fmt"
	"testing"
)

func TestNewLexState(t *testing.T) {
	filename := "base.jce"
	data := `
	/*test*/
module base
{
    enum Code
    {
        Success = 0,
        Error = 1
    };   
 
	/*  test */
    const short ERPC_VERSION   = 0x01;
    const int   TUP_VERSION    = 0x03;

    // test
    struct request
    {

        1  require byte   b;  // hhhhhh
    };
	/*
	sdf
	wer
	*/
};	
	`
	l := NewLexState(filename, []byte(data))
	for {
		token := l.NextToken()
		fmt.Printf("token:%+v, value:%v, line:%v\n", TokenMap[token.Type], token.Value, token.Line)
		if IsEOS(token.Type) {
			break
		}
	}
}
