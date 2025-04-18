package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
)

func loadTemplates() (Template, error) {
	err := fs.WalkDir(os.DirFS("."), "view/templates", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		fmt.Println(path, d, err)
		return nil
	})

	if err != nil {
		return Template{}, err
	}

	baseTemplates := template.Must(template.ParseGlob("view/templates/base/*.gohtml"))

	return Template{
		LoginPage: template.Must(template.Must(baseTemplates.Clone()).ParseGlob("view/templates/page/login.gohtml")),
		DebugPage: template.Must(template.Must(baseTemplates.Clone()).ParseGlob("view/templates/page/debug.gohtml")),
	}, nil
}

type Template struct {
	LoginPage *template.Template
	DebugPage *template.Template
}
