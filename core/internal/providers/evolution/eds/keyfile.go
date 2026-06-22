package eds

import "strings"

// keyFile is a minimal reader for the GLib key-file format that EDS uses to
// describe a source (the Source.Data property). Localized keys (DisplayName[de])
// are dropped in favour of the plain key.
type keyFile map[string]map[string]string

func parseKeyFile(data string) keyFile {
	out := keyFile{}
	section := ""
	for raw := range strings.SplitSeq(data, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case line == "", strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			section = line[1 : len(line)-1]
			if out[section] == nil {
				out[section] = map[string]string{}
			}
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || section == "" || strings.ContainsRune(key, '[') {
			continue
		}
		out[section][key] = strings.TrimSpace(value)
	}
	return out
}

func (k keyFile) hasSection(name string) bool {
	_, ok := k[name]
	return ok
}

func (k keyFile) get(section, key string) string {
	if s, ok := k[section]; ok {
		return s[key]
	}
	return ""
}
