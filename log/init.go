package log

import (
	"bufio"
	"os"
)

var (
	mapMode = map[string]Mode{
		"normal": NormalMode,
		"json":   JSONMode,
	}

	mapLevel = map[string]Level{
		"trace":  DebugLevel,
		"debug":  DebugLevel,
		"info":   InfoLevel,
		"warn":   WarnLevel,
		"error":  ErrorLevel,
		"dPanic": DPanicLevel,
		"panic":  PanicLevel,
		"fatal":  FatalLevel,
	}

	mapOutPutType = map[string]LoggerOutPutType{
		"fileoutput":              FileOutPut,
		"udpoutput":               UdpOutPut,
		"bothudpandfileoutput":    BothUdpAndFileOutPut,
		"bothudpandfilenotoutput": BothUdpAndFileNotOutPut,
	}

	mapRotateType = map[string]LoggerRotateType{
		"date": DateRotate,
		"size": SizeRotate,
	}

	confPath      = "../conf/config.toml"
	DefaultLogger *Logger
)

// 默认初始化，自动调用
func init() {
	DefaultLogger = &Logger{
		level:      ErrorLevel,
		depth:      4,
		mode:       NormalMode,
		outputType: 0,
		rotateType: 0,
		writer: bufio.ReadWriter{
			Writer: bufio.NewWriter(os.Stdout),
		},
	}
}
