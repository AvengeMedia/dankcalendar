import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: root

    property date displayDate: new Date()
    property int eventsVersion: 0
    readonly property int daysAhead: 14

    signal eventClicked(var event)

    Connections {
        target: DankCalService
        function onEventsUpdated() {
            root.eventsVersion++;
        }
    }

    function dayLabel(d) {
        const now = new Date();
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const diff = Math.round((d.getTime() - today.getTime()) / 86400000);
        switch (diff) {
        case 0:
            return I18n.tr("Today", "agenda section header for the current day");
        case 1:
            return I18n.tr("Tomorrow", "agenda section header for the next day");
        default:
            return Qt.formatDate(d, "dddd, MMM d");
        }
    }

    function durationLabel(ev) {
        if (ev.allDay)
            return "";
        const mins = Math.round((ev.end.getTime() - ev.start.getTime()) / 60000);
        if (mins <= 0)
            return "";
        const h = Math.floor(mins / 60);
        const m = mins % 60;
        if (h === 0)
            return I18n.tr("%1m", "event duration in minutes on agenda card, %1 is minutes").arg(m);
        return m === 0 ? I18n.tr("%1h", "event duration in hours on agenda card, %1 is hours").arg(h) : I18n.tr("%1h%2m", "event duration on agenda card, %1 is hours and %2 is minutes").arg(h).arg(m);
    }

    readonly property var sections: {
        eventsVersion;
        const out = [];
        for (let i = 0; i < daysAhead; i++) {
            const d = new Date(displayDate.getFullYear(), displayDate.getMonth(), displayDate.getDate() + i);
            const evs = DankCalService.eventsForDay(d);
            if (evs.length === 0)
                continue;
            const cards = evs.map(ev => {
                const card = Object.assign({}, ev);
                card.time = ev.allDay ? I18n.tr("All day", "time column label for all-day events on agenda card") : SettingsData.formatTime(ev.start);
                card.duration = durationLabel(ev);
                return card;
            });
            out.push({
                "label": dayLabel(d),
                "events": cards
            });
        }
        return out;
    }

    DankFlickable {
        anchors.fill: parent
        contentWidth: width
        contentHeight: agendaColumn.implicitHeight
        clip: true

        Column {
            id: agendaColumn
            width: root.width
            spacing: Theme.spacingM
            padding: 0

            StyledText {
                visible: root.sections.length === 0
                text: DankCalService.connected ? I18n.tr("Nothing scheduled in the next %1 days", "agenda empty state, %1 is the number of days ahead").arg(root.daysAhead) : I18n.tr("Waiting for the dankcalendar daemon...", "agenda placeholder while the daemon is not connected")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceVariantText
                width: parent.width
            }

            Repeater {
                model: ScriptModel {
                    values: root.sections
                }

                Column {
                    id: section
                    required property var modelData
                    width: root.width
                    spacing: Theme.spacingS

                    StyledText {
                        text: parent.modelData.label
                        font.pixelSize: Theme.fontSizeLarge
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        width: parent.width
                    }

                    Repeater {
                        model: ScriptModel {
                            values: section.modelData.events
                        }

                        StyledRect {
                            id: card
                            required property var modelData
                            width: root.width
                            height: Math.max(76, contentRow.implicitHeight + Theme.spacingM * 2)
                            color: Theme.surfaceContainer
                            radius: Theme.cornerRadius

                            Row {
                                id: contentRow
                                anchors.left: parent.left
                                anchors.right: parent.right
                                anchors.verticalCenter: parent.verticalCenter
                                anchors.leftMargin: Theme.spacingM
                                anchors.rightMargin: Theme.spacingM
                                spacing: Theme.spacingM

                                Rectangle {
                                    width: 4
                                    height: 44
                                    anchors.verticalCenter: parent.verticalCenter
                                    radius: 2
                                    color: card.modelData.color
                                }

                                Column {
                                    anchors.verticalCenter: parent.verticalCenter
                                    width: 84
                                    spacing: 2

                                    StyledText {
                                        text: card.modelData.time
                                        font.pixelSize: Theme.fontSizeMedium
                                        font.weight: Font.Medium
                                        color: Theme.surfaceText
                                        isMonospace: true
                                        width: parent.width
                                    }

                                    StyledText {
                                        text: card.modelData.duration
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.surfaceVariantText
                                        visible: text !== ""
                                        width: parent.width
                                    }
                                }

                                Column {
                                    anchors.verticalCenter: parent.verticalCenter
                                    width: parent.width - 4 - 84 - Theme.spacingM * 2
                                    spacing: Theme.spacingXS

                                    StyledText {
                                        text: card.modelData.title
                                        font.pixelSize: Theme.fontSizeLarge
                                        font.weight: Font.Medium
                                        color: Theme.surfaceText
                                        width: parent.width
                                        wrapMode: Text.WordWrap
                                        maximumLineCount: 2
                                        elide: Text.ElideRight
                                    }

                                    Row {
                                        spacing: Theme.spacingS

                                        Rectangle {
                                            width: 8
                                            height: 8
                                            radius: 4
                                            anchors.verticalCenter: parent.verticalCenter
                                            color: card.modelData.color
                                        }

                                        StyledText {
                                            text: card.modelData.calendar
                                            font.pixelSize: Theme.fontSizeSmall
                                            color: Theme.surfaceVariantText
                                            anchors.verticalCenter: parent.verticalCenter
                                        }
                                    }
                                }
                            }

                            StateLayer {
                                stateColor: Theme.primary
                                cornerRadius: parent.radius
                                onClicked: root.eventClicked(card.modelData)
                            }
                        }
                    }
                }
            }
        }
    }
}
