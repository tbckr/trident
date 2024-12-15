package report

type DomainReport struct {
	Hostname     string
	ApexDomain   string
	AlexaRank    int
	RecordReport RecordReport
}

type RecordReport struct {
	A    []ARecord
	AAAA []AAAARecord
	MX   []Record
	NS   []Record
	SOA  []Record
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

type AAAARecord struct {
	Record
	IP           string
	Organization string
}
