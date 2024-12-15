package shell

import (
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/tbckr/trident/pkg/report"
)

// TODO adapt this template to domain descriptions or create a new one
const tmpl = `Hostname:{{ tab }}{{ .Hostname }}
{{ if .ApexDomain }}Apex Domain:{{ tab }}{{ .ApexDomain }}{{ end }}
{{- if .AlexaRank }}Alexa Rank:{{ tab }}{{ .AlexaRank }}{{ end }}
{{- if .RecordReport.A }}
{{- range $index, $val := .RecordReport.A }}
{{ if eq $index 0 }}A Records:{{ end }}{{ tab }}{{ $val.IP }}{{ with $val.Organization }}{{ tab }}{{ . }}{{ end }}
{{- end }}
{{- end }}
{{- if .RecordReport.AAAA }}
{{- range $index, $val := .RecordReport.AAAA }}
{{ if eq $index 0 }}AAAA Records:{{ end }}{{ tab }}{{ $val.IP }}{{ with $val.Organization }}{{ tab }}{{ . }}{{ end }}
{{- end }}
{{- end }}
{{- if .RecordReport.MX }}
{{- range $index, $val := .RecordReport.MX }}
{{ if eq $index 0 }}MX Records:{{ end }}{{ tab }}{{ $val.Hostname }}{{ tab }}{{ $val.Priority }}{{ with $val.Organization }}{{ tab }}{{ . }}{{ end }}
{{- end }}
{{- end }}
{{- if .RecordReport.NS }}
{{- range $index, $val := .RecordReport.NS }}
{{ if eq $index 0 }}NS Records:{{ end }}{{ tab }}{{ $val.Nameserver }}{{ with $val.Organization }}{{ tab }}{{ . }}{{ end }}
{{- end }}
{{- end }}
{{- if .RecordReport.SOA }}
{{- range $index, $val := .RecordReport.SOA }}
{{ if eq $index 0 }}SOA Records:{{ end }}{{ tab }}{{ $val.Email }}{{ tab }}{{ $val.Ttl }}
{{- end }}
{{- end }}
{{- if .RecordReport.TXT }}
{{- range $index, $val := .RecordReport.TXT }}
{{ if eq $index 0 }}TXT Records:{{ end }}{{ tab }}{{ $val.Text }}
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

// TODO refactor this function to use the same template as WriteDomainReport
func (w *Writer) WriteDomainDescriptionReport(stdout io.Writer, description report.DomainDescriptionReport) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	data := description
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
