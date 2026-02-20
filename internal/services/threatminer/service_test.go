package threatminer_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/services"
	"github.com/tbckr/trident/internal/services/threatminer"
	"github.com/tbckr/trident/internal/testutil"
)

func newTestClient(t *testing.T) *req.Client {
	t.Helper()
	client := req.NewClient()
	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)
	return client
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

// ---------- Domain ----------

func TestRun_Domain(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/domain.php?q=example.com&rt=2",
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/domain_pdns.json")),
	)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/domain.php?q=example.com&rt=5",
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/domain_subdomains.json")),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*threatminer.Result)
	require.True(t, ok, "expected *threatminer.Result")

	assert.Equal(t, "example.com", result.Input)
	assert.Equal(t, "domain", result.InputType)
	assert.Len(t, result.PassiveDNS, 2)
	assert.Equal(t, "93.184.216.34", result.PassiveDNS[0].IP)
	assert.Equal(t, []string{"www.example.com", "mail.example.com", "api.example.com"}, result.Subdomains)
}

func TestRun_Domain_Empty(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/domain.php?q=example.com&rt=2",
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/empty.json")),
	)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/domain.php?q=example.com&rt=5",
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/empty.json")),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "example.com")
	require.NoError(t, err)

	result, ok := raw.(*threatminer.Result)
	require.True(t, ok)
	assert.True(t, result.IsEmpty())
}

// ---------- IP ----------

func TestRun_IP(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/host.php?q=8.8.8.8&rt=2",
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/ip_pdns.json")),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "8.8.8.8")
	require.NoError(t, err)

	result, ok := raw.(*threatminer.Result)
	require.True(t, ok, "expected *threatminer.Result")

	assert.Equal(t, "8.8.8.8", result.Input)
	assert.Equal(t, "ip", result.InputType)
	assert.Len(t, result.PassiveDNS, 1)
	assert.Equal(t, "dns.google", result.PassiveDNS[0].Domain)
}

// ---------- Hash ----------

func TestRun_Hash_MD5(t *testing.T) {
	client := newTestClient(t)
	hash := "d41d8cd98f00b204e9800998ecf8427e"
	httpmock.RegisterResponder(http.MethodGet,
		fmt.Sprintf("https://api.threatminer.org/v2/sample.php?q=%s&rt=1", hash),
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/hash_metadata.json")),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), hash)
	require.NoError(t, err)

	result, ok := raw.(*threatminer.Result)
	require.True(t, ok)
	assert.Equal(t, "hash", result.InputType)
	require.NotNil(t, result.HashInfo)
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", result.HashInfo.MD5)
	assert.Equal(t, "PE32", result.HashInfo.FileType)
}

func TestRun_Hash_SHA256(t *testing.T) {
	client := newTestClient(t)
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	httpmock.RegisterResponder(http.MethodGet,
		fmt.Sprintf("https://api.threatminer.org/v2/sample.php?q=%s&rt=1", hash),
		httpmock.NewBytesResponder(http.StatusOK, mustReadFile(t, "testdata/hash_metadata.json")),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), hash)
	require.NoError(t, err)

	result, ok := raw.(*threatminer.Result)
	require.True(t, ok)
	assert.Equal(t, "hash", result.InputType)
}

// ---------- Invalid input ----------

func TestRun_InvalidInput(t *testing.T) {
	client := newTestClient(t)
	svc := threatminer.NewService(client, testutil.NopLogger())

	for _, bad := range []string{"", "not valid!", "has space.com"} {
		_, err := svc.Run(context.Background(), bad)
		require.Error(t, err, "input %q should be invalid", bad)
		assert.ErrorIs(t, err, services.ErrInvalidInput)
	}
}

// ---------- HTTP errors ----------

func TestRun_HTTPFailure(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/host.php?q=8.8.8.8&rt=2",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	_, err := svc.Run(context.Background(), "8.8.8.8")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
}

func TestRun_NetworkError(t *testing.T) {
	client := newTestClient(t)
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/host.php?q=1.2.3.4&rt=2",
		httpmock.NewErrorResponder(fmt.Errorf("connection refused")),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	_, err := svc.Run(context.Background(), "1.2.3.4")
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrRequestFailed)
}

// ---------- ANSI sanitization ----------

func TestRun_ANSISanitization(t *testing.T) {
	client := newTestClient(t)
	body := `{"status_code":"200","status_message":"Results found.","results":[{"ip":"\u001b[31m1.2.3.4\u001b[0m","domain":"evil.com","first_seen":"","last_seen":""}]}`
	httpmock.RegisterResponder(http.MethodGet,
		"https://api.threatminer.org/v2/host.php?q=1.2.3.4&rt=2",
		httpmock.NewStringResponder(http.StatusOK, body),
	)

	svc := threatminer.NewService(client, testutil.NopLogger())
	raw, err := svc.Run(context.Background(), "1.2.3.4")
	require.NoError(t, err)
	result, ok := raw.(*threatminer.Result)
	require.True(t, ok)
	for _, e := range result.PassiveDNS {
		assert.NotContains(t, e.IP, "\x1b")
		assert.NotContains(t, e.Domain, "\x1b")
	}
}

// ---------- Result methods ----------

func TestResult_IsEmpty(t *testing.T) {
	assert.True(t, (&threatminer.Result{}).IsEmpty())
	assert.False(t, (&threatminer.Result{Subdomains: []string{"www.example.com"}}).IsEmpty())
}

func TestResult_WriteText_Domain(t *testing.T) {
	result := &threatminer.Result{
		Input:     "example.com",
		InputType: "domain",
		PassiveDNS: []threatminer.PDNSEntry{
			{IP: "1.2.3.4", Domain: "example.com", FirstSeen: "2021-01-01", LastSeen: "2024-01-01"},
		},
		Subdomains: []string{"www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4")
	assert.Contains(t, out, "www.example.com")
}

func TestResult_WriteText_Hash(t *testing.T) {
	result := &threatminer.Result{
		Input:     "d41d8cd98f00b204e9800998ecf8427e",
		InputType: "hash",
		HashInfo: &threatminer.HashMetadata{
			MD5:      "d41d8cd98f00b204e9800998ecf8427e",
			FileType: "PE32",
		},
	}
	var buf bytes.Buffer
	err := result.WriteText(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "d41d8cd98f00b204e9800998ecf8427e")
	assert.Contains(t, out, "PE32")
}

func TestResult_WritePlain_Domain(t *testing.T) {
	result := &threatminer.Result{
		Input:     "example.com",
		InputType: "domain",
		PassiveDNS: []threatminer.PDNSEntry{
			{IP: "1.2.3.4", Domain: "example.com"},
		},
		Subdomains: []string{"www.example.com"},
	}
	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "1.2.3.4 example.com")
	assert.Contains(t, out, "www.example.com")
}

func TestResult_WritePlain_Hash(t *testing.T) {
	result := &threatminer.Result{
		Input:     "d41d8cd98f00b204e9800998ecf8427e",
		InputType: "hash",
		HashInfo: &threatminer.HashMetadata{
			MD5:      "d41d8cd98f00b204e9800998ecf8427e",
			FileType: "PE32",
		},
	}
	var buf bytes.Buffer
	err := result.WritePlain(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "MD5: d41d8cd98f00b204e9800998ecf8427e")
	assert.Contains(t, out, "FileType: PE32")
}

func TestService_PAP(t *testing.T) {
	svc := threatminer.NewService(req.NewClient(), testutil.NopLogger())
	assert.Equal(t, "amber", svc.PAP().String())
}
