//go:build !withshell

package shellembed

import "embed"

// Untagged builds (tests, vet, plain `go build`) carry no embedded UI;
// config resolution then requires an explicit shell dir.
var distFS embed.FS
