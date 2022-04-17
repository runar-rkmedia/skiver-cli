package main

import (
	"os"
	"os/exec"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/runar-rkmedia/go-common/logger"
)

var l = logger.InitLogger(logger.LogConfig{
	Level:      "debug",
	Format:     "human",
	WithCaller: true,
})

func getUsage() string {

	c := exec.Command("go", "run", "main.go")

	out, err := c.CombinedOutput()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to get usage")
	}
	return string(out)
}

func main() {
	tmpl, err := prepareTemplates().ParseFiles("./templating/baseReadme.md")
	if err != nil {
		l.Fatal().Err(err).Msg("failed to read readme")
	}
	f, err := os.OpenFile("./README.md", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		l.Fatal().Err(err).Msg("failed to open readme for writing")
	}
	data := map[string]interface{}{
		"Header": "This file is generated.",
		"Usage":  getUsage(),
	}

	err = tmpl.ExecuteTemplate(f, "baseReadme.md", data)
	if err != nil {
		l.Fatal().Err(err).Msg("Failed during templating")
	}
	l.Info().Msg("Success")
}

func prepareTemplates() *template.Template {
	t := template.New("")
	funcMap := sprig.TxtFuncMap()

	t.Funcs(funcMap)
	return t
}
