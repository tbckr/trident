package output_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/output"
)

type fakeResult struct {
	Name string `json:"name"`
}

func (f *fakeResult) WriteTable(w io.Writer) error {
	_, err := w.Write([]byte("text:" + f.Name))
	return err
}

func (f *fakeResult) WriteText(w io.Writer) error {
	_, err := w.Write([]byte("text:" + f.Name))
	return err
}

func TestWrite_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatJSON, &fakeResult{Name: "test"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name"`)
	assert.Contains(t, buf.String(), `"test"`)
}

func TestWrite_Table(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatTable, &fakeResult{Name: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "text:hello", buf.String())
}

func TestWrite_Table_NotFormattable(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatTable, struct{ X int }{X: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support table output")
}

func TestWrite_Text(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatText, &fakeResult{Name: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "text:hello", buf.String())
}

func TestWrite_Text_NotFormattable(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatText, struct{ X int }{X: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support text output")
}

func TestWrite_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.Format("xml"), struct{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
}
