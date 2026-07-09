package uriopen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The KDE geo handler's real Exec line — quoted args that xdg-open chokes on.
const kdeGeoExec = `kde-geo-uri-handler --coordinate-template "https://www.google.com/maps/@<LAT>,<LON>,<Z>z" --query-template "https://www.google.com/maps/search/<Q>" --fallback "https://www.google.com/maps/" %u`

func TestBuildArgvQuotedTemplates(t *testing.T) {
	uri := "geo:0,0?q=35000%20Curtis%20Boulevard%2C%20Eastlake"
	argv, err := buildArgv(kdeGeoExec, uri)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"kde-geo-uri-handler",
		"--coordinate-template", "https://www.google.com/maps/@<LAT>,<LON>,<Z>z",
		"--query-template", "https://www.google.com/maps/search/<Q>",
		"--fallback", "https://www.google.com/maps/",
		uri,
	}, argv)
}

func TestBuildArgv(t *testing.T) {
	tests := []struct {
		name string
		exec string
		want []string
	}{
		{"embedded field code", `browser --url=%u --new-window`, []string{"browser", "--url=geo:1", "--new-window"}},
		{"no field code appends uri", `handler --flag`, []string{"handler", "--flag", "geo:1"}},
		{"deprecated codes dropped", `app %i %c %k %u`, []string{"app", "geo:1"}},
		{"literal percent", `app --pct=100%% %u`, []string{"app", "--pct=100%", "geo:1"}},
		{"escaped quote inside quotes", `app "say \"hi\"" %u`, []string{"app", `say "hi"`, "geo:1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argv, err := buildArgv(tt.exec, "geo:1")
			require.NoError(t, err)
			assert.Equal(t, tt.want, argv)
		})
	}
}

func TestBuildArgvUnterminatedQuote(t *testing.T) {
	_, err := buildArgv(`app "broken %u`, "geo:1")
	assert.Error(t, err)
}

func TestDefaultHandlerResolution(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "applications")
	require.NoError(t, os.MkdirAll(appDir, 0o755))

	writeFile(t, filepath.Join(appDir, "maps.desktop"), "[Desktop Entry]\nExec=maps %u\n")
	writeFile(t, filepath.Join(appDir, "backup.desktop"), "[Desktop Entry]\nExec=backup %u\n")
	writeFile(t, filepath.Join(dir, "mimeapps.list"), "[Default Applications]\nx-scheme-handler/geo=missing.desktop;maps.desktop\n")
	writeFile(t, filepath.Join(appDir, "mimeinfo.cache"), "[MIME Cache]\nx-scheme-handler/geo=backup.desktop;\n")

	mimeapps := []string{filepath.Join(dir, "mimeapps.list")}
	appDirs := []string{appDir}

	t.Run("default wins, missing entries skipped", func(t *testing.T) {
		assert.Equal(t, "maps.desktop", defaultHandler("x-scheme-handler/geo", mimeapps, appDirs))
	})

	t.Run("falls back to mimeinfo cache", func(t *testing.T) {
		assert.Equal(t, "backup.desktop", defaultHandler("x-scheme-handler/geo", nil, appDirs))
	})

	t.Run("unknown mime", func(t *testing.T) {
		assert.Equal(t, "", defaultHandler("x-scheme-handler/nope", mimeapps, appDirs))
	})
}

func TestFindDesktopFileDashSubdir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "vendor"), 0o755))
	writeFile(t, filepath.Join(dir, "vendor", "app.desktop"), "[Desktop Entry]\nExec=x\n")

	assert.Equal(t, filepath.Join(dir, "vendor", "app.desktop"), findDesktopFile("vendor-app.desktop", []string{dir}))
	assert.Equal(t, "", findDesktopFile("nope.desktop", []string{dir}))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
