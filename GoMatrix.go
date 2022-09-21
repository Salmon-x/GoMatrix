package GoMatrix

import (
	"golang.org/x/net/netutil"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"text/template"
)

// HandlerFunc定义使用的请求处理程序，替换成上下文

type HandlerFunc func(c *Context)

type HandlersChain []HandlerFunc

// 引擎实现ServeHTTP接口

type Engine struct {
	// 继承group的功能，之后都使用group来进行路由操作
	*RouterGroup
	router *router
	groups []*RouterGroup
	// 地址升级
	serverIp   string
	serverPort string
	// 限流，最大连接数
	maxConn int

	// 池优化
	pool sync.Pool

	// https支持
	isSsl bool
	crt   string
	key   string

	// 模板
	htmlTemplates *template.Template
	funcMap       template.FuncMap
}

// 初始化引擎

func New(serverIp, serverPort string, maxConn int) *Engine {
	engine := &Engine{
		router:     newRouter(),
		maxConn:    maxConn,
		serverIp:   serverIp,
		serverPort: serverPort,
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

func SslNew(serverIp, serverPort string, maxConn int, crt, key string) *Engine {
	engine := &Engine{
		router:     newRouter(),
		maxConn:    maxConn,
		serverIp:   serverIp,
		serverPort: serverPort,
		isSsl:      true,
		crt:        crt,
		key:        key,
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

// 分配池：当池中没有对象，则新建一个初始对象
func (engine *Engine) allocateContext() *Context {
	return &Context{engine: engine, index: -1}
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

func (group *RouterGroup) PATCH(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodPatch, pattern, handler)
}

func (group *RouterGroup) CONNECT(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodConnect, pattern, handler)
}

func (group *RouterGroup) OPTIONS(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodOptions, pattern, handler)
}

func (group *RouterGroup) TRACE(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodTrace, pattern, handler)
}

func (group *RouterGroup) HEAD(pattern string, handler HandlerFunc) {
	group.addRoute(http.MethodHead, pattern, handler)
}

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// 定义启动http服务器的方法
func (engine *Engine) Run() (err error) {
	var listener net.Listener
	log.Printf("start server on host %s port:%s ,workers:%d\n", engine.serverIp, engine.serverPort, engine.maxConn)
	listener, err = net.Listen("tcp", engine.serverIp+":"+engine.serverPort)
	if err != nil {
		return err
	}
	defer listener.Close()
	listener = netutil.LimitListener(listener, engine.maxConn)
	if engine.isSsl {
		return http.ServeTLS(listener, engine, engine.crt, engine.key)
	} else {
		return http.Serve(listener, engine)
	}
}

// 根据分组不同挂载不同的中间件

func (engine *Engine) addMiddlewares(c *Context) {
	var handles HandlersChain
	for _, group := range engine.groups {
		if strings.HasPrefix(c.Path, group.prefix) {
			handles = append(handles, group.middlewares...)
		}
	}
	c.middlewares = handles
}

// 实现Handler接口
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context)
	// 初始化上下文
	c.newContext(w, req)
	engine.addMiddlewares(c)
	engine.router.handle(c)
	engine.pool.Put(c)
}
