/*	This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/pkg/template/template.go
	Licensed under the Apache License.  http://www.apache.org/licenses/LICENSE-2.0
*/

package template

import (
	"bytes"
	"fmt"
	"log"
	text_template "text/template"

	"git.nwaonline.com/kubernetes/caddy-ingress/pkg/config"

	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/ingress/core/pkg/ingress"
	"k8s.io/ingress/core/pkg/watch"
)

const (
	slash         = "/"
	defBufferSize = 65535
	errNoChild    = "wait: no child processes"
)

// Template
type Template struct {
	tmpl    *text_template.Template
	fw      watch.FileWatcher
	s       int
	tmplBuf *bytes.Buffer
}

// NewTemplate returns a new Template instance or an
// error if the specified template contains errors
func NewTemplate(file string, onChange func()) (*Template, error) {
	tmpl := text_template.Must(text_template.New("Caddyfile.tmpl").Funcs(funcMap).ParseFiles(file))
	fw, err := watch.NewFileWatcher(file, onChange)
	if err != nil {
		return nil, err
	}

	return &Template{
		tmpl:    tmpl,
		fw:      fw,
		s:       defBufferSize,
		tmplBuf: bytes.NewBuffer(make([]byte, 0, defBufferSize)),
	}, nil
}

// Close removes the file watcher
func (t *Template) Close() {
	t.fw.Close()
}

// Write populates a buffer using the template with the Caddy configuration
// and the servers and upstreams created by the Ingress rules
func (t *Template) Write(conf config.TemplateConfig) ([]byte, error) {
	defer t.tmplBuf.Reset()

	defer func() {
		if t.s < t.tmplBuf.Cap() {
			log.Printf("adjusting template buffer size from %v to %v", t.s, t.tmplBuf.Cap())
			t.s = t.tmplBuf.Cap()
			t.tmplBuf = bytes.NewBuffer(make([]byte, 0, t.tmplBuf.Cap()))
		}
	}()

	err := t.tmpl.Execute(t.tmplBuf, conf)
	if err != nil && err.Error() != errNoChild {
		return nil, err
	}

	return t.tmplBuf.Bytes(), nil
}

var (
	funcMap = text_template.FuncMap{
		"empty": func(input interface{}) bool {
			check, ok := input.(string)
			if ok {
				return len(check) == 0
			}
			return true
		},
		"buildLocation": buildLocation,
		"cleanHostname": cleanHostname,
		"fromSpec":      fromSpec,
	}
)

func fromSpec(input interface{}) string {
	spec, ok := input.(api.ServiceSpec)
	if !ok {
		return ""
	}
	switch spec.Type {
	case "ClusterIP":
		// The ClusterIP ServiceType means the service has been assigned a
		// dedicated unique IP address within the server
		return spec.ClusterIP
	case "NodePort":
		// The NodePort ServiceType means the service is given a dedicated
		// unique port on every node in the cluster, which is provided by the
		// spec's ClusterIP
		return spec.ClusterIP
	case "LoadBalancer":
		// The LoadBalancer ServiceType creates a dedicated Load Balancer IP
		// on supported cloud providers
		return spec.LoadBalancerIP
	case "ExternalName":
		// ExternalName is the external reference that kubedns or equiv
		// will return as a CNAME record for this service
		return spec.ExternalName
	}
	return ""
}

// buildLocation produces the location string, if the ingress has redirects
// (specified through the ingress.kubernetes.io/rewrite-target annotation)
func buildLocation(input interface{}) string {
	location, ok := input.(*ingress.Location)
	if !ok {
		return slash
	}

	path := location.Path
	if len(location.Redirect.Target) > 0 && location.Redirect.Target != path {
		if path == slash {
			return fmt.Sprintf("%s", path)
		}
	}
	return path
}

// cleanHostname will replace the "_" hostname with ""
func cleanHostname(input interface{}) string {
	if hostname, ok := input.(string); ok {
		if hostname == "_" {
			return ""
		}
		return hostname
	}
	return ""
}
