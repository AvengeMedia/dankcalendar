# Contributing

Contributions are welcome and encouraged.

To contribute fork this repository, make your changes, and open a pull request.

## Setup

Install [prek](https://prek.j178.dev/) then activate pre-commit hooks:

```bash
prek install
```

The hooks run `gofmt -s`, `go vet`, `go mod tidy`, and the Go test suite for changes under `core/`, plus generic whitespace/YAML checks.

## VSCode Setup

This is a monorepo, the easiest thing to do is to open an editor in either `quickshell`, `core`, or both depending on which part of the project you are working on.

### QML (`quickshell` directory)

1. Install the [QML Extension](https://doc.qt.io/vscodeext/)
2. Configure `ctrl+shift+p` -> user preferences (json) with qmlls path

**Note:** Paths may vary by distribution. Below are examples for Arch Linux and Fedora.

**Arch Linux:**

```json
{
  "[qml]": {
    "editor.defaultFormatter": "qt-project.qmlls",
    "editor.formatOnSave": true
  },
  "qt-qml.doNotAskForQmllsDownload": true,
  "qt-qml.qmlls.customExePath": "/usr/lib/qt6/bin/qmlls",
  "qt-core.additionalQtPaths": [
    {
      "name": "Qt-6.x-linux-g++",
      "path": "/usr/bin/qmake"
    }
  ]
}
```

**Fedora:**

```json
{
  "[qml]": {
    "editor.defaultFormatter": "qt-project.qmlls",
    "editor.formatOnSave": true
  },
  "qt-qml.doNotAskForQmllsDownload": true,
  "qt-qml.qmlls.customExePath": "/usr/bin/qmlls",
  "qt-core.additionalQtPaths": [
    {
      "name": "Qt-6.x-Fedora-linux-g++",
      "path": "/usr/bin/qmake6"
    }
  ]
}
```

3. Create empty `.qmlls.ini` file in `quickshell/` directory

```bash
cd quickshell
touch .qmlls.ini
```

4. Make your changes, test, and open a pull request.

### I18n/Localization

When adding user-facing strings, ensure they are wrapped in `I18n.tr()` with context, for example.

```qml
import qs.Common

Text {
  text: I18n.tr("Hello World", "<This is context for the translators, example> Hello world greeting shown on the calendar header")
}
```

Preferably, try to keep new terms to a minimum and re-use existing terms where possible. See `quickshell/translations/en.json` for the list of existing terms. (This isn't always possible obviously, but instead of using `Auto-connect` you would use `Autoconnect` since it's already translated)

After adding or changing strings, regenerate the translation template from the repo root:

```bash
make i18n-extract
```

### GO (`core` directory)

1. Install the [Go Extension](https://code.visualstudio.com/docs/languages/go)
2. Ensure code is formatted with `make fmt`
3. Add appropriate test coverage and ensure tests pass with `make test`
4. Run `go mod tidy`
5. Open pull request

## Go guidelines

All Go commands below run from the `core/` directory (or via the root Makefile, which delegates).

### No CGO

The binary must build and test with `CGO_ENABLED=0`. The Makefile, CI, and pre-commit hooks all enforce this — do not add dependencies that require cgo (the SQLite driver is the pure-Go `modernc.org/sqlite` for this reason). The only exception is the CI race-detector step, which inherently needs cgo.

### Testing

- Tests use [testify](https://github.com/stretchr/testify): `assert`/`require` for plain tests, `suite` when tests share non-trivial setup (see `internal/sync/engine_test.go` or `api/calendar/handlers_test.go`).
- Tests must run sandboxed: no network, no system services, no pre-installed tools. Use `repo.OpenMemory` for an in-memory database, `t.TempDir()` for files, and `httptest`/`humatest` for HTTP. If a test needs named timezones, import `_ "time/tzdata"`.
- Prefer table-driven tests for pure functions.
- Run everything with `make test`, or `make cover` for a coverage report.

### Mocks

Mocks are generated with [mockery](https://vektra.github.io/mockery/) from `.mockery.yml` into `internal/mocks` — never write mocks by hand. To mock a new interface, add it under its package in `.mockery.yml` and run:

```bash
make mocks
```

Generated mocks use the expecter API:

```go
provider := mocks.NewMockProvider(t)
provider.EXPECT().ListCalendars(mock.Anything).Return(cals, nil)
```

### Style

- Use guard statements (early returns) instead of nested conditionals.
- Prefer `switch`/`case` over `if`/`else` chains.
- Keep comments sparse; explain constraints the code can't express, not what the code does.
- `gofmt -s` clean; `golangci-lint run` (config in `core/.golangci.yml`) must pass.

## Pull request

Include screenshots/video if applicable in your pull request if applicable, to visualize what your change is affecting.
