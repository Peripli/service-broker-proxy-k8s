package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-broker-proxy/pkg/middleware"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	"github.com/pmorie/osb-broker-lib/pkg/rest"
	osbserver "github.com/pmorie/osb-broker-lib/pkg/server"
	prom "github.com/prometheus/client_golang/prometheus"
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
	*osbserver.Server

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

	server, err := osbServer(osbConfig)
	if err != nil {
		return nil, err
	}

	server.Router.Use(middleware.LogRequest())

	return &Server{
		Server: server,
		Config: config,
	}, nil
}

// Use provides a way to plugin middleware in the Server
func (s Server) Use(middleware func(handler http.Handler) http.Handler) {
	s.Server.Router.Use(middleware)
}

// Run is the entrypoint of the Server. Run boots the application.
func (s Server) Run(group *sync.WaitGroup) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer waitWithTimeout(group, s.Config.Timeout)
	defer cancel()

	handleInterrupts(ctx, cancel)

	addr := ":" + strconv.Itoa(s.Config.Port)

	logrus.Info("Running Server...")
	if s.Config.TLSKey != "" && s.Config.TLSCert != "" {
		err = s.Server.RunTLS(ctx, addr, s.Config.TLSCert, s.Config.TLSKey)
	} else {
		err = s.Server.Run(ctx, addr)
	}

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		logrus.WithError(errors.WithStack(err)).Errorln("Error occurred while sbproxy was running")
	}
}

// handleInterrupts hannles OS interrupt signals by canceling the context
func handleInterrupts(ctx context.Context, cancel context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancel()
		case <-ctx.Done():
			return
		}
	}()
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

func osbServer(config *osb.ClientConfig) (*osbserver.Server, error) {
	businessLogic, err := osb.NewBusinessLogic(config)
	if err != nil {
		return nil, err
	}

	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := rest.NewAPISurface(businessLogic, osbMetrics)
	if err != nil {
		return nil, errors.Wrap(err, "error creating OSB API surface")
	}

	osbServer := osbserver.New(api, reg)
	router := mux.NewRouter()

	err = registerRoutes(Path, osbServer.Router, router)
	if err != nil {
		return nil, err
	}

	osbServer.Router = router
	return osbServer, nil
}

func registerRoutes(prefix string, fromRouter *mux.Router, toRouter *mux.Router) error {
	subRouter := toRouter.PathPrefix(prefix).Subrouter()
	return fromRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {

		path, err := route.GetPathTemplate()
		if err != nil {
			return errors.Wrap(err, "error getting path template")
		}

		methods, err := route.GetMethods()
		if err != nil {
			return errors.Wrap(err, "error getting route methods")
		}
		logrus.Info("Registering route with methods: ", methods, " and path: ", prefix+path)
		subRouter.Handle(path, route.GetHandler()).Methods(methods...)
		return nil
	})
}
