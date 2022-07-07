package util

import "text/template"

func sum(x, y int) int {
	return x + y
}

var TPLFuncMap = template.FuncMap{
	"sum": sum,
}
