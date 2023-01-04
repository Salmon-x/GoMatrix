package response

import (
	"net/http"
	"text/template"
)

type HTML struct {
	Template *template.Template
	Name     string
	Data     interface{}
}

type HTMLRender interface {
	Instance(string, interface{}) Render
}

type HTMLProduction struct {
	Template *template.Template
}

func (r HTMLProduction) Instance(name string, data interface{}) Render {
	return HTML{
		Template: r.Template,
		Name:     name,
		Data:     data,
	}
}

func (h HTML) Render(w http.ResponseWriter) (err error) {
	writeContentType(w, htmlContentType)
	if err = h.Template.ExecuteTemplate(w, h.Name, h.Data); err != nil {
		return err
	}
	return
}

func (h HTML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, htmlContentType)
}
