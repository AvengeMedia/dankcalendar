import QtQuick
import QtQuick.Controls
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets
import qs.DankCommon.Widgets
import "../Common/EventUtils.js" as EventUtils

FloatingWindow {
    id: eventModal

    property var event: ({})
    property bool editMode: false
    property bool createMode: false
    property bool confirmDelete: false
    property bool saving: false
    property string pendingResponse: ""
    property string formError: ""

    readonly property bool noWritableCalendars: DankCalService.writableCalendars().length === 0

    signal addCalendarRequested

    property string formTitle: ""
    property date formStartDate: new Date()
    property date formEndDate: new Date()
    property int formStartMinutes: 600
    property int formEndMinutes: 660
    property string formLocation: ""
    property string formDescription: ""
    property bool formAllDay: false
    property int formCalendarIndex: 0
    property var formReminders: []
    property var recurrencePickerItem: null

    // Recurrence is editable on new events and series masters; occurrence
    // rows (recurringId set) would need per-occurrence semantics we don't
    // model, so the control is hidden there.
    readonly property bool recurrenceEditable: createMode || !(event.recurringId || "")
    readonly property bool isOccurrence: (event.recurringId || "") !== "" && (event.recurrence || []).length > 0
    readonly property bool isRecurring: (event.recurrence || []).length > 0
    readonly property int maxReminders: 5

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
        saving = false;
        pendingResponse = "";
        formError = "";
        event = eventData || {};
        visible = true;
    }

    function _nextHalfHour() {
        const slot = 30 * 60000;
        return new Date(Math.ceil(Date.now() / slot) * slot);
    }

    // Prefill start: an explicit time is honored (ui.newEvent start=...); a
    // bare day (midnight) gets 10:00, except today, which rounds up to the
    // next half-hour slot so the prefill is never in the past.
    function _defaultStart(day) {
        if (!day)
            return _nextHalfHour();
        const base = new Date(day);
        if (base.getHours() !== 0 || base.getMinutes() !== 0 || base.getSeconds() !== 0)
            return base;
        const next = _nextHalfHour();
        if (base.getFullYear() === next.getFullYear() && base.getMonth() === next.getMonth() && base.getDate() === next.getDate())
            return next;
        base.setHours(10, 0, 0, 0);
        return base;
    }

    function showCreate(day, endDay) {
        const base = _defaultStart(day);
        if (!endDay) {
            _beginCreate(base, new Date(base.getTime() + SettingsData.defaultEventDurationMinutes * 60000), false);
            return;
        }
        base.setHours(0, 0, 0, 0);
        const end = new Date(endDay);
        end.setHours(0, 0, 0, 0);
        end.setDate(end.getDate() + 1);
        _beginCreate(base, end, true);
    }

    function showCreateTimed(start, end) {
        _beginCreate(new Date(start), new Date(end), false);
    }

    function _beginCreate(start, end, allDay) {
        event = {
            "title": "",
            "description": "",
            "location": "",
            "start": start,
            "end": end,
            "allDay": allDay,
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
        saving = false;
        pendingResponse = "";
        formError = "";
        _loadForm();
        editMode = true;
        visible = true;
    }

    function hide() {
        editMode = false;
        createMode = false;
        confirmDelete = false;
        saving = false;
        pendingResponse = "";
        visible = false;
    }

    onClosed: hide()

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
        let endDate = new Date(end);
        if (formAllDay)
            endDate.setDate(endDate.getDate() - 1);
        if (endDate.getTime() < start.getTime())
            endDate = new Date(start);
        formEndDate = endDate;
        if (!formAllDay && end.getTime() <= start.getTime())
            formEndMinutes = formStartMinutes + 60;

        const writable = DankCalService.writableCalendars();
        formCalendarIndex = 0;
        for (let i = 0; i < writable.length; i++) {
            if (writable[i].id === event.calendarId) {
                formCalendarIndex = i;
                break;
            }
        }

        const popupMinutes = [];
        const reminders = event.reminders || [];
        for (let i = 0; i < reminders.length; i++) {
            if (!_isPopup(reminders[i]))
                continue;
            if (popupMinutes.indexOf(reminders[i].minutes) === -1)
                popupMinutes.push(reminders[i].minutes);
        }
        formReminders = popupMinutes;
    }

    function addReminder() {
        const list = formReminders.slice();
        if (list.length >= maxReminders)
            return;
        let candidate = SettingsData.defaultReminderMinutes >= 0 ? SettingsData.defaultReminderMinutes : 10;
        if (list.indexOf(candidate) !== -1) {
            for (let i = 1; i < reminderOptions.length; i++) {
                if (list.indexOf(reminderOptions[i].value) === -1) {
                    candidate = reminderOptions[i].value;
                    break;
                }
            }
        }
        if (list.indexOf(candidate) !== -1)
            return;
        list.push(candidate);
        formReminders = list;
    }

    function setReminder(index, minutes) {
        const list = formReminders.slice();
        list[index] = minutes;
        formReminders = list;
    }

    function removeReminder(index) {
        const list = formReminders.slice();
        list.splice(index, 1);
        formReminders = list;
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

    // locationUrl makes the location row clickable: a URL location opens
    // directly, conference placeholders open the meeting link, and anything
    // else opens as a geo: search in the maps app.
    function locationUrl() {
        const loc = (event.location || "").trim();
        if (loc === "")
            return "";
        if (/^https?:\/\/\S+$/i.test(loc))
            return loc;
        if (/^www\.\S+$/i.test(loc))
            return "https://" + loc;
        if (event.meetingUrl)
            return event.meetingUrl;
        return "geo:0,0?q=" + encodeURIComponent(loc);
    }

    function recurrenceSummary() {
        const label = DankCalService.recurrenceLabel(event);
        if (label !== "")
            return label;
        return (event.recurringId || "") !== "" ? I18n.tr("Repeats", "generic recurrence label for an unrecognized rule") : "";
    }

    function reminderOptionLabel(minutes) {
        for (let i = 0; i < reminderOptions.length; i++) {
            if (reminderOptions[i].value === minutes)
                return reminderOptions[i].label;
        }
        return reminderText(minutes);
    }

    function setFormStartDate(value) {
        const startMidnight = new Date(formStartDate.getFullYear(), formStartDate.getMonth(), formStartDate.getDate());
        const endMidnight = new Date(formEndDate.getFullYear(), formEndDate.getMonth(), formEndDate.getDate());
        const spanDays = Math.round((endMidnight.getTime() - startMidnight.getTime()) / 86400000);
        formStartDate = value;
        const end = new Date(value);
        end.setDate(end.getDate() + Math.max(spanDays, 0));
        formEndDate = end;
    }

    function _formRange() {
        const y = formStartDate.getFullYear();
        const mo = formStartDate.getMonth();
        const d = formStartDate.getDate();
        const ey = formEndDate.getFullYear();
        const emo = formEndDate.getMonth();
        const ed = formEndDate.getDate();
        if (formAllDay) {
            const start = new Date(Date.UTC(y, mo, d));
            let end = new Date(Date.UTC(ey, emo, ed + 1));
            if (end.getTime() <= start.getTime())
                end = new Date(Date.UTC(y, mo, d + 1));
            return {
                "start": start,
                "end": end
            };
        }

        const start = new Date(y, mo, d, Math.floor(formStartMinutes / 60), formStartMinutes % 60);
        let end = new Date(ey, emo, ed, Math.floor(formEndMinutes / 60), formEndMinutes % 60);
        if (end.getTime() <= start.getTime())
            end = new Date(y, mo, d, Math.floor((formStartMinutes + 60) / 60), (formStartMinutes + 60) % 60);
        return {
            "start": start,
            "end": end
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

        // Non-popup reminders (e.g. email) are kept as-is; the editor only
        // manages popup reminders.
        const reminders = (event.reminders || []).filter(r => !_isPopup(r));
        const seen = [];
        for (let i = 0; i < formReminders.length; i++) {
            const minutes = formReminders[i];
            if (minutes < 0 || seen.indexOf(minutes) !== -1)
                continue;
            seen.push(minutes);
            reminders.push({
                "method": "popup",
                "minutes": minutes
            });
        }

        const fields = {
            "summary": formTitle.trim(),
            "description": formDescription,
            "location": formLocation.trim(),
            "start": range.start.toISOString(),
            "end": range.end.toISOString(),
            "allDay": formAllDay,
            "reminders": reminders
        };
        if (recurrenceEditable && recurrencePickerItem)
            fields.recurrence = recurrencePickerItem.currentRules();
        if (!createMode && isOccurrence)
            fields.occurrenceStart = EventUtils.wireTime(event.start, event.allDay);

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

    function removeEvent(occurrenceOnly) {
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
        }, occurrenceOnly ? EventUtils.wireTime(event.start, event.allDay) : undefined);
    }

    function respond(action) {
        if (!event.id || saving)
            return;
        if (isRecurring) {
            pendingResponse = action;
            return;
        }
        submitResponse(action, false);
    }

    function submitResponse(action, occurrenceOnly) {
        pendingResponse = "";
        saving = true;
        formError = "";
        DankCalService.rsvpEvent(event.id, action, response => {
            saving = false;
            if (response.error) {
                formError = response.error;
                return;
            }
            if (response.result)
                eventModal.event = DankCalService.eventFromResult(response.result);
        }, occurrenceOnly ? EventUtils.wireTime(event.start, event.allDay) : undefined);
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
        const day = SettingsData.formatDate(event.start, "dddd, MMM d, yyyy");
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

    readonly property real contentNaturalHeight: contentLoader.item ? contentLoader.item.naturalHeight : 0
    readonly property real chromeHeight: 48 + Theme.spacingL * 2 + (editMode ? 60 : 0)

    title: createMode ? I18n.tr("New event", "event modal window title when creating") : (editMode ? I18n.tr("Edit event", "event modal window title when editing") : I18n.tr("Event", "event modal window title when viewing"))
    minimumSize: Qt.size(460, 560)
    implicitWidth: Math.max(minimumSize.width, Theme.modalWidth(parentWindow, screen, 560))
    implicitHeight: Math.max(minimumSize.height, Theme.modalHeight(parentWindow, screen, Math.max(720, chromeHeight + contentNaturalHeight)))
    color: Theme.surface
    visible: false

    Column {
        anchors.fill: parent
        spacing: 0

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

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
                    visible: !eventModal.editMode && !eventModal.event.readOnly && !!eventModal.event.id && eventModal.confirmDelete && eventModal.isRecurring
                    text: I18n.tr("Delete occurrence", "event details button to delete only this occurrence of a recurring event")
                    buttonHeight: 32
                    backgroundColor: Theme.error
                    textColor: Theme.primaryText
                    anchors.verticalCenter: parent.verticalCenter
                    onClicked: eventModal.removeEvent(true)
                }

                DankButton {
                    visible: !eventModal.editMode && !eventModal.event.readOnly && !!eventModal.event.id && eventModal.confirmDelete
                    text: eventModal.isRecurring ? I18n.tr("Delete series", "event details button to confirm deleting a whole recurring series") : I18n.tr("Confirm delete", "event details button to confirm deleting the event")
                    buttonHeight: 32
                    backgroundColor: eventModal.isRecurring ? "transparent" : Theme.error
                    textColor: eventModal.isRecurring ? Theme.error : Theme.primaryText
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
                id: contentLoader
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
            readonly property real naturalHeight: {
                var h = titleBlock.implicitHeight + Theme.spacingL + metaBlock.implicitHeight;
                if (rsvpBlock.visible)
                    h += Theme.spacingL + rsvpBlock.implicitHeight;
                if (attendeesBlock.visible)
                    h += Theme.spacingL + attendeesBlock.implicitHeight;
                if (descBlock.visible)
                    h += Theme.spacingL + 24 + descBlock.spacing + descriptionText.implicitHeight + Theme.spacingM * 2;
                return h;
            }

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
                    width: parent.width
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
                    accent: link ? Theme.primary : Theme.surfaceVariantText
                    link: linkUrl !== ""
                    linkUrl: eventModal.locationUrl()
                    visible: primary !== ""
                }

                MetaRow {
                    iconName: "repeat"
                    primary: eventModal.recurrenceSummary()
                    visible: primary !== ""
                }

                MetaRow {
                    iconName: "videocam"
                    primary: I18n.tr("Join video call", "event details row that opens the meeting link")
                    secondary: eventModal.event.meetingUrl || ""
                    accent: Theme.primary
                    link: true
                    linkUrl: eventModal.event.meetingUrl || ""
                    visible: secondary !== ""
                }

                MetaRow {
                    iconName: "notifications"
                    primary: eventModal.reminderSummary()
                    visible: primary !== ""
                }

                MetaRow {
                    iconName: "link"
                    primary: eventModal.event.url || ""
                    accent: Theme.primary
                    link: true
                    linkUrl: eventModal.event.url || ""
                    visible: primary !== ""
                }
            }

            Column {
                id: rsvpBlock
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: metaBlock.bottom
                anchors.topMargin: Theme.spacingL
                spacing: Theme.spacingS
                visible: !!eventModal.event.canRespond

                Row {
                    spacing: Theme.spacingS

                    DankIcon {
                        name: "event_available"
                        size: Theme.iconSize - 4
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("Your response", "event details RSVP section label")
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                Row {
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingL + Theme.iconSize - 4
                    spacing: Theme.spacingS
                    visible: eventModal.pendingResponse === ""

                    DankButton {
                        text: I18n.tr("Accept", "RSVP accept button")
                        buttonHeight: 32
                        backgroundColor: eventModal.event.myResponse === "accepted" ? Theme.success : Theme.surfaceContainer
                        textColor: eventModal.event.myResponse === "accepted" ? Theme.primaryText : Theme.surfaceText
                        enabled: !eventModal.saving
                        onClicked: eventModal.respond("accept")
                    }

                    DankButton {
                        text: I18n.tr("Maybe", "RSVP tentative button")
                        buttonHeight: 32
                        backgroundColor: eventModal.event.myResponse === "tentative" ? Theme.warning : Theme.surfaceContainer
                        textColor: eventModal.event.myResponse === "tentative" ? Theme.primaryText : Theme.surfaceText
                        enabled: !eventModal.saving
                        onClicked: eventModal.respond("tentative")
                    }

                    DankButton {
                        text: I18n.tr("Decline", "RSVP decline button")
                        buttonHeight: 32
                        backgroundColor: eventModal.event.myResponse === "declined" ? Theme.error : Theme.surfaceContainer
                        textColor: eventModal.event.myResponse === "declined" ? Theme.primaryText : Theme.surfaceText
                        enabled: !eventModal.saving
                        onClicked: eventModal.respond("decline")
                    }
                }

                Row {
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingL + Theme.iconSize - 4
                    spacing: Theme.spacingS
                    visible: eventModal.pendingResponse !== ""

                    DankButton {
                        text: I18n.tr("This event", "RSVP scope button replying for one occurrence of a recurring event")
                        buttonHeight: 32
                        backgroundColor: Theme.primary
                        textColor: Theme.primaryText
                        enabled: !eventModal.saving
                        onClicked: eventModal.submitResponse(eventModal.pendingResponse, true)
                    }

                    DankButton {
                        text: I18n.tr("All events", "RSVP scope button replying for the whole recurring series")
                        buttonHeight: 32
                        backgroundColor: Theme.surfaceContainer
                        textColor: Theme.surfaceText
                        enabled: !eventModal.saving
                        onClicked: eventModal.submitResponse(eventModal.pendingResponse, false)
                    }

                    DankActionButton {
                        circular: false
                        iconName: "close"
                        iconColor: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                        onClicked: eventModal.pendingResponse = ""
                    }
                }
            }

            Column {
                id: attendeesBlock
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: rsvpBlock.visible ? rsvpBlock.bottom : metaBlock.bottom
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
                anchors.top: attendeesBlock.visible ? attendeesBlock.bottom : (rsvpBlock.visible ? rsvpBlock.bottom : metaBlock.bottom)
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

        DankFlickable {
            readonly property real naturalHeight: editColumn.implicitHeight

            clip: true
            contentWidth: width
            contentHeight: editColumn.implicitHeight

            Column {
                id: editColumn
                width: parent.width
                spacing: Theme.spacingM

                DankTextField {
                    width: parent.width
                    placeholderText: I18n.tr("Add title", "event form placeholder for title input")
                    text: eventModal.formTitle
                    onTextChanged: eventModal.formTitle = text
                    Component.onCompleted: forceActiveFocus()
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingM

                    DankDatePicker {
                        width: (parent.width - allDayRow.width - dateDash.width - Theme.spacingM * 3) / 2
                        firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
                        selectedDate: eventModal.formStartDate
                        onDateSelected: value => eventModal.setFormStartDate(value)
                    }

                    StyledText {
                        id: dateDash

                        text: "–"
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    DankDatePicker {
                        width: (parent.width - allDayRow.width - dateDash.width - Theme.spacingM * 3) / 2
                        firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
                        selectedDate: eventModal.formEndDate
                        onDateSelected: value => eventModal.formEndDate = value
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
                    leftIconName: "place"
                    placeholderText: I18n.tr("Location", "event form placeholder for location input")
                    text: eventModal.formLocation
                    onTextChanged: eventModal.formLocation = text
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingM
                    visible: !(eventModal.createMode && eventModal.noWritableCalendars)

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
                    visible: eventModal.createMode && eventModal.noWritableCalendars

                    DankIcon {
                        name: "calendar_add_on"
                        size: Theme.iconSize - 6
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Column {
                        width: parent.width - (Theme.iconSize - 6) - Theme.spacingM
                        spacing: Theme.spacingXS

                        StyledText {
                            width: parent.width
                            text: I18n.tr("No calendars yet. Add one to start creating events.", "event form guidance when there are no writable calendars")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.surfaceVariantText
                            wrapMode: Text.WordWrap
                        }

                        DankButton {
                            text: I18n.tr("Add a calendar", "event form button to add a calendar when none exist")
                            iconName: "add"
                            buttonHeight: 36
                            backgroundColor: Theme.primary
                            textColor: Theme.primaryText
                            onClicked: {
                                eventModal.addCalendarRequested();
                                eventModal.hide();
                            }
                        }
                    }
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingM
                    visible: eventModal.recurrenceEditable

                    DankIcon {
                        name: "repeat"
                        size: Theme.iconSize - 6
                        color: Theme.surfaceVariantText
                        anchors.top: parent.top
                        anchors.topMargin: (40 - (Theme.iconSize - 6)) / 2
                    }

                    Column {
                        width: parent.width - (Theme.iconSize - 6) - Theme.spacingM
                        spacing: Theme.spacingXS

                        DankRecurrencePicker {
                            id: recurrencePicker
                            width: parent.width
                            rules: eventModal.event.recurrence || []
                            startDate: eventModal.formStartDate
                            allDay: eventModal.formAllDay
                            Component.onCompleted: eventModal.recurrencePickerItem = this
                            Component.onDestruction: {
                                if (eventModal.recurrencePickerItem === this)
                                    eventModal.recurrencePickerItem = null;
                            }
                        }

                        StyledText {
                            visible: !eventModal.createMode && recurrencePicker.isRecurring
                            text: I18n.tr("Changes apply to the whole series", "event form note when editing a recurring event")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.surfaceVariantText
                            width: parent.width
                            horizontalAlignment: Text.AlignLeft
                            wrapMode: Text.WordWrap
                        }
                    }
                }

                Row {
                    width: parent.width
                    spacing: Theme.spacingM
                    visible: !eventModal.recurrenceEditable && eventModal.isOccurrence

                    DankIcon {
                        name: "repeat"
                        size: Theme.iconSize - 6
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Column {
                        width: parent.width - (Theme.iconSize - 6) - Theme.spacingM
                        spacing: 0

                        StyledText {
                            text: eventModal.recurrenceSummary()
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.surfaceText
                            width: parent.width
                            horizontalAlignment: Text.AlignLeft
                            wrapMode: Text.WordWrap
                        }

                        StyledText {
                            text: I18n.tr("Changes apply to the whole series", "event form note when editing a recurring event")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.surfaceVariantText
                            width: parent.width
                            horizontalAlignment: Text.AlignLeft
                            wrapMode: Text.WordWrap
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
                        anchors.top: parent.top
                        anchors.topMargin: (40 - (Theme.iconSize - 6)) / 2
                    }

                    Column {
                        width: parent.width - (Theme.iconSize - 6) - Theme.spacingM
                        spacing: Theme.spacingS

                        Repeater {
                            model: ScriptModel {
                                values: eventModal.formReminders
                            }

                            Row {
                                id: reminderRow

                                required property int index
                                required property var modelData

                                width: parent.width
                                spacing: Theme.spacingS

                                DankDropdown {
                                    width: parent.width - removeButton.width - Theme.spacingS
                                    options: eventModal.reminderOptions.slice(1).map(o => o.label)
                                    currentValue: eventModal.reminderOptionLabel(reminderRow.modelData)
                                    onValueChanged: value => {
                                        for (let i = 0; i < eventModal.reminderOptions.length; i++) {
                                            if (eventModal.reminderOptions[i].label === value) {
                                                eventModal.setReminder(reminderRow.index, eventModal.reminderOptions[i].value);
                                                return;
                                            }
                                        }
                                    }
                                }

                                DankActionButton {
                                    id: removeButton
                                    iconName: "close"
                                    iconColor: Theme.surfaceVariantText
                                    onClicked: eventModal.removeReminder(reminderRow.index)
                                    anchors.verticalCenter: parent.verticalCenter
                                }
                            }
                        }

                        DankButton {
                            text: I18n.tr("Add reminder", "event form button to add another reminder")
                            iconName: "add"
                            buttonHeight: 36
                            visible: eventModal.formReminders.length < eventModal.maxReminders
                            onClicked: eventModal.addReminder()
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
                            Keys.onTabPressed: nextItemInFocusChain(true).forceActiveFocus()
                            Keys.onBacktabPressed: nextItemInFocusChain(false).forceActiveFocus()

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
    }

    component MetaRow: Item {
        property string iconName: ""
        property string primary: ""
        property string secondary: ""
        property color accent: Theme.surfaceVariantText
        property bool link: false
        property string linkUrl: ""

        width: parent.width
        height: Math.max(32, infoColumn.implicitHeight)

        DankIcon {
            id: rowIcon
            name: parent.iconName
            size: Theme.iconSize - 4
            color: parent.accent
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
        }

        Column {
            id: infoColumn
            anchors.left: rowIcon.right
            anchors.leftMargin: Theme.spacingM
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            spacing: 0

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

        MouseArea {
            anchors.fill: parent
            enabled: parent.link && parent.linkUrl !== ""
            cursorShape: enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
            onClicked: DankCalService.openUri(parent.linkUrl)
        }
    }

    Shortcut {
        sequence: "Escape"
        enabled: eventModal.visible
        onActivated: {
            if (eventModal.confirmDelete) {
                eventModal.confirmDelete = false;
                return;
            }
            if (eventModal.editMode && !eventModal.createMode) {
                eventModal.editMode = false;
                return;
            }
            eventModal.hide();
        }
    }

    FloatingWindowControls {
        id: windowControls
        targetWindow: eventModal
    }
}
