package httpauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vectors from RFC 7616 §3.9.1.
func TestAuthorizationMatchesRFC7616Vectors(t *testing.T) {
	const (
		nonce  = "7ypf/xlj9XXwfDPEoM4URrv/xwf94BcCAzFZH4GiTo0v"
		cnonce = "f2/wE4q74E6zIJEtWaHKaf5wv/H5QzzpXusqGemxURZJ"
		opaque = "FQhe/qaU925kfnzjCev0ciny7QMkPqMAFRtzCUYo5tdS"
	)
	cases := []struct {
		header   string
		response string
	}{
		{
			header:   `Digest realm="http-auth@example.org", qop="auth, auth-int", algorithm=MD5, nonce="` + nonce + `", opaque="` + opaque + `"`,
			response: "8ca523f5e9506fed4657c9700eebdbec",
		},
		{
			header:   `Digest realm="http-auth@example.org", qop="auth, auth-int", algorithm=SHA-256, nonce="` + nonce + `", opaque="` + opaque + `"`,
			response: "753927fa0e85d155564e2e272a28d1802ca10daf4496794697cf8db5856cb6c1",
		},
	}
	for _, tc := range cases {
		ch := parseChallenge([]string{tc.header})
		require.NotNil(t, ch, tc.header)
		got := ch.authorization("Mufasa", "Circle of Life", "GET", "/dir/index.html", 1, cnonce)
		assert.Contains(t, got, `response="`+tc.response+`"`)
		assert.Contains(t, got, `opaque="`+opaque+`"`)
		assert.Contains(t, got, "qop=auth, nc=00000001")
	}
}

func TestParseChallengePicksSupportedDigest(t *testing.T) {
	headers := []string{
		`Basic realm="dav"`,
		`Digest realm="dav", qop="auth-int", nonce="n1", algorithm=MD5, Digest realm="dav", qop="auth", nonce="n2", algorithm=SHA-256, stale=TRUE`,
	}
	ch := parseChallenge(headers)
	require.NotNil(t, ch)
	assert.Equal(t, "n2", ch.nonce)
	assert.Equal(t, "SHA-256", ch.algorithm)
	assert.True(t, ch.stale)
	assert.Nil(t, parseChallenge([]string{`Bearer realm="x"`, `Digest realm="x", nonce="n", algorithm=SHA-1`}))
}

func TestReadValueUnescapesQuotedString(t *testing.T) {
	ch := parseChallenge([]string{`Digest realm="say \"hi\" \\ bye",nonce=abc`})
	require.NotNil(t, ch)
	assert.Equal(t, `say "hi" \ bye`, ch.realm)
	assert.Equal(t, "abc", ch.nonce)
}
