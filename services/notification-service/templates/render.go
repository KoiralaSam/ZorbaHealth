package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

//go:embed *.tmpl
var templateFS embed.FS

func RenderText(name string, data any) (string, error) {
	tmpl, err := texttemplate.ParseFS(templateFS, name)
	if err != nil {
		return "", fmt.Errorf("parse text template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute text template %s: %w", name, err)
	}
	return buf.String(), nil
}

func RenderHTML(name string, data any) (string, error) {
	tmpl, err := template.ParseFS(templateFS, name)
	if err != nil {
		return "", fmt.Errorf("parse html template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute html template %s: %w", name, err)
	}
	return buf.String(), nil
}
