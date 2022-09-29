package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	debug = false
)

func StartDebug() {
	debug = true

	f, err := os.Create("demo.log")
	if err != nil {
		panic(err)
	}

	log.SetOutput(io.MultiWriter(f, os.Stdout))
}

func Debugf(format string, a ...any) {
	if !debug {
		return
	}

	log.Printf(fmt.Sprintf("[DEBUG] %s", fmt.Sprintf(format, a...)))
}

func Errorf(format string, a ...any) {
	log.Printf(fmt.Sprintf("[ERROR] %s", fmt.Sprintf(format, a...)))
}

func Raw(a ...any) {
	if !debug {
		return
	}
	log.Println(a...)
}
