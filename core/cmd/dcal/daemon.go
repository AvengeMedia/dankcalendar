package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"

	auth_handler "github.com/AvengeMedia/dankcalendar/core/api/auth"
	calendar_handler "github.com/AvengeMedia/dankcalendar/core/api/calendar"
	"github.com/AvengeMedia/dankcalendar/core/api/middleware"
	"github.com/AvengeMedia/dankcalendar/core/api/server"
	"github.com/AvengeMedia/dankcalendar/core/config"
	"github.com/AvengeMedia/dankcalendar/core/errdefs"
	"github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/AvengeMedia/dankcalendar/core/internal/colorscheme"
	"github.com/AvengeMedia/dankcalendar/core/internal/invitations"
	"github.com/AvengeMedia/dankcalendar/core/internal/ipc"
	dankkeyring "github.com/AvengeMedia/dankcalendar/core/internal/keyring"
	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/notify"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/caldav"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/evolution"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/google"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/ical"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/local"
	"github.com/AvengeMedia/dankcalendar/core/internal/providers/microsoft"
	"github.com/AvengeMedia/dankcalendar/core/internal/reminders"
	"github.com/AvengeMedia/dankcalendar/core/internal/rsvp"
	"github.com/AvengeMedia/dankcalendar/core/internal/sync"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the dcal daemon (IPC + HTTP, no UI)",
	RunE:  runDaemon,
}

type daemonServices struct {
	ipc         *ipc.Server
	httpSrv     *http.Server
	httpAddr    string
	syncEngine  *sync.Engine
	reminders   *reminders.Engine
	invitations *invitations.Engine
	notifier    *notify.Client
	repo        *repo.Repo
	registry    *calendar.Registry
	secrets     calendar.SecretStore
	broker      *auth_handler.CallbackBroker
	flows       *oauth.FlowRegistry
	ipcErrCh    <-chan error
	httpErrCh   <-chan error
}

func (s *daemonServices) socketPath() string {
	if s == nil || s.ipc == nil {
		return ""
	}
	return s.ipc.SocketPath()
}

func (s *daemonServices) close() {
	if s == nil {
		return
	}
	if s.syncEngine != nil {
		s.syncEngine.Stop()
	}
	if s.reminders != nil {
		s.reminders.Stop()
	}
	if s.invitations != nil {
		s.invitations.Stop()
	}
	if s.notifier != nil {
		s.notifier.Close()
	}
	if s.httpSrv != nil {
		shutdownHTTP(s.httpSrv)
	}
	if s.ipc != nil {
		s.ipc.Close()
	}
	if s.repo != nil {
		s.repo.Close()
	}
}

func bootDaemonServices(ctx context.Context) (*daemonServices, error) {
	cfg := config.New()

	dbPath := cfg.DatabasePath
	if dbPath == "" {
		path, err := paths.DatabasePath()
		if err != nil {
			return nil, err
		}
		dbPath = path
	}

	client, err := repo.OpenFile(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	r := repo.New(client)

	registry := calendar.NewRegistry()
	registry.Register(local.Factory{})
	registry.Register(google.Factory{})
	registry.Register(caldav.Factory{})
	registry.Register(microsoft.Factory{})
	registry.Register(ical.Factory{})
	registry.Register(evolution.Factory{})

	keyringStore := dankkeyring.Open()
	secrets := dankkeyring.NewSecretStore(keyringStore, repo.NewSecretStore(r))
	flows := oauth.NewFlowRegistry()
	broker := auth_handler.NewCallbackBroker()

	syncEngine := sync.NewEngine(r, registry, secrets, 5*time.Minute)
	bus := ipc.NewEventBus()
	syncEngine.SetNotifier(bus.Publish)
	syncEngine.WatchMutations(client)

	colorSchemeWatcher := colorscheme.New(bus.Publish)
	if err := colorSchemeWatcher.Start(); err != nil {
		log.Warnf("color-scheme portal watcher unavailable: %v", err)
	}

	notifier, err := notify.New()
	if err != nil {
		log.Warnf("desktop notifications unavailable: %v", err)
		notifier = nil
	}
	var sender reminders.Sender
	if notifier != nil {
		sender = notifier
	}
	remindersEngine := reminders.NewEngine(r, sender, time.Hour)
	remindersEngine.SetPublisher(bus.Publish)
	remindersEngine.WatchMutations(client)

	var invitationsSender invitations.Sender
	if notifier != nil {
		invitationsSender = notifier
	}
	invitationsEngine := invitations.NewEngine(r, rsvp.Stores{
		Repo:     r,
		Registry: registry,
		Secrets:  secrets,
	}, invitationsSender, time.Hour)
	invitationsEngine.SetPublisher(bus.Publish)
	invitationsEngine.WatchMutations(client)

	// A single notification dispatcher fans out to both engines; each ignores
	// notification ids it does not own.
	if notifier != nil {
		notifier.SetHandlers(
			func(id uint32, action string) {
				remindersEngine.HandleAction(id, action)
				invitationsEngine.HandleAction(id, action)
			},
			func(id uint32) {
				remindersEngine.HandleClosed(id)
				invitationsEngine.HandleClosed(id)
			},
		)
	}

	httpSrv, httpAddr, httpErrCh, err := startHTTP(ctx, cfg, r, registry, secrets, broker)
	if err != nil {
		r.Close()
		return nil, err
	}

	deps := ipc.Deps{
		Repo:        r,
		Registry:    registry,
		Secrets:     secrets,
		Broker:      broker,
		Flows:       flows,
		HTTPAddr:    httpAddr,
		Sync:        syncEngine,
		Reminders:   remindersEngine,
		Bus:         bus,
		Pending:     &ipc.PendingOpen{},
		Version:     Version,
		ColorScheme: colorSchemeWatcher,
	}
	ipcSrv, ipcErrCh, err := startIPC(ctx, deps)
	if err != nil {
		shutdownHTTP(httpSrv)
		r.Close()
		return nil, err
	}

	syncEngine.Start(ctx)
	remindersEngine.Start(ctx)
	invitationsEngine.Start(ctx)

	return &daemonServices{
		ipc:         ipcSrv,
		httpSrv:     httpSrv,
		httpAddr:    httpAddr,
		syncEngine:  syncEngine,
		reminders:   remindersEngine,
		invitations: invitationsEngine,
		notifier:    notifier,
		repo:        r,
		registry:    registry,
		secrets:     secrets,
		broker:      broker,
		flows:       flows,
		ipcErrCh:    ipcErrCh,
		httpErrCh:   httpErrCh,
	}, nil
}

func runDaemon(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := bootDaemonServices(ctx)
	if err != nil {
		return err
	}
	defer svc.close()

	log.Infof("dcal daemon ready (ipc=%s http=%s)", svc.socketPath(), svc.httpAddr)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		log.Infof("received %s, shutting down", sig)
	case err := <-svc.ipcErrCh:
		if err != nil {
			return fmt.Errorf("ipc server: %w", err)
		}
	case err := <-svc.httpErrCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}
	return nil
}

func startIPC(ctx context.Context, deps ipc.Deps) (*ipc.Server, <-chan error, error) {
	srv := ipc.NewServer(deps)
	if err := srv.Listen(); err != nil {
		return nil, nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()
	return srv, errCh, nil
}

func startHTTP(ctx context.Context, cfg *config.Config, r *repo.Repo, registry *calendar.Registry, secrets calendar.SecretStore, broker *auth_handler.CallbackBroker) (*http.Server, string, <-chan error, error) {
	if cfg.DisableHTTP {
		errCh := make(chan error, 1)
		return nil, "", errCh, nil
	}

	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	callbackHandler := auth_handler.NewCallbackHandler(broker)
	router.Get("/oauth/callback", callbackHandler)

	// Microsoft requires registering the bare http://localhost redirect
	// (path matched exactly, port ignored), so callbacks also land on "/".
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "" {
			callbackHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("dcal daemon"))
	})

	router.Group(func(rt chi.Router) {
		rt.Use(middleware.Logger)

		huma.NewError = errdefs.HumaErrorFunc

		humaCfg := server.NewHumaConfig("Dank Calendar API", Version)
		humaCfg.DocsPath = ""
		api := humachi.New(rt, humaCfg)

		mw := middleware.NewMiddleware(cfg, api)
		api.UseMiddleware(mw.Recoverer)

		rt.Get("/docs", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Dank Calendar API</title><meta charset="utf-8"/></head><body><script id="api-reference" data-url="/openapi.json"></script><script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script></body></html>`))
		})

		srvImpl := &server.Server{
			Cfg:      cfg,
			Repo:     r,
			Registry: registry,
			Secrets:  secrets,
		}

		group := huma.NewGroup(api, "/v1")
		group.UseModifier(func(op *huma.Operation, next func(*huma.Operation)) {
			op.Tags = []string{"Calendar"}
			next(op)
		})
		calendar_handler.RegisterHandlers(srvImpl, group)
	})

	addr := cfg.APIAddr
	if addr == "" {
		addr = "127.0.0.1:0"
	}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	listener, err := boundListener(httpServer.Addr)
	if err != nil {
		return nil, "", nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		err := httpServer.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	go func() {
		<-ctx.Done()
		shutdownHTTP(httpServer)
	}()

	return httpServer, listener.Addr().String(), errCh, nil
}

func shutdownHTTP(srv *http.Server) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
