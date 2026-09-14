import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets
import qs.DankCommon.Widgets

FloatingWindow {
    id: importModal

    property string ics: ""
    property string fileName: ""
    property string method: ""
    property var items: []
    property bool loading: false
    property bool importing: false
    property string errorText: ""
    property int calendarIndex: 0

    readonly property var writable: DankCalService.writableCalendars()
    readonly property bool noWritableCalendars: writable.length === 0
    readonly property var pendingItems: items.filter(item => !item.existing)
    readonly property var targetCalendar: writable.length > 0 ? writable[Math.min(calendarIndex, writable.length - 1)] : null

    signal openEventRequested(var event)
    signal addCalendarRequested

    function show(data, name) {
        ics = data;
        fileName = name || "";
        method = "";
        items = [];
        errorText = "";
        importing = false;
        loading = true;
        calendarIndex = _defaultCalendarIndex();
        visible = true;
        DankCalService.parseIcs(ics, response => {
            loading = false;
            if (response.error) {
                errorText = response.error;
                return;
            }
            const result = response.result || {};
            method = result.method || "";
            items = (result.events || []).map(entry => ({
                        "event": DankCalService.eventFromResult(entry.event),
                        "existing": entry.existing ? DankCalService.eventFromResult(entry.existing) : null
                    }));
        });
    }

    function hide() {
        visible = false;
    }

    onClosed: hide()

    function _defaultCalendarIndex() {
        const preferred = DankCalService.defaultCalendar();
        if (!preferred)
            return 0;
        for (let i = 0; i < writable.length; i++) {
            if (writable[i].id === preferred.id)
                return i;
        }
        return 0;
    }

    function importPending() {
        if (importing || !targetCalendar || pendingItems.length === 0)
            return;
        importing = true;
        errorText = "";
        DankCalService.importIcs(ics, targetCalendar.id, pendingItems.map(item => item.event.uid), response => {
            importing = false;
            if (response.error) {
                errorText = response.error;
                return;
            }
            const imported = ((response.result || {}).events || []).filter(entry => !entry.existing);
            hide();
            if (imported.length === 1)
                openEventRequested(DankCalService.eventFromResult(imported[0].event));
        });
    }

    function openExisting(event) {
        hide();
        openEventRequested(event);
    }

    function kindLabel() {
        switch (method) {
        case "REQUEST":
            return I18n.tr("Meeting invitation", "import dialog subtitle for an iCalendar REQUEST");
        case "CANCEL":
            return I18n.tr("Meeting cancellation", "import dialog subtitle for an iCalendar CANCEL");
        case "REPLY":
            return I18n.tr("Meeting reply", "import dialog subtitle for an iCalendar REPLY");
        default:
            return I18n.tr("Calendar file", "import dialog subtitle for a plain iCalendar file");
        }
    }

    function subtitle() {
        if (fileName === "")
            return kindLabel();
        return kindLabel() + " · " + fileName;
    }

    function timeLabel(ev) {
        if (!ev.start)
            return "";
        const day = SettingsData.formatDate(ev.start, "dddd, MMM d, yyyy");
        if (ev.allDay)
            return I18n.tr("%1 · All day", "event details time label for all-day events").arg(day);
        return day + " · " + SettingsData.formatTime(ev.start) + " – " + SettingsData.formatTime(ev.end);
    }

    function organizerLabel(ev) {
        if (!ev.organizer)
            return "";
        const who = ev.organizer.displayName || ev.organizer.email || "";
        if (who === "")
            return "";
        return I18n.tr("Organized by %1", "import row line naming the meeting organizer").arg(who);
    }

    function importLabel() {
        if (importing)
            return I18n.tr("Importing...", "import dialog button while the import runs");
        if (pendingItems.length > 1)
            return I18n.tr("Import %1 events", "import dialog button naming how many events will be imported").arg(pendingItems.length);
        return I18n.tr("Import", "import dialog button to import the events");
    }

    function statusText() {
        if (loading)
            return I18n.tr("Reading file...", "import dialog status while parsing");
        if (items.length > 0 && pendingItems.length === 0)
            return I18n.tr("Everything in this file is already on your calendar.", "import dialog status when no event is new");
        return "";
    }

    readonly property real chromeHeight: header.height + footer.height + Theme.spacingL * 2
    readonly property real contentNaturalHeight: contentColumn.implicitHeight

    title: I18n.tr("Import events", "import modal window title")
    minimumSize: Qt.size(460, 360)
    implicitWidth: Math.max(minimumSize.width, Theme.modalWidth(parentWindow, screen, 560))
    implicitHeight: Math.max(minimumSize.height, Theme.modalHeight(parentWindow, screen, Math.max(420, chromeHeight + contentNaturalHeight)))
    color: Theme.surface
    visible: false

    Column {
        anchors.fill: parent
        spacing: 0

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        DankWindowHeader {
            id: header
            width: parent.width
            z: 10
            controls: windowControls
            title: I18n.tr("Import events", "import modal header")
            iconName: "download"
            onCloseRequested: importModal.hide()
        }

        Item {
            width: parent.width
            height: parent.height - header.height - footer.height - Theme.dividerWidth

            DankFlickable {
                anchors.fill: parent
                anchors.margins: Theme.spacingL
                clip: true
                contentWidth: width
                contentHeight: contentColumn.implicitHeight

                Column {
                    id: contentColumn
                    width: parent.width
                    spacing: Theme.spacingM

                    StyledText {
                        width: parent.width
                        text: importModal.subtitle()
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.surfaceVariantText
                        elide: Text.ElideMiddle
                    }

                    StyledText {
                        width: parent.width
                        text: importModal.statusText()
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceVariantText
                        wrapMode: Text.WordWrap
                        visible: text !== ""
                    }

                    Repeater {
                        model: importModal.items

                        StyledRect {
                            id: row
                            required property var modelData

                            readonly property var event: modelData.event
                            readonly property var existing: modelData.existing

                            width: parent.width
                            height: rowColumn.implicitHeight + Theme.spacingM * 2
                            radius: Theme.cornerRadiusM
                            color: Theme.surfaceContainerLow

                            Rectangle {
                                width: Theme.spacingXS
                                radius: Theme.cornerRadiusXXS
                                anchors.left: parent.left
                                anchors.leftMargin: Theme.spacingS
                                anchors.top: parent.top
                                anchors.bottom: parent.bottom
                                anchors.topMargin: Theme.spacingM
                                anchors.bottomMargin: Theme.spacingM
                                color: row.existing ? row.existing.color : Theme.primary
                            }

                            Column {
                                id: rowColumn
                                anchors.left: parent.left
                                anchors.right: parent.right
                                anchors.top: parent.top
                                anchors.leftMargin: Theme.spacingM + Theme.spacingS + Theme.spacingXS
                                anchors.rightMargin: Theme.spacingM
                                anchors.topMargin: Theme.spacingM
                                spacing: Theme.spacingXS

                                StyledText {
                                    width: parent.width
                                    text: row.event.title
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Theme.fontWeightMedium
                                    color: Theme.surfaceText
                                    elide: Text.ElideRight
                                }

                                StyledText {
                                    width: parent.width
                                    text: importModal.timeLabel(row.event)
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    elide: Text.ElideRight
                                }

                                StyledText {
                                    width: parent.width
                                    text: row.event.location
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    elide: Text.ElideRight
                                    visible: text !== ""
                                }

                                StyledText {
                                    width: parent.width
                                    text: importModal.organizerLabel(row.event)
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    elide: Text.ElideRight
                                    visible: text !== ""
                                }

                                Row {
                                    width: parent.width
                                    spacing: Theme.spacingM
                                    visible: !!row.existing

                                    StyledText {
                                        text: row.existing ? I18n.tr("Already in %1", "import row note naming the calendar that holds this event").arg(row.existing.calendar) : ""
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.warning
                                        anchors.verticalCenter: parent.verticalCenter
                                    }

                                    DankButton {
                                        text: I18n.tr("Open", "import row button to open the event already on the calendar")
                                        buttonHeight: Theme.buttonHeightXS
                                        backgroundColor: Theme.secondaryContainer
                                        textColor: Theme.onSecondaryContainer
                                        anchors.verticalCenter: parent.verticalCenter
                                        onClicked: importModal.openExisting(row.existing)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        Rectangle {
            width: parent.width
            height: Theme.dividerWidth
            color: Theme.outlineVariant
        }

        Item {
            id: footer
            width: parent.width
            height: Theme.buttonHeightS + Theme.spacingM * 2

            StyledText {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingL
                anchors.right: actions.left
                anchors.rightMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                text: importModal.errorText
                color: Theme.error
                font.pixelSize: Theme.fontSizeSmall
                elide: Text.ElideRight
                visible: importModal.errorText !== ""
            }

            Row {
                id: actions
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingL
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingS

                DankDropdown {
                    width: Theme.fieldDefaultWidth
                    anchors.verticalCenter: parent.verticalCenter
                    visible: !importModal.noWritableCalendars
                    enabled: !importModal.importing
                    options: importModal.writable.map(c => c.name)
                    currentValue: importModal.targetCalendar ? importModal.targetCalendar.name : ""
                    onValueChanged: value => {
                        for (let i = 0; i < importModal.writable.length; i++) {
                            if (importModal.writable[i].name === value) {
                                importModal.calendarIndex = i;
                                return;
                            }
                        }
                    }
                }

                DankButton {
                    text: I18n.tr("Cancel", "import dialog button to close without importing")
                    backgroundColor: "transparent"
                    textColor: Theme.primary
                    onClicked: importModal.hide()
                }

                DankButton {
                    visible: importModal.noWritableCalendars
                    text: I18n.tr("Add a calendar", "import dialog button to add a calendar when none can hold events")
                    iconName: "add"
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    onClicked: {
                        importModal.addCalendarRequested();
                        importModal.hide();
                    }
                }

                DankButton {
                    visible: !importModal.noWritableCalendars
                    text: importModal.importLabel()
                    iconName: "check"
                    busy: importModal.importing
                    backgroundColor: Theme.primary
                    textColor: Theme.primaryText
                    enabled: !importModal.importing && !importModal.loading && importModal.pendingItems.length > 0
                    onClicked: importModal.importPending()
                }
            }
        }
    }

    Shortcut {
        sequence: "Escape"
        enabled: importModal.visible
        onActivated: importModal.hide()
    }

    FloatingWindowControls {
        id: windowControls
        targetWindow: importModal
    }
}
