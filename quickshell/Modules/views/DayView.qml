import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: root

    property date displayDate: new Date()
    property int eventsVersion: 0

    signal eventClicked(var event)

    readonly property int hourCount: 24
    readonly property real hourHeight: 56
    readonly property real timeColumnWidth: 72

    Connections {
        target: DankCalService
        function onEventsUpdated() {
            root.eventsVersion++;
        }
    }

    DankTooltipV2 {
        id: chipTooltip
    }

    function eventTooltip(ev) {
        if (ev.allDay)
            return ev.title + " · " + I18n.tr("All day", "all-day marker in event tooltip") + (ev.calendar ? " · " + ev.calendar : "");
        return ev.title + " · " + SettingsData.formatTime(ev.start) + " – " + SettingsData.formatTime(ev.end) + (ev.calendar ? " · " + ev.calendar : "");
    }

    function hourLabel(hour) {
        if (hour === 0)
            return "";
        if (SettingsData.use24HourTime)
            return (hour < 10 ? "0" + hour : hour) + ":00";
        const h = hour % 12 === 0 ? 12 : hour % 12;
        return hour < 12 ? I18n.tr("%1 AM", "morning hour label in time gutter").arg(h) : I18n.tr("%1 PM", "afternoon hour label in time gutter").arg(h);
    }

    readonly property var dayEvents: {
        eventsVersion;
        return DankCalService.eventsForDay(displayDate);
    }

    readonly property var allDayEvents: dayEvents.filter(ev => ev.allDay)

    readonly property var timedEvents: {
        const dayStart = new Date(displayDate.getFullYear(), displayDate.getMonth(), displayDate.getDate());
        const dayEnd = new Date(dayStart.getTime() + 86400000);
        const out = [];
        for (let i = 0; i < dayEvents.length; i++) {
            const ev = dayEvents[i];
            if (ev.allDay)
                continue;
            const s = Math.max(ev.start.getTime(), dayStart.getTime());
            const e = Math.min(ev.end.getTime(), dayEnd.getTime());
            const decorated = Object.assign({}, ev);
            decorated.startHour = (s - dayStart.getTime()) / 3600000;
            decorated.durationHours = Math.max((e - s) / 3600000, 0.5);
            out.push(decorated);
        }
        return out;
    }

    Column {
        id: allDayStrip
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.leftMargin: root.timeColumnWidth
        spacing: 2
        visible: root.allDayEvents.length > 0

        Repeater {
            model: ScriptModel {
                values: root.allDayEvents
            }

            Rectangle {
                required property var modelData
                width: parent.width
                height: 22
                radius: 4
                clip: true
                color: Theme.withAlpha(modelData.color, 0.22)
                border.color: modelData.color
                border.width: 1

                StyledText {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.leftMargin: Theme.spacingS
                    anchors.rightMargin: Theme.spacingS
                    anchors.verticalCenter: parent.verticalCenter
                    text: modelData.title + "  ·  " + I18n.tr("all day", "suffix on all-day event chip in day view")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.surfaceText
                    wrapMode: Text.NoWrap
                    maximumLineCount: 1
                    elide: Text.ElideRight
                }

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    hoverEnabled: true
                    onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                    onExited: chipTooltip.hide()
                    onClicked: {
                        chipTooltip.hide();
                        root.eventClicked(parent.modelData);
                    }
                }
            }
        }
    }

    DankFlickable {
        anchors.top: allDayStrip.visible ? allDayStrip.bottom : parent.top
        anchors.topMargin: allDayStrip.visible ? Theme.spacingS : 0
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        contentHeight: root.hourHeight * root.hourCount
        clip: true

        Item {
            width: parent.width
            height: root.hourHeight * root.hourCount

            Column {
                anchors.left: parent.left
                width: root.timeColumnWidth
                spacing: 0

                Repeater {
                    model: root.hourCount

                    Item {
                        required property int index
                        width: parent.width
                        height: root.hourHeight

                        StyledText {
                            anchors.right: parent.right
                            anchors.rightMargin: Theme.spacingS
                            anchors.top: parent.top
                            anchors.topMargin: -6
                            text: root.hourLabel(index)
                            font.pixelSize: 11
                            color: Theme.surfaceVariantText
                            isMonospace: true
                        }
                    }
                }
            }

            Item {
                anchors.right: parent.right
                width: parent.width - root.timeColumnWidth
                height: parent.height

                Repeater {
                    model: root.hourCount

                    Rectangle {
                        required property int index
                        y: index * root.hourHeight
                        width: parent.width
                        height: 1
                        color: Theme.gridLine
                    }
                }

                Repeater {
                    model: ScriptModel {
                        values: root.timedEvents
                    }

                    Rectangle {
                        required property var modelData
                        x: 8
                        y: modelData.startHour * root.hourHeight
                        width: parent.width - 16
                        height: modelData.durationHours * root.hourHeight - 4
                        radius: Theme.cornerRadiusSmall
                        clip: true
                        color: Theme.withAlpha(modelData.color, 0.22)
                        border.color: modelData.color
                        border.width: 1

                        Row {
                            anchors.fill: parent
                            anchors.margins: Theme.spacingS
                            spacing: Theme.spacingS

                            Rectangle {
                                width: 3
                                height: parent.height - 4
                                anchors.verticalCenter: parent.verticalCenter
                                radius: 1.5
                                color: parent.parent.modelData.color
                            }

                            Column {
                                anchors.verticalCenter: parent.verticalCenter
                                width: parent.width - 12

                                StyledText {
                                    text: parent.parent.parent.modelData.title
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.Medium
                                    color: Theme.surfaceText
                                    width: parent.width
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
                                    elide: Text.ElideRight
                                }

                                StyledText {
                                    visible: parent.parent.parent.modelData.durationHours >= 0.75
                                    text: parent.parent.parent.modelData.calendar
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    width: parent.width
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
                                    elide: Text.ElideRight
                                }
                            }
                        }

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            hoverEnabled: true
                            onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                            onExited: chipTooltip.hide()
                            onClicked: {
                                chipTooltip.hide();
                                root.eventClicked(parent.modelData);
                            }
                        }
                    }
                }
            }
        }
    }
}
