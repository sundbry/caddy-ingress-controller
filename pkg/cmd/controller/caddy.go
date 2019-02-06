/*	This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/pkg/cmd/controller/nginx.go
	Licensed under the Apache License.  http://www.apache.org/licenses/LICENSE-2.0
*/

package main

import (
	"github.com/mholt/caddy"
	_ "github.com/mholt/caddy/caddyhttp"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/spf13/pflag"

	"k8s.io/ingress/controllers/caddy/pkg/config"
	cdy_template "k8s.io/ingress/controllers/caddy/pkg/template"
	"k8s.io/ingress/controllers/caddy/pkg/version"

	api "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"

	"k8s.io/ingress/core/pkg/ingress"
	"k8s.io/ingress/core/pkg/ingress/defaults"
	"k8s.io/ingress/core/pkg/net/dns"
)

const (
	cdyHealthHost = ""
	cdyHealthPort = 12015
	cdyHealthPath = "/healthz"
)

var (
	tmplPath        = "/etc/Caddyfile.tmpl"
	defIngressClass = "caddy"
)

func newCaddyController() ingress.Controller {
	h, err := dns.GetSystemNameServers()
	if err != nil {
		log.Printf("unexpected error reading system nameservers: %v", err)
	}

	c := &CaddyController{
		configmap: &api.ConfigMap{},
		resolver:  h,
	}

	var onChange func()
	onChange = func() {
		template, err := cdy_template.NewTemplate(tmplPath, onChange)
		if err != nil {
			log.Printf(`
-------------------------------------------------------------------------------
Error loading new template: %v
-------------------------------------------------------------------------------
`, err)
			return
		}

		c.t.Close()
		c.t = template
		log.Print("new Caddy template loaded")
	}

	cdyTpl, err := cdy_template.NewTemplate(tmplPath, onChange)
	if err != nil {
		log.Fatalf("invalid Caddy template: %v", err)
	}

	c.t = cdyTpl
	c.caddyFileContent = nil

	go c.Start()

	return ingress.Controller(c)
}

type CaddyController struct {
	t *cdy_template.Template

	configmap *api.ConfigMap

	storeLister ingress.StoreLister

	resolver []net.IP

	watchClass string
	namespace  string
	instance *caddy.Instance
	caddyFileContent []byte
}

func (c *CaddyController) Start() {
	log.Print("Starting Caddy controller...")

	caddy.AppName = "caddy-ingress-controller"
	caddy.AppVersion = version.RELEASE
	caddy.SetCAAgreement(true)
	caddy.RegisterCaddyfileLoader("caddy-ingress-controller", c)

	caddyfile, err := caddy.LoadCaddyfile("http")
	if err != nil {
		log.Fatal(err)
	}

	c.instance, err = caddy.Start(caddyfile)
	if err != nil {
		log.Fatal(err)
	}
}

// https://godoc.org/github.com/mholt/caddy#Loader
func (c *CaddyController) Load(serverType string) (caddy.Input, error) {
	if c.caddyFileContent == nil {
		return nil, nil
	} else {
		return caddy.CaddyfileInput{
			Contents: c.caddyFileContent,
			Filepath: "configmap",
			ServerTypeName: serverType,
		}, nil
	}
}

func (c CaddyController) BackendDefaults() defaults.Backend {
	if c.configmap == nil {
		d := config.NewDefault()
		return d.Backend
	}

	return cdy_template.ReadConfig(c.configmap.Data).Backend
}

// Info returns build information
func (c CaddyController) Info() *ingress.BackendInfo {
	return &ingress.BackendInfo{
		Name:       "caddy",
		Release:    version.RELEASE,
		Build:      version.COMMIT,
		Repository: version.REPO,
	}
}

func (c *CaddyController) ConfigureFlags(flags *pflag.FlagSet) {
}

func (c CaddyController) OverrideFlags(flags *pflag.FlagSet) {
	ic, _ := flags.GetString("ingress-class")

	if ic == "" {
		ic = defIngressClass
	}

	if ic != defIngressClass {
		log.Printf("only Ingress with class %v will be processed by this controller", ic)
	}

	flags.Set("ingress-class", ic)
}

func (c CaddyController) DefaultIngressClass() string {
	return defIngressClass
}

func (c CaddyController) SetConfig(cfgMap *api.ConfigMap) {
	c.configmap = cfgMap

	if cfgMap == nil {
		return
	}
}

// SetListers sets the configured store listers in the generic ingress controller
func (c CaddyController) SetListers(lister ingress.StoreLister) {
	c.storeLister = lister
}

func (c *CaddyController) UpdateIngressStatus(*extensions.Ingress) []api.LoadBalancerIngress {
	return nil
}

// OnUpdate is called by syncQueue in  https://github.com/aledbf/ingress-controller/blob/master/pkg/ingress/controller/controller.go#L82
// periodically to keep the configuration in sync.
//
// convert configmap to custom configuration object (different in each implementation)
// write the custom template (complexity depends on the implementation)
// write the configuration file
// returning nil implies the backend will be reloaded
// if an error is returned the update should be re-queued
func (c *CaddyController) OnUpdate(ingressCfg ingress.Configuration) error {
	cfg := cdy_template.ReadConfig(c.configmap.Data)
	cfg.Resolver = c.resolver

	var ingressCfgJson []byte
	ingressCfgJson, _ = json.Marshal(ingressCfg)
	log.Printf(
`
-----------------------------------------------
Loading Configuration Struct
-----------------------------------------------
%v
`, string(ingressCfgJson))

	content, err := c.t.Write(config.TemplateConfig{
		Backends:    ingressCfg.Backends,
		Servers:     ingressCfg.Servers,
		TCPBackends: ingressCfg.TCPEndpoints,
		UDPBackends: ingressCfg.UDPEndpoints,
		HealthzHost: cdyHealthHost,
		HealthzPort: cdyHealthPort,
		HealthzPath: cdyHealthPath,
		Cfg:         cfg,
	})
	if err != nil {
		return err
	}

	c.caddyFileContent = content
	caddyfile, err := caddy.LoadCaddyfile("http")
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Printf(
`
-----------------------------------------------
Loading Caddyfile
-----------------------------------------------
%v
`, string(caddyfile.Body()))

	c.instance, err = c.instance.Restart(caddyfile)
	if err != nil {
	log.Fatal(err)
	return err
	}

	return nil
}

// == HealthCheck ==

// Name returns the HealthCheck name
func (c CaddyController) Name() string {
	return "Caddy Controller"
}

// Check performs a healthcheck
func (c CaddyController) Check(_ *http.Request) error {
	log.Printf("Performing Caddy Controller Health Check")
	res, err := http.Get(fmt.Sprintf("http://%v%v:%v", cdyHealthHost, cdyHealthPath, cdyHealthPort))
	if err != nil {
		log.Printf("error with health check: %v", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("ingress controller is not healthy")
	}
	return nil
}
