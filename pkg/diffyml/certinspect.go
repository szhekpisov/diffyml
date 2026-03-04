// certinspect.go - X.509 certificate inspection helpers.
package diffyml

import (
	"github.com/szhekpisov/diffyml/pkg/diffyml/internal/format"
)

func IsPEMCertificate(s string) bool   { return format.IsPEMCertificate(s) }
func FormatCertificate(s string) string { return format.FormatCertificate(s) }
