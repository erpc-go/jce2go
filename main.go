package main

import (
	"flag"
	"path"

	"github.com/edte/jce2go/log"
)

var (
	// 最终生成的代码根目录
	outdir string
	// 启动 debug 模式
	debug bool

	modulePath string

	jsonOmitEmpty bool
)

func main() {
	flag.StringVar(&outdir, "o", "", "which dir to put generated code")
	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.BoolVar(&jsonOmitEmpty, "json", false, "enable json tag")
	flag.StringVar(&modulePath, "mod", "", "model path")

	flag.Parse()

	if debug {
		log.StartDebug()
	}

	for _, filename := range flag.Args() {
		if path.Ext(filename) != ".jce" {
			continue
		}

		// log.Debugf("begin parse file, name: %s", filename)

		gen := NewGenerate(filename, modulePath, outdir)
		gen.Gen()
	}
}
