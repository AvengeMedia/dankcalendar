import QtQuick
import QtQuick.Controls
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

FloatingWindow {
    id: eventModal

    property var event: ({})
    property bool editMode: false
    property bool createMode: false
    property bool confirmDelete: false
    property bool saving: false
    property string formError: ""

    property string formTitle: ""
    property date formStartDate: new Date()
    property int formStartMinutes: 600
    property int formEndMinutes: 660
    property string formLocation: ""
    property string formDescription: ""
    property bool formAllDay: false
    property int formCalendarIndex: 0
    property int formReminderMinutes: -1

    readonly property var reminderOptions: [
        {
            label: I18n.tr("None", "event reminder dropdown option"),
            value: -1
        },
        {
            label: I18n.tr("At start", "event reminder dropdown option"),
            value: 0
        },
        {
            label: I18n.tr("5 minutes before", "event reminder dropdown option"),
            value: 5
        },
        {
            label: I18n.tr("10 minutes before", "event reminder dropdown option"),
            value: 10
        },
        {
            label: I18n.tr("15 minutes before", "event reminder dropdown option"),
            value: 15
        },
        {
            label: I18n.tr("30 minutes before", "event reminder dropdown option"),
            value: 30
        },
        {
            label: I18n.tr("1 hour before", "event reminder dropdown option"),
            value: 60
        },
        {
            label: I18n.tr("2 hours before", "event reminder dropdown option"),
            value: 120
        },
        {
            label: I18n.tr("1 day before", "event reminder dropdown option"),
            value: 1440
        },
        {
            label: I18n.tr("2 days before", "event reminder dropdown option"),
            value: 2880
        },
        {
            label: I18n.tr("1 week before", "event reminder dropdown option"),
            value: 10080
        }
    ]
    readonly property bool descriptionIsHtml: /<[a-z][^>]*>/i.test(event.description || "")

    function show(eventData) {
        createMode = false;
        editMode = false;
        confirmDelete = false;
        formError = "";
        event = eventData || {};
        visible = true;
    }

    function showCreate(day) {
        const base = day ? new Date(day) : new Date();
        base.setHours(10, 0, 0, 0);
        const end = new Date(base.getTime() + SettingsData.defaultEventDurationMinutes * 60000);
        event = {
            "title": "",
            "description": "",
            "location": "",
            "start": base,
            "end": end,
            "allDay": false,
            "attendees": [],
            "reminders": SettingsData.defaultReminderMinutes >= 0 ? [
                {
                    "method": "popup",
                    "minutes": SettingsData.defaultReminderMinutes
                }
            ] : []
        };
        createMode = true;
        confirmDelete = false;
        formError = "";
        _loadForm();
        editMode = true;
        visible = true;
    }

    function hide() {
        editMode = false;
        createMode = false;
        visible = false;
    }

    function beginEdit() {
        _loadForm();
        formError = "";
        editMode = true;
    }

    function _loadForm() {
        formTitle = event.title || "";
        formLocation = event.location || "";
        formDescription = event.description || "";
        formAllDay = !!event.allDay;
        const start = event.start ? new Date(event.start) : new Date();
        const end = event.end ? new Date(event.end) : new Date(start.getTime() + 3600000);
        formStartDate = start;
        formStartMinutes = start.getHours() * 60 + start.getMinutes();
        formEndMinutes = end.getHours() * 60 + end.getMinutes();
        if (!formAllDay && formEndMinutes <= formStartMinutes)
            formEndMinutes = formStartMinutes + 60;

        const writable = DankCalService.writableCalendars();
        formCalendarIndex = 0;
        for (let i = 0; i < writable.length; i++) {
            if (writable[i].id === event.calendarId) {
                formCalendarIndex = i;
                break;
            }
        }

        formReminderMinutes = -1;
        const reminders = event.reminders || [];
        for (let i = 0; i < reminders.length; i++) {
            if (_isPopup(reminders[i])) {
                formReminderMinutes = reminders[i].minutes;
                break;
            }
        }
    }

    function _isPopup(reminder) {
        const method = reminder.method || "popup";
        return method === "popup" || method === "display";
    }

    function reminderText(minutes) {
        if (minutes < 0)
            return "";
        if (minutes === 0)
            return I18n.tr("At start", "event reminder dropdown option");
        if (minutes % 10080 === 0)
            return I18n.tr("%1w before", "event details short reminder offset in weeks").arg(minutes / 10080);
        if (minutes % 1440 === 0)
            return I18n.tr("%1d before", "event details short reminder offset in days").arg(minutes / 1440);
        if (minutes % 60 === 0)
            return I18n.tr("%1h before", "event details short reminder offset in hours").arg(minutes / 60);
        return I18n.tr("%1m before", "event details short reminder offset in minutes").arg(minutes);
    }

    function reminderSummary() {
        const labels = (event.reminders || []).filter(r => _isPopup(r)).map(r => reminderText(r.minutes));
        return labels.join(" · ");
    }

    function reminderOptionLabel(minutes) {
        for (let i = 0; i < reminderOptions.length; i++) {
            if (reminderOptions[i].value === minutes)
                return reminderOptions[i].label;
        }
        return reminderText(minutes);
    }

    function _formRange() {
        const y = formStartDate.getFullYear();
        const mo = formStartDate.getMonth();
        const d = formStartDate.getDate();
        if (formAllDay)
            return {
                "start": new Date(y, mo, d),
                "end": new Date(y, mo, d + 1)
            };

        let endMinutes = formEndMinutes;
        if (endMinutes <= formStartMinutes)
            endMinutes = formStartMinutes + 60;
        return {
            "start": new Date(y, mo, d, Math.floor(formStartMinutes / 60), formStartMinutes % 60),
            "end": new Date(y, mo, d, Math.floor(endMinutes / 60), endMinutes % 60)
        };
    }

    function save() {
        if (formTitle.trim() === "") {
            formError = I18n.tr("Title is required", "event form validation error for missing title");
            return;
        }
        const range = _formRange();
        const writable = DankCalService.writableCalendars();
        if (writable.length === 0) {
            formError = I18n.tr("No writable calendar available", "event form error when no calendar allows creating events");
            return;
        }
        const cal = writable[Math.min(formCalendarIndex, writable.length - 1)];

        // Non-popup reminders (e.g. email) are kept as-is; the dropdown only
        // manages the popup reminder.
        const reminders = (event.reminders || []).filter(r => !_isPopup(r));
        if (formReminderMinutes >= 0)
            reminders.push({
                "method": "popup",
                "minutes": formReminderMinutes
            });

        const fields = {
            "summary": formTitle.trim(),
            "description": formDescription,
            "location": formLocation.trim(),
            "start": range.start.toISOString(),
            "end": range.end.toISOString(),
            "allDay": formAllDay,
            "reminders": reminders
        };

        saving = true;
        formError = "";
        const done = response => {
            saving = false;
            if (response.error) {
                formError = response.error;
                return;
            }
            hide();
        };

        if (createMode) {
            fields.calendarId = cal.id;
            DankCalService.createEvent(fields, done);
        } else {
            DankCalService.updateEvent(event.id, fields, done);
        }
    }

    function removeEvent() {
        if (!confirmDelete) {
            confirmDelete = true;
            return;
        }
        saving = true;
        DankCalService.deleteEvent(event.id, response => {
            saving = false;
            confirmDelete = false;
            if (response.error) {
                formError = response.error;
                return;
            }
            hide();
        });
    }

    function _styleAnchors(html) {
        return html.replace(/<a\s([^>]*)>/gi, (m, attrs) => {
            const cleaned = attrs.replace(/style="[^"]*"/gi, "");
            return "<a style=\"text-decoration:none; color:" + Theme.primary + ";\" " + cleaned + ">";
        });
    }

    function _inlineMarkdown(line) {
        let out = line.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
        out = out.replace(/\\([\\`*_{}[\]()#+\-.!~>])/g, "$1");
        out = out.replace(/(?:https?:\/\/|www\.)[^\s<>)\]]*[^\s<>)\].,;:!?"']/g, (m, offset, s) => {
            const prev = offset > 0 ? s[offset - 1] : "";
            if (prev === "(" || prev === "[" || prev === "\"" || prev === "'")
                return m;
            const href = m.startsWith("www.") ? "https://" + m : m;
            return "<a href=\"" + href + "\">" + m + "</a>";
        });
        out = out.replace(/\[([^\]]+)\]\(([^()\s]+)\)/g, "<a href=\"$2\">$1</a>");
        out = out.replace(/\*\*([^*]+)\*\*/g, "<b>$1</b>");
        out = out.replace(/(^|[^*])\*([^*\s][^*]*)\*/g, "$1<i>$2</i>");
        return out;
    }

    // Descriptions arrive as HTML (Google web UI) or markdown/plain text
    // (Microsoft converted server-side, CalDAV, local). Both paths render
    // as RichText with About-page style anchors: markdown import strips
    // inline styles and bakes the palette link blue, so linkColor alone
    // cannot recolor links.
    function descriptionRichText() {
        const raw = (event.description || "").trim();
        if (descriptionIsHtml)
            return _styleAnchors(raw);

        const parts = [];
        let list = "";
        const closeList = () => {
            if (list === "")
                return;
            parts.push("</" + list + ">");
            list = "";
        };

        const lines = raw.split("\n");
        for (let i = 0; i < lines.length; i++) {
            const ul = lines[i].match(/^\s*[-*+]\s+(.+)$/);
            const ol = lines[i].match(/^\s*\d+[.)]\s+(.+)$/);
            if (ul || ol) {
                const tag = ul ? "ul" : "ol";
                if (list !== tag) {
                    closeList();
                    parts.push("<" + tag + ">");
                    list = tag;
                }
                parts.push("<li>" + _inlineMarkdown((ul || ol)[1]) + "</li>");
                continue;
            }
            closeList();
            parts.push(_inlineMarkdown(lines[i]) + "<br/>");
        }
        closeList();
        return _styleAnchors(parts.join("").replace(/<br\/>$/, ""));
    }

    function timeLabel() {
        if (!event.start)
            return "";
        const day = Qt.formatDate(event.start, "dddd, MMM d, yyyy");
        if (event.allDay)
            return I18n.tr("%1 · All day", "event details time label for all-day events").arg(day);
        return day + " · " + SettingsData.formatTime(event.start) + " – " + SettingsData.formatTime(event.end);
    }

    function attendeeSummary() {
        const list = event.attendees || [];
        if (list.length === 0)
            return "";
        let accepted = 0;
        for (let i = 0; i < list.length; i++) {
            if (list[i].status === "accepted")
                accepted++;
        }
        return I18n.tr("· %1 invited · %2 accepted", "event details attendee count summary").arg(list.length).arg(accepted);
    }

    title: createMode ? I18n.tr("New event", "event modal window title when creating") : (editMode ? I18n.tr("Edit event", "event modal window title when editing") : I18n.tr("Event", "event modal window title when viewing"))
    minimumSize: Qt.size(460, 560)
    implicitWidth: 560
    implicitHeight: 720
    color: Theme.surface
    visible: false

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

                Rectangle {
                    width: 12
                    height: 12
                    radius: 3
                    color: {
                        if (!eventModal.editMode)
                            return eventModal.event.color || Theme.primary;
                        const writable = DankCalService.writableCalendars();
                        if (writable.length === 0)
                            return Theme.primary;
                        return writable[Math.min(eventModal.formCalendarIndex, writable.length - 1)].color || Theme.primary;
                    }
                    anchors.verticalCenter: parent.verticalCenter
                }

                StyledText {
                    text: eventModal.createMode ? I18n.tr("New event", "event modal header when creating") : (eventModal.editMode ? I18n.tr("Edit event", "event modal header when editing") : I18n.tr("Event details", "event modal header when viewing"))
                    font.pixelSize: Theme.fontSizeXLarge
                    font.weight: Font.Medium
                    color: Theme.surfaceText
                    anchors.verticalCenter: parent.verticalCenter
                }
            }

            Row {
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingXS

                DankActionButton {
                    visible: !eventModal.editMode && !eventModal.event.readOnly && !!eventModal.event.id
                    circular: false
                    iconName: "edit"
                    iconColor: Theme.surfaceText
                    onClicked: eventModal.beginEdit()
                }

                DankButton {
                    visible: !eventModal.editMode && !eventModal.event.readOnly && !!eventModal.event.id && eventModal.confirmDelete
                    text: I18n.tr("Confirm delete", "event details button to confirm deleting the event")
                    buttonHeight: 32
                    backgroundColor: Theme.error
                    textColor: Theme.primaryText
                    anchors.verticalCenter: parent.verticalCenter
                    onClicked: eventModal.removeEvent()
                }

                DankActionButton {
                    visible: !eventModal.editMode && !eventModal.event.readOnly && !!eventModal.event.id && !eventModal.confirmDelete
                    circular: false
                    iconName: "delete_outline"
                    iconColor: Theme.error
                    onClicked: eventModal.removeEvent()
                }

                DankActionButton {
                    circular: false
                    iconName: "close"
                    iconColor: Theme.surfaceText
                    onClicked: eventModal.hide()
                }
            }
        }

        Item {
            width: parent.width
            height: parent.height - 48 - (footer.visible ? footer.height + 1 : 0)

            Loader {
                anchors.fill: parent
                anchors.margins: Theme.spacingL
                sourceComponent: eventModal.editMode ? editComponent : detailComponent
            }
        }

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.outlineLight
            visible: footer.visible
        }

        Item {
            id: footer
            width: parent.width
            height: 59
            visible: eventModal.editMode

            StyledText {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingL
                anchors.verticalCenter: parent.verticalCenter
                text: eventModal.formError
                color: Theme.error
                font.pixelSize: Theme.fontSizeSmall
                visible: eventModal.formError !== ""
                elide: Text.ElideRight
                width: parent.width / 2
            }

            Row {
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingL
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingS

                DankButton {
                    text: I18n.tr("Cancel", "event form button to cancel editing")
                    buttonHeight: 38
                    backgroundColor: "transparent"
                    textColor: Theme.surfaceText
                    onClicked: {
                        if (eventModal.createMode) {
                            eventModal.hide();
                            return;
                        }
                        eventModal.editMode = false;
                    }
                }

                DankButton {
                    text: eventModal.saving ? I18n.tr("Saving...", "event details save button while saving") : I18n.tr("Save", "event details button to save changes")
                    iconName: "check"
                    buttonHeight: 38
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    onClicked: {
                        if (!eventModal.saving)
                            eventModal.save();
                    }
                }
            }
        }
    }

    Component {
        id: detailComponent

        Item {
            Column {
                id: titleBlock
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                spacing: Theme.spacingXS

                StyledText {
                    text: eventModal.event.title || I18n.tr("(untitled)", "event details fallback title when event has no title")
                    font.pixelSize: 24
                    font.weight: Font.Medium
                    color: Theme.surfaceText
                    width: parent.width
                    wrapMode: Text.WordWrap
                    maximumLineCount: 2
                    elide: Text.ElideRight
                }

                StyledText {
                    text: eventModal.timeLabel()
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.surfaceVariantText
                }
            }

            Column {
                id: metaBlock
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: titleBlock.bottom
                anchors.topMargin: Theme.spacingL
                spacing: Theme.spacingS

                MetaRow {
                    iconName: "calendar_month"
                    primary: eventModal.event.calendar || ""
                    secondary: eventModal.event.account || ""
                    accent: eventModal.event.color || Theme.primary
                    visible: primary !== ""
                }

                MetaRow {
                    iconName: "place"
                    primary: eventModal.event.location || ""
                    visible: primary !== ""
                }

                MetaRow {
                    iconName: "notifications"
                    primary: eventModal.reminderSummary()
                    visible: primary !== ""
                }

                MetaRow {
                    iconName: "link"
                    primary: eventModal.event.url || ""
                    link: true
                    visible: primary !== ""

                    TapHandler {
                        onTapped: Qt.openUrlExternally(eventModal.event.url)
                    }
                }
            }

            Column {
                id: attendeesBlock
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: metaBlock.bottom
                anchors.topMargin: Theme.spacingL
                spacing: Theme.spacingS
                visible: (eventModal.event.attendees || []).length > 0

                Row {
                    spacing: Theme.spacingS

                    DankIcon {
                        name: "people"
                        size: Theme.iconSize - 4
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("Attendees", "event details section label for attendee list")
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: eventModal.attendeeSummary()
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                Repeater {
                    model: ScriptModel {
                        values: (eventModal.event.attendees || []).slice(0, 8)
                    }

                    Item {
                        id: attendeeItem
                        required property var modelData
                        readonly property string displayName: modelData.displayName || modelData.email || ""
                        readonly property string status: modelData.status || "needsAction"
                        width: parent.width
                        height: 36

                        Row {
                            anchors.left: parent.left
                            anchors.leftMargin: Theme.spacingL + Theme.iconSize - 4
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: Theme.spacingS

                            Rectangle {
                                width: 24
                                height: 24
                                radius: 12
                                anchors.verticalCenter: parent.verticalCenter
                                color: Theme.withAlpha(Theme.primary, 0.18)

                                StyledText {
                                    anchors.centerIn: parent
                                    text: attendeeItem.displayName.charAt(0).toUpperCase()
                                    font.pixelSize: Theme.fontSizeSmall
                                    font.weight: Font.Medium
                                    color: Theme.primary
                                }
                            }

                            StyledText {
                                text: attendeeItem.displayName
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.surfaceText
                                anchors.verticalCenter: parent.verticalCenter
                            }

                            StyledText {
                                text: attendeeItem.modelData.email && attendeeItem.modelData.displayName ? attendeeItem.modelData.email : ""
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.surfaceVariantText
                                anchors.verticalCenter: parent.verticalCenter
                            }
                        }

                        Item {
                            anchors.right: parent.right
                            anchors.verticalCenter: parent.verticalCenter
                            width: 90
                            height: 22

                            Rectangle {
                                anchors.fill: parent
                                radius: 11
                                color: {
                                    switch (attendeeItem.status) {
                                    case "accepted":
                                        return Theme.withAlpha(Theme.success, 0.18);
                                    case "declined":
                                        return Theme.withAlpha(Theme.error, 0.18);
                                    default:
                                        return Theme.withAlpha(Theme.warning, 0.18);
                                    }
                                }

                                StyledText {
                                    anchors.centerIn: parent
                                    text: attendeeItem.status
                                    font.pixelSize: Theme.fontSizeSmall
                                    font.weight: Font.Medium
                                    elide: Text.ElideRight
                                    color: {
                                        switch (attendeeItem.status) {
                                        case "accepted":
                                            return Theme.success;
                                        case "declined":
                                            return Theme.error;
                                        default:
                                            return Theme.warning;
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }

            Column {
                id: descBlock
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: attendeesBlock.visible ? attendeesBlock.bottom : metaBlock.bottom
                anchors.bottom: parent.bottom
                anchors.topMargin: Theme.spacingL
                spacing: Theme.spacingS
                visible: (eventModal.event.description || "") !== ""

                Row {
                    spacing: Theme.spacingS

                    DankIcon {
                        name: "notes"
                        size: Theme.iconSize - 4
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("Description", "event details section label for description")
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                StyledRect {
                    width: parent.width
                    height: parent.height - parent.spacing - 24
                    color: Theme.surfaceContainer
                    radius: Theme.cornerRadius

                    DankFlickable {
                        anchors.fill: parent
                        anchors.margins: Theme.spacingM
                        clip: true
                        contentWidth: width
                        contentHeight: descriptionText.implicitHeight

                        StyledText {
                            id: descriptionText
                            text: eventModal.descriptionRichText()
                            textFormat: Text.RichText
                            linkColor: Theme.primary
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.surfaceText
                            width: parent.width
                            wrapMode: Text.WordWrap
                            onLinkActivated: link => Qt.openUrlExternally(link)

                            HoverHandler {
                                enabled: descriptionText.hoveredLink !== ""
                                cursorShape: Qt.PointingHandCursor
                            }
                        }
                    }
                }
            }
        }
    }

    Component {
        id: editComponent

        Column {
            spacing: Theme.spacingM

            DankTextField {
                width: parent.width
                placeholderText: I18n.tr("Add title", "event form placeholder for title input")
                text: eventModal.formTitle
                onTextChanged: eventModal.formTitle = text
            }

            Row {
                width: parent.width
                spacing: Theme.spacingM

                DankDatePicker {
                    width: parent.width - allDayRow.width - Theme.spacingM
                    firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
                    selectedDate: eventModal.formStartDate
                    onDateSelected: value => eventModal.formStartDate = value
                }

                Row {
                    id: allDayRow

                    spacing: Theme.spacingS
                    anchors.verticalCenter: parent.verticalCenter

                    DankToggle {
                        checked: eventModal.formAllDay
                        onToggled: checked => eventModal.formAllDay = checked
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("All day", "event form toggle label for all-day events")
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }
            }

            Row {
                width: parent.width
                spacing: Theme.spacingM
                visible: !eventModal.formAllDay

                DankTimePicker {
                    width: (parent.width - dash.width - Theme.spacingM * 2) / 2
                    use24Hour: SettingsData.use24HourTime
                    minutes: eventModal.formStartMinutes
                    onTimeSelected: value => {
                        const duration = eventModal.formEndMinutes - eventModal.formStartMinutes;
                        eventModal.formStartMinutes = value;
                        eventModal.formEndMinutes = value + Math.max(duration, 0);
                    }
                }

                StyledText {
                    id: dash

                    text: "–"
                    color: Theme.surfaceVariantText
                    anchors.verticalCenter: parent.verticalCenter
                }

                DankTimePicker {
                    width: (parent.width - dash.width - Theme.spacingM * 2) / 2
                    use24Hour: SettingsData.use24HourTime
                    minutes: eventModal.formEndMinutes
                    onTimeSelected: value => eventModal.formEndMinutes = value
                }
            }

            DankTextField {
                width: parent.width
                iconName: "place"
                placeholderText: I18n.tr("Location", "event form placeholder for location input")
                text: eventModal.formLocation
                onTextChanged: eventModal.formLocation = text
            }

            Row {
                width: parent.width
                spacing: Theme.spacingM

                DankIcon {
                    name: "calendar_month"
                    size: Theme.iconSize - 6
                    color: Theme.surfaceVariantText
                    anchors.verticalCenter: parent.verticalCenter
                }

                DankDropdown {
                    readonly property var writable: DankCalService.writableCalendars()

                    width: parent.width - (Theme.iconSize - 6) - Theme.spacingM
                    enabled: eventModal.createMode
                    opacity: enabled ? 1 : 0.5
                    options: writable.map(c => c.name)
                    currentValue: writable.length > 0 ? writable[Math.min(eventModal.formCalendarIndex, writable.length - 1)].name : ""
                    onValueChanged: value => {
                        for (let i = 0; i < writable.length; i++) {
                            if (writable[i].name === value) {
                                eventModal.formCalendarIndex = i;
                                return;
                            }
                        }
                    }
                }
            }

            Row {
                width: parent.width
                spacing: Theme.spacingM

                DankIcon {
                    name: "notifications"
                    size: Theme.iconSize - 6
                    color: Theme.surfaceVariantText
                    anchors.verticalCenter: parent.verticalCenter
                }

                DankDropdown {
                    width: parent.width - (Theme.iconSize - 6) - Theme.spacingM
                    options: eventModal.reminderOptions.map(o => o.label)
                    currentValue: eventModal.reminderOptionLabel(eventModal.formReminderMinutes)
                    onValueChanged: value => {
                        for (let i = 0; i < eventModal.reminderOptions.length; i++) {
                            if (eventModal.reminderOptions[i].label === value) {
                                eventModal.formReminderMinutes = eventModal.reminderOptions[i].value;
                                return;
                            }
                        }
                    }
                }
            }

            StyledRect {
                width: parent.width
                height: 150
                color: Theme.surfaceContainer
                radius: Theme.cornerRadius
                border.width: 1
                border.color: descArea.activeFocus ? Theme.primary : Theme.outlineLight

                DankFlickable {
                    anchors.fill: parent
                    anchors.margins: Theme.spacingS
                    clip: true
                    contentWidth: width

                    TextArea.flickable: TextArea {
                        id: descArea

                        wrapMode: TextEdit.Wrap
                        background: null
                        color: Theme.surfaceText
                        selectionColor: Theme.primarySelected
                        selectedTextColor: Theme.surfaceText
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeMedium
                        text: eventModal.formDescription
                        onTextChanged: eventModal.formDescription = text

                        Text {
                            anchors.fill: parent
                            anchors.leftMargin: descArea.leftPadding
                            anchors.topMargin: descArea.topPadding
                            visible: descArea.length === 0 && descArea.preeditText.length === 0
                            text: I18n.tr("Add description", "event form placeholder for description text area")
                            color: Theme.surfaceVariantText
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            wrapMode: Text.Wrap
                        }
                    }
                }
            }
        }
    }

    component MetaRow: Row {
        property string iconName: ""
        property string primary: ""
        property string secondary: ""
        property color accent: Theme.surfaceVariantText
        property bool link: false

        spacing: Theme.spacingM
        width: parent.width
        height: 32

        DankIcon {
            name: parent.iconName
            size: Theme.iconSize - 4
            color: parent.accent
            anchors.verticalCenter: parent.verticalCenter
        }

        Column {
            anchors.verticalCenter: parent.verticalCenter
            spacing: 0
            width: parent.width - Theme.iconSize - 4 - Theme.spacingM

            StyledText {
                text: parent.parent.primary
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.Medium
                color: parent.parent.link ? Theme.primary : Theme.surfaceText
                width: parent.width
                elide: Text.ElideRight
            }

            StyledText {
                visible: text !== ""
                text: parent.parent.secondary
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.surfaceVariantText
                width: parent.width
                elide: Text.ElideRight
            }
        }
    }

    FloatingWindowControls {
        id: windowControls
        targetWindow: eventModal
    }
}
