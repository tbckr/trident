package writer

import (
	"io"

	"github.com/tbckr/trident/pkg/report"
)

type Writer interface {
	WriteDomainReport(out io.Writer, report report.DomainReport) error
}
