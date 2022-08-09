package GoMatrix

import "log"

type RouterGroup struct {
	// 当前分组前缀
	prefix      string
	// 分组挂载的中间件
	middlewares []HandlerFunc
	// 父节点
	parent      *RouterGroup
	// 引擎统一化协调管理
	engine      *Engine
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
