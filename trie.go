package GoMatrix

import (
	"strings"
)

// 树节点需要存储的信息

type node struct {
	pattern  string  // 待匹配路由，例如 /p/:lang
	part     string  // 路由中的一部分，例如 :lang
	children []*node // 子节点
	isWild   bool    // 是否精确匹配，part 含有 : 或 * 时为true
}

type methodTree struct {
	method string
	root   *node
}

type methodTrees []methodTree

// 第一个匹配成功的节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func (n *node) insert(pattern string, parts []string, height int) {
	// parts为处理过的路由组：路由为/func/:cid   [func :cid] <-- parts为
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	// 取子节点
	part := parts[height]
	// 匹配子节点
	child := n.matchChild(part)
	if child == nil {
		// 如果没有匹配到则新建一个子节点，将其加入父节点
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	// 递归插入
	child.insert(pattern, parts, height+1)
}

// TODO 改成循环查询，人工建堆栈改良循环
func (n *node) search(parts []string, height int) *node {
	// 解析到最后一层或者是匹配到*号，则认为该节点是最子节点
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}
	// 查询该height下所有子节点
	part := parts[height]
	children := n.matchChildren(part)
	for _, child := range children {
		// 递归查询子节点
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}

func (trees methodTrees) get(method string) *node {
	for _, tree := range trees {
		if tree.method == method {
			return tree.root
		}
	}
	return nil
}
