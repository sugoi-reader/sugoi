package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

var templateStore *template.Template

func RenderPage(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	var err error
	r.ParseForm()

	if templateStore == nil || config.Debug {
		tmpl := template.New("")

		tmpl = tmpl.Funcs(template.FuncMap{"noescape": func(str string) template.HTML {
			return template.HTML(str)
		}})

		tmpl = tmpl.Funcs(template.FuncMap{"trd": func(str string) template.HTML {
			s := r.FormValue(str)
			return template.HTML(s)
		}})

		tmpl = tmpl.Funcs(template.FuncMap{"getBuildTime": func() template.HTML {
			return template.HTML(buildStr)
		}})

		tmpl = tmpl.Funcs(template.FuncMap{"getBuildDiff": func() template.HTML {
			return template.HTML(BuildDiff())
		}})

		tmpl, err = tmpl.ParseGlob("templates/*.gohtml")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		tmpl, err = tmpl.ParseGlob("pages/*.gohtml")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		// _, err := tmpl.Parse(`{{define "content"}}{{end}}`)

		fmt.Println(tmpl.DefinedTemplates())

		templateStore = tmpl
	}

	err = templateStore.ExecuteTemplate(w, name, data)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
}

func RenderError(w http.ResponseWriter, r *http.Request, message string) {
	data := struct {
		Title string
		Error string
	}{}

	data.Error = message
	data.Title = "Error"
	debugPrintf(message)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(400)
	RenderPage(w, r, "error.gohtml", data)
}

func SetCacheHeader(w http.ResponseWriter, r *http.Request, age uint) {
	h := w.Header()

	h.Set("Cache-Control", fmt.Sprintf("public, max-age=%d", age))

	ageD, _ := time.ParseDuration(fmt.Sprintf("%ds", age))
	cacheUntil := time.Now().Add(ageD)
	h.Set("Expires", cacheUntil.Format(http.TimeFormat))
}

type JsonResponse struct {
	Message string
}

type JsonError struct {
	Error string
}
