package detect

import "strings"

// txtPattern maps a TXT record substring to a provider name and service type.
type txtPattern struct {
	substring   string
	provider    string
	serviceType ServiceType
}

var txtPatterns = []txtPattern{
	// SPF-based email provider detection
	{substring: "include:_spf.google.com", provider: "Google Workspace", serviceType: TypeEmail},
	{substring: "include:spf.protection.outlook.com", provider: "Microsoft 365", serviceType: TypeEmail},
	{substring: "include:_spf.salesforce.com", provider: "Salesforce", serviceType: TypeEmail},
	{substring: "include:spf.pphosted.com", provider: "Proofpoint", serviceType: TypeEmail},
	{substring: "include:spf.mimecast.com", provider: "Mimecast", serviceType: TypeEmail},
	{substring: "include:sendgrid.net", provider: "SendGrid", serviceType: TypeEmail},
	{substring: "include:servers.mcsv.net", provider: "Mailchimp", serviceType: TypeEmail},
	// Domain ownership verification tokens
	{substring: "google-site-verification=", provider: "Google", serviceType: TypeVerification},
	{substring: "MS=ms", provider: "Microsoft", serviceType: TypeVerification},
	{substring: "facebook-domain-verification=", provider: "Facebook", serviceType: TypeVerification},
	{substring: "hs-site-verification=", provider: "HubSpot", serviceType: TypeVerification},
	{substring: "atlassian-domain-verification=", provider: "Atlassian", serviceType: TypeVerification},
	{substring: "docusign=", provider: "DocuSign", serviceType: TypeVerification},
	{substring: "adobe-idp-site-verification=", provider: "Adobe", serviceType: TypeVerification},
	{substring: "zoom-domain-verification=", provider: "Zoom", serviceType: TypeVerification},
	{substring: "stripe-verification=", provider: "Stripe", serviceType: TypeVerification},
	{substring: "apple-domain-verification=", provider: "Apple", serviceType: TypeVerification},
}

// TXTRecord matches TXT record values against known provider patterns.
// Each txt value is checked against all patterns using substring matching.
// Duplicate detections (same provider+txt) are suppressed.
func TXTRecord(txts []string) []Detection {
	seen := map[string]bool{}
	var detections []Detection
	for _, txt := range txts {
		for _, p := range txtPatterns {
			if !strings.Contains(txt, p.substring) {
				continue
			}
			key := p.provider + ":" + txt
			if seen[key] {
				continue
			}
			seen[key] = true
			detections = append(detections, Detection{
				Type:     p.serviceType,
				Provider: p.provider,
				Evidence: txt,
				Source:   "txt",
			})
		}
	}
	return detections
}
