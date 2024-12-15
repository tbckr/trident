package shell

import (
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/tbckr/trident/pkg/report"
)

const tmpl = `Hostname:{{ tab }}{{ .Hostname }}
{{ if .ApexDomain }}Apex Domain:{{ tab }}{{ .ApexDomain }}{{ end }}
{{- if .AlexaRank }}Alexa Rank:{{ tab }}{{ .AlexaRank }}{{ end }}
{{- if .RecordReport.A }}
{{- range $index, $val := .RecordReport.A }}
{{ if eq $index 0 }}A Records:{{ end }}{{ tab }}{{ $val.IP }}{{ with $val.Organization }}{{ tab }}{{ . }}{{ end }}
{{- end }}
{{- end }}
`

type Writer struct {
	t template.Template
}

func NewShellWriter() (*Writer, error) {
	t, err := template.New("shell").Funcs(customFuncs()).Parse(tmpl)
	if err != nil {
		return nil, err
	}
	return &Writer{
		t: *t,
	}, nil
}

func customFuncs() template.FuncMap {
	f := template.FuncMap{
		"tab": func() string {
			return "\t"
		},
	}
	return f
}

func (w *Writer) WriteDomainReport(out io.Writer, domainReport report.DomainReport) error {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	data := domainReport
	err := w.t.Execute(tw, data)
	if err != nil {
		return err
	}
	err = tw.Flush()
	if err != nil {
		return err
	}
	return nil
}
