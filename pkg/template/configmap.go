/*	This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/pkg/template/configmap.go
	Licensed under the Apache License.  http://www.apache.org/licenses/LICENSE-2.0
*/

package template

import (
	"github.com/golang/glog"
	"github.com/mitchellh/mapstructure"

	"git.nwaonline.com/kubernetes/caddy-ingress/pkg/config"
)

// ReadConfig obtains the configuration defined by the user merged with the defaults.
func ReadConfig(src map[string]string) config.Configuration {
	conf := map[string]string{}
	if src != nil {
		// copy the configmap data because the content is altered
		for k, v := range src {
			conf[k] = v
		}
	}

	to := config.NewDefault()

	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		WeaklyTypedInput: true,
		Result:           &to,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		glog.Warningf("unexpected error merging defaults: %v", err)
	}

	err = decoder.Decode(conf)
	if err != nil {
		glog.Warningf("unexpected error merging defaults: %v", err)
	}

	return to
}
