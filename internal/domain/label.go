package domain

import "regexp"

var dnsLabelPattern = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,30}[a-z0-9])?$`)

// LabelRule describes IsDNSLabel for error messages.
const LabelRule = "must be 1 to 32 lowercase letters, digits or '-', starting with a letter and not ending with '-'"

// IsDNSLabel reports whether s is safe as a part of Docker object names and
// hostnames. App, environment, service and volume names all follow it.
func IsDNSLabel(s string) bool { return dnsLabelPattern.MatchString(s) }
