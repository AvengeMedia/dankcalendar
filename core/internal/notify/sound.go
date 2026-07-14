package notify

import (
	"bytes"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
)

// Fallback chime for systems without a sound theme installed (see assets/CREDITS).
//
//go:embed assets/message-new-instant.wav
var reminderChime []byte

type soundPlayer struct {
	mu   sync.Mutex
	argv map[string][]string
}

func (p *soundPlayer) play(name string) bool {
	p.mu.Lock()
	if p.argv == nil {
		p.argv = make(map[string][]string)
	}
	argv, resolved := p.argv[name]
	if !resolved {
		argv = resolveArgv(name)
		p.argv[name] = argv
	}
	p.mu.Unlock()
	if argv == nil {
		return false
	}

	if !startSound(argv) {
		return false
	}
	// A two-pulse alarm is both more audible and recognizably different from
	// the single chime used for calendar events.
	if name == SoundTask {
		go func() {
			time.Sleep(400 * time.Millisecond)
			startSound(argv)
		}()
	}
	return true
}

func startSound(argv []string) bool {
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		log.Debugf("notify: play sound: %v", err)
		return false
	}
	go func() { _ = cmd.Wait() }()
	return true
}

func resolveArgv(name string) []string {
	file := themeSoundFile(name)
	if file == "" {
		extracted, err := chimeFile()
		if err != nil {
			log.Debugf("notify: extract chime: %v", err)
			return nil
		}
		file = extracted
	}

	for _, player := range []string{"pw-play", "paplay", "mpv", "ffplay"} {
		path, err := exec.LookPath(player)
		if err != nil {
			continue
		}
		switch player {
		case "mpv":
			return []string{path, "--no-terminal", file}
		case "ffplay":
			return []string{path, "-nodisp", "-autoexit", "-loglevel", "quiet", file}
		default:
			return []string{path, file}
		}
	}
	return nil
}

func themeSoundFile(name string) string {
	themes := make([]string, 0, 2)
	if theme := currentSoundTheme(); theme != "" && theme != "freedesktop" {
		themes = append(themes, theme)
	}
	themes = append(themes, "freedesktop")

	for _, theme := range themes {
		for _, dir := range append([]string{xdg.DataHome}, xdg.DataDirs...) {
			if dir == "" {
				continue
			}
			for _, ext := range []string{".oga", ".ogg", ".wav", ".mp3", ".flac"} {
				file := filepath.Join(dir, "sounds", theme, "stereo", name+ext)
				if _, err := os.Stat(file); err == nil {
					return file
				}
			}
		}
	}
	return ""
}

func currentSoundTheme() string {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.sound", "theme-name").Output()
	if err != nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(string(out)), "'")
}

// chimeFile materializes the embedded fallback sample to a real path players can open.
func chimeFile() (string, error) {
	dir, err := paths.CacheDir()
	if err != nil {
		return "", err
	}

	file := filepath.Join(dir, "message-new-instant.wav")
	if existing, err := os.ReadFile(file); err == nil && bytes.Equal(existing, reminderChime) {
		return file, nil
	}
	if err := os.WriteFile(file, reminderChime, 0o644); err != nil {
		return "", err
	}
	return file, nil
}
