/*
This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/pkg/cmd/controller/main.go
Licensed under the Apache License 2.0 https://www.apache.org/licenses/LICENSE-2.0
*/

package main

import (
	"log"

	"k8s.io/ingress/core/pkg/ingress/controller"
)

func main() {
	dc := newCaddyController()
	ic := controller.NewIngressController(dc)
	defer func() {
		log.Printf("Shutting down ingress controller...")
		ic.Stop()
	}()
	ic.Start()
}
