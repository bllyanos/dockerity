package domain

// AllServices is the override key that applies to every service of an environment.
const AllServices = "*"

// ResourceOverride replaces a service's memory or CPU limit in one
// environment. Empty fields keep the spec's value.
type ResourceOverride struct {
	Memory string `json:"memory,omitempty"`
	CPUs   string `json:"cpus,omitempty"`
}

// ResourceOverrides maps a service name (or AllServices) to its override.
type ResourceOverrides map[string]ResourceOverride

// For returns the override of service: the service's own fields win over
// the environment-wide ones.
func (o ResourceOverrides) For(service string) ResourceOverride {
	out := o[AllServices]
	if s, ok := o[service]; ok {
		if s.Memory != "" {
			out.Memory = s.Memory
		}
		if s.CPUs != "" {
			out.CPUs = s.CPUs
		}
	}
	return out
}
