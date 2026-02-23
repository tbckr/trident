package detect_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services/detect"
)

func TestMultiResult_WriteTable(t *testing.T) {
	m := &detect.MultiResult{}
	m.Results = []*detect.Result{
		{
			Input: "example.com",
			Detections: []detect.Detection{
				{Type: "CDN", Provider: "AWS CloudFront", Evidence: "foo.cloudfront.net."},
			},
		},
		{
			Input: "example.org",
			Detections: []detect.Detection{
				{Type: "Email", Provider: "Google Workspace", Evidence: "aspmx.l.google.com."},
			},
		},
	}

	var buf bytes.Buffer
	err := m.WriteTable(&buf)
	require.NoError(t, err)
	out := buf.String()

	assert.Contains(t, out, "DOMAIN")
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "PROVIDER")
	assert.Contains(t, out, "EVIDENCE")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "example.org")
	assert.Contains(t, out, "AWS CloudFront")
	assert.Contains(t, out, "Google Workspace")

	// example.com should appear before example.org
	comIdx := bytes.Index(buf.Bytes(), []byte("example.com"))
	orgIdx := bytes.Index(buf.Bytes(), []byte("example.org"))
	assert.Less(t, comIdx, orgIdx)
}
