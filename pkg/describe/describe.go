package describe

import (
	"context"

	"github.com/tbckr/trident/pkg/report"
)

type DomainDescriber interface {
	DescribeDomain(ctx context.Context, domain string) (report.DomainDescriptionReport, error)
}
