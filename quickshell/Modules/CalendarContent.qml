import QtQuick
import qs.Common
import qs.Services
import qs.Widgets
import "views"

Item {
    id: root

    property string currentView: "month"
    property date displayDate: new Date()
    property date selectedDate: new Date()
    property string selectedEventKey: ""
    property date today: new Date()

    signal todayRequested
    signal previousRequested
    signal nextRequested
    signal shiftDaysRequested(int days)
    signal createEventRequested
    signal eventClicked(var event)
    signal createTaskRequested
    signal taskClicked(var task)
    signal settingsRequested
    signal searchRequested
    signal goToDateRequested
    signal daySelected(date day)
    signal dayActivated(date day)
    signal viewDayRequested(date day)

    function headerTitle() {
        switch (currentView) {
        case "day":
            return Qt.formatDate(displayDate, "dddd, MMMM d, yyyy");
        case "week":
            return SettingsData.monthName(displayDate.getMonth()) + " " + displayDate.getFullYear();
        case "agenda":
            return I18n.tr("Agenda", "header title when the agenda view is active");
        case "tasks":
            return I18n.tr("Tasks", "header title when the tasks view is active");
        default:
            return SettingsData.monthName(displayDate.getMonth()) + " " + displayDate.getFullYear();
        }
    }

    Column {
        anchors.fill: parent
        spacing: 0

        Item {
            width: parent.width
            height: 56

            Row {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingL
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingM

                DankButton {
                    text: I18n.tr("Today", "header button that jumps to the current date")
                    iconName: "today"
                    buttonHeight: 36
                    horizontalPadding: Theme.spacingM
                    backgroundColor: "transparent"
                    textColor: Theme.surfaceText
                    onClicked: root.todayRequested()
                }

                DankActionButton {
                    iconName: I18n.isRtl ? "chevron_right" : "chevron_left"
                    iconSize: Theme.iconSize - 4
                    iconColor: Theme.surfaceText
                    circular: true
                    onClicked: root.previousRequested()
                    anchors.verticalCenter: parent.verticalCenter
                }

                DankActionButton {
                    iconName: I18n.isRtl ? "chevron_left" : "chevron_right"
                    iconSize: Theme.iconSize - 4
                    iconColor: Theme.surfaceText
                    circular: true
                    onClicked: root.nextRequested()
                    anchors.verticalCenter: parent.verticalCenter
                }

                Rectangle {
                    width: headerTitleText.implicitWidth + Theme.spacingM * 2
                    height: 36
                    radius: Theme.cornerRadius
                    color: "transparent"
                    anchors.verticalCenter: parent.verticalCenter

                    StyledText {
                        id: headerTitleText
                        anchors.centerIn: parent
                        text: root.headerTitle()
                        font.pixelSize: Theme.fontSizeXLarge
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                    }

                    StateLayer {
                        stateColor: Theme.primary
                        cornerRadius: parent.radius
                        onClicked: root.goToDateRequested()
                    }
                }
            }

            Row {
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingL
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingS

                DankActionButton {
                    iconName: "refresh"
                    iconColor: DankCalService.eventsLoading ? Theme.primary : Theme.surfaceText
                    circular: true
                    enabled: DankCalService.connected
                    onClicked: DankCalService.refreshAll()

                    RotationAnimation on rotation {
                        running: DankCalService.eventsLoading
                        from: 0
                        to: 360
                        duration: 900
                        loops: Animation.Infinite
                        onRunningChanged: {
                            if (!running)
                                rotation = 0;
                        }
                    }
                }

                DankActionButton {
                    iconName: "search"
                    iconColor: Theme.surfaceText
                    circular: true
                    onClicked: root.searchRequested()
                }

                DankActionButton {
                    iconName: "settings"
                    iconColor: Theme.surfaceText
                    circular: true
                    onClicked: root.settingsRequested()
                }
            }
        }

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.outlineLight
        }

        Item {
            width: parent.width
            height: parent.height - 57

            Loader {
                id: viewLoader
                anchors.fill: parent
                anchors.margins: Theme.spacingM

                sourceComponent: {
                    switch (root.currentView) {
                    case "day":
                        return dayComponent;
                    case "week":
                        return weekComponent;
                    case "agenda":
                        return agendaComponent;
                    case "tasks":
                        return tasksComponent;
                    default:
                        return monthComponent;
                    }
                }
            }

            Component {
                id: monthComponent
                MonthView {
                    displayDate: root.displayDate
                    today: root.today
                    selectedDate: root.selectedDate
                    onDaySelected: day => root.daySelected(day)
                    onDayActivated: day => root.dayActivated(day)
                    onViewDayRequested: day => root.viewDayRequested(day)
                    onEventClicked: ev => root.eventClicked(ev)
                    onPreviousRequested: root.previousRequested()
                    onNextRequested: root.nextRequested()
                }
            }

            Component {
                id: weekComponent
                WeekView {
                    displayDate: root.displayDate
                    today: root.today
                    selectedDate: root.selectedDate
                    selectedEventKey: root.selectedEventKey
                    onEventClicked: ev => root.eventClicked(ev)
                    onShiftDaysRequested: days => root.shiftDaysRequested(days)
                }
            }

            Component {
                id: dayComponent
                DayView {
                    displayDate: root.displayDate
                    selectedEventKey: root.selectedEventKey
                    onEventClicked: ev => root.eventClicked(ev)
                    onPreviousRequested: root.previousRequested()
                    onNextRequested: root.nextRequested()
                }
            }

            Component {
                id: agendaComponent
                AgendaView {
                    displayDate: root.displayDate
                    selectedEventKey: root.selectedEventKey
                    onEventClicked: ev => root.eventClicked(ev)
                }
            }

            Component {
                id: tasksComponent
                TasksView {
                    onTaskClicked: task => root.taskClicked(task)
                    onCreateTaskRequested: root.createTaskRequested()
                }
            }
        }
    }
}
