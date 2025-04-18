package main

import (
	"fmt"
	"html/template"
	"io"
	"net/url"

	"github.com/Masterminds/sprig/v3"
)

func loadTemplates(config Config) (Template, error) {
	//var executor TemplateExecutor
	//
	//if config.Debug {
	//	executor = DebugTemplateExecutor{"view/templates/base/*.gohtml"}
	//} else {
	//	executor = ReleaseTemplateExecutor{
	//		template.Must(template.ParseGlob("")).Funcs(sprig.FuncMap()),
	//	}
	//}

	return Template{DebugTemplateExecutor{"view/templates/base/*.gohtml"}}, nil
	//_ = executor

	//baseTemplates := template.Must(template.ParseGlob("view/templates/base/*.gohtml")).Funcs(sprig.FuncMap())

	//return Template{
	//	LoginPage: template.Must(template.Must(baseTemplates.Clone()).ParseGlob("view/templates/page/login.gohtml")),
	//DebugPage: template.Must(template.Must(baseTemplates.Clone()).ParseGlob("view/templates/page/debug.gohtml")),
	//}, nil
}

type Template struct {
	Executor TemplateExecutor
	//LoginPage *template.Template
	//DebugPage *template.Template
}

type TemplateExecutor interface {
	ExecuteTemplate(wr io.Writer, name string, data any) error
}

type DebugTemplateExecutor struct {
	Glob string
}

func (e DebugTemplateExecutor) ExecuteTemplate(wr io.Writer, name string, data any) error {
	t := template.Must(template.New("").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"setQuery": setQuery,
		"seq64":    seq64,
	}).ParseGlob(e.Glob))

	return template.Must(t.ParseFiles("view/templates/page/"+name)).ExecuteTemplate(wr, name, data)
}

type ReleaseTemplateExecutor struct {
	Template *template.Template
}

func (e ReleaseTemplateExecutor) ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	return e.Template.ExecuteTemplate(wr, name, data)
}

func setQuery(u *url.URL, key string, value any) string {
	q := u.Query()

	q.Set(key, fmt.Sprint(value))

	return u.Path + "?" + q.Encode()
}

func seq64(start int64, end int64) []int64 {
	var r []int64
	for i := start; i < end; i++ {
		r = append(r, i)
	}
	return r
}
