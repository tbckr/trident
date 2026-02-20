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

func (f *fakeResult) WriteText(w io.Writer) error {
	_, err := w.Write([]byte("text:" + f.Name))
	return err
}

func (f *fakeResult) WritePlain(w io.Writer) error {
	_, err := w.Write([]byte("plain:" + f.Name))
	return err
}

func TestWrite_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatJSON, &fakeResult{Name: "test"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name"`)
	assert.Contains(t, buf.String(), `"test"`)
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

func TestWrite_Plain(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatPlain, &fakeResult{Name: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "plain:hello", buf.String())
}

func TestWrite_Plain_NotFormattable(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.FormatPlain, struct{ X int }{X: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support plain output")
}

func TestWrite_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, output.Format("xml"), struct{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
}
