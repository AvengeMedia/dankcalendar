import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: root

    property int currentIndex: 0

    signal addAccountRequested

    readonly property var weekStartOptions: [
        {
            label: I18n.tr("System default (%1)", "week start dropdown option").arg(SettingsData.dayName(SettingsData.localeFirstDayOfWeek, Locale.LongFormat)),
            value: -1
        },
        {
            label: SettingsData.dayName(0, Locale.LongFormat),
            value: 0
        },
        {
            label: SettingsData.dayName(1, Locale.LongFormat),
            value: 1
        },
        {
            label: SettingsData.dayName(6, Locale.LongFormat),
            value: 6
        }
    ]

    readonly property var durationOptions: [
        {
            label: I18n.tr("15 minutes", "default event duration option"),
            value: 15
        },
        {
            label: I18n.tr("30 minutes", "default event duration option"),
            value: 30
        },
        {
            label: I18n.tr("45 minutes", "default event duration option"),
            value: 45
        },
        {
            label: I18n.tr("1 hour", "default event duration option"),
            value: 60
        },
        {
            label: I18n.tr("1.5 hours", "default event duration option"),
            value: 90
        },
        {
            label: I18n.tr("2 hours", "default event duration option"),
            value: 120
        }
    ]

    readonly property var reminderOptions: [
        {
            label: I18n.tr("None", "default reminder dropdown option"),
            value: -1
        },
        {
            label: I18n.tr("At start", "default reminder dropdown option"),
            value: 0
        },
        {
            label: I18n.tr("5 minutes before", "default reminder dropdown option"),
            value: 5
        },
        {
            label: I18n.tr("10 minutes before", "default reminder dropdown option"),
            value: 10
        },
        {
            label: I18n.tr("15 minutes before", "default reminder dropdown option"),
            value: 15
        },
        {
            label: I18n.tr("30 minutes before", "default reminder dropdown option"),
            value: 30
        },
        {
            label: I18n.tr("1 hour before", "default reminder dropdown option"),
            value: 60
        }
    ]

    readonly property var snoozeOptions: [
        {
            label: I18n.tr("5 minutes", "snooze duration dropdown option"),
            value: 5
        },
        {
            label: I18n.tr("10 minutes", "snooze duration dropdown option"),
            value: 10
        },
        {
            label: I18n.tr("15 minutes", "snooze duration dropdown option"),
            value: 15
        },
        {
            label: I18n.tr("30 minutes", "snooze duration dropdown option"),
            value: 30
        }
    ]

    readonly property var allDayDayOptions: [
        {
            label: I18n.tr("On the day", "all-day reminder day dropdown option"),
            value: 0
        },
        {
            label: I18n.tr("1 day before", "all-day reminder day dropdown option"),
            value: 1
        },
        {
            label: I18n.tr("2 days before", "all-day reminder day dropdown option"),
            value: 2
        },
        {
            label: I18n.tr("1 week before", "all-day reminder day dropdown option"),
            value: 7
        }
    ]

    function minutesFromClock(text) {
        const parts = (text || "").split(":");
        if (parts.length !== 2)
            return 540;
        const total = parseInt(parts[0], 10) * 60 + parseInt(parts[1], 10);
        return isNaN(total) ? 540 : total;
    }

    function clockFromMinutes(minutes) {
        const h = Math.floor(minutes / 60);
        const m = minutes % 60;
        return (h < 10 ? "0" : "") + h + ":" + (m < 10 ? "0" : "") + m;
    }

    function optionLabels(options) {
        return options.map(o => o.label);
    }

    function labelForValue(options, value) {
        for (let i = 0; i < options.length; i++) {
            if (options[i].value === value)
                return options[i].label;
        }
        return options[0].label;
    }

    function valueForLabel(options, label) {
        for (let i = 0; i < options.length; i++) {
            if (options[i].label === label)
                return options[i].value;
        }
        return options[0].value;
    }

    Loader {
        anchors.fill: parent
        sourceComponent: {
            switch (root.currentIndex) {
            case 0:
                return generalPage;
            case 1:
                return appearancePage;
            case 2:
                return calendarsPage;
            case 3:
                return accountsPage;
            case 4:
                return notificationsPage;
            case 5:
                return aboutPage;
            default:
                return generalPage;
            }
        }
    }

    component SectionHeader: Column {
        property string title: ""
        property string subtitle: ""

        width: parent.width
        spacing: Theme.spacingXS

        StyledText {
            text: parent.title
            font.pixelSize: Theme.fontSizeXLarge
            font.weight: Font.Medium
            color: Theme.surfaceText
        }

        StyledText {
            visible: text !== ""
            text: parent.subtitle
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceVariantText
        }
    }

    component SettingsRow: StyledRect {
        id: settingsRow
        property string label: ""
        property string description: ""
        default property alias trailingContent: trailing.data

        width: parent.width
        height: 64
        color: Theme.surfaceContainer
        radius: Theme.cornerRadius

        Column {
            anchors.left: parent.left
            anchors.leftMargin: Theme.spacingL
            anchors.right: trailing.left
            anchors.rightMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            spacing: 2

            StyledText {
                text: settingsRow.label
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.Medium
                color: Theme.surfaceText
                width: parent.width
                elide: Text.ElideRight
            }

            StyledText {
                visible: text !== ""
                text: settingsRow.description
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.surfaceVariantText
                width: parent.width
                elide: Text.ElideRight
            }
        }

        Row {
            id: trailing
            anchors.right: parent.right
            anchors.rightMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS
        }
    }

    Component {
        id: generalPage
        DankFlickable {
            clip: true
            contentWidth: width
            contentHeight: generalColumn.implicitHeight

            Column {
                id: generalColumn
                width: parent.width
                padding: Theme.spacingL
                spacing: Theme.spacingL

                SectionHeader {
                    title: I18n.tr("General", "general settings section header")
                    subtitle: I18n.tr("Defaults follow your locale unless overridden.", "general settings section subtitle")
                }

                Column {
                    width: parent.width - Theme.spacingL * 2
                    spacing: Theme.spacingS

                    SettingsRow {
                        label: I18n.tr("Start at login", "autostart setting label")
                        description: I18n.tr("Launch Dank Calendar in the background when you log in.", "autostart setting description")

                        DankToggle {
                            anchors.verticalCenter: parent.verticalCenter
                            checked: DankCalService.autostartEnabled
                            toggleEnabled: DankCalService.connected
                            onToggled: checked => DankCalService.setAutostart(checked)
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Close behavior", "close behavior setting label")
                        description: I18n.tr("What the window's close button does.", "close behavior setting description")

                        DankButtonGroup {
                            anchors.verticalCenter: parent.verticalCenter
                            buttonHeight: 36
                            minButtonWidth: 90
                            model: [I18n.tr("Minimize", "close behavior button group option to hide to tray"), I18n.tr("Quit", "close behavior button group option to quit the app")]
                            currentIndex: SettingsData.closeBehavior === "quit" ? 1 : 0
                            onSelectionChanged: (index, selected) => {
                                if (!selected)
                                    return;
                                SettingsData.closeBehavior = index === 1 ? "quit" : "minimize";
                            }
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Start week on", "week start setting label")
                        description: I18n.tr("First day shown in week and month views.", "week start setting description")

                        DankDropdown {
                            anchors.verticalCenter: parent.verticalCenter
                            dropdownWidth: 220
                            options: root.optionLabels(root.weekStartOptions)
                            currentValue: root.labelForValue(root.weekStartOptions, SettingsData.firstDayOfWeek)
                            onValueChanged: value => SettingsData.firstDayOfWeek = root.valueForLabel(root.weekStartOptions, value)
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Time format", "time format setting label")
                        description: I18n.tr("Auto follows your locale (%1).", "time format setting description").arg(SettingsData.localeUses24Hour ? I18n.tr("24-hour", "locale time format name in time format description") : I18n.tr("12-hour", "locale time format name in time format description"))

                        DankButtonGroup {
                            anchors.verticalCenter: parent.verticalCenter
                            buttonHeight: 36
                            minButtonWidth: 56
                            model: [I18n.tr("Auto", "time format button group option"), I18n.tr("12h", "time format button group option"), I18n.tr("24h", "time format button group option")]
                            currentIndex: {
                                switch (SettingsData.timeFormat) {
                                case "12h":
                                    return 1;
                                case "24h":
                                    return 2;
                                default:
                                    return 0;
                                }
                            }
                            onSelectionChanged: (index, selected) => {
                                if (!selected)
                                    return;
                                switch (index) {
                                case 1:
                                    SettingsData.timeFormat = "12h";
                                    break;
                                case 2:
                                    SettingsData.timeFormat = "24h";
                                    break;
                                default:
                                    SettingsData.timeFormat = "auto";
                                    break;
                                }
                            }
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Show week numbers", "week numbers setting label")
                        description: I18n.tr("Display ISO week numbers in month view.", "week numbers setting description")

                        DankToggle {
                            anchors.verticalCenter: parent.verticalCenter
                            checked: SettingsData.showWeekNumbers
                            onToggled: checked => SettingsData.showWeekNumbers = checked
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Default event duration", "default event duration setting label")
                        description: I18n.tr("Length used when creating events.", "default event duration setting description")

                        DankDropdown {
                            anchors.verticalCenter: parent.verticalCenter
                            dropdownWidth: 150
                            options: root.optionLabels(root.durationOptions)
                            currentValue: root.labelForValue(root.durationOptions, SettingsData.defaultEventDurationMinutes)
                            onValueChanged: value => SettingsData.defaultEventDurationMinutes = root.valueForLabel(root.durationOptions, value)
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Default reminder", "default reminder setting label")
                        description: I18n.tr("Initial reminder set on new events.", "default reminder setting description")

                        DankDropdown {
                            anchors.verticalCenter: parent.verticalCenter
                            dropdownWidth: 180
                            options: root.optionLabels(root.reminderOptions)
                            currentValue: root.labelForValue(root.reminderOptions, SettingsData.defaultReminderMinutes)
                            onValueChanged: value => SettingsData.defaultReminderMinutes = root.valueForLabel(root.reminderOptions, value)
                        }
                    }
                }
            }
        }
    }

    Component {
        id: appearancePage
        DankFlickable {
            clip: true
            contentWidth: width
            contentHeight: appearanceColumn.implicitHeight

            Column {
                id: appearanceColumn
                width: parent.width
                padding: Theme.spacingL
                spacing: Theme.spacingL

                SectionHeader {
                    title: I18n.tr("Appearance", "appearance settings section header")
                    subtitle: I18n.tr("Colors are shared with DankMaterialShell when available.", "appearance settings section subtitle")
                }

                Column {
                    width: parent.width - Theme.spacingL * 2
                    spacing: Theme.spacingS

                    SettingsRow {
                        label: I18n.tr("Theme", "theme mode setting label")
                        description: {
                            switch (SettingsData.themeMode) {
                            case "light":
                                return I18n.tr("Always use the light theme.", "theme mode setting description");
                            case "dark":
                                return I18n.tr("Always use the dark theme.", "theme mode setting description");
                            default:
                                return PortalService.available ? I18n.tr("Following the system color scheme (currently %1).", "theme mode setting description").arg(Theme.isLightMode ? I18n.tr("light", "current scheme name in theme mode description") : I18n.tr("dark", "current scheme name in theme mode description")) : I18n.tr("System preference unavailable — using dark.", "theme mode setting description");
                            }
                        }

                        DankButtonGroup {
                            anchors.verticalCenter: parent.verticalCenter
                            buttonHeight: 36
                            minButtonWidth: 56
                            model: [I18n.tr("Auto", "theme mode button group option"), I18n.tr("Light", "theme mode button group option"), I18n.tr("Dark", "theme mode button group option")]
                            currentIndex: {
                                switch (SettingsData.themeMode) {
                                case "light":
                                    return 1;
                                case "dark":
                                    return 2;
                                default:
                                    return 0;
                                }
                            }
                            onSelectionChanged: (index, selected) => {
                                if (!selected)
                                    return;
                                switch (index) {
                                case 1:
                                    SettingsData.themeMode = "light";
                                    break;
                                case 2:
                                    SettingsData.themeMode = "dark";
                                    break;
                                default:
                                    SettingsData.themeMode = "auto";
                                    break;
                                }
                            }
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Color source", "color source setting label")
                        description: I18n.tr("DMS dynamic colors detected when available.", "color source setting description")

                        StyledText {
                            anchors.verticalCenter: parent.verticalCenter
                            text: Theme.colorsLoaded ? I18n.tr("DMS dynamic", "color source status value") : I18n.tr("Stock palette", "color source status value")
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.colorsLoaded ? Theme.success : Theme.surfaceVariantText
                        }
                    }
                }
            }
        }
    }

    Component {
        id: calendarsPage
        DankFlickable {
            id: calendarsFlickable

            property var actionCalendar: null

            clip: true
            contentWidth: width
            contentHeight: calendarsColumn.implicitHeight

            Column {
                id: calendarsColumn
                width: parent.width
                padding: Theme.spacingL
                spacing: Theme.spacingL

                SectionHeader {
                    title: I18n.tr("Calendars", "calendars settings section header")
                    subtitle: I18n.tr("Visibility, names, and removal per calendar.", "calendars settings section subtitle")
                }

                Column {
                    width: parent.width - Theme.spacingL * 2
                    spacing: Theme.spacingS

                    StyledText {
                        visible: DankCalService.calendars.length === 0
                        text: DankCalService.connected ? I18n.tr("No calendars yet. Add an account first.", "calendars page empty state") : I18n.tr("Backend not connected.", "calendars page empty state")
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.surfaceVariantText
                    }

                    Repeater {
                        model: ScriptModel {
                            values: DankCalService.calendars
                        }

                        StyledRect {
                            id: calendarRow
                            required property var modelData
                            readonly property bool renamed: !!modelData.providerName && modelData.providerName !== modelData.name

                            width: parent.width
                            height: 64
                            color: Theme.surfaceContainer
                            radius: Theme.cornerRadius

                            Row {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingL
                                anchors.rightMargin: Theme.spacingM
                                spacing: Theme.spacingM

                                Rectangle {
                                    width: 18
                                    height: 18
                                    radius: 5
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: calendarRow.modelData.color
                                }

                                Column {
                                    anchors.verticalCenter: parent.verticalCenter
                                    width: parent.width - 18 - 168 - Theme.spacingM * 2

                                    StyledText {
                                        text: calendarRow.modelData.name
                                        font.pixelSize: Theme.fontSizeMedium
                                        font.weight: Font.Medium
                                        color: Theme.surfaceText
                                        elide: Text.ElideRight
                                        width: parent.width
                                    }

                                    StyledText {
                                        text: {
                                            let line = calendarRow.modelData.accountName || calendarRow.modelData.accountId || "";
                                            if (calendarRow.renamed)
                                                line += " · " + I18n.tr("synced as \"%1\"", "renamed calendar provider name suffix in calendar list").arg(calendarRow.modelData.providerName);
                                            if (calendarRow.modelData.readOnly)
                                                line += " · " + I18n.tr("read-only", "read-only suffix in calendar list");
                                            return line;
                                        }
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.surfaceVariantText
                                        elide: Text.ElideRight
                                        width: parent.width
                                    }
                                }

                                Row {
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: Theme.spacingXS

                                    DankActionButton {
                                        iconName: "edit"
                                        iconColor: Theme.surfaceText
                                        onClicked: calendarRenameDialog.show(calendarRow.modelData)
                                    }

                                    DankActionButton {
                                        iconName: !!calendarRow.modelData.reminders ? "notifications_active" : "notifications"
                                        iconColor: !!calendarRow.modelData.reminders ? Theme.primary : Theme.surfaceText
                                        onClicked: calendarRemindersDialog.show(calendarRow.modelData)
                                    }

                                    DankActionButton {
                                        iconName: "delete_outline"
                                        iconColor: Theme.error
                                        onClicked: {
                                            calendarsFlickable.actionCalendar = calendarRow.modelData;
                                            calendarDeleteConfirm.show({
                                                title: I18n.tr("Delete \"%1\"?", "delete calendar confirmation title").arg(calendarRow.modelData.name),
                                                message: I18n.tr("Removes this calendar and its events from Dank Calendar. If the provider still offers it, it will come back on the next sync.", "delete calendar confirmation message"),
                                                confirmText: I18n.tr("Delete", "delete calendar confirmation button"),
                                                danger: true
                                            });
                                        }
                                    }

                                    DankToggle {
                                        anchors.verticalCenter: parent.verticalCenter
                                        checked: !calendarRow.modelData.hidden
                                        onToggled: checked => DankCalService.setCalendarHidden(calendarRow.modelData.id, !checked)
                                    }
                                }
                            }
                        }
                    }
                }
            }

            RenameCalendarDialog {
                id: calendarRenameDialog
            }

            CalendarRemindersDialog {
                id: calendarRemindersDialog
            }

            ConfirmDialog {
                id: calendarDeleteConfirm
                onConfirmed: {
                    if (calendarsFlickable.actionCalendar)
                        DankCalService.deleteCalendar(calendarsFlickable.actionCalendar.id);
                }
            }
        }
    }

    Component {
        id: accountsPage
        DankFlickable {
            id: accountsFlickable

            property var actionAccount: null

            clip: true
            contentWidth: width
            contentHeight: accountsColumn.implicitHeight

            function providerMeta(kind) {
                switch (kind) {
                case "google":
                    return {
                        "label": "Google",
                        "icon": "mail",
                        "color": Theme.primary
                    };
                case "caldav":
                    return {
                        "label": "CalDAV",
                        "icon": "cloud",
                        "color": Theme.secondary
                    };
                case "icloud":
                    return {
                        "label": "iCloud",
                        "icon": "cloud_circle",
                        "color": Theme.info
                    };
                case "microsoft":
                    return {
                        "label": "Microsoft",
                        "icon": "business",
                        "color": Theme.warning
                    };
                case "local":
                    return {
                        "label": I18n.tr("Local", "local provider name in account list"),
                        "icon": "folder",
                        "color": Theme.success
                    };
                }
                return {
                    "label": kind,
                    "icon": "account_circle",
                    "color": Theme.primary
                };
            }

            Column {
                id: accountsColumn
                width: parent.width
                padding: Theme.spacingL
                spacing: Theme.spacingL

                SectionHeader {
                    title: I18n.tr("Accounts", "accounts settings section header")
                    subtitle: DankCalService.connected ? I18n.tr("Connected calendar providers.", "accounts settings section subtitle") : I18n.tr("Backend not connected.", "accounts settings section subtitle")
                }

                Column {
                    width: parent.width - Theme.spacingL * 2
                    spacing: Theme.spacingS

                    StyledText {
                        visible: DankCalService.accounts.length === 0
                        text: I18n.tr("No accounts yet. Click \"Add account\" to connect one.", "accounts page empty state")
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.surfaceVariantText
                    }

                    Repeater {
                        model: ScriptModel {
                            values: DankCalService.accounts
                        }

                        StyledRect {
                            id: accountRow
                            required property var modelData
                            readonly property var meta: providerMeta(DankCalService.accountFlavor(modelData))

                            width: parent.width
                            height: 80
                            color: Theme.surfaceContainer
                            radius: Theme.cornerRadius

                            Row {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingL
                                anchors.rightMargin: Theme.spacingM
                                spacing: Theme.spacingM

                                Rectangle {
                                    width: 40
                                    height: 40
                                    radius: 20
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: Theme.withAlpha(parent.parent.meta.color, 0.18)

                                    DankIcon {
                                        anchors.centerIn: parent
                                        name: parent.parent.parent.meta.icon
                                        size: Theme.iconSize - 4
                                        color: parent.parent.parent.meta.color
                                    }
                                }

                                Column {
                                    anchors.verticalCenter: parent.verticalCenter
                                    width: parent.width - 40 - 80 - Theme.spacingM * 3
                                    spacing: 2

                                    StyledText {
                                        text: parent.parent.parent.meta.label
                                        font.pixelSize: Theme.fontSizeMedium
                                        font.weight: Font.Medium
                                        color: Theme.surfaceText
                                    }

                                    StyledText {
                                        text: DankCalService.accountLabel(parent.parent.parent.modelData)
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.surfaceText
                                        elide: Text.ElideRight
                                        width: parent.width
                                    }

                                    StyledText {
                                        readonly property bool needsReauth: parent.parent.parent.modelData.needsReauth === true
                                        readonly property bool authorized: parent.parent.parent.modelData.authorized !== false
                                        text: needsReauth ? I18n.tr("Sign-in expired — reconnect to keep syncing", "account status when oauth needs re-auth") : (authorized ? I18n.tr("Connected", "account status in account list") : I18n.tr("Not authorized — remove this account and add it again", "account status in account list"))
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: (needsReauth || !authorized) ? Theme.error : Theme.surfaceVariantText
                                    }
                                }

                                Row {
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: Theme.spacingXS

                                    DankActionButton {
                                        visible: accountRow.modelData.needsReauth === true
                                        iconName: "login"
                                        iconColor: Theme.primary
                                        onClicked: DankCalService.reconnectAccount(accountRow.modelData)
                                    }

                                    DankActionButton {
                                        iconName: "refresh"
                                        iconColor: Theme.surfaceText
                                        onClicked: DankCalService.refreshAccount(parent.parent.parent.modelData.id)
                                    }

                                    DankActionButton {
                                        iconName: "delete_outline"
                                        iconColor: Theme.error
                                        onClicked: {
                                            accountsFlickable.actionAccount = accountRow.modelData;
                                            accountRemoveConfirm.show({
                                                title: I18n.tr("Remove \"%1\"?", "remove account confirmation title").arg(DankCalService.accountLabel(accountRow.modelData)),
                                                message: I18n.tr("Removes this account and its calendars and events from Dank Calendar. Nothing is deleted from the provider.", "remove account confirmation message"),
                                                confirmText: I18n.tr("Remove", "remove account confirmation button"),
                                                danger: true
                                            });
                                        }
                                    }
                                }
                            }
                        }
                    }

                    DankButton {
                        text: I18n.tr("Add account", "add account button on accounts page")
                        iconName: "add"
                        buttonHeight: 40
                        backgroundColor: Theme.primary
                        textColor: Theme.primaryText
                        onClicked: root.addAccountRequested()
                    }
                }
            }

            ConfirmDialog {
                id: accountRemoveConfirm
                onConfirmed: {
                    if (accountsFlickable.actionAccount)
                        DankCalService.removeAccount(accountsFlickable.actionAccount.id);
                }
            }
        }
    }

    Component {
        id: notificationsPage
        DankFlickable {
            clip: true
            contentWidth: width
            contentHeight: notificationsColumn.implicitHeight

            Column {
                id: notificationsColumn
                width: parent.width
                padding: Theme.spacingL
                spacing: Theme.spacingL

                SectionHeader {
                    title: I18n.tr("Notifications", "notifications settings section header")
                    subtitle: I18n.tr("Reminders and desktop alerts.", "notifications settings section subtitle")
                }

                Column {
                    width: parent.width - Theme.spacingL * 2
                    spacing: Theme.spacingS

                    SettingsRow {
                        label: I18n.tr("Enable reminders", "reminders toggle label")
                        description: I18n.tr("Desktop notifications for upcoming events.", "reminders toggle description")

                        DankToggle {
                            anchors.verticalCenter: parent.verticalCenter
                            checked: SettingsData.remindersEnabled
                            onToggled: checked => SettingsData.remindersEnabled = checked
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Keep until dismissed", "persistent reminders toggle label")
                        description: I18n.tr("Reminders stay on screen until you act on them.", "persistent reminders toggle description")

                        DankToggle {
                            anchors.verticalCenter: parent.verticalCenter
                            checked: SettingsData.reminderPersist
                            toggleEnabled: SettingsData.remindersEnabled
                            onToggled: checked => SettingsData.reminderPersist = checked
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Snooze duration", "snooze duration setting label")
                        description: I18n.tr("How long the snooze button postpones a reminder.", "snooze duration setting description")

                        DankDropdown {
                            anchors.verticalCenter: parent.verticalCenter
                            dropdownWidth: 150
                            enabled: SettingsData.remindersEnabled
                            opacity: enabled ? 1 : 0.5
                            options: root.optionLabels(root.snoozeOptions)
                            currentValue: root.labelForValue(root.snoozeOptions, SettingsData.snoozeMinutes)
                            onValueChanged: value => SettingsData.snoozeMinutes = root.valueForLabel(root.snoozeOptions, value)
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Show all-day reminders", "all-day reminders toggle label")
                        description: I18n.tr("Notify for all-day events without their own reminders.", "all-day reminders toggle description")

                        DankToggle {
                            anchors.verticalCenter: parent.verticalCenter
                            checked: SettingsData.allDayReminders
                            toggleEnabled: SettingsData.remindersEnabled
                            onToggled: checked => SettingsData.allDayReminders = checked
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("All-day reminder time", "all-day reminder time setting label")
                        description: I18n.tr("When all-day event reminders fire.", "all-day reminder time setting description")

                        DankDropdown {
                            anchors.verticalCenter: parent.verticalCenter
                            dropdownWidth: 160
                            enabled: SettingsData.remindersEnabled && SettingsData.allDayReminders
                            opacity: enabled ? 1 : 0.5
                            options: root.optionLabels(root.allDayDayOptions)
                            currentValue: root.labelForValue(root.allDayDayOptions, SettingsData.allDayReminderDaysBefore)
                            onValueChanged: value => SettingsData.allDayReminderDaysBefore = root.valueForLabel(root.allDayDayOptions, value)
                        }

                        DankTimePicker {
                            anchors.verticalCenter: parent.verticalCenter
                            enabled: SettingsData.remindersEnabled && SettingsData.allDayReminders
                            opacity: enabled ? 1 : 0.5
                            use24Hour: SettingsData.use24HourTime
                            minutes: root.minutesFromClock(SettingsData.allDayReminderTime)
                            onTimeSelected: value => SettingsData.allDayReminderTime = root.clockFromMinutes(value)
                        }
                    }

                    SettingsRow {
                        label: I18n.tr("Test notification", "test notification setting label")
                        description: DankCalService.connected ? I18n.tr("Verify desktop notifications are working.", "test notification setting description") : I18n.tr("Backend not connected.", "test notification setting description when backend is unavailable")

                        DankButton {
                            anchors.verticalCenter: parent.verticalCenter
                            text: I18n.tr("Send test", "test notification button")
                            buttonHeight: 36
                            enabled: DankCalService.connected
                            backgroundColor: Theme.primary
                            textColor: Theme.primaryText
                            onClicked: DankCalService.sendTestReminder()
                        }
                    }
                }
            }
        }
    }

    Component {
        id: aboutPage
        SettingsAboutPage {}
    }
}
