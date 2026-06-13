package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
)

type Server struct {
	deps     Deps
	listener net.Listener
	socket   string
	bus      *EventBus

	mu      sync.Mutex
	stopped bool
}

func NewServer(deps Deps) *Server {
	bus := deps.Bus
	if bus == nil {
		bus = NewEventBus()
		deps.Bus = bus
	}
	return &Server{deps: deps, bus: bus}
}

func (s *Server) Listen() error {
	cleanupStaleSockets()

	socketPath := paths.SocketPath()
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		listener.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	s.listener = listener
	s.socket = socketPath

	log.Infof("ipc listening on %s", socketPath)
	return nil
}

func (s *Server) SocketPath() string { return s.socket }

func (s *Server) Bus() *EventBus { return s.bus }

func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		return errors.New("listen has not been called")
	}

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			stopped := s.stopped
			s.mu.Unlock()
			if stopped {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		go s.handleConnection(ctx, conn)
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	s.stopped = true

	if s.listener != nil {
		s.listener.Close()
	}
	if s.socket != "" {
		_ = os.Remove(s.socket)
	}
	return nil
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	caps := Capabilities{
		APIVersion:   APIVersion,
		Capabilities: []string{"accounts", "calendars", "events", "reminders", "subscribe", "ui", "system"},
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(caps); err != nil {
		return
	}

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	writer := newConnWriter(conn)
	subscriber := s.bus.NewSubscriber(connCtx, writer)
	defer subscriber.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var req Request
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &req); err != nil {
			writer.WriteResponse(Response[any]{ID: 0, Error: "invalid json: " + err.Error()})
			continue
		}
		go Route(connCtx, writer, req, s.deps, subscriber)
	}
}

func cleanupStaleSockets() {
	dir := paths.SocketDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		switch {
		case !strings.HasPrefix(entry.Name(), "dankcal-"):
			continue
		case !strings.HasSuffix(entry.Name(), ".sock"):
			continue
		}

		pidStr := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "dankcal-"), ".sock")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		if !processAlive(pid) {
			path := filepath.Join(dir, entry.Name())
			_ = os.Remove(path)
			log.Debugf("removed stale socket %s", path)
		}
	}
}

func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := proc.Signal(syscallSignalZero); err != nil {
		return false
	}
	return true
}

func FindRunningSocket() (string, error) {
	dir := paths.SocketDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		switch {
		case !strings.HasPrefix(entry.Name(), "dankcal-"):
			continue
		case !strings.HasSuffix(entry.Name(), ".sock"):
			continue
		}

		path := filepath.Join(dir, entry.Name())
		conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
		if err != nil {
			continue
		}
		conn.Close()
		return path, nil
	}
	return "", errors.New("no running dankcal socket found")
}
