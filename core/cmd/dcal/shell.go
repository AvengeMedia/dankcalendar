package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
)

var (
	isSessionManaged bool
	startHidden      bool
)

func getProcessExitCode(state *os.ProcessState) int {
	if state == nil {
		return 1
	}
	if code := state.ExitCode(); code != -1 {
		return code
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return 128 + int(status.Signal())
		}
	}
	return 1
}

func hasSystemdRun() bool {
	_, err := exec.LookPath("systemd-run")
	return err == nil
}

func appendLogEnv(env []string) []string {
	if v := os.Getenv("DANKCAL_LOG_LEVEL"); v != "" {
		env = append(env, "DANKCAL_LOG_LEVEL="+v)
	}
	if v := os.Getenv("DANKCAL_LOG_FILE"); v != "" {
		env = append(env, "DANKCAL_LOG_FILE="+v)
	}
	if rules := log.GetQtLoggingRules(); rules != "" {
		env = append(env, "QT_LOGGING_RULES="+rules)
	}
	return env
}

func appendQtEnv(env []string) []string {
	if os.Getenv("QT_QPA_PLATFORMTHEME") == "" {
		env = append(env, "QT_QPA_PLATFORMTHEME=gtk3")
	}
	if os.Getenv("QT_QPA_PLATFORMTHEME_QT6") == "" {
		env = append(env, "QT_QPA_PLATFORMTHEME_QT6=gtk3")
	}
	if os.Getenv("QT_QPA_PLATFORM") == "" {
		env = append(env, "QT_QPA_PLATFORM=wayland;xcb")
	}
	return env
}

func pidFilePath() string {
	return filepath.Join(runtimeDir(), fmt.Sprintf("%s%d%s", pidFilePrefix, os.Getpid(), pidFileExtension))
}

func writePIDFile(childPID int) error {
	return os.WriteFile(pidFilePath(), []byte(strconv.Itoa(childPID)), 0o644)
}

func removePIDFile() {
	os.Remove(pidFilePath())
}

func getAllPIDs() []int {
	dir := runtimeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var pids []int

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, pidFilePrefix) || !strings.HasSuffix(name, pidFileExtension) {
			continue
		}

		pidFile := filepath.Join(dir, name)
		data, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}

		childPID, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			os.Remove(pidFile)
			continue
		}

		proc, err := os.FindProcess(childPID)
		if err != nil {
			os.Remove(pidFile)
			continue
		}

		if err := proc.Signal(syscall.Signal(0)); err != nil {
			os.Remove(pidFile)
			continue
		}

		pids = append(pids, childPID)

		parentPIDStr := strings.TrimPrefix(name, pidFilePrefix)
		parentPIDStr = strings.TrimSuffix(parentPIDStr, pidFileExtension)
		if parentPID, err := strconv.Atoi(parentPIDStr); err == nil {
			if parentProc, err := os.FindProcess(parentPID); err == nil {
				if err := parentProc.Signal(syscall.Signal(0)); err == nil {
					pids = append(pids, parentPID)
				}
			}
		}
	}

	return pids
}

func writeConfigState() error {
	return os.WriteFile(activeConfigStateFile(), []byte(configPath), 0o644)
}

func clearConfigState() {
	os.Remove(activeConfigStateFile())
}

func runShellInteractive(session bool) error {
	isSessionManaged = session
	fmt.Fprintf(os.Stderr, "dcal %s\n", Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := bootDaemonServices(ctx)
	if err != nil {
		return fmt.Errorf("starting backend: %w", err)
	}
	defer svc.close()

	if err := writeConfigState(); err != nil {
		log.Warnf("failed to write config state: %v", err)
	}
	defer clearConfigState()

	log.Infof("dcal backend ready (ipc=%s http=%s)", svc.socketPath(), svc.httpAddr)
	log.Infof("starting UI (config=%s)", configPath)

	cmd := buildUICommand(ctx, svc.socketPath())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return runShell(cmd, false)
}

func runShellDaemon(session bool) error {
	isSessionManaged = session
	isDaemonChild := slices.Contains(os.Args, "--daemon-child")

	if !isDaemonChild {
		fmt.Fprintf(os.Stderr, "dcal %s\n", Version)

		args := []string{"run", "-d", "--daemon-child"}
		if startHidden {
			args = append(args, "--hidden")
		}
		if customConfigPath != "" {
			args = append([]string{"-c", customConfigPath}, args...)
		}

		cmd := exec.Command(os.Args[0], args...)
		cmd.Env = os.Environ()
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("starting daemon: %w", err)
		}

		log.Infof("dcal daemon started (pid=%d)", cmd.Process.Pid)
		return nil
	}

	fmt.Fprintf(os.Stderr, "dcal %s\n", Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := bootDaemonServices(ctx)
	if err != nil {
		return fmt.Errorf("starting backend: %w", err)
	}
	defer svc.close()

	if err := writeConfigState(); err != nil {
		log.Warnf("failed to write config state: %v", err)
	}
	defer clearConfigState()

	log.Infof("dcal backend ready (ipc=%s http=%s)", svc.socketPath(), svc.httpAddr)

	cmd := buildUICommand(ctx, svc.socketPath())

	devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open /dev/null: %w", err)
	}
	defer devNull.Close()

	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	return runShell(cmd, true)
}

func buildUICommand(ctx context.Context, socketPath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "qs", "-p", configPath)
	env := append(os.Environ(),
		"DANKCAL_SOCKET="+socketPath,
		"QS_APP_ID=com.danklinux.dankcalendar",
	)

	if startHidden {
		env = append(env, "DANKCAL_START_HIDDEN=1")
	}

	if isSessionManaged && hasSystemdRun() {
		env = append(env, "DANKCAL_DEFAULT_LAUNCH_PREFIX=systemd-run --user --scope")
	}

	env = appendQtEnv(env)
	env = appendLogEnv(env)
	cmd.Env = env
	return cmd
}

func runShell(cmd *exec.Cmd, silent bool) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting UI: %w", err)
	}

	if err := writePIDFile(cmd.Process.Pid); err != nil {
		log.Warnf("failed to write pid file: %v", err)
	}
	defer removePIDFile()

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	exitChan := make(chan error, 1)
	go func() {
		if err := cmd.Wait(); err != nil {
			exitChan <- fmt.Errorf("UI exited: %w", err)
			return
		}
		exitChan <- fmt.Errorf("UI exited")
	}()

	for {
		select {
		case sig := <-sigChan:
			if sig == syscall.SIGUSR1 {
				if isSessionManaged {
					log.Infof("received SIGUSR1, exiting for systemd restart")
					cmd.Process.Signal(syscall.SIGTERM)
					os.Exit(1)
				}
				log.Infof("received SIGUSR1, spawning detached restart")
				execDetachedRestart(os.Getpid())
				return nil
			}

			select {
			case <-exitChan:
				os.Exit(getProcessExitCode(cmd.ProcessState))
			case <-time.After(500 * time.Millisecond):
			}

			if !silent {
				log.Infof("\nreceived %v, shutting down", sig)
			}
			cmd.Process.Signal(syscall.SIGTERM)
			return nil

		case err := <-exitChan:
			if !silent && err != nil {
				log.Error(err)
			}
			if cmd.Process != nil {
				cmd.Process.Signal(syscall.SIGTERM)
			}
			os.Exit(getProcessExitCode(cmd.ProcessState))
		}
	}
}

func execDetachedRestart(targetPID int) {
	selfPath, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.Command(selfPath, "restart-detached", strconv.Itoa(targetPID))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Start()
}

func runDetachedRestart(targetPIDStr string) {
	targetPID, err := strconv.Atoi(targetPIDStr)
	if err != nil {
		return
	}

	time.Sleep(200 * time.Millisecond)

	if proc, err := os.FindProcess(targetPID); err == nil {
		proc.Signal(syscall.SIGTERM)
	}

	time.Sleep(500 * time.Millisecond)

	killShell()
	_ = runShellDaemon(false)
}

func restartShell() {
	pids := getAllPIDs()
	if len(pids) == 0 {
		log.Info("no running dcal instances; starting daemon")
		_ = runShellDaemon(false)
		return
	}

	currentPid := os.Getpid()
	unique := make(map[int]bool)
	for _, pid := range pids {
		if pid != currentPid {
			unique[pid] = true
		}
	}

	for pid := range unique {
		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Errorf("finding process %d: %v", pid, err)
			continue
		}
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			continue
		}
		if err := proc.Signal(syscall.SIGUSR1); err != nil {
			log.Errorf("sending SIGUSR1 to %d: %v", pid, err)
			continue
		}
		log.Infof("sent SIGUSR1 to dcal pid=%d", pid)
	}
}

func killShell() {
	pids := getAllPIDs()
	if len(pids) == 0 {
		log.Info("no running dcal instances")
		return
	}

	currentPid := os.Getpid()
	unique := make(map[int]bool)
	for _, pid := range pids {
		if pid != currentPid {
			unique[pid] = true
		}
	}

	for pid := range unique {
		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Errorf("finding process %d: %v", pid, err)
			continue
		}
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			continue
		}
		if err := proc.Kill(); err != nil {
			log.Errorf("killing process %d: %v", pid, err)
			continue
		}
		log.Infof("killed dcal pid=%d", pid)
	}

	dir := runtimeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, pidFilePrefix) && strings.HasSuffix(name, pidFileExtension) {
			os.Remove(filepath.Join(dir, name))
		}
	}
}
