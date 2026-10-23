package domain

import "fmt"

// Docker object names owned by Dockerity (docs/design.md section 5). All name
// parts are DNS labels, so the results are valid Docker names.

// ImageTag is the tag of a service image built from a commit.
func ImageTag(app, service, commitSHA string) string {
	return fmt.Sprintf("dkt/%s/%s:%s", app, service, commitSHA)
}

// NetworkName is the network shared by an environment's containers.
func NetworkName(app, env string) string {
	return fmt.Sprintf("dkt-%s-%s", app, env)
}

// ContainerName names one container of a deployment.
func ContainerName(app, env, service, deploymentShortID string) string {
	return fmt.Sprintf("dkt-%s-%s-%s-%s", app, env, service, deploymentShortID)
}

// VolumeName names a stateful service's persistent volume in an environment.
func VolumeName(app, env, service, volume string) string {
	return fmt.Sprintf("dkt-%s-%s-%s-%s", app, env, service, volume)
}

// LabelCommit records the commit an image was built from.
const LabelCommit = "dockerity.commit"

// ImageLabels are put on every image Dockerity builds.
func ImageLabels(app, service, commitSHA string) map[string]string {
	return map[string]string{LabelApp: app, LabelService: service, LabelCommit: commitSHA}
}
