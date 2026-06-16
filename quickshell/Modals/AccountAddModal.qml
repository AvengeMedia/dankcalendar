import QtQuick
import QtQuick.Controls
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Modals.FileBrowser
import qs.Services
import qs.Widgets

FloatingWindow {
    id: accountModal

    property string selectedProvider: ""
    property int wizardStep: 0
    property string clientId: ""
    property string clientSecret: ""
    property string pendingState: ""
    property string pendingAuthUrl: ""
    property string completedAccountId: ""
    property string completedEmail: ""
    property string flowError: ""
    property bool flowInProgress: false

    readonly property var providers: DankCalService.providers
    readonly property bool isMicrosoft: selectedProvider === "microsoft"
    readonly property var setupSteps: isMicrosoft ? DankCalService.microsoftSetupSteps : DankCalService.googleSetupSteps
    readonly property string screenshotDir: isMicrosoft ? "../assets/microsoft-setup/" : "../assets/google-setup/"
    readonly property int guideCount: setupSteps.length > 0 ? setupSteps.length : (isMicrosoft ? 4 : 5)
    readonly property int credsStep: guideCount + 1
    readonly property int browserStepIndex: guideCount + 2
    readonly property int doneStep: guideCount + 3

    function show() {
        resetState();
        visible = true;
    }

    function hide() {
        cancelPendingFlow();
        visible = false;
    }

    function resetState() {
        selectedProvider = "";
        wizardStep = 0;
        clientId = "";
        clientSecret = "";
        pendingState = "";
        pendingAuthUrl = "";
        completedAccountId = "";
        completedEmail = "";
        flowError = "";
        flowInProgress = false;
        expandedScreenshot = "";
    }

    function cancelPendingFlow() {
        if (pendingState === "")
            return;
        if (isMicrosoft)
            DankCalService.cancelMicrosoftFlow(pendingState);
        else
            DankCalService.cancelGoogleFlow(pendingState);
        pendingState = "";
    }

    function startOAuthFlow() {
        flowError = "";
        flowInProgress = true;

        const onStarted = response => {
            if (response.error) {
                flowInProgress = false;
                flowError = response.error;
                return;
            }
            const result = response.result || {};
            pendingState = result.state || "";
            pendingAuthUrl = result.authUrl || "";
            wizardStep = browserStepIndex;
            Qt.openUrlExternally(result.authUrl);
            const complete = isMicrosoft ? DankCalService.completeMicrosoftFlow : DankCalService.completeGoogleFlow;
            complete(result.state, () => {
                flowInProgress = false;
            });
        };

        if (isMicrosoft)
            DankCalService.startMicrosoftFlow(clientId, onStarted);
        else
            DankCalService.startGoogleFlow("", clientId, clientSecret, onStarted);
    }

    function providerName(id) {
        for (let i = 0; i < providers.length; i++) {
            if (providers[i].id === id)
                return providers[i].name;
        }
        switch (id) {
        case "google":
            return "Google";
        case "microsoft":
            return "Microsoft";
        case "caldav":
            return "CalDAV";
        case "icloud":
            return "iCloud";
        case "local":
            return I18n.tr("Local", "local provider name in account add modal");
        }
        return id;
    }

    function providerIcon(id) {
        switch (id) {
        case "google":
            return "mail";
        case "microsoft":
            return "business";
        case "icloud":
            return "cloud_circle";
        case "local":
            return "folder";
        default:
            return "cloud";
        }
    }

    function finishSimpleAdd(accountId) {
        completedAccountId = accountId || "";
        completedEmail = completedAccountId;
        flowInProgress = false;
        wizardStep = doneStep;
    }

    property var pickerCallback: null
    property string expandedScreenshot: ""

    function openFilePicker(opts, callback) {
        pickerCallback = callback;
        pickerLoader.active = true;
        const picker = pickerLoader.item;
        picker.browserTitle = opts.title || I18n.tr("Select file", "default title for file picker in account add modal");
        picker.fileExtensions = opts.extensions || ["*.*"];
        picker.folderMode = !!opts.folderMode;
        picker.open();
    }

    function importCredentialsFile(path) {
        if (credentialsFile.path === path) {
            _applyCredentialsJson(credentialsFile.text());
            return;
        }
        credentialsFile.path = path;
    }

    function _applyCredentialsJson(content) {
        let doc;
        try {
            doc = JSON.parse(content);
        } catch (e) {
            flowError = I18n.tr("That file is not valid JSON.", "error when imported credentials file is not json");
            return;
        }
        const client = doc.installed || doc.web;
        if (!client || !client.client_id) {
            flowError = I18n.tr("No OAuth client found in that file — expected the client_secret_….json downloaded from Google.", "error when credentials json lacks an oauth client");
            return;
        }
        clientId = client.client_id;
        clientSecret = client.client_secret || "";
        flowError = "";
    }

    FileView {
        id: credentialsFile
        onLoaded: accountModal._applyCredentialsJson(credentialsFile.text())
        onLoadFailed: accountModal.flowError = I18n.tr("Could not read that file.", "error when credentials file cannot be read")
    }

    Loader {
        id: pickerLoader
        active: false
        sourceComponent: FileBrowserModal {
            parentModal: accountModal
            onFileSelected: path => {
                const cb = accountModal.pickerCallback;
                accountModal.pickerCallback = null;
                close();
                if (cb)
                    cb(path);
            }
        }
    }

    title: I18n.tr("Add account", "account add modal window title")
    minimumSize: Qt.size(560, 540)
    implicitWidth: 640
    implicitHeight: 620
    color: Theme.surface
    visible: false

    Connections {
        target: DankCalService
        function onGoogleFlowCompleted(accountId, email) {
            if (accountId === "")
                return;
            accountModal.completedAccountId = accountId;
            accountModal.completedEmail = email;
            accountModal.flowInProgress = false;
            accountModal.wizardStep = accountModal.doneStep;
        }
        function onGoogleFlowFailed(state, error) {
            if (state !== accountModal.pendingState)
                return;
            accountModal.flowError = error;
            accountModal.flowInProgress = false;
        }
        function onMicrosoftFlowCompleted(accountId, email) {
            if (accountId === "")
                return;
            accountModal.completedAccountId = accountId;
            accountModal.completedEmail = email;
            accountModal.flowInProgress = false;
            accountModal.wizardStep = accountModal.doneStep;
        }
        function onMicrosoftFlowFailed(state, error) {
            if (state !== accountModal.pendingState)
                return;
            accountModal.flowError = error;
            accountModal.flowInProgress = false;
        }
    }

    Column {
        anchors.fill: parent
        spacing: 0

        Item {
            width: parent.width
            height: 48
            z: 10

            MouseArea {
                anchors.fill: parent
                onPressed: windowControls.tryStartMove()
            }

            Rectangle {
                anchors.fill: parent
                color: Theme.surfaceContainer
                opacity: 0.5
            }

            Row {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingL
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingM

                DankActionButton {
                    visible: accountModal.selectedProvider !== ""
                    circular: false
                    iconName: "arrow_back"
                    iconColor: Theme.surfaceText
                    anchors.verticalCenter: parent.verticalCenter
                    onClicked: {
                        accountModal.cancelPendingFlow();
                        if (accountModal.wizardStep > 0) {
                            accountModal.wizardStep -= 1;
                        } else {
                            accountModal.selectedProvider = "";
                        }
                    }
                }

                DankIcon {
                    name: "person_add"
                    size: Theme.iconSize
                    color: Theme.primary
                    anchors.verticalCenter: parent.verticalCenter
                }

                StyledText {
                    text: accountModal.selectedProvider === "" ? I18n.tr("Add account", "account add modal header title") : I18n.tr("Connect %1", "account add header when a provider is selected").arg(accountModal.providerName(accountModal.selectedProvider))
                    font.pixelSize: Theme.fontSizeXLarge
                    font.weight: Font.Medium
                    color: Theme.surfaceText
                    anchors.verticalCenter: parent.verticalCenter
                }
            }

            DankActionButton {
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                circular: false
                iconName: "close"
                iconColor: Theme.surfaceText
                onClicked: accountModal.hide()
            }
        }

        Item {
            width: parent.width
            height: parent.height - 48
            clip: true

            Loader {
                anchors.fill: parent
                anchors.margins: Theme.spacingL
                sourceComponent: {
                    switch (accountModal.selectedProvider) {
                    case "":
                        return providerPicker;
                    case "caldav":
                    case "icloud":
                        return caldavForm;
                    case "local":
                        return localForm;
                    default:
                        return oauthWizard;
                    }
                }
            }
        }
    }

    Rectangle {
        id: previewOverlay
        anchors.fill: parent
        z: 100
        visible: accountModal.expandedScreenshot !== ""
        color: Qt.rgba(0, 0, 0, 0.85)

        property bool zoomed: false

        onVisibleChanged: zoomed = false

        MouseArea {
            anchors.fill: parent
            onClicked: accountModal.expandedScreenshot = ""
        }

        Flickable {
            id: previewFlick
            anchors.fill: parent
            anchors.topMargin: 52
            anchors.bottomMargin: 36
            anchors.leftMargin: Theme.spacingL
            anchors.rightMargin: Theme.spacingL
            clip: true
            contentWidth: previewImage.width
            contentHeight: previewImage.height
            flickableDirection: Flickable.HorizontalAndVerticalFlick
            interactive: previewOverlay.zoomed

            Image {
                id: previewImage
                source: accountModal.expandedScreenshot
                asynchronous: true
                fillMode: Image.PreserveAspectFit
                sourceSize.width: 2048
                width: previewOverlay.zoomed ? Math.max(previewFlick.width, implicitWidth) : previewFlick.width
                height: previewOverlay.zoomed ? Math.max(previewFlick.height, implicitHeight) : previewFlick.height

                MouseArea {
                    anchors.fill: parent
                    cursorShape: previewOverlay.zoomed ? Qt.OpenHandCursor : Qt.ZoomInCursor
                    onClicked: previewOverlay.zoomed = !previewOverlay.zoomed
                }
            }
        }

        DankActionButton {
            anchors.top: parent.top
            anchors.right: parent.right
            anchors.margins: Theme.spacingM
            circular: false
            iconName: "close"
            iconColor: "white"
            onClicked: accountModal.expandedScreenshot = ""
        }

        StyledText {
            anchors.bottom: parent.bottom
            anchors.bottomMargin: Theme.spacingS
            anchors.horizontalCenter: parent.horizontalCenter
            text: previewOverlay.zoomed ? I18n.tr("Drag to pan · click to fit", "hint while a setup screenshot is zoomed") : I18n.tr("Click to zoom · Esc to close", "hint while a setup screenshot is fit to the window")
            font.pixelSize: Theme.fontSizeSmall
            color: Qt.rgba(1, 1, 1, 0.7)
        }

        Shortcut {
            sequence: "Escape"
            enabled: previewOverlay.visible
            onActivated: accountModal.expandedScreenshot = ""
        }
    }

    Shortcut {
        sequence: "Escape"
        enabled: accountModal.expandedScreenshot === ""
        onActivated: accountModal.hide()
    }

    FloatingWindowControls {
        id: windowControls
        targetWindow: accountModal
        z: 101
    }

    Component {
        id: providerPicker

        DankFlickable {
            clip: true
            contentWidth: width
            contentHeight: pickerColumn.implicitHeight + Theme.spacingL * 2

            Column {
                id: pickerColumn
                width: parent.width
                spacing: Theme.spacingM

                StyledText {
                    text: I18n.tr("Choose a provider", "heading for provider selection list")
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.Medium
                    color: Theme.surfaceText
                }

                StyledText {
                    text: I18n.tr("All credentials are stored in your system keyring when available.", "subtitle under provider selection heading")
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.surfaceVariantText
                }

                Grid {
                    columns: 2
                    columnSpacing: Theme.spacingM
                    rowSpacing: Theme.spacingM
                    width: parent.width

                    Repeater {
                        model: ScriptModel {
                            values: accountModal.providers && accountModal.providers.length > 0 ? accountModal.providers : [
                                {
                                    "id": "google",
                                    "name": "Google",
                                    "description": I18n.tr("Sign in with your Google account.", "google provider description in picker"),
                                    "implemented": true
                                },
                                {
                                    "id": "microsoft",
                                    "name": "Microsoft",
                                    "description": I18n.tr("Outlook, Office 365 and Exchange Online calendars.", "microsoft provider description in picker"),
                                    "implemented": true
                                },
                                {
                                    "id": "caldav",
                                    "name": "CalDAV",
                                    "description": I18n.tr("Connect any CalDAV-compatible server.", "caldav provider description in picker"),
                                    "implemented": true
                                },
                                {
                                    "id": "icloud",
                                    "name": "iCloud",
                                    "description": I18n.tr("Apple iCloud calendars via app-specific password.", "icloud provider description in picker"),
                                    "implemented": true
                                },
                                {
                                    "id": "local",
                                    "name": I18n.tr("Local", "local provider name in account add modal"),
                                    "description": I18n.tr("A folder of .ics files on this machine.", "local provider description in picker"),
                                    "implemented": true
                                }
                            ]
                        }

                        StyledRect {
                            required property var modelData
                            width: (parent.width - Theme.spacingM) / 2
                            height: 132
                            color: Theme.surfaceContainer
                            radius: Theme.cornerRadius
                            opacity: modelData.implemented ? 1 : 0.55

                            Column {
                                anchors.fill: parent
                                anchors.margins: Theme.spacingM
                                spacing: Theme.spacingS

                                Row {
                                    spacing: Theme.spacingS
                                    width: parent.width

                                    Rectangle {
                                        width: 36
                                        height: 36
                                        radius: 18
                                        color: Theme.withAlpha(Theme.primary, 0.18)

                                        DankIcon {
                                            anchors.centerIn: parent
                                            name: accountModal.providerIcon(parent.parent.parent.parent.modelData.id)
                                            size: Theme.iconSize - 4
                                            color: Theme.primary
                                        }
                                    }

                                    StyledText {
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: parent.parent.parent.modelData.name
                                        font.pixelSize: Theme.fontSizeMedium
                                        font.weight: Font.Medium
                                        color: Theme.surfaceText
                                    }
                                }

                                StyledText {
                                    text: parent.parent.modelData.description
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    width: parent.width
                                    wrapMode: Text.WordWrap
                                }

                                Item {
                                    width: parent.width
                                    height: 1
                                }

                                Rectangle {
                                    visible: !parent.parent.modelData.implemented
                                    width: badgeText.implicitWidth + Theme.spacingM
                                    height: 22
                                    radius: 11
                                    color: Theme.withAlpha(Theme.warning, 0.18)

                                    StyledText {
                                        id: badgeText
                                        anchors.centerIn: parent
                                        text: I18n.tr("Coming soon", "badge on unimplemented provider card")
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.warning
                                    }
                                }
                            }

                            StateLayer {
                                stateColor: Theme.primary
                                cornerRadius: parent.radius
                                disabled: !parent.modelData.implemented
                                onClicked: {
                                    if (!parent.modelData.implemented)
                                        return;
                                    accountModal.selectedProvider = parent.modelData.id;
                                    accountModal.wizardStep = 0;
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    Component {
        id: oauthWizard

        DankFlickable {
            clip: true
            contentWidth: width
            contentHeight: wizardColumn.implicitHeight + Theme.spacingL * 2

            Column {
                id: wizardColumn
                width: parent.width
                spacing: Theme.spacingL

                Row {
                    spacing: Theme.spacingXS
                    width: parent.width

                    Repeater {
                        model: accountModal.doneStep + 1
                        Rectangle {
                            required property int index
                            width: (wizardColumn.width - Theme.spacingXS * accountModal.doneStep) / (accountModal.doneStep + 1)
                            height: 4
                            radius: 2
                            color: index <= accountModal.wizardStep ? Theme.primary : Theme.outlineLight
                        }
                    }
                }

                StyledText {
                    text: {
                        const step = accountModal.wizardStep;
                        switch (step) {
                        case 0:
                            return I18n.tr("Sign in with %1", "oauth wizard intro step heading").arg(accountModal.providerName(accountModal.selectedProvider));
                        case accountModal.credsStep:
                            return accountModal.isMicrosoft ? I18n.tr("Enter your client ID", "microsoft credentials step heading") : I18n.tr("Paste your credentials", "google credentials step heading");
                        case accountModal.browserStepIndex:
                            return I18n.tr("Authorize in your browser", "browser authorization step heading");
                        case accountModal.doneStep:
                            return I18n.tr("Account connected", "wizard success step heading");
                        }
                        if (step >= 1 && step <= accountModal.guideCount) {
                            const guide = accountModal.setupSteps[step - 1];
                            return guide && guide.title ? I18n.tr("Step %1: %2", "setup guide step heading with title").arg(step).arg(guide.title) : I18n.tr("Step %1", "setup guide step heading without title").arg(step);
                        }
                        return "";
                    }
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.Medium
                    color: Theme.surfaceText
                }

                Loader {
                    width: parent.width
                    sourceComponent: {
                        const step = accountModal.wizardStep;
                        switch (step) {
                        case 0:
                            return introStep;
                        case accountModal.credsStep:
                            return credentialsStep;
                        case accountModal.browserStepIndex:
                            return browserStep;
                        case accountModal.doneStep:
                            return successStep;
                        }
                        if (step >= 1 && step <= accountModal.guideCount)
                            return guideStep;
                        return null;
                    }
                }
            }
        }
    }

    Component {
        id: introStep
        Column {
            spacing: Theme.spacingM
            width: parent.width

            StyledText {
                text: accountModal.isMicrosoft ? I18n.tr("Microsoft requires you to register your own app to read calendar data on your behalf. This is a one-time setup that takes about two minutes — no client secret needed.", "microsoft oauth setup intro text") : I18n.tr("Google requires you to create your own OAuth client to read calendar data on your behalf. This is a one-time setup that takes about three minutes.", "google oauth setup intro text")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceText
                width: parent.width
                wrapMode: Text.WordWrap
            }

            StyledRect {
                width: parent.width
                height: childrenRect.height + Theme.spacingM * 2
                color: Theme.withAlpha(Theme.info, 0.10)
                radius: Theme.cornerRadius

                Column {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.margins: Theme.spacingM
                    spacing: Theme.spacingXS

                    Row {
                        spacing: Theme.spacingS
                        DankIcon {
                            name: "lock"
                            size: Theme.iconSize - 6
                            color: Theme.info
                            anchors.verticalCenter: parent.verticalCenter
                        }
                        StyledText {
                            text: I18n.tr("Credentials stay on this machine", "security note heading in oauth intro")
                            font.pixelSize: Theme.fontSizeMedium
                            font.weight: Font.Medium
                            color: Theme.surfaceText
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }
                    StyledText {
                        text: I18n.tr("Your client ID, secret, and refresh token are stored in the system keyring (libsecret/kwallet) when available, or an encrypted local file.", "security note body in oauth intro")
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceVariantText
                        width: parent.width
                        wrapMode: Text.WordWrap
                    }
                }
            }

            DankButton {
                text: I18n.tr("Begin setup", "button to start oauth setup guide")
                iconName: "arrow_forward"
                buttonHeight: 44
                backgroundColor: Theme.primary
                textColor: Theme.primaryText
                onClicked: accountModal.wizardStep = 1
            }
        }
    }

    Component {
        id: guideStep
        Column {
            id: guideColumn
            spacing: Theme.spacingM
            width: parent.width

            property var step: {
                if (!accountModal.setupSteps || accountModal.setupSteps.length === 0)
                    return null;
                return accountModal.setupSteps[accountModal.wizardStep - 1];
            }

            StyledText {
                visible: guideColumn.step !== null
                text: guideColumn.step ? guideColumn.step.description : ""
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceText
                width: parent.width
                wrapMode: Text.WordWrap
            }

            StyledText {
                visible: guideColumn.step === null
                text: I18n.tr("Loading setup instructions…", "placeholder while setup guide steps load")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceVariantText
            }

            StyledRect {
                visible: !!(guideColumn.step && guideColumn.step.note)
                width: parent.width
                height: visible ? noteRow.implicitHeight + Theme.spacingM * 2 : 0
                color: Theme.withAlpha(Theme.info, 0.10)
                radius: Theme.cornerRadius

                Row {
                    id: noteRow
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.leftMargin: Theme.spacingM
                    anchors.rightMargin: Theme.spacingM
                    spacing: Theme.spacingS

                    DankIcon {
                        name: "info"
                        size: Theme.iconSize - 4
                        color: Theme.info
                    }

                    StyledText {
                        width: parent.width - (Theme.iconSize - 4) - Theme.spacingS
                        text: guideColumn.step && guideColumn.step.note ? guideColumn.step.note : ""
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceText
                        wrapMode: Text.WordWrap
                    }
                }
            }

            // Drop matching PNGs into quickshell/assets/google-setup/ or
            // microsoft-setup/ to render a thumbnail under the step's
            // description (click to enlarge). The frame is hidden until the
            // image loads, so missing files degrade silently.
            StyledRect {
                visible: screenshot.status === Image.Ready
                width: parent.width
                height: visible ? 180 : 0
                color: Theme.surfaceContainer
                radius: Theme.cornerRadius
                clip: true

                Image {
                    id: screenshot
                    anchors.fill: parent
                    anchors.margins: Theme.spacingS
                    source: guideColumn.step && guideColumn.step.screenshot ? Qt.resolvedUrl(accountModal.screenshotDir + guideColumn.step.screenshot) : ""
                    fillMode: Image.PreserveAspectFit
                    asynchronous: true
                    sourceSize.width: 1024
                }

                Rectangle {
                    anchors.right: parent.right
                    anchors.bottom: parent.bottom
                    anchors.margins: Theme.spacingS
                    width: zoomHint.implicitWidth + Theme.spacingM
                    height: 24
                    radius: 12
                    color: Qt.rgba(0, 0, 0, 0.6)

                    Row {
                        id: zoomHint
                        anchors.centerIn: parent
                        spacing: 4

                        DankIcon {
                            name: "zoom_in"
                            size: 14
                            color: "white"
                            anchors.verticalCenter: parent.verticalCenter
                        }

                        StyledText {
                            text: I18n.tr("Enlarge", "hint on setup screenshot thumbnail")
                            font.pixelSize: 11
                            color: "white"
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }
                }

                StateLayer {
                    stateColor: Theme.primary
                    cornerRadius: parent.radius
                    onClicked: accountModal.expandedScreenshot = screenshot.source.toString()
                }
            }

            Row {
                visible: !!(guideColumn.step && guideColumn.step.url)
                spacing: Theme.spacingS

                DankButton {
                    text: (guideColumn.step && guideColumn.step.urlLabel) || I18n.tr("Open", "fallback label for guide step link button")
                    iconName: "open_in_new"
                    buttonHeight: 44
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    onClicked: {
                        if (guideColumn.step && guideColumn.step.url)
                            Qt.openUrlExternally(guideColumn.step.url);
                    }
                }

                DankButton {
                    text: I18n.tr("Copy link", "button to copy a setup guide url to the clipboard")
                    iconName: "content_copy"
                    buttonHeight: 44
                    backgroundColor: Theme.surfaceContainer
                    textColor: Theme.surfaceText
                    onClicked: {
                        if (guideColumn.step && guideColumn.step.url)
                            Quickshell.clipboardText = guideColumn.step.url;
                    }
                }
            }

            Row {
                width: parent.width
                spacing: Theme.spacingS
                layoutDirection: Qt.RightToLeft

                DankButton {
                    text: accountModal.wizardStep === accountModal.guideCount ? (accountModal.isMicrosoft ? I18n.tr("Enter client ID", "last guide step button to microsoft credentials") : I18n.tr("Paste credentials", "last guide step button to google credentials")) : I18n.tr("Continue", "guide step next button")
                    iconName: "arrow_forward"
                    buttonHeight: 40
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    onClicked: accountModal.wizardStep += 1
                }

                DankButton {
                    text: I18n.tr("Back", "back button in account setup wizard")
                    buttonHeight: 40
                    backgroundColor: "transparent"
                    textColor: Theme.surfaceText
                    onClicked: accountModal.wizardStep -= 1
                }
            }
        }
    }

    Component {
        id: credentialsStep

        Column {
            spacing: Theme.spacingM
            width: parent.width

            StyledText {
                text: accountModal.isMicrosoft ? I18n.tr("Paste the Application (client) ID from your app registration's Overview page.", "microsoft credentials step instructions") : I18n.tr("Import the client_secret_….json you downloaded when creating the client, or paste the Client ID and Secret manually.", "google credentials step instructions")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceText
                width: parent.width
                wrapMode: Text.WordWrap
            }

            DankButton {
                visible: !accountModal.isMicrosoft
                text: I18n.tr("Import client_secret.json", "button to import google oauth client json")
                iconName: "upload_file"
                buttonHeight: 40
                backgroundColor: Theme.surfaceContainer
                textColor: Theme.surfaceText
                onClicked: accountModal.openFilePicker({
                    "title": I18n.tr("Select client_secret.json", "file picker title for google client json"),
                    "extensions": ["*.json"]
                }, path => accountModal.importCredentialsFile(path))
            }

            DankTextField {
                width: parent.width
                label: I18n.tr("Client ID", "oauth client id field label")
                placeholderText: accountModal.isMicrosoft ? "00000000-0000-0000-0000-000000000000" : "xxxxxxxxxxxx.apps.googleusercontent.com"
                iconName: "key"
                onTextChanged: accountModal.clientId = text

                Binding on text {
                    value: accountModal.clientId
                }
            }

            DankTextField {
                visible: !accountModal.isMicrosoft
                width: parent.width
                label: I18n.tr("Client Secret", "oauth client secret field label")
                placeholderText: "GOCSPX-…"
                iconName: "lock"
                echoMode: TextInput.Password
                onTextChanged: accountModal.clientSecret = text

                Binding on text {
                    value: accountModal.clientSecret
                }
            }

            StyledText {
                visible: accountModal.flowError !== ""
                text: accountModal.flowError
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.error
                width: parent.width
                wrapMode: Text.WordWrap
            }

            Row {
                width: parent.width
                spacing: Theme.spacingS
                layoutDirection: Qt.RightToLeft

                DankButton {
                    text: accountModal.flowInProgress ? I18n.tr("Starting…", "connect button while oauth flow starts") : I18n.tr("Connect", "account setup button to start provider sign-in")
                    iconName: "check"
                    buttonHeight: 40
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    enabled: !accountModal.flowInProgress && accountModal.clientId.length > 0 && (accountModal.isMicrosoft || accountModal.clientSecret.length > 0)
                    onClicked: accountModal.startOAuthFlow()
                }

                DankButton {
                    text: I18n.tr("Back", "back button in account setup wizard")
                    buttonHeight: 40
                    backgroundColor: "transparent"
                    textColor: Theme.surfaceText
                    enabled: !accountModal.flowInProgress
                    onClicked: accountModal.wizardStep = accountModal.guideCount
                }
            }
        }
    }

    Component {
        id: browserStep

        Column {
            spacing: Theme.spacingM
            width: parent.width

            Row {
                spacing: Theme.spacingM

                BusyIndicator {
                    width: 24
                    height: 24
                    running: accountModal.flowInProgress
                }

                StyledText {
                    anchors.verticalCenter: parent.verticalCenter
                    text: accountModal.flowError !== "" ? I18n.tr("Authorization failed", "browser step heading on oauth error") : I18n.tr("Waiting for browser…", "browser step heading while waiting for oauth")
                    font.pixelSize: Theme.fontSizeMedium
                    font.weight: Font.Medium
                    color: accountModal.flowError !== "" ? Theme.error : Theme.surfaceText
                }
            }

            StyledText {
                visible: accountModal.flowError === ""
                text: I18n.tr("Approve access in your browser. The window will return here automatically when %1 sends us back the authorization code.", "browser step body while waiting for oauth").arg(accountModal.providerName(accountModal.selectedProvider))
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceVariantText
                width: parent.width
                wrapMode: Text.WordWrap
            }

            StyledText {
                visible: accountModal.flowError !== ""
                text: accountModal.flowError
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.error
                width: parent.width
                wrapMode: Text.WordWrap
            }

            StyledText {
                visible: accountModal.flowError.indexOf("server_error") !== -1
                text: I18n.tr("Microsoft often returns server_error for the first few minutes after an app registration is created or changed. Wait a minute or two and try again.", "hint about microsoft server_error during oauth")
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.surfaceVariantText
                width: parent.width
                wrapMode: Text.WordWrap
            }

            DankButton {
                visible: accountModal.flowError === "" && accountModal.pendingAuthUrl !== ""
                text: I18n.tr("Open browser again", "button to reopen oauth url in browser")
                iconName: "open_in_new"
                buttonHeight: 40
                backgroundColor: Theme.surfaceContainer
                textColor: Theme.surfaceText
                onClicked: Qt.openUrlExternally(accountModal.pendingAuthUrl)
            }

            Row {
                width: parent.width
                spacing: Theme.spacingS
                layoutDirection: Qt.RightToLeft

                DankButton {
                    visible: accountModal.flowError !== ""
                    text: I18n.tr("Try again", "button to retry oauth after failure")
                    iconName: "refresh"
                    buttonHeight: 40
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    onClicked: {
                        accountModal.cancelPendingFlow();
                        accountModal.flowError = "";
                        accountModal.wizardStep = accountModal.credsStep;
                    }
                }

                DankButton {
                    text: I18n.tr("Cancel", "button to cancel oauth browser step")
                    buttonHeight: 40
                    backgroundColor: "transparent"
                    textColor: Theme.surfaceText
                    onClicked: {
                        accountModal.cancelPendingFlow();
                        accountModal.wizardStep = accountModal.credsStep;
                    }
                }
            }
        }
    }

    Component {
        id: successStep

        Column {
            spacing: Theme.spacingM
            width: parent.width

            Row {
                spacing: Theme.spacingM

                Rectangle {
                    width: 40
                    height: 40
                    radius: 20
                    color: Theme.withAlpha(Theme.success, 0.18)

                    DankIcon {
                        anchors.centerIn: parent
                        name: "check"
                        size: Theme.iconSize - 4
                        color: Theme.success
                    }
                }

                Column {
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: 2

                    StyledText {
                        text: I18n.tr("Connected", "success step status heading")
                        font.pixelSize: Theme.fontSizeLarge
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                    }

                    StyledText {
                        text: accountModal.completedEmail
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.surfaceVariantText
                    }
                }
            }

            StyledText {
                text: I18n.tr("Your calendars will sync in the background. You can manage this account from Settings → Accounts.", "success step body about background sync")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceText
                width: parent.width
                wrapMode: Text.WordWrap
            }

            Row {
                width: parent.width
                spacing: Theme.spacingS
                layoutDirection: Qt.RightToLeft

                DankButton {
                    text: I18n.tr("Done", "button to close modal after account added")
                    iconName: "check"
                    buttonHeight: 40
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    onClicked: accountModal.hide()
                }
            }
        }
    }

    Component {
        id: caldavForm

        DankFlickable {
            id: caldavRoot
            clip: true
            contentWidth: width
            contentHeight: caldavColumn.implicitHeight + Theme.spacingL * 2

            readonly property bool isICloud: accountModal.selectedProvider === "icloud"
            property string serverUrl: isICloud ? "https://caldav.icloud.com" : ""
            property string username: ""
            property string password: ""
            property string displayName: ""
            property bool busy: false
            property bool done: false
            property string error: ""

            Column {
                id: caldavColumn
                width: parent.width
                spacing: Theme.spacingM
                visible: !caldavRoot.done

                StyledText {
                    text: caldavRoot.isICloud ? I18n.tr("Sign in with your Apple ID and an app-specific password. Regular Apple ID passwords will not work.", "icloud sign-in form instructions") : I18n.tr("Enter your CalDAV server details. The connection is verified before the account is saved.", "caldav sign-in form instructions")
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.surfaceText
                    width: parent.width
                    wrapMode: Text.WordWrap
                }

                Row {
                    visible: caldavRoot.isICloud
                    spacing: Theme.spacingS

                    DankButton {
                        text: I18n.tr("Generate app-specific password", "button to open apple app password page")
                        iconName: "open_in_new"
                        buttonHeight: 40
                        backgroundColor: Theme.surfaceContainer
                        textColor: Theme.surfaceText
                        onClicked: Qt.openUrlExternally("https://account.apple.com/account/manage")
                    }

                    DankButton {
                        text: I18n.tr("Copy link", "button to copy a setup guide url to the clipboard")
                        iconName: "content_copy"
                        buttonHeight: 40
                        backgroundColor: Theme.surfaceContainer
                        textColor: Theme.surfaceText
                        onClicked: Quickshell.clipboardText = "https://account.apple.com/account/manage"
                    }
                }

                DankTextField {
                    visible: !caldavRoot.isICloud
                    width: parent.width
                    label: I18n.tr("Server URL", "caldav server url field label")
                    placeholderText: "https://dav.example.com"
                    iconName: "cloud"
                    text: caldavRoot.serverUrl
                    onTextChanged: caldavRoot.serverUrl = text
                }

                DankTextField {
                    width: parent.width
                    label: caldavRoot.isICloud ? I18n.tr("Apple ID", "icloud username field label") : I18n.tr("Username", "caldav username field label")
                    placeholderText: caldavRoot.isICloud ? "you@icloud.com" : I18n.tr("username", "caldav username field placeholder")
                    iconName: "person"
                    text: caldavRoot.username
                    onTextChanged: caldavRoot.username = text
                }

                DankTextField {
                    width: parent.width
                    label: caldavRoot.isICloud ? I18n.tr("App-specific password", "icloud password field label") : I18n.tr("Password", "caldav password field label")
                    placeholderText: caldavRoot.isICloud ? "xxxx-xxxx-xxxx-xxxx" : I18n.tr("password", "caldav password field placeholder")
                    iconName: "lock"
                    echoMode: TextInput.Password
                    text: caldavRoot.password
                    onTextChanged: caldavRoot.password = text
                }

                DankTextField {
                    width: parent.width
                    label: I18n.tr("Display name (optional)", "optional account display name field label")
                    placeholderText: caldavRoot.isICloud ? "iCloud" : I18n.tr("My server", "caldav display name field placeholder")
                    iconName: "badge"
                    text: caldavRoot.displayName
                    onTextChanged: caldavRoot.displayName = text
                }

                StyledText {
                    visible: caldavRoot.error !== ""
                    text: caldavRoot.error
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.error
                    width: parent.width
                    wrapMode: Text.WordWrap
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingS
                    layoutDirection: Qt.RightToLeft

                    DankButton {
                        text: caldavRoot.busy ? I18n.tr("Connecting…", "connect button while caldav connection verifies") : I18n.tr("Connect", "account setup button to start provider sign-in")
                        iconName: "check"
                        buttonHeight: 40
                        backgroundColor: Theme.primary
                        textColor: Theme.primaryText
                        enabled: !caldavRoot.busy && caldavRoot.serverUrl.length > 0 && caldavRoot.username.length > 0 && caldavRoot.password.length > 0
                        onClicked: {
                            caldavRoot.busy = true;
                            caldavRoot.error = "";
                            DankCalService.addCalDAVAccount(caldavRoot.serverUrl, caldavRoot.username, caldavRoot.password, caldavRoot.displayName, response => {
                                caldavRoot.busy = false;
                                if (response.error) {
                                    caldavRoot.error = response.error;
                                    return;
                                }
                                const result = response.result || {};
                                accountModal.completedAccountId = result.accountId || "";
                                accountModal.completedEmail = result.displayName || result.accountId || "";
                                caldavRoot.done = true;
                            });
                        }
                    }
                }
            }

            Loader {
                width: parent.width
                visible: caldavRoot.done
                sourceComponent: caldavRoot.done ? successStep : null
            }
        }
    }

    Component {
        id: localForm

        DankFlickable {
            id: localRoot
            clip: true
            contentWidth: width
            contentHeight: localColumn.implicitHeight + Theme.spacingL * 2

            property string rootPath: ""
            property string displayName: ""
            property bool busy: false
            property bool done: false
            property string error: ""

            Column {
                id: localColumn
                width: parent.width
                spacing: Theme.spacingM
                visible: !localRoot.done

                StyledText {
                    text: I18n.tr("Point at a directory of .ics files. Each subdirectory or .ics file becomes a calendar — compatible with vdirsyncer/khal layouts. The directory is created if it doesn't exist.", "local calendar directory form instructions")
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.surfaceText
                    width: parent.width
                    wrapMode: Text.WordWrap
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingS

                    DankTextField {
                        id: localDirField
                        width: parent.width - 44 - Theme.spacingS
                        label: I18n.tr("Directory", "local calendar directory field label")
                        placeholderText: "/home/you/.local/share/calendars"
                        iconName: "folder"
                        text: localRoot.rootPath
                        onTextChanged: localRoot.rootPath = text
                    }

                    DankActionButton {
                        circular: false
                        iconName: "folder_open"
                        iconColor: Theme.surfaceText
                        anchors.verticalCenter: localDirField.verticalCenter
                        onClicked: accountModal.openFilePicker({
                            "title": I18n.tr("Choose calendar directory", "folder picker title for local calendar directory"),
                            "folderMode": true
                        }, path => localDirField.text = path)
                    }
                }

                DankTextField {
                    width: parent.width
                    label: I18n.tr("Display name (optional)", "optional account display name field label")
                    placeholderText: I18n.tr("Local calendars", "local account display name field placeholder")
                    iconName: "badge"
                    text: localRoot.displayName
                    onTextChanged: localRoot.displayName = text
                }

                StyledText {
                    visible: localRoot.error !== ""
                    text: localRoot.error
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.error
                    width: parent.width
                    wrapMode: Text.WordWrap
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingS
                    layoutDirection: Qt.RightToLeft

                    DankButton {
                        text: localRoot.busy ? I18n.tr("Adding…", "add button while local account is created") : I18n.tr("Add", "button to add local calendar account")
                        iconName: "check"
                        buttonHeight: 40
                        backgroundColor: Theme.primary
                        textColor: Theme.primaryText
                        enabled: !localRoot.busy && localRoot.rootPath.length > 0
                        onClicked: {
                            localRoot.busy = true;
                            localRoot.error = "";
                            DankCalService.addLocalAccount(localRoot.rootPath, localRoot.displayName, response => {
                                localRoot.busy = false;
                                if (response.error) {
                                    localRoot.error = response.error;
                                    return;
                                }
                                const result = response.result || {};
                                accountModal.completedAccountId = result.accountId || "";
                                accountModal.completedEmail = result.displayName || result.accountId || "";
                                localRoot.done = true;
                            });
                        }
                    }
                }
            }

            Loader {
                width: parent.width
                visible: localRoot.done
                sourceComponent: localRoot.done ? successStep : null
            }
        }
    }
}
