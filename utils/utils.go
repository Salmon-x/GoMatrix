package utils

import "unsafe"

// 路由规则校验
func Assert1(guard bool, text string) {
	if !guard {
		panic(text)
	}
}

// 字符串转换为byte

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
