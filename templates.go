package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"

	"github.com/Masterminds/sprig/v3"
	"github.com/valyala/bytebufferpool"

	"app/view"
)

const baseTemplates = "view/templates/base/*.gohtml"

func loadTemplates(config Config) (Template, error) {
	if config.Debug {
		return Template{
			EditSubject: &devPage[view.SubjectPatchEdit]{name: "subject-edit.gohtml"},
			EditEpisode: &devPage[view.EpisodePatchEdit]{name: "episode-edit.gohtml"},
		}, nil
	}

	base := template.Must(template.New("").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"setQuery": setQuery,
		"seq64":    seq64,
	}).ParseFS(templateFiles, baseTemplates))

	return Template{
		EditSubject: loadProdPage[view.SubjectPatchEdit](base, "subject-edit.gohtml"),
		EditEpisode: loadProdPage[view.EpisodePatchEdit](base, "episode-edit.gohtml"),
	}, nil
}

func loadProdPage[T any](base *template.Template, name string) *prodPage[T] {
	raw, err := fs.ReadFile(templateFiles, "view/templates/page/"+name)
	if err != nil {
		panic(err)
	}

	return &prodPage[T]{
		tmpl: template.Must(template.Must(base.Clone()).Parse(string(raw))),
	}
}

type Template struct {
	EditSubject Page[view.SubjectPatchEdit]
	EditEpisode Page[view.EpisodePatchEdit]
}

type Page[T any] interface {
	Execute(wr io.Writer, data T) error
}

type prodPage[T any] struct {
	pool bytebufferpool.Pool
	tmpl *template.Template
}

func (p *prodPage[T]) Execute(wr io.Writer, data T) error {
	buf := p.pool.Get()
	defer p.pool.Put(buf)

	err := p.tmpl.Execute(buf, data)
	if err != nil {
		return err
	}

	_, _ = buf.WriteTo(wr)
	return nil
}

type devPage[T any] struct {
	name string
}

func (p *devPage[T]) Execute(wr io.Writer, data T) error {
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
