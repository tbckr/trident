package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type Formatter interface {
	Format(data interface{}) (string, error)
}

type TableFormatter struct {
	Header []string
}

func (f *TableFormatter) Format(data interface{}) (string, error) {
	rows, ok := data.([][]string)
	if !ok {
		return "", fmt.Errorf("invalid data type for table formatter")
	}

	var sb strings.Builder
	table := tablewriter.NewWriter(&sb)

	header := make([]any, len(f.Header))
	for i, h := range f.Header {
		header[i] = h
	}
	table.Header(header...)

	// table.SetBorder(false) // Let's see if this version has a similar method if needed
	if err := table.Bulk(rows); err != nil {
		return "", err
	}
	if err := table.Render(); err != nil {
		return "", err
	}

	return sb.String(), nil
}

type JSONFormatter struct{}

func (f *JSONFormatter) Format(data interface{}) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type PlainFormatter struct{}

func (f *PlainFormatter) Format(data interface{}) (string, error) {
	switch v := data.(type) {
	case []string:
		return strings.Join(v, "\n"), nil
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func NewFormatter(format string, header []string) Formatter {
	switch strings.ToLower(format) {
	case "json":
		return &JSONFormatter{}
	case "plain":
		return &PlainFormatter{}
	default:
		return &TableFormatter{Header: header}
	}
}
