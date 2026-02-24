package detect

var emailPatterns = []pattern{
	{"google.com", "Google Workspace"},
	{"googlemail.com", "Google Workspace"},
	{"mail.protection.outlook.com", "Microsoft 365"},
	{"eo.outlook.com", "Microsoft 365"},
	{"pphosted.com", "Proofpoint"},
	{"mimecast.com", "Mimecast"},
	{"barracudanetworks.com", "Barracuda"},
	{"sendgrid.net", "SendGrid"},
	{"mailgun.org", "Mailgun"},
	{"zoho.com", "ZOHO Mail"},
	{"emailsrvr.com", "Rackspace Email"},
	{"messagelabs.com", "Broadcom Email Security"},
}

// EmailProvider matches MX exchange hostnames against known email provider
// patterns and returns one Detection per unique (provider, evidence) pair.
func EmailProvider(mxHosts []string) []Detection {
	var detections []Detection
	seen := map[string]bool{}
	for _, host := range mxHosts {
		for _, p := range emailPatterns {
			if matchSuffix(host, p.suffix) {
				key := p.provider + ":" + host
				if seen[key] {
					continue
				}
				seen[key] = true
				detections = append(detections, Detection{
					Type:     TypeEmail,
					Provider: p.provider,
					Evidence: host,
					Source:   "mx",
				})
			}
		}
	}
	return detections
}
