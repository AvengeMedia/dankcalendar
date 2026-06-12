import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: root

    property date displayDate: new Date()
    property date today: new Date()
    property date selectedDate: new Date()
    property int eventsVersion: 0

    signal daySelected(date day)
    signal eventClicked(var event)

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

    readonly property int firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
    readonly property real weekGutter: SettingsData.showWeekNumbers ? 28 : 0

    function isoWeekNumber(d) {
        const date = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
        const dayNum = date.getUTCDay() || 7;
        date.setUTCDate(date.getUTCDate() + 4 - dayNum);
        const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1));
        return Math.ceil(((date - yearStart) / 86400000 + 1) / 7);
    }

    readonly property int gridYear: displayDate.getFullYear()
    readonly property int gridMonth: displayDate.getMonth()

    readonly property date firstOfMonth: {
        const d = new Date(gridYear, gridMonth, 1);
        return d;
    }

    readonly property int leadingDays: {
        const offset = firstOfMonth.getDay() - firstDayOfWeek;
        return offset < 0 ? offset + 7 : offset;
    }

    function isSameDay(a, b) {
        return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
    }

    function cellDate(index) {
        return new Date(gridYear, gridMonth, 1 + index - leadingDays);
    }

    Column {
        anchors.fill: parent
        spacing: 0

        Row {
            width: parent.width
            height: 32

            Item {
                width: root.weekGutter
                height: parent.height
            }

            Repeater {
                model: 7
                Item {
                    required property int index
                    width: (parent.width - root.weekGutter) / 7
                    height: parent.height

                    StyledText {
                        anchors.centerIn: parent
                        text: SettingsData.dayName((index + root.firstDayOfWeek) % 7)
                        font.pixelSize: Theme.fontSizeSmall
                        font.weight: Font.Medium
                        color: Theme.surfaceVariantText
                    }
                }
            }
        }

        Row {
            width: parent.width
            height: parent.height - 32

            Column {
                visible: SettingsData.showWeekNumbers
                width: root.weekGutter

                Repeater {
                    model: 6

                    Item {
                        required property int index
                        width: parent.width
                        height: grid.cellHeight

                        StyledText {
                            anchors.centerIn: parent
                            // ISO weeks are Monday-based; the row's 4th day avoids
                            // ambiguity when the grid starts on another day
                            text: root.isoWeekNumber(root.cellDate(index * 7 + 3))
                            font.pixelSize: 10
                            color: Theme.surfaceVariantText
                            isMonospace: true
                        }
                    }
                }
            }

            Grid {
                id: grid
                width: parent.width - root.weekGutter
                height: parent.height
                columns: 7
                rows: 6

                readonly property real cellWidth: width / columns
                readonly property real cellHeight: height / rows

                Repeater {
                    model: 42

                    Item {
                        id: dayCell
                        required property int index

                        readonly property date cellDate: root.cellDate(index)
                        readonly property bool inCurrentMonth: cellDate.getMonth() === root.gridMonth
                        readonly property bool isToday: root.isSameDay(cellDate, root.today)
                        readonly property bool isSelected: root.isSameDay(cellDate, root.selectedDate)
                        readonly property var cellEvents: {
                            root.eventsVersion;
                            return DankCalService.eventsForDay(cellDate);
                        }

                        // 36px day badge area + 14px reserved for "+N more"; 20px per chip
                        readonly property int maxChips: Math.max(0, Math.min(3, Math.floor((height - 50) / 20)))

                        width: grid.cellWidth
                        height: grid.cellHeight

                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                            border.width: 1
                            border.color: Theme.outlineLight
                        }

                        Rectangle {
                            visible: parent.isSelected
                            anchors.fill: parent
                            anchors.margins: 1
                            color: Theme.primaryBackground
                            border.color: Theme.primary
                            border.width: 1
                            radius: Theme.cornerRadiusSmall
                        }

                        Item {
                            id: dayBadge
                            anchors.top: parent.top
                            anchors.left: parent.left
                            anchors.topMargin: Theme.spacingS
                            anchors.leftMargin: Theme.spacingS
                            width: 28
                            height: 28

                            Rectangle {
                                anchors.fill: parent
                                radius: width / 2
                                color: Theme.primary
                                visible: parent.parent.isToday
                            }

                            StyledText {
                                anchors.centerIn: parent
                                text: parent.parent.cellDate.getDate()
                                font.pixelSize: parent.parent.isToday ? Theme.fontSizeMedium : Theme.fontSizeSmall
                                font.weight: parent.parent.isToday ? Font.Medium : Font.Normal
                                color: {
                                    if (parent.parent.isToday)
                                        return Theme.primaryText;
                                    if (!parent.parent.inCurrentMonth)
                                        return Theme.surfaceVariantText;
                                    return Theme.surfaceText;
                                }
                                opacity: parent.parent.inCurrentMonth ? 1.0 : 0.5
                            }
                        }

                        Column {
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.bottom: parent.bottom
                            anchors.margins: Theme.spacingXS
                            spacing: 2
                            clip: true
                            // Above the cell's day-select MouseArea so chip clicks win.
                            z: 1

                            Repeater {
                                model: ScriptModel {
                                    values: dayCell.cellEvents.slice(0, dayCell.maxChips)
                                }

                                Rectangle {
                                    required property var modelData
                                    width: parent.width
                                    height: 18
                                    radius: 4
                                    clip: true
                                    color: Theme.withAlpha(modelData.color, 0.18)

                                    Row {
                                        anchors.left: parent.left
                                        anchors.right: parent.right
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.leftMargin: 4
                                        anchors.rightMargin: 4
                                        spacing: 4

                                        Rectangle {
                                            width: 3
                                            height: 12
                                            radius: 1.5
                                            color: parent.parent.modelData.color
                                            anchors.verticalCenter: parent.verticalCenter
                                        }

                                        StyledText {
                                            anchors.verticalCenter: parent.verticalCenter
                                            text: parent.parent.modelData.title
                                            font.pixelSize: 11
                                            color: Theme.surfaceText
                                            wrapMode: Text.NoWrap
                                            maximumLineCount: 1
                                            elide: Text.ElideRight
                                            width: parent.width - 10
                                        }
                                    }

                                    MouseArea {
                                        anchors.fill: parent
                                        cursorShape: Qt.PointingHandCursor
                                        hoverEnabled: true
                                        onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                                        onExited: chipTooltip.hide()
                                        onClicked: mouse => {
                                            mouse.accepted = true;
                                            chipTooltip.hide();
                                            root.eventClicked(parent.modelData);
                                        }
                                    }
                                }
                            }

                            StyledText {
                                visible: parent.parent.cellEvents.length > parent.parent.maxChips
                                text: I18n.tr("+%1 more", "overflow label in month grid day cell, %1 is the number of hidden events").arg(parent.parent.cellEvents.length - parent.parent.maxChips)
                                font.pixelSize: 10
                                color: Theme.surfaceVariantText
                            }
                        }

                        MouseArea {
                            anchors.fill: parent
                            onClicked: root.daySelected(parent.cellDate)
                        }
                    }
                }
            }
        }
    }
}
