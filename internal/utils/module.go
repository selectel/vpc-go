package utils

import "runtime/debug"

const modulePath = "github.com/selectel/vpc-go"

var ModuleVersion = getModuleVersion()

func getModuleVersion() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, dependency := range buildInfo.Deps {
			if dependency.Path == modulePath {
				return dependency.Version
			}
		}
	}

	return "v0.0.0"
}
