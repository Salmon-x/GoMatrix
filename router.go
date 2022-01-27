package GoMatrix

import (
	"net/http"
	"strings"
)

// 抽离router
type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 解析路由

func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)
	key := method + "-" + pattern
	// 查询该方法路由树
	_, ok := r.roots[method]
	if !ok {
		// 如果没有该树，则新建一个
		r.roots[method] = &node{}
	}
	// 向树内插入路由
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}


func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	// 路由树
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}
	// 搜索路由
	n := root.search(searchParts, 0)
	//fmt.Println(n) // &{/func/:cid :cid [] true}
	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			// 如果该子节点以:或者*开头
			if part[0] == ':' {
				// 将值赋值到params
				params[part[1:]] = searchParts[index]
			}
			// 如果*开头并且长度大于1，则这个节点的值赋值到params，并停止循环
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}


func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		r.handlers[key](c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}

