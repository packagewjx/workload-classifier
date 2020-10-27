package main

import (
	"github.com/packagewjx/workload-classifier/pkg/schedulerplugin/featureaware"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	"k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

func main() {
	cmd := app.NewSchedulerCommand(func(registry runtime.Registry) error {
		_ = registry.Register(featureaware.PluginName, featureaware.New)
		return nil
	})

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
