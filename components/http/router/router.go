package main

import (
	"net/url"
)

const (
	NotFound         = -1
	MethodNotAllowed = -2
)

type Router struct {
	outputs map[string][]*Output
}

// New returns a new Router.
func NewRouter() *Router {
	return &Router{make(map[string][]*Output)}
}

// Looks up the router and returns the output port index or -1 for not found,
// HTTP status code and new URI with resolved :params as GET variables
func (p *Router) Route(method, uri string) (int, url.Values) {
	for _, ph := range p.outputs[method] {
		if params, ok := ph.try(uri); ok {
			return ph.Index, params
		}
	}

	//allowed := make([]string, 0, len(p.outputs))
	allowedCount := 0
	for meth, outputs := range p.outputs {
		if meth == method {
			continue
		}
		for _, ph := range outputs {
			if _, ok := ph.try(uri); ok {
				//allowed = append(allowed, meth)
				allowedCount++
			}
		}
	}

	if allowedCount == 0 {
		return NotFound, nil
	}

	return MethodNotAllowed, nil
}

// Head will register a pattern with a handler for HEAD requests.
func (p *Router) Head(pat string, outputIndex int) {
	p.Add("HEAD", pat, outputIndex)
}

// Get will register a pattern with a handler for GET requests.
// It also registers pat for HEAD requests. If this needs to be overridden, use
// Head before Get with pat.
func (p *Router) Get(pat string, outputIndex int) {
	p.Add("HEAD", pat, outputIndex)
	p.Add("GET", pat, outputIndex)
}

// Post will register a pattern with a handler for POST requests.
func (p *Router) Post(pat string, outputIndex int) {
	p.Add("POST", pat, outputIndex)
}

// Put will register a pattern with a handler for PUT requests.
func (p *Router) Put(pat string, outputIndex int) {
	p.Add("PUT", pat, outputIndex)
}

// Del will register a pattern with a handler for DELETE requests.
func (p *Router) Del(pat string, outputIndex int) {
	p.Add("DELETE", pat, outputIndex)
}

// Options will register a pattern with a handler for OPTIONS requests.
func (p *Router) Options(pat string, outputIndex int) {
	p.Add("OPTIONS", pat, outputIndex)
}

// Add will register a pattern with a handler for meth requests.
func (p *Router) Add(meth, pat string, outputIndex int) {
	p.outputs[meth] = append(p.outputs[meth], &Output{outputIndex, pat})
	n := len(pat)
	if n > 0 && pat[n-1] == '/' {
		p.Add(meth, pat[:n-1], outputIndex)
	}
}

// Tail returns the trailing string in path after the final slash for a pat ending with a slash.
//
// Examples:
//
//	Tail("/hello/:title/", "/hello/mr/mizerany") == "mizerany"
//	Tail("/:a/", "/x/y/z")                       == "y/z"
//
func Tail(pat, path string) string {
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(pat):
			if pat[len(pat)-1] == '/' {
				return path[i:]
			}
			return ""
		case pat[j] == ':':
			var nextc byte
			_, nextc, j = match(pat, isAlnum, j+1)
			_, _, i = match(path, matchPart(nextc), i)
		case path[i] == pat[j]:
			i++
			j++
		default:
			return ""
		}
	}
	return ""
}

type Output struct {
	Index int
	pat   string
}

func (ph *Output) try(path string) (url.Values, bool) {
	p := make(url.Values)
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(ph.pat):
			if ph.pat != "/" && len(ph.pat) > 0 && ph.pat[len(ph.pat)-1] == '/' {
				return p, true
			}
			return nil, false
		case ph.pat[j] == ':':
			var name, val string
			var nextc byte
			name, nextc, j = match(ph.pat, isAlnum, j+1)
			val, _, i = match(path, matchPart(nextc), i)
			p.Add(":"+name, val)
		case path[i] == ph.pat[j]:
			i++
			j++
		default:
			return nil, false
		}
	}
	if j != len(ph.pat) {
		return nil, false
	}
	return p, true
}

func matchPart(b byte) func(byte) bool {
	return func(c byte) bool {
		return c != b && c != '/'
	}
}

func match(s string, f func(byte) bool, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) {
		j++
	}
	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

func isAlpha(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
