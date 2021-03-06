package liberty

import (
	"fmt"
	"net/http"
	"strings"
)

// an imlementation of a ternary search tree for web/api routing
type tree struct {
	root   *node
	router *Router
}

type node struct {
	v        byte
	lt       *node
	eq       *node
	gt       *node
	handlers mHandlers
	varName  string
}

func (n *node) String() string {
	return fmt.Sprintf(
		"[value: %s, varName: %s, handlers: %T]",
		string(n.v),
		n.varName,
		n.handlers,
	)
}

func (t *tree) handle(nd *node, pattern *pattern, index int) *node {
	v := pattern.str[index]

	if nd == nil {
		nd = &node{v: v}
	}

	varName, ok := pattern.varNameAt(index)
	if ok {
		nd.varName = varName
	}

	if v < nd.v {
		nd.lt = t.handle(nd.lt, pattern, index)
	} else if v > nd.v {
		nd.gt = t.handle(nd.gt, pattern, index)
	} else if index < (len(pattern.str) - 1) {
		nd.eq = t.handle(nd.eq, pattern, index+1)
	} else {
		if nd.handlers == nil {
			nd.handlers = make(mHandlers, 0)
		}
		nd.handlers[pattern.method] = pattern.handler
	}

	return nd
}

func (t *tree) match(method method, path string, ctx *Context) http.Handler {
	var i int
	var match int
	var char byte

	n := t.root
	l := len(path)

	for i < l {
		char = path[i]
		switch {
		default:
			n = n.eq
			i++
		case n == nil || n.v == 0x0:
			return t.router.NotFound
		case char == '/' && n.eq != nil && (n.eq.v == ':' || n.eq.v == '*'):
			match = i + 1
			for match < l && path[match] != '/' {
				match++
			}
			ctx.Params.Add(n.eq.varName, path[i+1:match])

			nextSegment := strings.IndexByte(path[i+1:], '/')
			lastNode := nextSegment == -1 || n.eq.v == '*'
			i = i + 1 + nextSegment

			n = n.eq
			var sc byte
			var si int

			searchPath := string(n.v) + n.varName
			if !lastNode { //  && n.v != '*' { //  TODO WTF graham ??? tests ???
				searchPath = searchPath + "/"
			}
			sl := len(searchPath)

			for si < sl {
				sc = searchPath[si]
				switch {
				default:
					n = n.eq
					si++
				case sc < n.v:
					n = n.lt
				case sc > n.v:
					n = n.gt
				case si == sl-1:
					si++
				}
			}

			if lastNode {
				return n.handlers[method]
			}

			continue
		case char < n.v:
			n = n.lt
		case char > n.v:
			n = n.gt
		case i == l-1:
			return n.handlers[method]
		}
	}

	return t.router.NotFound
}

func (t *tree) longestPrefix(mthd method, key string, ctx *Context) http.Handler {
	if len(key) < 1 {
		return http.HandlerFunc(http.NotFound)
	}

	length := prefix(t.root, key, 0)

	return t.match(mthd, key[0:length], ctx)
}

func prefix(n *node, key string, index int) int {
	if index == len(key) || n == nil {
		return 0
	}

	length := 0
	recLen := 0
	v := key[index]

	if v < n.v {
		recLen = prefix(n.lt, key, index)
	} else if v > n.v {
		recLen = prefix(n.gt, key, index)
	} else {
		if n.v != 0x0 {
			length = index + 1
		}
		recLen = prefix(n.eq, key, index+1)
	}
	if length > recLen {
		return length
	}

	return recLen
}
