//go:build withshell

package shellembed

import "embed"

// dist is populated from the repo's quickshell/ tree by `make sync-shell`
// before any tagged build; it is never committed.
//
//go:embed dist
var distFS embed.FS
