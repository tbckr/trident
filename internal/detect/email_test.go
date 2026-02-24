package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

func TestDetectEmailProvider_GoogleWorkspace(t *testing.T) {
	detections := detect.EmailProvider([]string{"aspmx.l.google.com."})
	require.Len(t, detections, 1)
	assert.Equal(t, detect.TypeEmail, detections[0].Type)
	assert.Equal(t, "Google Workspace", detections[0].Provider)
	assert.Equal(t, "aspmx.l.google.com.", detections[0].Evidence)
	assert.Equal(t, "mx", detections[0].Source)
}

func TestDetectEmailProvider_Microsoft365(t *testing.T) {
	detections := detect.EmailProvider([]string{"contoso-com.mail.protection.outlook.com."})
	require.Len(t, detections, 1)
	assert.Equal(t, "Microsoft 365", detections[0].Provider)
}

func TestDetectEmailProvider_UnknownHost(t *testing.T) {
	assert.Empty(t, detect.EmailProvider([]string{"mail.unknown-provider.example."}))
}

func TestDetectEmailProvider_EmptyInput(t *testing.T) {
	assert.Empty(t, detect.EmailProvider(nil))
	assert.Empty(t, detect.EmailProvider([]string{}))
}

func TestDetectEmailProvider_KnownProviders(t *testing.T) {
	tests := []struct {
		host     string
		provider string
	}{
		{"aspmx.l.google.com.", "Google Workspace"},
		{"mx.googlemail.com.", "Google Workspace"},
		{"tenant.mail.protection.outlook.com.", "Microsoft 365"},
		{"tenant.eo.outlook.com.", "Microsoft 365"},
		{"mx.pphosted.com.", "Proofpoint"},
		{"us-smtp-inbound-1.mimecast.com.", "Mimecast"},
		{"mail.barracudanetworks.com.", "Barracuda"},
		{"mx.sendgrid.net.", "SendGrid"},
		{"mxa.mailgun.org.", "Mailgun"},
		{"mx.zoho.com.", "ZOHO Mail"},
		{"mail.emailsrvr.com.", "Rackspace Email"},
		{"smtp.messagelabs.com.", "Broadcom Email Security"},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			detections := detect.EmailProvider([]string{tt.host})
			require.Len(t, detections, 1)
			assert.Equal(t, tt.provider, detections[0].Provider)
		})
	}
}

func TestDetectEmailProvider_DuplicateSuppression(t *testing.T) {
	hosts := []string{"aspmx.l.google.com.", "aspmx.l.google.com."}
	detections := detect.EmailProvider(hosts)
	require.Len(t, detections, 1)
	assert.Equal(t, "Google Workspace", detections[0].Provider)
}
