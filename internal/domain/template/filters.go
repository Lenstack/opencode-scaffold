package template

import (
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"kebabcase":  strcase.ToKebab,
		"snakecase":  strcase.ToSnake,
		"camelcase":  strcase.ToCamel,
		"pascalcase": strcase.ToCamel,
		"titlecase":  strings.Title,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"replace":    strings.ReplaceAll,
		"trim":       strings.TrimSpace,
	}
}
