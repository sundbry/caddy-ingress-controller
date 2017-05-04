// This package will contain general Caddy configuration.

package config

import (
	"k8s.io/ingress/core/pkg/ingress"
	"k8s.io/ingress/core/pkg/ingress/defaults"
)

const (
	// bodySize = "1m"

	// `log stdout`
	// Send all logs to stdout
	logLocation = "stdout"
)

var (
	SSLDirectory = "/root/.caddy"
)

// Configuration represents the content of /etc/Caddyfile
type Configuration struct {
	defaults.Backend `json:",squash"`
	LogLocation      string
}

func NewDefault() Configuration {
	cfg := Configuration{
		LogLocation: logLocation,
	}

	return cfg
}

// TemplateConfig contains the configuration to render the file /etc/Caddyfile
type TemplateConfig struct {
	Backends    []*ingress.Backend
	Servers     []*ingress.Server
	TCPBackends []ingress.L4Service
	UDPBackends []ingress.L4Service
	HealthzHost string
	HealthzPort int
	HealthzURI  string
	Cfg         Configuration
}
