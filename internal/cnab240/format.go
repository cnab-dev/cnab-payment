package cnab240

import (
	"strconv"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// FmtDate formats t as DDMMYYYY in t's own location. Calendar dates are
// therefore written exactly as given; a layout that writes an instant must
// convert it to its local time first.
func FmtDate(t time.Time) string { return t.Format("02012006") }

// FmtTime formats t as HHMMSS in t's own location.
func FmtTime(t time.Time) string { return t.Format("150405") }

// FmtAmount formats a non-negative amount as zero-padded cents, width
// digits wide.
func FmtAmount(a core.Amount, width int) string {
	return PadL(strconv.FormatInt(int64(a), 10), width, '0')
}

// FmtInt formats a non-negative n zero-padded to width digits.
func FmtInt(n, width int) string {
	return PadL(strconv.Itoa(n), width, '0')
}
