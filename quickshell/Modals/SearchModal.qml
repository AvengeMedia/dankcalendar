pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Popup {
    id: root

    signal eventSelected(var event)

    property var results: []
    property bool loading: false
    property string activeQuery: ""
    property string errorText: ""
    property int selectedIndex: 0

    readonly property bool hasQuery: searchInput.text.trim().length > 0
    readonly property real searchBarH: 56
    readonly property real rowH: 64
    readonly property real statusH: 92
    readonly property real maxResultsH: Math.min(430, (parent ? parent.height : 600) * 0.55)

    readonly property var rows: {
        const now = new Date();
        const upcoming = [];
        const past = [];
        for (let i = 0; i < results.length; i++) {
            const ev = results[i];
            if (ev.end >= now)
                upcoming.push(ev);
            else
                past.push(ev);
        }
        upcoming.sort((a, b) => a.start - b.start);
        past.sort((a, b) => b.start - a.start);
        return upcoming.concat(past).map(ev => ({
                    "rowId": ev.id + "|" + ev.start.getTime(),
                    "event": ev
                }));
    }

    readonly property real resultsH: {
        if (!hasQuery)
            return 0;
        if (rows.length === 0)
            return statusH;
        return Math.min(rows.length * rowH + Theme.spacingS, maxResultsH);
    }

    onRowsChanged: selectedIndex = 0

    function show() {
        open();
    }

    function _dateLabel(ev) {
        const now = new Date();
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const day = new Date(ev.start.getFullYear(), ev.start.getMonth(), ev.start.getDate());
        const diff = Math.round((day.getTime() - today.getTime()) / 86400000);
        switch (diff) {
        case -1:
            return I18n.tr("Yesterday", "search result date label for previous day");
        case 0:
            return I18n.tr("Today", "search result date label for current day");
        case 1:
            return I18n.tr("Tomorrow", "search result date label for next day");
        }
        if (ev.start.getFullYear() === now.getFullYear())
            return Qt.formatDate(ev.start, "MMM d");
        return Qt.formatDate(ev.start, "MMM d, yyyy");
    }

    function _subtitle(ev) {
        const parts = [];
        parts.push(ev.allDay ? I18n.tr("All day", "search result subtitle for all-day events") : SettingsData.formatTime(ev.start));
        if (ev.calendar)
            parts.push(ev.calendar);
        if (ev.location)
            parts.push(ev.location);
        return parts.join(" · ");
    }

    function _statusTitle() {
        if (loading)
            return I18n.tr("Searching", "search status title while search is running");
        if (errorText !== "")
            return I18n.tr("Search failed", "search status title when search errors");
        if (activeQuery === "")
            return I18n.tr("Keep typing", "search status title when query is too short");
        return I18n.tr("No matching events", "search status title when no results found");
    }

    function _statusSubtitle() {
        if (loading)
            return "";
        if (errorText !== "")
            return errorText;
        if (activeQuery === "")
            return I18n.tr("Type at least 2 characters to search events.", "search status hint when query is too short");
        return I18n.tr("Try a different search.", "search status hint when no results found");
    }

    function _activate(index) {
        if (index < 0 || index >= rows.length)
            return;
        const ev = rows[index].event;
        eventSelected(ev);
        close();
    }

    function _selectDelta(delta) {
        if (rows.length === 0)
            return;
        selectedIndex = Math.max(0, Math.min(rows.length - 1, selectedIndex + delta));
        resultsList.positionViewAtIndex(selectedIndex, ListView.Contain);
    }

    function _handleKey(event) {
        switch (event.key) {
        case Qt.Key_Down:
            _selectDelta(1);
            event.accepted = true;
            return;
        case Qt.Key_Up:
            _selectDelta(-1);
            event.accepted = true;
            return;
        case Qt.Key_PageDown:
            _selectDelta(6);
            event.accepted = true;
            return;
        case Qt.Key_PageUp:
            _selectDelta(-6);
            event.accepted = true;
            return;
        case Qt.Key_Return:
        case Qt.Key_Enter:
            if (rows.length > 0)
                _activate(selectedIndex);
            else
                _runSearch();
            event.accepted = true;
            return;
        case Qt.Key_Escape:
            close();
            event.accepted = true;
            return;
        }
        event.accepted = false;
    }

    function _runSearch() {
        searchDebounce.stop();
        const q = searchInput.text.trim();
        if (q.length < 2) {
            activeQuery = "";
            loading = false;
            errorText = "";
            results = [];
            return;
        }

        activeQuery = q;
        loading = true;
        DankCalService.searchEvents(q, response => {
            if (root.activeQuery !== q)
                return;
            root.loading = false;
            root.errorText = response.error || "";
            root.results = response.events || [];
        });
    }

    parent: Overlay.overlay
    x: Math.round((parent.width - width) / 2)
    y: Math.round(parent.height * 0.12)
    width: Math.min(600, parent.width - Theme.spacingXL * 2)
    modal: true
    padding: Theme.spacingS
    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside

    onOpened: searchInput.forceActiveFocus()
    onClosed: {
        searchDebounce.stop();
        searchInput.text = "";
        activeQuery = "";
        loading = false;
        errorText = "";
        results = [];
    }

    Overlay.modal: Rectangle {
        color: Qt.rgba(0, 0, 0, 0.4)
    }

    background: Rectangle {
        color: Theme.surfaceContainer
        radius: Theme.cornerRadiusLarge
        border.width: 1
        border.color: Theme.outlineMedium
    }

    Timer {
        id: searchDebounce
        interval: 250
        repeat: false
        onTriggered: root._runSearch()
    }

    contentItem: Item {
        implicitHeight: root.searchBarH + resultsContainer.height

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        Rectangle {
            id: searchBar
            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            height: root.searchBarH
            radius: Theme.cornerRadius
            color: Theme.surfaceContainerHigh

            Rectangle {
                id: leadingWell
                width: 36
                height: 36
                radius: height / 2
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                color: searchInput.activeFocus ? Theme.primaryContainer : Theme.surfaceContainerHighest

                Behavior on color {
                    ColorAnimation {
                        duration: Theme.shorterDuration
                        easing.type: Theme.standardEasing
                    }
                }

                DankIcon {
                    anchors.centerIn: parent
                    name: "search"
                    size: 20
                    color: searchInput.activeFocus ? Theme.primary : Theme.surfaceVariantText
                }
            }

            DankActionButton {
                id: clearButton
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                iconName: "close"
                iconSize: 16
                visible: searchInput.text.length > 0
                onClicked: {
                    searchInput.text = "";
                    searchInput.forceActiveFocus();
                }
            }

            Text {
                anchors.left: leadingWell.right
                anchors.leftMargin: Theme.spacingM
                anchors.right: clearButton.left
                anchors.rightMargin: Theme.spacingS
                anchors.verticalCenter: parent.verticalCenter
                text: I18n.tr("Search events", "search modal input placeholder")
                font.family: Theme.fontFamily
                font.pixelSize: 18
                font.weight: Font.Medium
                color: Theme.outlineButton
                visible: searchInput.text.length === 0
                clip: true
            }

            TextInput {
                id: searchInput
                anchors.left: leadingWell.right
                anchors.leftMargin: Theme.spacingM
                anchors.right: clearButton.left
                anchors.rightMargin: Theme.spacingS
                anchors.verticalCenter: parent.verticalCenter
                font.family: Theme.fontFamily
                font.pixelSize: 18
                font.weight: Font.Medium
                color: Theme.surfaceText
                selectionColor: Theme.primary
                selectedTextColor: Theme.primaryText
                clip: true
                focus: true
                onTextChanged: searchDebounce.restart()
                Keys.onPressed: event => root._handleKey(event)
            }
        }

        Item {
            id: resultsContainer
            anchors.top: searchBar.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            height: root.resultsH
            clip: true

            Behavior on height {
                NumberAnimation {
                    duration: 110
                    easing.type: Theme.standardEasing
                }
            }

            DankListView {
                id: resultsList
                anchors.fill: parent
                anchors.topMargin: Theme.spacingS
                clip: true
                visible: root.rows.length > 0

                model: ScriptModel {
                    values: root.rows
                    objectProp: "rowId"
                }

                delegate: Item {
                    id: delegateRoot
                    required property var modelData
                    required property int index

                    width: resultsList.width
                    height: root.rowH

                    Rectangle {
                        anchors.fill: parent
                        anchors.topMargin: 3
                        anchors.bottomMargin: 3
                        radius: Theme.cornerRadius
                        color: delegateRoot.index === root.selectedIndex ? Theme.primaryPressed : "transparent"

                        Behavior on color {
                            ColorAnimation {
                                duration: Theme.shorterDuration
                                easing.type: Theme.standardEasing
                            }
                        }

                        Rectangle {
                            id: iconWell
                            width: 40
                            height: 40
                            radius: Theme.cornerRadius
                            anchors.left: parent.left
                            anchors.leftMargin: Theme.spacingS
                            anchors.verticalCenter: parent.verticalCenter
                            color: Theme.surfaceContainerHigh
                            border.width: 1
                            border.color: Theme.withAlpha(Theme.outline, 0.12)

                            DankIcon {
                                anchors.centerIn: parent
                                name: "event"
                                size: 22
                                color: delegateRoot.modelData.event.color
                            }
                        }

                        Column {
                            anchors.left: iconWell.right
                            anchors.leftMargin: Theme.spacingM
                            anchors.right: datePill.left
                            anchors.rightMargin: Theme.spacingM
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 2

                            StyledText {
                                width: parent.width
                                text: delegateRoot.modelData.event.title
                                font.pixelSize: Theme.fontSizeMedium
                                font.weight: Font.Medium
                                color: Theme.surfaceText
                                maximumLineCount: 1
                                elide: Text.ElideRight
                            }

                            StyledText {
                                width: parent.width
                                text: root._subtitle(delegateRoot.modelData.event)
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.surfaceVariantText
                                maximumLineCount: 1
                                elide: Text.ElideRight
                                visible: text !== ""
                            }
                        }

                        Rectangle {
                            id: datePill
                            width: dateText.implicitWidth + Theme.spacingS * 2
                            height: 22
                            radius: height / 2
                            anchors.right: parent.right
                            anchors.rightMargin: Theme.spacingS
                            anchors.verticalCenter: parent.verticalCenter
                            color: Theme.surfaceVariantAlpha

                            StyledText {
                                id: dateText
                                anchors.centerIn: parent
                                text: root._dateLabel(delegateRoot.modelData.event)
                                font.pixelSize: Theme.fontSizeSmall - 1
                                color: Theme.surfaceVariantText
                            }
                        }

                        StateLayer {
                            stateColor: Theme.primary
                            cornerRadius: parent.radius
                            onClicked: root._activate(delegateRoot.index)
                        }
                    }
                }
            }

            Row {
                anchors.centerIn: parent
                spacing: Theme.spacingM
                visible: root.hasQuery && root.rows.length === 0

                Rectangle {
                    width: 40
                    height: 40
                    radius: Theme.cornerRadius
                    anchors.verticalCenter: parent.verticalCenter
                    color: Theme.surfaceContainerHigh

                    DankIcon {
                        anchors.centerIn: parent
                        name: root.errorText !== "" ? "error" : root.activeQuery === "" ? "search" : "search_off"
                        size: 22
                        color: Theme.surfaceVariantText
                        visible: !root.loading
                    }

                    DankSpinner {
                        anchors.centerIn: parent
                        size: 22
                        visible: root.loading
                    }
                }

                Column {
                    anchors.verticalCenter: parent.verticalCenter
                    width: Math.min(420, resultsContainer.width - 88)
                    spacing: 2

                    StyledText {
                        width: parent.width
                        text: root._statusTitle()
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        elide: Text.ElideRight
                    }

                    StyledText {
                        width: parent.width
                        text: root._statusSubtitle()
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceVariantText
                        maximumLineCount: 2
                        wrapMode: Text.WordWrap
                        elide: Text.ElideRight
                        visible: text !== ""
                    }
                }
            }
        }
    }
}
