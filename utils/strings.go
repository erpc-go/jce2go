package utils

import "strings"

// path2ProtoName 函数接受一个文件路径字符串 path 作为参数，并从中提取文件名（不带扩展名）作为协议名。以下是函数的工作原理：

// 使用 strings.LastIndex 函数找到路径中最后一个斜杠（/）的位置。如果找不到斜杠或者斜杠位于路径末尾，则将 iBegin 设置为 0，否则将其设置为斜杠后面的位置。
// 使用 strings.LastIndex 函数找到路径中最后一个 .jce 扩展名的位置。如果找不到扩展名，则将 iEnd 设置为路径的长度。
// 使用切片操作从路径中提取子字符串，范围从 iBegin 到 iEnd。这将得到不带扩展名的文件名。
// 最后，函数返回提取的协议名。

// 例如，对于输入路径 "/path/to/my_protocol.jce"，Path2PackageName 函数将返回 "my_protocol"。
func Path2PackageName(path string, suffix string) string {
	iBegin := strings.LastIndex(path, "/")
	if iBegin == -1 || iBegin >= len(path)-1 {
		iBegin = 0
	} else {
		iBegin++
	}
	iEnd := strings.LastIndex(path, suffix)
	if iEnd == -1 {
		iEnd = len(path)
	}

	return path[iBegin:iEnd]
}
