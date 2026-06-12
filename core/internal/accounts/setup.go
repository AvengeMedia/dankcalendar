package accounts

type SetupStep struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
	URLLabel    string `json:"urlLabel,omitempty"`
	Screenshot  string `json:"screenshot,omitempty"`
}

// Screenshot filenames are resolved by the GUI wizard relative to its
// google-setup/microsoft-setup asset directories; missing files degrade
// silently.
func GoogleSetupSteps() []SetupStep {
	return []SetupStep{
		{
			Title:       "Create a Google Cloud project",
			Description: "Pick any name (e.g. \"dankcal\"). You'll only do this once.",
			URL:         "https://console.cloud.google.com/projectcreate",
			URLLabel:    "Open Google Cloud Console",
			Screenshot:  "01-create-project.png",
		},
		{
			Title:       "Enable the Calendar API",
			Description: "With the new project selected, click \"Enable\" on the Calendar API page.",
			URL:         "https://console.cloud.google.com/apis/library/calendar-json.googleapis.com",
			URLLabel:    "Enable Calendar API",
			Screenshot:  "02-enable-api.png",
		},
		{
			Title:       "Configure the Google Auth Platform",
			Description: "Click \"Get started\" on the overview page. App name: anything (e.g. \"dankcal\"). User support email: your own address. Audience: \"External\". Then agree and finish — nothing here is ever published.",
			URL:         "https://console.cloud.google.com/auth/overview",
			URLLabel:    "Open Google Auth Platform",
			Screenshot:  "03-auth-platform.png",
		},
		{
			Title:       "Add yourself as a test user",
			Description: "On the Audience page, under \"Test users\" click \"Add users\" and enter the Google account you want to sync. Only test users can sign in while the app is unpublished.",
			URL:         "https://console.cloud.google.com/auth/audience",
			URLLabel:    "Open Audience page",
			Screenshot:  "04-test-users.png",
		},
		{
			Title:       "Create an OAuth client",
			Description: "On the Clients page click \"Create client\", choose \"Desktop app\", and create it. Copy the Client ID and Client Secret for the next step.",
			URL:         "https://console.cloud.google.com/auth/clients",
			URLLabel:    "Open Clients page",
			Screenshot:  "05-client-create.png",
		},
	}
}

func MicrosoftSetupSteps() []SetupStep {
	return []SetupStep{
		{
			Title:       "Start a new app registration",
			Description: "Open App registrations in the Microsoft Entra portal and click \"New registration\". You'll only do this once.",
			URL:         "https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade",
			URLLabel:    "Open App registrations",
			Screenshot:  "01-new-registration.png",
		},
		{
			Title:       "Fill the form and register",
			Description: "Name: anything (e.g. \"dankcal\"). Supported account types: \"Any Entra ID Tenant + Personal Microsoft accounts\". Redirect URI: choose \"Public client/native (mobile & desktop)\" and enter http://localhost. Then click Register.",
			Screenshot:  "01-register-form.png",
		},
		{
			Title:       "Copy the Application (client) ID",
			Description: "From the app's Overview page, copy the \"Application (client) ID\". No client secret is needed for a desktop app.",
			Screenshot:  "03-client-id.png",
		},
	}
}
