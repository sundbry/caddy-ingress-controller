/*	This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/pkg/cmd/controller/nginx.go
	Licensed under the Apache License.  http://www.apache.org/licenses/LICENSE-2.0
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/pflag"

	"git.nwaonline.com/kubernetes/caddy-ingress/pkg/config"
	cdy_template "git.nwaonline.com/kubernetes/caddy-ingress/pkg/template"
	"git.nwaonline.com/kubernetes/caddy-ingress/pkg/version"

	api "k8s.io/client-go/pkg/api/v1"

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
	cfgPath         = "/etc/Caddyfile"
	binary          = "/usr/bin/caddy"
	defIngressClass = "caddy"
	ingressCfgJson  = []byte{}
)

func newCaddyController() ingress.Controller {
	cdy := os.Getenv("CADDY_BINARY")

	if cdy == "" {
		cdy = binary
	}

	h, err := dns.GetSystemNameServers()
	if err != nil {
		log.Printf("unexpected error reading system nameservers: %v", err)
	}

	c := &CaddyController{
		binary:    cdy,
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

	go c.Start()

	return ingress.Controller(c)
}

type CaddyController struct {
	t *cdy_template.Template

	configmap *api.ConfigMap

	storeLister ingress.StoreLister

	binary   string
	resolver []net.IP

	watchClass string
	namespace  string

	cmd *exec.Cmd
}

func (c *CaddyController) Start() {
	log.Print("starting Caddy process...")

	done := make(chan error, 1)
	c.cmd = exec.Command(
		c.binary,
		"-conf", cfgPath,
		"-log", "stdout",
		"-ca", "https://acme-staging.api.letsencrypt.org/directory",
	)
	c.start(done)
	for {
		err := <-done
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			log.Fatalf(`
-------------------------------------------------------------------------------
Caddy process died (%v): %v
-------------------------------------------------------------------------------
`, waitStatus.ExitStatus(), err)
		}
	}
}

func (c *CaddyController) start(done chan error) {
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
	if err := c.cmd.Start(); err != nil {
		log.Fatalf("caddy error: %v", err)
		done <- err
		return
	}

	go func() {
		done <- c.cmd.Wait()
	}()
}

// Reload checks if the running configuration file is different
// from the specified and reload Caddy if required
func (c CaddyController) Reload(data []byte) ([]byte, bool, error) {
	if !c.isReloadRequired(data) {
		return []byte("Reload not required"), false, nil
	}

	err := ioutil.WriteFile(cfgPath, data, 0644)
	if err != nil {
		return nil, false, err
	}

	log.Printf(`
		-----------------------------------------------
		Caddyfile
		-----------------------------------------------
		%v
		`, string(data))
	log.Printf(`
		-----------------------------------------------
		Configuration Struct
		-----------------------------------------------
		%v
		`, string(ingressCfgJson))

	// signal the Caddy process to reload the configuration
	err = c.cmd.Process.Signal(syscall.SIGUSR1)
	return []byte{}, true, err
}

func (c CaddyController) isReloadRequired(data []byte) bool {
	src, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return false
	}

	if !bytes.Equal(src, data) {
		return true
	}

	return false
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

// OnUpdate is called by syncQueue in  https://github.com/aledbf/ingress-controller/blob/master/pkg/ingress/controller/controller.go#L82
// periodically to keep the configuration in sync.
//
// convert configmap to custom configuration object (different in each implementation)
// write the custom template (complexity depends on the implementation)
// write the configuration file
// returning nil implies the backend will be reloaded
// if an error is returned the update should be re-queued
func (c *CaddyController) OnUpdate(ingressCfg ingress.Configuration) ([]byte, error) {
	cfg := cdy_template.ReadConfig(c.configmap.Data)
	cfg.Resolver = c.resolver

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
		return nil, err
	}

	// TODO: Validate config template results

	ingressCfgJson, _ = json.Marshal(ingressCfg)

	return content, nil
}

// == HealthCheck ==

// Name returns the HealthCheck name
func (c CaddyController) Name() string {
	return "Caddy Controller"
}

// Check performs a healthcheck
func (c CaddyController) Check(_ *http.Request) error {
	res, err := http.Get(fmt.Sprintf("http://%v%v", cdyHealthHost, cdyHealthPath))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("ingress controller is not healthy")
	}
	return nil
}
