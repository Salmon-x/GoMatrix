package GoMatrix

import (
	"net/http"
	"strings"
)

// 抽离router
type router struct {
	// 路由树
	trees    methodTrees
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		trees:    make(methodTrees, 0, 9),
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
	assert1(pattern[0] == '/', "path must begin with '/'")
	assert1(method != "", "HTTP method can not be empty")

	root := r.trees.get(method)
	if root == nil {
		root = new(node)
		r.trees = append(r.trees, methodTree{method: method, root: root})
	}
	parts := parsePattern(pattern)
	key := method + "-" + pattern
	// 向树内插入路由
	root.insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	var root *node
	for i := range r.trees {
		if r.trees[i].method != method {
			continue
		}
		root = r.trees[i].root
		break
	}
	if root == nil {
		return nil, nil
	}
	// 搜索路由
	n := root.search(searchParts, 0)
	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			// 如果该子节点以:或者*开头
			if part[0] == ':' {
				// 将值赋值到params
				params[part[1:]] = searchParts[index]
			} else if part[0] == '*' && len(part) > 1 {
				// 如果*开头并且长度大于1，则这个节点的值赋值到params，并停止循环
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
