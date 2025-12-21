package config

// Config represents the complete Trident configuration.
type Config struct {
	// Global settings
	Global GlobalConfig `yaml:"global" mapstructure:"global"`

	// API keys for external services (Phase 3, structured for future use)
	APIKeys APIKeysConfig `yaml:"api_keys" mapstructure:"api_keys"`
}

// GlobalConfig holds global application settings.
type GlobalConfig struct {
	// Output format: text, json, plain
	Output string `yaml:"output" mapstructure:"output"`

	// Number of concurrent workers for bulk processing
	Concurrency int `yaml:"concurrency" mapstructure:"concurrency"`

	// Permissible Actions Protocol limit: red, amber, green, white
	PAPLimit string `yaml:"pap_limit" mapstructure:"pap_limit"`

	// Proxy URL (supports HTTP, HTTPS, SOCKS5)
	Proxy string `yaml:"proxy" mapstructure:"proxy"`

	// Custom User-Agent string
	UserAgent string `yaml:"user_agent" mapstructure:"user_agent"`

	// Enable output defanging
	Defang bool `yaml:"defang" mapstructure:"defang"`

	// Disable output defanging (overrides automatic defanging)
	NoDefang bool `yaml:"no_defang" mapstructure:"no_defang"`
}

// APIKeysConfig holds API keys for all external services.
// Phase 1 services (dns, asn, crtsh, threatminer, pgp) do not require keys.
// Phase 3 services are included for future-proofing.
type APIKeysConfig struct {
	// Phase 3 services
	BinaryEdge     string `yaml:"binaryedge" mapstructure:"binaryedge"`
	Censys         string `yaml:"censys" mapstructure:"censys"`
	CertSpotter    string `yaml:"certspotter" mapstructure:"certspotter"`
	CIRCL          string `yaml:"circl" mapstructure:"circl"`
	FullContact    string `yaml:"fullcontact" mapstructure:"fullcontact"`
	GitHub         string `yaml:"github" mapstructure:"github"`
	GreyNoise      string `yaml:"greynoise" mapstructure:"greynoise"`
	HIBP           string `yaml:"hibp" mapstructure:"hibp"`
	Hunter         string `yaml:"hunter" mapstructure:"hunter"`
	HybridAnalysis string `yaml:"hybrid_analysis" mapstructure:"hybrid_analysis"`
	IPInfo         string `yaml:"ipinfo" mapstructure:"ipinfo"`
	IP2LocationIO  string `yaml:"ip2locationio" mapstructure:"ip2locationio"`
	Koodous        string `yaml:"koodous" mapstructure:"koodous"`
	MalShare       string `yaml:"malshare" mapstructure:"malshare"`
	MISP           string `yaml:"misp" mapstructure:"misp"`
	NumVerify      string `yaml:"numverify" mapstructure:"numverify"`
	OpenCage       string `yaml:"opencage" mapstructure:"opencage"`
	OTX            string `yaml:"otx" mapstructure:"otx"`
	PermaCC        string `yaml:"permacc" mapstructure:"permacc"`
	PassiveTotal   string `yaml:"passivetotal" mapstructure:"passivetotal"`
	PulseDive      string `yaml:"pulsedive" mapstructure:"pulsedive"`
	SafeBrowsing   string `yaml:"safebrowsing" mapstructure:"safebrowsing"`
	SecurityTrails string `yaml:"securitytrails" mapstructure:"securitytrails"`
	Shodan         string `yaml:"shodan" mapstructure:"shodan"`
	SpyOnWeb       string `yaml:"spyonweb" mapstructure:"spyonweb"`
	Telegram       string `yaml:"telegram" mapstructure:"telegram"`
	ThreatCrowd    string `yaml:"threatcrowd" mapstructure:"threatcrowd"`
	ThreatGrid     string `yaml:"threatgrid" mapstructure:"threatgrid"`
	TotalHash      string `yaml:"totalhash" mapstructure:"totalhash"`
	Twitter        string `yaml:"twitter" mapstructure:"twitter"`
	URLHaus        string `yaml:"urlhaus" mapstructure:"urlhaus"`
	URLScan        string `yaml:"urlscan" mapstructure:"urlscan"`
	VirusTotal     string `yaml:"virustotal" mapstructure:"virustotal"`
	XForce         string `yaml:"xforce" mapstructure:"xforce"`
	Zetalytics     string `yaml:"zetalytics" mapstructure:"zetalytics"`
}

// NewDefaultConfig returns a Config with sensible defaults.
func NewDefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Output:      "text",
			Concurrency: 10,
			PAPLimit:    "white",
			Proxy:       "",
			UserAgent:   "",
			Defang:      false,
			NoDefang:    false,
		},
		APIKeys: APIKeysConfig{},
	}
}
