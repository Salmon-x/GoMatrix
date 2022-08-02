package GoMatrix

// 路由规则校验
func assert1(guard bool, text string) {
	if !guard {
		panic(text)
	}
}
