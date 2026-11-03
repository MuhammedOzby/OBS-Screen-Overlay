package lib

import (
	"html/template"
)

// panelTemplates lists the HTML template files parsed for the admin panel.
// Paths are relative to the application's working directory.
var panelTemplates = []string{
	"template/index.html.tmpl",
	"template/new.html.tmpl",
	"template/edit.html.tmpl",
	"template/access.html.tmpl",
	"template/devices.html.tmpl",
}

// LoadTemplates parses the three panel template files and returns a single
// parsed *html/template.Template. The template is parsed once and reused, so
// it is safe to pass to handlers.
func LoadTemplates() (*template.Template, error) {
	tpl, err := template.ParseFiles(panelTemplates...)
	if err != nil {
		return nil, err
	}
	return tpl, nil
}
