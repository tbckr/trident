package doh

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"

	"github.com/tbckr/trident/internal/apperr"
)

const txtTestName = "1.2.3.4.origin.asn.cymru.com."

// txtAnswer is one answer record of a hand-built DNS response.
type txtAnswer func(t *testing.T, b *dnsmessage.Builder, hdr dnsmessage.ResourceHeader)

// txtRecord builds a TXT answer holding the given character-strings.
func txtRecord(chunks ...string) txtAnswer {
	return func(t *testing.T, b *dnsmessage.Builder, hdr dnsmessage.ResourceHeader) {
		t.Helper()
		hdr.Type = dnsmessage.TypeTXT
		require.NoError(t, b.TXTResource(hdr, dnsmessage.TXTResource{TXT: chunks}))
	}
}

// cnameRecord builds a CNAME answer.
func cnameRecord(target string) txtAnswer {
	return func(t *testing.T, b *dnsmessage.Builder, hdr dnsmessage.ResourceHeader) {
		t.Helper()
		hdr.Type = dnsmessage.TypeCNAME
		require.NoError(t, b.CNAMEResource(hdr, dnsmessage.CNAMEResource{CNAME: dnsmessage.MustNewName(target)}))
	}
}

// buildTXTWire packs a DNS response with exact bytes via dnsmessage.Builder.
func buildTXTWire(t *testing.T, rcode dnsmessage.RCode, answers ...txtAnswer) []byte {
	t.Helper()
	name := dnsmessage.MustNewName(txtTestName)
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, RecursionDesired: true, RCode: rcode})
	require.NoError(t, b.StartQuestions())
	require.NoError(t, b.Question(dnsmessage.Question{Name: name, Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET}))
	if len(answers) > 0 {
		require.NoError(t, b.StartAnswers())
		for _, a := range answers {
			a(t, &b, dnsmessage.ResourceHeader{Name: name, Class: dnsmessage.ClassINET, TTL: 300})
		}
	}
	data, err := b.Finish()
	require.NoError(t, err)
	return data
}

// newTXTResolver returns a resolver on a mocked client that answers every DoH
// query with the given wire data.
func newTXTResolver(t *testing.T, data []byte) *TXTResolver {
	t.Helper()
	client := newMockClient(t)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+Endpoint,
		httpmock.NewBytesResponder(http.StatusOK, data))
	return NewTXTResolver(client)
}

func TestTXTResolver_RawBytes(t *testing.T) {
	// miekg/dns would present these as escapes (\", \\, \009, \195\188); the
	// resolver must return the bytes exactly as net.Resolver.LookupTXT does.
	want := []string{`a"b`, `c\d`, "tab\there", "ü"}
	answers := make([]txtAnswer, 0, len(want))
	for _, s := range want {
		answers = append(answers, txtRecord(s))
	}
	r := newTXTResolver(t, buildTXTWire(t, dnsmessage.RCodeSuccess, answers...))

	got, err := r.LookupTXT(context.Background(), txtTestName)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTXTResolver_ConcatenatesCharacterStrings(t *testing.T) {
	r := newTXTResolver(t, buildTXTWire(t, dnsmessage.RCodeSuccess,
		txtRecord("15169 | 8.8.8.0/24", " | US")))

	got, err := r.LookupTXT(context.Background(), txtTestName)
	require.NoError(t, err)
	assert.Equal(t, []string{"15169 | 8.8.8.0/24 | US"}, got)
}

func TestTXTResolver_MultipleRecordsInOrder(t *testing.T) {
	r := newTXTResolver(t, buildTXTWire(t, dnsmessage.RCodeSuccess,
		txtRecord("third"), txtRecord("first"), txtRecord("sec", "ond")))

	got, err := r.LookupTXT(context.Background(), txtTestName)
	require.NoError(t, err)
	assert.Equal(t, []string{"third", "first", "second"}, got)
}

func TestTXTResolver_IgnoresNonTXTAnswers(t *testing.T) {
	r := newTXTResolver(t, buildTXTWire(t, dnsmessage.RCodeSuccess,
		cnameRecord("alias.example.com."), txtRecord("v=spf1 -all")))

	got, err := r.LookupTXT(context.Background(), txtTestName)
	require.NoError(t, err)
	assert.Equal(t, []string{"v=spf1 -all"}, got)
}

func TestTXTResolver_NXDOMAINAndNoData(t *testing.T) {
	tests := []struct {
		name  string
		rcode dnsmessage.RCode
	}{
		{name: "nxdomain", rcode: dnsmessage.RCodeNameError},
		{name: "no data", rcode: dnsmessage.RCodeSuccess},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTXTResolver(t, buildTXTWire(t, tt.rcode))

			got, err := r.LookupTXT(context.Background(), txtTestName)
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestTXTResolver_ServFail(t *testing.T) {
	r := newTXTResolver(t, buildTXTWire(t, dnsmessage.RCodeServerFailure))

	got, err := r.LookupTXT(context.Background(), txtTestName)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrRequestFailed)
	assert.Contains(t, err.Error(), "RCodeServerFailure")
	assert.Nil(t, got)
}

func TestTXTResolver_HTTPError(t *testing.T) {
	client := newMockClient(t)
	httpmock.RegisterResponder(http.MethodGet, "=~^"+Endpoint,
		httpmock.NewStringResponder(http.StatusBadGateway, "bad gateway"))
	r := NewTXTResolver(client)

	got, err := r.LookupTXT(context.Background(), txtTestName)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrRequestFailed)
	assert.Nil(t, got)
}

// TestTXTResolver_TruncatedResponse cuts a well-formed NOERROR response at
// each structural boundary, so every step of the wire parser fails in turn.
// A non-DNS body such as "garbage bytes" would not do: it is long enough for a
// header whose flags decode to SERVFAIL, so it never reaches the parser.
func TestTXTResolver_TruncatedResponse(t *testing.T) {
	const (
		headerLen      = 12 // fixed DNS header
		questionFixed  = 4  // QTYPE + QCLASS after the name
		answerFixedLen = 10 // TYPE + CLASS + TTL + RDLENGTH after the name
	)
	// A response without answers is exactly header + question.
	questionEnd := len(buildTXTWire(t, dnsmessage.RCodeSuccess))
	nameLen := questionEnd - headerLen - questionFixed
	answerHeaderEnd := questionEnd + nameLen + answerFixedLen

	tests := []struct {
		name string
		wire func(t *testing.T) []byte
	}{
		{
			name: "header cut short",
			wire: func(t *testing.T) []byte {
				return buildTXTWire(t, dnsmessage.RCodeSuccess, txtRecord("v=spf1 -all"))[:headerLen-1]
			},
		},
		{
			name: "question cut inside the name",
			wire: func(t *testing.T) []byte {
				return buildTXTWire(t, dnsmessage.RCodeSuccess, txtRecord("v=spf1 -all"))[:headerLen+3]
			},
		},
		{
			name: "answer section missing",
			wire: func(t *testing.T) []byte {
				return buildTXTWire(t, dnsmessage.RCodeSuccess, txtRecord("v=spf1 -all"))[:questionEnd]
			},
		},
		{
			name: "answer header cut inside the fixed fields",
			wire: func(t *testing.T) []byte {
				return buildTXTWire(t, dnsmessage.RCodeSuccess, txtRecord("v=spf1 -all"))[:answerHeaderEnd-3]
			},
		},
		{
			name: "non-TXT answer body cut short",
			wire: func(t *testing.T) []byte {
				data := buildTXTWire(t, dnsmessage.RCodeSuccess, cnameRecord("alias.example.com."))
				return data[:len(data)-1]
			},
		},
		{
			name: "TXT answer body cut short",
			wire: func(t *testing.T) []byte {
				data := buildTXTWire(t, dnsmessage.RCodeSuccess, txtRecord("v=spf1 -all"))
				return data[:len(data)-1]
			},
		},
		{
			name: "second answer cut short after a good one",
			wire: func(t *testing.T) []byte {
				data := buildTXTWire(t, dnsmessage.RCodeSuccess, txtRecord("first"), txtRecord("second"))
				return data[:len(data)-1]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTXTResolver(t, tt.wire(t))

			got, err := r.LookupTXT(context.Background(), txtTestName)
			require.Error(t, err)
			assert.ErrorIs(t, err, apperr.ErrRequestFailed)
			assert.Contains(t, err.Error(), "failed to parse DNS response")
			assert.Nil(t, got, "a partial answer must not leak out of a failed parse")
		})
	}
}
