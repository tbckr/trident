package detect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tbckr/trident/internal/detect"
)

// allTXTPatterns is the full set of TXT patterns used across TXT tests.
var allTXTPatterns = []detect.TXTPattern{
	{Substring: "include:_spf.google.com", Provider: "Google Workspace", Type: detect.TypeEmail},
	{Substring: "include:spf.protection.outlook.com", Provider: "Microsoft 365", Type: detect.TypeEmail},
	{Substring: "include:_spf.salesforce.com", Provider: "Salesforce", Type: detect.TypeEmail},
	{Substring: "include:spf.pphosted.com", Provider: "Proofpoint", Type: detect.TypeEmail},
	{Substring: "include:spf.mimecast.com", Provider: "Mimecast", Type: detect.TypeEmail},
	{Substring: "include:sendgrid.net", Provider: "SendGrid", Type: detect.TypeEmail},
	{Substring: "include:servers.mcsv.net", Provider: "Mailchimp", Type: detect.TypeEmail},
	{Substring: "google-site-verification=", Provider: "Google", Type: detect.TypeVerification},
	{Substring: "MS=ms", Provider: "Microsoft", Type: detect.TypeVerification},
	{Substring: "facebook-domain-verification=", Provider: "Facebook", Type: detect.TypeVerification},
	{Substring: "hs-site-verification=", Provider: "HubSpot", Type: detect.TypeVerification},
	{Substring: "atlassian-domain-verification=", Provider: "Atlassian", Type: detect.TypeVerification},
	{Substring: "docusign=", Provider: "DocuSign", Type: detect.TypeVerification},
	{Substring: "adobe-idp-site-verification=", Provider: "Adobe", Type: detect.TypeVerification},
	{Substring: "zoom-domain-verification=", Provider: "Zoom", Type: detect.TypeVerification},
	{Substring: "stripe-verification=", Provider: "Stripe", Type: detect.TypeVerification},
	{Substring: "apple-domain-verification=", Provider: "Apple", Type: detect.TypeVerification},
}

func newTXTDetector() *detect.Detector {
	return detect.NewDetector(detect.Patterns{TXT: allTXTPatterns})
}

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
	d := newTXTDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections := d.TXTRecord([]string{tt.txt})
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
	d := newTXTDetector()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections := d.TXTRecord([]string{tt.txt})
			require.Len(t, detections, 1)
			assert.Equal(t, detect.TypeVerification, detections[0].Type)
			assert.Equal(t, tt.provider, detections[0].Provider)
			assert.Equal(t, tt.txt, detections[0].Evidence)
			assert.Equal(t, "txt", detections[0].Source)
		})
	}
}

func TestTXTRecord_NoMatch(t *testing.T) {
	detections := newTXTDetector().TXTRecord([]string{
		"v=spf1 ip4:192.0.2.0/24 ~all",
		"some-other-token=irrelevant",
	})
	assert.Empty(t, detections)
}

func TestTXTRecord_Empty(t *testing.T) {
	d := newTXTDetector()
	assert.Empty(t, d.TXTRecord(nil))
	assert.Empty(t, d.TXTRecord([]string{}))
}

func TestTXTRecord_Deduplication(t *testing.T) {
	// Same TXT value passed twice — should produce only one detection.
	txt := "google-site-verification=abc123"
	detections := newTXTDetector().TXTRecord([]string{txt, txt})
	require.Len(t, detections, 1)
	assert.Equal(t, "Google", detections[0].Provider)
}

func TestTXTRecord_Multiple(t *testing.T) {
	txts := []string{
		"v=spf1 include:_spf.google.com ~all",
		"google-site-verification=abc123",
		"MS=ms12345678",
	}
	detections := newTXTDetector().TXTRecord(txts)
	require.Len(t, detections, 3)
}
