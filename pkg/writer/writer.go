package writer

import (
	"io"
	"text/template"
)

const shell = `{{ .domain }}`

type TemplateWriter struct {
	tmpl *template.Template
}

func NewTemplateWriter() (*TemplateWriter, error) {
	tmpl, err := template.New("shell").Parse(shell)
	if err != nil {
		return &TemplateWriter{}, err
	}
	return &TemplateWriter{
		tmpl: tmpl,
	}, nil
}

func (w TemplateWriter) Fprint(out io.Writer) error {
	data := map[string]interface{}{
		"domain": "test",
	}
	return w.tmpl.Execute(out, data)
}
