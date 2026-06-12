# Google account setup screenshots

Drop PNGs into this directory and the Add Account → Google wizard will display
them under each step's description. Each step in `core/internal/ipc/accounts.go
:: googleSetupSteps` references its image by filename via the `screenshot`
field; missing files degrade silently (the image frame stays hidden).

The steps follow the current Google Auth Platform flow (consent screen
configuration moved there from the old "OAuth consent screen" page).

Expected filenames:

| Step | Wizard title                       | Filename                |
| ---- | ---------------------------------- | ----------------------- |
| 1    | Create a Google Cloud project      | `01-create-project.png` |
| 2    | Enable the Calendar API            | `02-enable-api.png`     |
| 3    | Configure the Google Auth Platform | `03-auth-platform.png`  |
| 4    | Add yourself as a test user        | `04-test-users.png`     |
| 5    | Create an OAuth client             | `05-client-create.png`  |

PNGs are loaded via `Qt.resolvedUrl("../assets/google-setup/<file>")` from
`quickshell/Modals/AccountAddModal.qml`. They are scaled to the wizard width
with `Image.PreserveAspectFit`; aim for ~1200px wide captures.
