package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"

	"github.com/Masterminds/sprig/v3"
	"github.com/valyala/bytebufferpool"
)

const baseTemplates = "view/templates/base/*.gohtml"

func loadTemplates(config Config) (Template, error) {
	if config.Debug {
		return Template{
			NewSubjectPatch: &devPage{name: "subject-edit.gohtml"},
			Debug:           &devPage{name: "debug.gohtml"},
			Login:           &devPage{name: "login.gohtml"},
		}, nil
	}

	base := template.Must(template.New("").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"setQuery": setQuery,
		"seq64":    seq64,
	}).ParseFS(templateFiles, baseTemplates))

	return Template{
		NewSubjectPatch: loadProdPage(base, "subject-edit.gohtml"),
		Debug:           loadProdPage(base, "debug.gohtml"),
		Login:           loadProdPage(base, "login.gohtml"),
	}, nil
}

func loadProdPage(base *template.Template, name string) *prodPage {
	raw, err := fs.ReadFile(templateFiles, "view/templates/page/"+name)
	if err != nil {
		panic(err)
	}

	return &prodPage{
		tmpl: template.Must(template.Must(base.Clone()).Parse(string(raw))),
	}
}

type Template struct {
	NewSubjectPatch Page
	Debug           Page
	Login           Page
}

type Page interface {
	Execute(wr io.Writer, data any) error
}

type prodPage struct {
	pool bytebufferpool.Pool
	tmpl *template.Template
}

func (p *prodPage) Execute(wr io.Writer, data any) error {
	buf := p.pool.Get()
	defer p.pool.Put(buf)

	err := p.tmpl.Execute(buf, data)
	if err != nil {
		return err
	}

	_, _ = buf.WriteTo(wr)
	return nil
}

type devPage struct {
	name string
}

func (p *devPage) Execute(wr io.Writer, data any) error {
	t := template.Must(template.New("").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"setQuery": setQuery,
		"seq64":    seq64,
	}).ParseGlob(baseTemplates))

	return template.Must(t.ParseFiles("view/templates/page/"+p.name)).ExecuteTemplate(wr, p.name, data)
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
