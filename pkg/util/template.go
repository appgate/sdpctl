package util

import (
	"text/template"
	"time"
)

func sum(x, y int) int {
	return x + y
}

func now() string {
	return time.Now().Format(time.RFC3339)
}

var TPLFuncMap = template.FuncMap{
	"sum": sum,
	"now": now,
}
