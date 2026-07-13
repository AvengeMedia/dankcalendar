package notify

import (
	"bytes"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adrg/xdg"

	"github.com/AvengeMedia/dankcalendar/core/internal/log"
	"github.com/AvengeMedia/dankcalendar/core/internal/paths"
)

// Fallback chime for systems without a sound theme installed (see assets/CREDITS).
//
//go:embed assets/message-new-instant.wav
var reminderChime []byte

type soundPlayer struct {
	once sync.Once
	argv []string
}

func (p *soundPlayer) play() bool {
	p.once.Do(func() { p.argv = resolveArgv() })
	if p.argv == nil {
		return false
	}

	cmd := exec.Command(p.argv[0], p.argv[1:]...)
	if err := cmd.Start(); err != nil {
		log.Debugf("notify: play sound: %v", err)
		return false
	}
	go func() { _ = cmd.Wait() }()
	return true
}

func resolveArgv() []string {
	file := themeSoundFile(SoundReminder)
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
