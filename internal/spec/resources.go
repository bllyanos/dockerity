package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MinMemoryBytes is Docker's smallest allowed memory limit.
const MinMemoryBytes = 6 * 1024 * 1024

// MaxCPUs is a sanity bound for cpus.
const MaxCPUs = 1024

var memoryPattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)([kmg]?)b?$`)

// ParseMemory converts a compose-style memory value ("512m", "1g", "1048576")
// to bytes.
func ParseMemory(s string) (int64, error) {
	m := memoryPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if m == nil {
		return 0, fmt.Errorf("memory %q must be a number with an optional k, m or g suffix", s)
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	mult := map[string]float64{"": 1, "k": 1 << 10, "m": 1 << 20, "g": 1 << 30}[m[2]]
	bytes := int64(n * mult)
	if bytes < MinMemoryBytes {
		return 0, fmt.Errorf("memory %q is below Docker's minimum of 6m", s)
	}
	return bytes, nil
}

// ParseCPUs converts a cpus value ("0.5", "2") to a positive float.
func ParseCPUs(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v <= 0 || v > MaxCPUs {
		return 0, fmt.Errorf("cpus %q must be a number greater than 0 and at most %d", s, MaxCPUs)
	}
	return v, nil
}
