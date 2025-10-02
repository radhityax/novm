package main

import (
	"html/template"
)

var tmplFuncs = template.FuncMap {
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"add": func(a, b int) int {
		return a+b
	},
	"sub": func(a, b int) int {
		return a-b
	},
}
