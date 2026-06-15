# Microsoft account setup screenshots

Drop PNGs into this directory and the Add Account → Microsoft wizard will
display them under each step's description. Each step in
`core/internal/accounts/setup.go :: MicrosoftSetupSteps` references its image by
filename via the `screenshot` field; missing files degrade silently (the image
frame stays hidden).

Expected filenames:

| Step | Wizard title                     | Filename                       |
| ---- | -------------------------------- | ------------------------------ |
| 1    | Make sure you have a directory   | `00-azure-ad-missing-error.png` |
| 2    | Start a new app registration     | `01-new-registration.png`      |
| 3    | Fill the form and register       | `01-register-form.png`         |
| 4    | Copy the Application (client) ID | `03-client-id.png`             |

Step 1 captures the "…not contained within any directory. The ability to create
applications outside of a directory has been deprecated" warning Microsoft shows
on App registrations for accounts without a Microsoft Entra directory — the most
common reason setup fails. The fix is to create a free directory (sign up for
Azure, or the Microsoft 365 Developer Program).

(`01-new-app.png` is an unused near-duplicate of the form capture.)

The redirect URI users register is the bare `http://localhost` — Entra ignores
the port for loopback URIs but matches the path exactly, so the daemon serves
the Microsoft OAuth callback on `/` (Google uses `/oauth/callback`).

PNGs are loaded via `Qt.resolvedUrl("../assets/microsoft-setup/<file>")` from
`quickshell/Modals/AccountAddModal.qml`. They are scaled to the wizard width
with `Image.PreserveAspectFit`; aim for ~1200px wide captures.
