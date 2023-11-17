package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Logger struct {
	level      Level
	depth      int
	mode       Mode
	outputType LoggerOutPutType
	rotateType LoggerRotateType
	writer     bufio.ReadWriter
}

// 设置日志等级(All,Trace,Debug,Info,Warn,Error,DPanic,Panic,Fatal)
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

func (l *Logger) SetModle(mode Mode) {
	l.mode = mode
}

func (l *Logger) Allow(level Level, mode Mode) bool {
	if l.outputType == BothUdpAndFileNotOutPut {
		return false
	}
	if l.level > level || l.mode != mode {
		return false
	}
	return true
}

// ---------------------Debug---------------------
func (l *Logger) Debug(f string, p ...interface{}) {
	l.write(DebugLevel, f, p...)
}

// ---------------------Info---------------------
func (l *Logger) Info(f string, p ...interface{}) {
	l.write(InfoLevel, f, p...)
}

// ---------------------Warn---------------------

func (l *Logger) Warn(f string, p ...interface{}) {
	l.write(WarnLevel, f, p...)
}

// ---------------------Error---------------------

func (l *Logger) Error(f string, p ...interface{}) {
	l.write(ErrorLevel, f, p...)
}

func (l *Logger) ErrorJSON(d interface{}) {
	l.writeJSON(ErrorLevel, d)
}

// ---------------------Panic---------------------
func (l *Logger) Panic(f string, p ...interface{}) {
	l.write(PanicLevel, f, p...)
	panic("")
}

// ---------------------Fatal---------------------

func (l *Logger) Fatal(f string, p ...interface{}) {
	l.write(FatalLevel, f, p...)
}

// ---------------All（最高的打印级别）---------------------
func (l *Logger) Raw(f string, p ...interface{}) {
	s := fmt.Sprintf(f, p...)
	l.writer.WriteString(s)
	l.writer.Flush()
}

func (l *Logger) write(level Level, f string, p ...interface{}) {
	if !l.Allow(level, NormalMode) {
		return
	}

	// [2023-11-17 17:11:09.2058935][ERROR][push_status.go:183][util.(*PushStatusHelper).SaveStatus]appid:1000176, uid:718778733, qz.Do err:(code:-13104, msg:)
	fileMessage := GetFileMessageStruct(l.depth)
	s := fmt.Sprintf("[%s][%s][%s:%d][%s]%s\n", time.Now().Format("2006-01-02 15:04:05.9999999"), level.String(), fileMessage.fileName, fileMessage.line, fileMessage.funcName, fmt.Sprintf(f, p...))
	l.writer.WriteString(s)
	l.writer.Flush()
}

func (l *Logger) writeJSON(level Level, data interface{}) {
	if !l.Allow(level, NormalMode) {
		return
	}

	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	var str bytes.Buffer
	_ = json.Indent(&str, b, "", "    ")
	l.writer.Write(str.Bytes())
	l.writer.Flush()
}
