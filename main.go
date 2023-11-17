package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/erpc-go/jce2go/generate"
	"github.com/erpc-go/jce2go/log"
)

var (
	// 最终生成的代码根目录
	outdir string
	// 启动 debug 模式
	debug bool

	modulePath string

	jsonOmitEmpty bool

	noOptional bool

	addTag bool
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jce2go [OPTION] <jcefile>\n")
		fmt.Fprintf(os.Stderr, "jce2go support type: bool byte short int long float double vector map\n")
		fmt.Fprintf(os.Stderr, "supported [OPTION]:\n")
		flag.PrintDefaults()
	}

	flag.StringVar(&modulePath, "mod", "", "model path(default github.com/erpc-go/jce2go)")
	flag.StringVar(&outdir, "o", "", "which dir to put generated code")
	flag.BoolVar(&jsonOmitEmpty, "json", false, "enable json tag")
	flag.BoolVar(&addTag, "tag", false, "set default struct tag")
	flag.BoolVar(&noOptional, "no-optional", false, "do not package optional fields")
	flag.BoolVar(&debug, "debug", false, "enable debug mode")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if debug {
		log.DefaultLogger.SetLevel(log.DebugLevel)
	}

	for _, filename := range flag.Args() {
		if path.Ext(filename) != ".jce" {
			continue
		}

		log.Debug("begin parse file, name: %s", filename)

		gen := generate.NewGenerate(filename, modulePath, outdir, jsonOmitEmpty)
		gen.Gen()
	}
}
