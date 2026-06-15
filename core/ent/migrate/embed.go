package migrate

import "embed"

// Migrations holds the versioned, goose-format SQL files applied on startup.
// Generate new ones with `go run -mod=mod ./ent/migrate/main.go -name <name>`.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// InitialVersion is the goose version of the first migration. Databases created
// before versioned migrations existed are stamped at this version so goose
// applies only the migrations that came after, instead of recreating tables.
const InitialVersion int64 = 20260615203534

// ReauthFieldsVersion adds accounts.needs_reauth and accounts.auth_error. A
// legacy database that already has those columns is stamped at this version too.
const ReauthFieldsVersion int64 = 20260615203550
