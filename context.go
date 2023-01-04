package GoMatrix

import (
	"github.com/Salmon-x/GoMatrix/response"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
)

type H map[string]interface{}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

const abortIndex int8 = math.MaxInt8 / 2

type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request
	// 请求信息
	Path   string
	Method string
	Params map[string]string
	// query参数缓存
	queryCache url.Values
	// 响应信息
	StatusCode int
	engine     *Engine

	// 中间件实现
	index       int8
	middlewares HandlersChain
}

func (c *Context) newContext(w http.ResponseWriter, req *http.Request) {
	c.Writer = w
	c.Req = req
	c.Path = req.URL.Path
	c.Method = req.Method
	c.index = -1
}

// Form参数

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// Query参数

func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.initQueryCache()
	if values, ok := c.queryCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

func (c *Context) initQueryCache() {
	if c.queryCache == nil {
		if c.Req != nil {
			c.queryCache = c.Req.URL.Query()
		} else {
			c.queryCache = url.Values{}
		}
	}
}

// 构造响应状态码

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// 构造响应头

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// 构造string响应

func (c *Context) String(code int, format string, values ...interface{}) {
	c.Render(code, response.String{Format: format, Value: values})
}

func (c *Context) Fail(code int, err string) {
	c.index = int8(len(c.middlewares))
	c.JSON(code, H{"message": err})
}

// 构造json响应

func (c *Context) JSON(code int, obj interface{}) {
	c.Render(code, response.JSON{Data: obj})
}

// 构造xml响应

func (c *Context) XML(code int, obj interface{}) {
	c.Render(code, response.XML{Data: obj})
}

// 构造文件响应

func (c *Context) Data(code int, data []byte) {
	c.Render(code, response.Data{Data: data})
}

// 构造HTML响应

func (c *Context) HTML(code int, name string, data interface{}) {
	c.Render(code, c.engine.htmlTemplates.Instance(name, data))
}

// 从上下文中读取param参数

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func (c *Context) Download(file string, filename ...string) {
	var (
		fName string
		err   error
	)
	c.SetHeader("Content-Description", "File Transfer")
	c.SetHeader("Content-Transfer-Encoding", "binary")
	c.SetHeader("Expires", "0")
	c.SetHeader("Cache-Control", "must-revalidate")
	c.SetHeader("Pragma", "public")
	c.SetHeader("Accept-Ranges", "bytes")
	c.SetHeader("Content-Type", "application/octet-stream")
	if err != nil {
		http.ServeFile(c.Writer, c.Req, file)
		return
	}
	if len(filename) > 0 && filename[0] != "" {
		fName = filename[0]
	} else {
		fName = filepath.Base(file)
	}
	fn := url.PathEscape(fName)
	if fName == fn {
		fn = "filename=" + fn
	} else {
		fn = "filename=" + fName + "; filename*=utf-8''" + fn
	}
	c.SetHeader("Content-Disposition", "attachment; "+fn)
	c.ServeFile(file)
}

func (c *Context) GetHeader(key string) string {
	return c.Req.Header.Get(key)
}

// 中间件的流转，主要通过index的移位来决定执行哪个middlewares中的中间件

func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.middlewares)) {
		c.middlewares[c.index](c)
		c.index++
	}
}

func (c *Context) Abort() {
	c.index = abortIndex
}

func (c *Context) Render(code int, r response.Render) {
	r.WriteContentType(c.Writer)
	c.Status(code)
	// 一丢丢小问题，不影响使用
	//if !bodyAllowedForStatus(code) {
	//	r.WriteContentType(c.Writer)
	//	return
	//}
	if err := r.Render(c.Writer); err != nil {
		panic(err)
	}
}

func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == http.StatusNoContent:
		return false
	case status == http.StatusNotModified:
		return false
	}
	return true
}
