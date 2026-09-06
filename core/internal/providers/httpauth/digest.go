package httpauth

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"slices"
	"strings"
)

type challenge struct {
	realm     string
	nonce     string
	opaque    string
	algorithm string
	qop       bool
	stale     bool
	sess      bool
	newHash   func() hash.Hash
}

func parseChallenge(headers []string) *challenge {
	for _, header := range headers {
		for _, raw := range splitChallenges(header) {
			if !strings.EqualFold(raw.scheme, "Digest") {
				continue
			}
			if ch := newChallenge(raw.params); ch != nil {
				return ch
			}
		}
	}
	return nil
}

func newChallenge(params map[string]string) *challenge {
	if params["nonce"] == "" {
		return nil
	}
	name := strings.ToUpper(params["algorithm"])
	if name == "" {
		name = "MD5"
	}
	ch := &challenge{
		realm:     params["realm"],
		nonce:     params["nonce"],
		opaque:    params["opaque"],
		algorithm: params["algorithm"],
		stale:     strings.EqualFold(params["stale"], "true"),
		sess:      strings.HasSuffix(name, "-SESS"),
	}
	switch strings.TrimSuffix(name, "-SESS") {
	case "MD5":
		ch.newHash = md5.New
	case "SHA-256":
		ch.newHash = sha256.New
	case "SHA-512-256":
		ch.newHash = sha512.New512_256
	default:
		return nil
	}
	if qop := params["qop"]; qop != "" {
		if !slices.Contains(splitList(qop), "auth") {
			return nil
		}
		ch.qop = true
	}
	return ch
}

func (c *challenge) authorization(username, password, method, uri string, nonceCount uint32, cnonce string) string {
	a1 := c.digest(username + ":" + c.realm + ":" + password)
	if c.sess {
		a1 = c.digest(a1 + ":" + c.nonce + ":" + cnonce)
	}
	a2 := c.digest(method + ":" + uri)

	fields := []string{
		"username=" + quote(username),
		"realm=" + quote(c.realm),
		"nonce=" + quote(c.nonce),
		"uri=" + quote(uri),
	}
	var response string
	switch {
	case c.qop:
		nc := fmt.Sprintf("%08x", nonceCount)
		response = c.digest(a1 + ":" + c.nonce + ":" + nc + ":" + cnonce + ":auth:" + a2)
		fields = append(fields, "qop=auth", "nc="+nc, "cnonce="+quote(cnonce))
	default:
		response = c.digest(a1 + ":" + c.nonce + ":" + a2)
		if c.sess {
			fields = append(fields, "cnonce="+quote(cnonce))
		}
	}
	fields = append(fields, "response="+quote(response))
	if c.algorithm != "" {
		fields = append(fields, "algorithm="+c.algorithm)
	}
	if c.opaque != "" {
		fields = append(fields, "opaque="+quote(c.opaque))
	}
	return "Digest " + strings.Join(fields, ", ")
}

func (c *challenge) digest(s string) string {
	h := c.newHash()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func quote(s string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
	return `"` + escaped + `"`
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	for i, part := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(part))
	}
	return parts
}

type rawChallenge struct {
	scheme string
	params map[string]string
}

// splitChallenges parses one WWW-Authenticate value, which may carry several
// comma-separated challenges (RFC 9110 §11.6.1).
func splitChallenges(header string) []rawChallenge {
	var out []rawChallenge
	rest := header
	for {
		rest = strings.TrimLeft(rest, " \t,")
		if rest == "" {
			return out
		}
		token, after := readToken(rest)
		if token == "" {
			return out
		}
		after = strings.TrimLeft(after, " \t")
		if !strings.HasPrefix(after, "=") {
			out = append(out, rawChallenge{scheme: token, params: map[string]string{}})
			rest = after
			continue
		}
		value, after := readValue(strings.TrimLeft(after[1:], " \t"))
		if len(out) > 0 {
			out[len(out)-1].params[strings.ToLower(token)] = value
		}
		rest = after
	}
}

func readToken(s string) (string, string) {
	end := strings.IndexAny(s, " \t,=")
	if end < 0 {
		return s, ""
	}
	return s[:end], s[end:]
}

func readValue(s string) (string, string) {
	if !strings.HasPrefix(s, `"`) {
		return readToken(s)
	}
	var value strings.Builder
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 < len(s) {
				i++
				value.WriteByte(s[i])
			}
		case '"':
			return value.String(), s[i+1:]
		default:
			value.WriteByte(s[i])
		}
	}
	return value.String(), ""
}
