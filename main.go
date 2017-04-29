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
