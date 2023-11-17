package log

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

type FileMessageData struct {
	currentTime time.Time
	fileName    string
	line        int
	funcName    string
}

// GetFileMessage 获得时间戳、格式化时间、文件名、函数名
func GetFileMessage(depth int) (time.Time, string, int, string) {
	currentTime := time.Now()
	pc, fileName, line, ok := runtime.Caller(depth)
	if !ok {
		return currentTime, "", 0, ""
	}

	fileName = filepath.Base(fileName)

	var funcName string
	splitName := strings.Split(runtime.FuncForPC(pc).Name(), "/")
	if len(splitName) > 0 {
		funcName = splitName[len(splitName)-1]
	}
	return currentTime, fileName, line, funcName
}

func GetFileMessageStruct(depth int) FileMessageData {
	currentTime, fileName, line, funcName := GetFileMessage(depth + 1)
	return FileMessageData{
		currentTime: currentTime,
		fileName:    fileName,
		line:        line,
		funcName:    funcName,
	}
}

// StrToBytes 高效的类型转换
func StrToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// GetBasicInfo 获得进程ID、进程名、本地IP地址
func GetBasicInfo() (int, string, string) {
	var (
		pid     int
		name    string
		localIP string
	)
	pid = os.Getpid()
	localIP, _ = GetLocalIP()
	name = filepath.Base(os.Args[0])
	return pid, name, localIP
}

func GetLocalIP() (ip string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, addr := range addrs {
		ipAddr, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipAddr.IP.IsLoopback() {
			continue
		}
		if !ipAddr.IP.IsGlobalUnicast() {
			continue
		}
		return ipAddr.IP.String(), nil
	}
	return "", errors.New("cant get local ip")
}
