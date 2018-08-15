package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-broker-proxy/pkg/middleware"
	"github.com/Peripli/service-broker-proxy/pkg/osb"

	smOsb "github.com/Peripli/service-manager/api/osb"
	smWeb "github.com/Peripli/service-manager/pkg/web"
	smServer "github.com/Peripli/service-manager/server"

	"github.com/sirupsen/logrus"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"

	// Path for the Proxy OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

// Server type is the starting point of the proxy application. It glues the proxy REST API and the timed
// jobs for broker registrations
type Server struct {
	server *smServer.Server

	Config *Config
}

// New builds a new Server from the provided configuration using the provided platform client. The
// platform client is used by the Server to call to the platform during broker creation and deletion.
func New(config *Config, osbConfig *osb.ClientConfig) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if err := osbConfig.Validate(); err != nil {
		return nil, err
	}

	logging.Setup(config.LogLevel, config.LogFormat)

	osbAdapter, err := osb.NewOsbAdapter(osbConfig)
	if err != nil {
		return nil, err
	}

	api := &smWeb.API{
		Controllers: []smWeb.Controller{
			smOsb.NewController(osbAdapter),
		},
	}
	server := smServer.New(smServer.Settings{
		Port:            config.Port,
		RequestTimeout:  config.Timeout,
		ShutdownTimeout: config.Timeout,
	}, api)
	server.Router.Use(middleware.LogRequest())

	return &Server{
		server: server,
		Config: config,
	}, nil
}

func (s *Server) Use(middleware func(handler http.Handler) http.Handler) {
	s.server.Router.Use(middleware)
}

// Run is the entrypoint of the Server. Run boots the application.
func (s Server) Run(group *sync.WaitGroup) {
	// var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer waitWithTimeout(group, s.Config.Timeout)
	defer cancel()

	logrus.Info("Running Server...")
	s.server.Run(ctx)
}

// waitWithTimeout waits for a WaitGroup to finish for a certain duration and times out afterwards
// WaitGroup parameter should be pointer or else the copy won't get notified about .Done() calls
func waitWithTimeout(group *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		group.Wait()
	}()
	select {
	case <-c:
		logrus.Debug("Timeout WaitGroup ", group, " finished successfully")
	case <-time.After(timeout):
		logrus.Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
