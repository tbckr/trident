package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

func TestTXTRecord_SPFEmail(t *testing.T) {
	tests := []struct {
		name     string
		txt      string
		provider string
	}{
		{"google workspace", "v=spf1 include:_spf.google.com ~all", "Google Workspace"},
		{"microsoft 365", "v=spf1 include:spf.protection.outlook.com ~all", "Microsoft 365"},
		{"salesforce", "v=spf1 include:_spf.salesforce.com ~all", "Salesforce"},
		{"proofpoint", "v=spf1 include:spf.pphosted.com ~all", "Proofpoint"},
		{"mimecast", "v=spf1 include:spf.mimecast.com ~all", "Mimecast"},
		{"sendgrid", "v=spf1 include:sendgrid.net ~all", "SendGrid"},
		{"mailchimp", "v=spf1 include:servers.mcsv.net ~all", "Mailchimp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections := detect.TXTRecord([]string{tt.txt})
			require.Len(t, detections, 1)
			assert.Equal(t, detect.TypeEmail, detections[0].Type)
			assert.Equal(t, tt.provider, detections[0].Provider)
			assert.Equal(t, tt.txt, detections[0].Evidence)
			assert.Equal(t, "txt", detections[0].Source)
		})
	}
}

func TestTXTRecord_Verification(t *testing.T) {
	tests := []struct {
		name     string
		txt      string
		provider string
	}{
		{"google", "google-site-verification=abc123", "Google"},
		{"microsoft", "MS=ms12345678", "Microsoft"},
		{"facebook", "facebook-domain-verification=xyz789", "Facebook"},
		{"hubspot", "hs-site-verification=token123", "HubSpot"},
		{"atlassian", "atlassian-domain-verification=abc", "Atlassian"},
		{"docusign", "docusign=abc123", "DocuSign"},
		{"adobe", "adobe-idp-site-verification=token", "Adobe"},
		{"zoom", "zoom-domain-verification=xyz", "Zoom"},
		{"stripe", "stripe-verification=abc", "Stripe"},
		{"apple", "apple-domain-verification=abc123", "Apple"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections := detect.TXTRecord([]string{tt.txt})
			require.Len(t, detections, 1)
			assert.Equal(t, detect.TypeVerification, detections[0].Type)
			assert.Equal(t, tt.provider, detections[0].Provider)
			assert.Equal(t, tt.txt, detections[0].Evidence)
			assert.Equal(t, "txt", detections[0].Source)
		})
	}
}

func TestTXTRecord_NoMatch(t *testing.T) {
	detections := detect.TXTRecord([]string{
		"v=spf1 ip4:192.0.2.0/24 ~all",
		"some-other-token=irrelevant",
	})
	assert.Empty(t, detections)
}

func TestTXTRecord_Empty(t *testing.T) {
	assert.Empty(t, detect.TXTRecord(nil))
	assert.Empty(t, detect.TXTRecord([]string{}))
}

func TestTXTRecord_Deduplication(t *testing.T) {
	// Same TXT value passed twice — should produce only one detection.
	txt := "google-site-verification=abc123"
	detections := detect.TXTRecord([]string{txt, txt})
	require.Len(t, detections, 1)
	assert.Equal(t, "Google", detections[0].Provider)
}

func TestTXTRecord_Multiple(t *testing.T) {
	txts := []string{
		"v=spf1 include:_spf.google.com ~all",
		"google-site-verification=abc123",
		"MS=ms12345678",
	}
	detections := detect.TXTRecord(txts)
	require.Len(t, detections, 3)
}
