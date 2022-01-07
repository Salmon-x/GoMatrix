package GoMatrix

import (
	"fmt"
	"log"
	"net/http"
)

// HandlerFunc定义使用的请求处理程序，替换成上下文

type HandlerFunc func(c *Context)

// 引擎实现ServeHTTP接口

type Engine struct {
	router *router
}

// 初始化引擎

func New() *Engine {
	return &Engine{router: newRouter()}
}


func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %s", method, pattern)
	engine.router.addRoute(method, pattern, handler)
}

// GET定义添加GET请求的方法
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// 定义添加POST请求的方法
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

// 定义添加PUT请求的方法
func (engine *Engine) PUT(pattern string, handler HandlerFunc) {
	engine.addRoute("PUT", pattern, handler)
}

func (engine *Engine)DELETE(pattern string, handler HandlerFunc) {
	engine.addRoute("DELETE", pattern, handler)
}

// 定义启动http服务器的方法
func (engine *Engine) Run(addr string) (err error) {
	fmt.Printf("ListenAndServe %s \n", addr)
	return http.ListenAndServe(addr, engine)
}

// 实现Handler接口
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}