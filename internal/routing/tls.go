package routing

import "go.yaml.in/yaml/v3"

// tlsFile is the name of the shared TLS configuration in the dynamic directory.
const tlsFile = "dkt-tls.yml"

type tlsConfig struct {
	TLS tlsSection `yaml:"tls"`
}

type tlsSection struct {
	Stores map[string]tlsStore `yaml:"stores"`
}

type tlsStore struct {
	DefaultCertificate certFiles `yaml:"defaultCertificate"`
}

type certFiles struct {
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

// CertificateConfig makes the given wildcard certificate Traefik's default.
func CertificateConfig(certFile, keyFile string) ([]byte, error) {
	return yaml.Marshal(tlsConfig{TLS: tlsSection{Stores: map[string]tlsStore{
		"default": {DefaultCertificate: certFiles{CertFile: certFile, KeyFile: keyFile}},
	}}})
}
