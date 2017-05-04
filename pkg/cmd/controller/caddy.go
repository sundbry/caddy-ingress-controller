/*	This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/pkg/cmd/controller/nginx.go
	Licensed under the Apache License.  http://www.apache.org/licenses/LICENSE-2.0
*/

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	cdy_template "git.nwaonline.com/kubernetes/caddy-ingress/pkg/template"

	api "k8s.io/client-go/pkg/api/v1"

	"k8s.io/ingress/controllers/nginx/pkg/config"
	"k8s.io/ingress/core/pkg/ingress"
	"k8s.io/ingress/core/pkg/ingress/defaults"
	"k8s.io/ingress/core/pkg/net/dns"
	"k8s.io/ingress/core/pkg/net/ssl"
)

type statusModule string

const (
	cdyHealthHost = "localhost"
	cdyHealthPort = 12015
	cdyHealthPath = "/healthz"

	defaultStatusModule statusModule = "default"

	errNoChild = "wait: no child processes"
)

var (
	tmplPath        = "/etc/caddy/template/caddy.tmpl"
	cfgPath         = "/etc/caddy/Caddyfile"
	binary          = "/usr/sbin/caddy"
	defIngressClass = "caddy"
)

func newCaddyController() ingress.Controller {
	cdy := os.Getenv("CADDY_BINARY")

	if cdy == "" {
		cdy = binary
	}

	h, err := dns.GetSystemNameServers()
	if err != nil {
		glog.Warningf("unexpected error reading system nameservers: %v", err)
	}

	c := &CaddyController{
		binary:    cdy,
		configmap: &api.ConfigMap{},
		resolver:  h,
		proxy:     &proxy{},
	}

	listener, err := net.Listen("tcp", ":443")
	if err != nil {
		glog.Fatalf("%v", err)
	}

	// start goroutine that accepts tcp connections in port 443
	go func() {
		for {
			var conn net.Conn
			var err error

			conn, err = listener.Accept()

			if err != nil {
				glog.Warningf("unexpected error accepting tcp connection: %v", err)
				continue
			}

			glog.V(3).Infof("remote address %s to local %s", conn.RemoteAddr(), conn.LocalAddr())
			go c.proxy.Handle(conn)
		}
	}()

	var onChange func()
	onChange = func() {
		template, err := cdy_template.NewTemplate(tmplPath, onChange)
		if err != nil {
			glog.Errorf(`
-------------------------------------------------------------------------------
Error loading new template: %v
-------------------------------------------------------------------------------
`, err)
			return
		}

		c.t.Close()
		c.t = template
		glog.Info("new Caddy template loaded")
	}

	cdyTpl, err := cdy_template.NewTemplate(tmplPath, onChange)
	if err != nil {
		glog.Fatalf("invalid Caddy template: %v", err)
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

	cmdArgs []string

	watchClass string
	namespace  string

	statusModule statusModule

	proxy *proxy

	cmd *exec.Cmd
}

func (c *CaddyController) Start() {
	glog.Info("starting Caddy process...")

	done := make(chan error, 1)
	c.cmd = exec.Command(c.binary, "-conf", cfgPath)
	c.start(c.cmd, done)
	for {
		err := <-done
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			glog.Warningf(`
-------------------------------------------------------------------------------
Caddy process died (%v): %v
-------------------------------------------------------------------------------
`, waitStatus.ExitStatus(), err)
			c.cmd.Process.Release()
			c.cmd = exec.Command(c.binary, "-conf", cfgPath)
			c.start(done)
		}
	}
}

func (c *CaddyController) start(done chan error) {
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
	if err := c.cmd.Start(); err != nil {
		glog.Fatalf("caddy error: %v", err)
		done <- err
		return
	}

	c.cmdArgs = c.cmd.Args

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

	// signal the Caddy process to reload the configuration
	err = c.cmd.Process.Signal(syscall.SIGUSR1)
	return []byte{}, true, err
}

func (c CaddyController) isReloadRequired(data []byte) bool {
	in, err := os.Open(cfgPath)
	if err != nil {
		return false
	}

	src, err := ioutil.ReadAll(in)
	in.Close()
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
		Release:    "0.0.1",
		Build:      "git-00000000",
		Repository: "git://github.com/zikes/caddy-ingress-controller",
	}
}

func (c CaddyController) OverrideFlags(flags *pflag.FlagSet) {
	ic, _ := flags.GetString("ingress-class")
	wc, _ := flags.GetString("watch-namespace")

	if ic == "" {
		ic = defIngressClass
	}

	if ic != defIngressClass {
		glog.Warningf("only Ingress with class %v will be processed by this controller", ic)
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
	var longestName int
	for _, srv := range ingressCfg.Servers {
		if longestName < len(srv.Hostname) {
			longestName = len(srv.Hostname)
		}
	}

	cfg = cdy_template.ReadConfig(c.configmap.Data)
	cfg.Resolver = c.resolver

	c.setupMonitor(defaultStatusModule)

	setHeaders := map[string]string{}
	if cfg.ProxySetHeaders != "" {
		cfgMap, exists, err := c.storeLister.ConfigMap.GetByKey(cfg.ProxySetHeaders)
		if err != nil {
			glog.Warningf("unexpected error reading configmap %v: %v", cfg.ProxySetHeaders, err)
		}
		if exists {
			setHeaders = cfgMap.(*api.ConfigMap).Data
		}
	}

	sslDHParam := ""
	if cfg.SSLDHParam != "" {
		secretName := cfg.SSLDHParam
		s, exists, err := c.storeLister.Secret.GetByKey(secretName)
		if err != nil {
			glog.Warningf("unexpected error reading secret %v: %v", secretName, err)
		}

		if exists {
			secret := s.(*api.Secret)
			nsSecName := strings.Replace(secretName, "/", "-", -1)

			dh, ok := secret.Data["dhparam.pem"]
			if ok {
				pemFileName, err := ssl.AddOrUpdateDHParam(nsSecName, dh)
				if err != nil {
					glog.Warningf("unexpected error adding or updating dhparam %v: %v", nsSecName, err)
				} else {
					sslDHParam = pemFileName
				}
			}
		}
	}

	cfg.SSLDHParam = sslDHParam

	content, err := c.t.Write(config.TemplateConfig{
		ProxySetHeaders:     setHeaders,
		Backends:            ingressCfg.Backends,
		PassthroughBackends: ingressCfg.PassthroughBackends,
		Servers:             ingressCfg.Servers,
		TCPBackends:         ingressCfg.TCPEndpoints,
		UDPBackends:         ingressCfg.UDPEndpoints,
		HealthzHost:         cdyHealthHost,
		HealthzPort:         cdyHealthPort,
		HealthzURI:          cdyHealthPath,
		CustomErrors:        len(cfg.CustomHTTPErrors) > 0,
		Cfg:                 cfg,
	})
	if err != nil {
		return nil, err
	}

	// TODO: Validate config template results

	servers := []*server{}
	for _, pb := range ingressCfg.PassthroughBackends {
		svc := pb.Service
		if svc == nil {
			glog.Warningf("missing service for PassthroughBackends %v", pb.Backend)
			continue
		}
		port, err := strconv.Atoi(pb.Port.String())
		if err != nil {
			for _, sp := range svc.Spec.Ports {
				if sp.Name == pb.Port.String() {
					port = int(sp.Port)
					break
				}
			}
		} else {
			for _, sp := range svc.Spec.Ports {
				if sp.Port == int32(port) {
					port = int(sp.Port)
					break
				}
			}
		}

		servers = append(servers, &server{
			Hostname: pb.Hostname,
			IP:       svc.Spec.ClusterIP,
			Port:     port,
		})
	}
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
