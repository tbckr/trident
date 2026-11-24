package doh

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/imroc/req/v3"
	"golang.org/x/net/dns/dnsmessage"

	"github.com/tbckr/trident/internal/apperr"
)

// TXTResolver looks up TXT records over DoH and returns them the way
// [net.Resolver.LookupTXT] does: one string per record, raw bytes.
//
// It does not go through [MakeDoHRequest], whose parser (miekg/dns) renders
// TXT data in presentation format: '"' and '\' get a backslash and bytes
// outside 0x20-0x7E become \DDD. Lookups whose answers are parsed by the
// caller (Team Cymru) need the bytes untouched.
type TXTResolver struct {
	client *req.Client
}

// NewTXTResolver returns a TXTResolver that sends its queries through client.
func NewTXTResolver(client *req.Client) *TXTResolver {
	return &TXTResolver{client: client}
}

// LookupTXT returns the TXT records of name, one string per record in answer
// order, with the character-strings of a record concatenated. Non-TXT answers
// (such as a CNAME chain) are skipped. NXDOMAIN and an answer without TXT
// records both return (nil, nil); any other failure rcode, a transport error
// or an unparsable response is an [apperr.ErrRequestFailed].
func (r *TXTResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	data, err := exchange(ctx, r.client, name, uint16(dnsmessage.TypeTXT))
	if err != nil {
		return nil, err
	}

	var p dnsmessage.Parser
	hdr, err := p.Start(data)
	if err != nil {
		return nil, errParse(name, err)
	}
	switch hdr.RCode {
	case dnsmessage.RCodeSuccess:
	case dnsmessage.RCodeNameError:
		return nil, nil
	default:
		return nil, fmt.Errorf("%w: quad9 returned rcode %s for %q type TXT", apperr.ErrRequestFailed, hdr.RCode, name)
	}
	if err := p.SkipAllQuestions(); err != nil {
		return nil, errParse(name, err)
	}

	var records []string
	for {
		ah, err := p.AnswerHeader()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			return records, nil
		}
		if err != nil {
			return nil, errParse(name, err)
		}
		if ah.Type != dnsmessage.TypeTXT {
			if err := p.SkipAnswer(); err != nil {
				return nil, errParse(name, err)
			}
			continue
		}
		txt, err := p.TXTResource()
		if err != nil {
			return nil, errParse(name, err)
		}
		records = append(records, strings.Join(txt.TXT, ""))
	}
}

// errParse reports a response that could not be parsed as DNS wire format.
func errParse(name string, err error) error {
	return fmt.Errorf("%w: failed to parse DNS response for %q: %v", apperr.ErrRequestFailed, name, err)
}
