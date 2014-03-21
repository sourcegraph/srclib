package gog

import (
	"strings"
)

// isCgoUnsafePointerConversionError checks if this is an error like the kind
// triggered by the grapher testdata file github.com/charles/c_go/ptr.go.
func isCgoUnsafePointerConversionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "cannot convert ") && (strings.HasSuffix(s, "(variable with invalid type) to Pointer") || strings.HasSuffix(s, "(variable with invalid type) to unsafe.Pointer"))
}

// isCgoConversionToInvalidTypeError checks if this is an error like the kind
// triggered by the grapher testdata file github.com/charles/c_go/num.go.
func isCgoConversionToInvalidTypeError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "cannot convert ") && strings.HasSuffix(s, " to invalid type")
}
