package report

type DomainReport struct {
	Hostname     string
	ApexDomain   string
	AlexaRank    int
	RecordReport RecordReport
}

type RecordReport struct {
	A    []ARecord
	AAAA []AaaaRecord
	MX   []MxRecord
	NS   []NsRecord
	SOA  []SoaRecord
	TXT  []TxtRecord
}

type Record struct {
	Type      string
	FirstSeen string
}

type ARecord struct {
	Record
	IP           string
	Organization string
}

type AaaaRecord struct {
	Record
	IP           string
	Organization string
}

type MxRecord struct {
	Record
	Hostname     string
	Priority     int
	Organization string
}

type NsRecord struct {
	Record
	Nameserver   string
	Organization string
}

type SoaRecord struct {
	Record
	Email string
	Ttl   int
}

type TxtRecord struct {
	Record
	Text string
}
