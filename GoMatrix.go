package GoMatrix

import (
	"golang.org/x/net/netutil"
	"log"
	"net"
	"net/http"
)

// HandlerFunc定义使用的请求处理程序，替换成上下文

type HandlerFunc func(c *Context)

// 引擎实现ServeHTTP接口

type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
	// 地址升级
	serverIp string
	serverPort string
	// 限流，最大连接数
	maxConn int
}

type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	parent      *RouterGroup
	engine      *Engine
}

// 初始化引擎

func New(serverIp,serverPort string, maxConn int) *Engine {
	engine := &Engine{
		router: newRouter(),
		maxConn: maxConn,
		serverIp: serverIp,
		serverPort: serverPort,
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// GET定义添加GET请求的方法
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodGet, pattern, handler)
}

// 定义添加POST请求的方法
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPost, pattern, handler)
}

// 定义添加PUT请求的方法
func (group *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPut, pattern, handler)
}

func (group *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodDelete, pattern, handler)
}

// 定义启动http服务器的方法
func (engine *Engine) Run() (err error) {
	var listener net.Listener
	log.Printf("start server on host %s port:%s ,workers:%d\n", engine.serverIp, engine.serverPort, engine.maxConn)
	listener, err = net.Listen("tcp", ":"+engine.serverPort)
	if err != nil {
		return err
	}
	defer listener.Close()
	listener = netutil.LimitListener(listener, engine.maxConn)
	return http.Serve(listener, engine)
}

// 实现Handler接口
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}
