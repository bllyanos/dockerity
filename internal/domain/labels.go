// Package domain holds Dockerity's entities and pure rules. It must not depend
// on Docker, SQLite or HTTP packages.
package domain

// Label keys put on every Docker object Dockerity manages (docs/design.md section 5).
const (
	LabelApp        = "dockerity.app"
	LabelEnv        = "dockerity.env"
	LabelService    = "dockerity.service"
	LabelDeployment = "dockerity.deployment"
	// LabelSystem marks Dockerity itself and Traefik. Objects carrying it with
	// value "true" are excluded from every deploy, cleanup and GC operation.
	LabelSystem = "dockerity.system"
)

// IsSystem reports whether labels mark a system object that must never be touched.
func IsSystem(labels map[string]string) bool {
	return labels[LabelSystem] == "true"
}
