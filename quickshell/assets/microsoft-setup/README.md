# Microsoft account setup screenshots

Drop PNGs into this directory and the Add Account → Microsoft wizard will
display them under each step's description. Each step in
`core/internal/ipc/accounts_providers.go :: microsoftSetupSteps` references its
image by filename via the `screenshot` field; missing files degrade silently
(the image frame stays hidden).

Expected filenames:

| Step | Wizard title                     | Filename                  |
| ---- | -------------------------------- | ------------------------- |
| 1    | Start a new app registration     | `01-new-registration.png` |
| 2    | Fill the form and register       | `01-register-form.png`    |
| 3    | Copy the Application (client) ID | `03-client-id.png`        |

(`01-new-app.png` is an unused near-duplicate of the form capture.)

The redirect URI users register is the bare `http://localhost` — Entra ignores
the port for loopback URIs but matches the path exactly, so the daemon serves
the Microsoft OAuth callback on `/` (Google uses `/oauth/callback`).

PNGs are loaded via `Qt.resolvedUrl("../assets/microsoft-setup/<file>")` from
`quickshell/Modals/AccountAddModal.qml`. They are scaled to the wizard width
with `Image.PreserveAspectFit`; aim for ~1200px wide captures.
