package GoMatrix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
)

type H map[string]interface{}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

type Context struct {
	// 原对象
	Writer http.ResponseWriter
	Req    *http.Request
	// 请求信息
	Path   string
	Method string
	Params map[string]string
	// 响应信息
	StatusCode int
}

// 生成新的上下文
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

// Form参数

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// Query参数

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
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
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// 构造json响应

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 构造文件响应

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

// 构造HTML响应

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}

// 从上下文中读取param参数

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func (c *Context) Download(file string, filename ...string) {
	var (
		fName string
		err error
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

func (c *Context)GetHeader(key string) string {
	return c.Req.Header.Get(key)
}
