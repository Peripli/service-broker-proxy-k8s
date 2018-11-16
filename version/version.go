package version

import "github.com/Peripli/service-manager/pkg/log"

// GitCommit is the commit id, injected by the build
var GitCommit string

// Version is the SM version, injected by the build
var Version string

// Log writes the Service Manager version info in the log
func Log() {
	log.D().Infof("Service Broker Proxy Version: %s (%s)", Version, GitCommit)
}
