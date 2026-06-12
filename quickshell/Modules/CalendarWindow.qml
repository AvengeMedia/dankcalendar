import QtQuick
import Quickshell
import qs.Common
import qs.Modals
import qs.Services
import qs.Widgets

FloatingWindow {
    id: window

    signal hideRequested

    property bool isCompactMode: width < 760
    property bool menuVisible: !isCompactMode
    property string currentView: "month"
    property date displayDate: new Date()
    property date selectedDate: new Date()
    property date today: new Date()

    function shiftDisplayDate(direction) {
        const d = new Date(displayDate);
        switch (currentView) {
        case "day":
            d.setDate(d.getDate() + direction);
            break;
        case "week":
            d.setDate(d.getDate() + direction * 7);
            break;
        case "agenda":
            d.setDate(d.getDate() + direction * 7);
            break;
        default:
            d.setMonth(d.getMonth() + direction);
            break;
        }
        displayDate = d;
    }

    function goToToday() {
        const t = new Date();
        today = t;
        displayDate = t;
        selectedDate = t;
    }

    function openSettings() {
        settingsLoader.active = true;
        settingsLoader.item.show();
    }

    function openAddAccount() {
        accountLoader.active = true;
        accountLoader.item.show();
    }

    function openEventDetails(event) {
        eventLoader.active = true;
        eventLoader.item.show(event);
    }

    function openCreateEvent() {
        eventLoader.active = true;
        eventLoader.item.showCreate(selectedDate);
    }

    function openSearch() {
        searchLoader.active = true;
        searchLoader.item.show();
    }

    function goToEvent(event) {
        displayDate = event.start;
        selectedDate = event.start;
        openEventDetails(event);
    }

    onDisplayDateChanged: DankCalService.focusDate = displayDate
    Component.onCompleted: DankCalService.focusDate = displayDate

    title: I18n.tr("Calendar", "main window title")
    minimumSize: Qt.size(560, 400)
    implicitWidth: 1100
    implicitHeight: screen ? Math.min(820, screen.height - 100) : 820
    color: Theme.surface
    visible: true

    onIsCompactModeChanged: {
        if (!isCompactMode)
            menuVisible = true;
    }

    FocusScope {
        anchors.fill: parent
        focus: true

        Shortcut {
            sequences: [StandardKey.Find]
            onActivated: window.openSearch()
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
                    onDoubleClicked: windowControls.tryToggleMaximize()
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
                        visible: window.isCompactMode
                        circular: false
                        iconName: "menu"
                        iconSize: Theme.iconSize - 4
                        iconColor: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                        onClicked: window.menuVisible = !window.menuVisible
                    }

                    DankIcon {
                        name: "calendar_month"
                        size: Theme.iconSize
                        color: Theme.primary
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("Calendar", "main window title bar text")
                        font.pixelSize: Theme.fontSizeXLarge
                        color: Theme.surfaceText
                        font.weight: Font.Medium
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                Row {
                    anchors.right: parent.right
                    anchors.rightMargin: Theme.spacingM
                    anchors.top: parent.top
                    anchors.topMargin: Theme.spacingM
                    spacing: Theme.spacingXS

                    DankActionButton {
                        visible: windowControls.supported
                        circular: false
                        iconName: window.maximized ? "fullscreen_exit" : "fullscreen"
                        iconSize: Theme.iconSize - 4
                        iconColor: Theme.surfaceText
                        onClicked: windowControls.tryToggleMaximize()
                    }

                    DankActionButton {
                        circular: false
                        iconName: "close"
                        iconSize: Theme.iconSize - 4
                        iconColor: Theme.surfaceText
                        onClicked: window.hideRequested()
                    }
                }
            }

            Item {
                width: parent.width
                height: parent.height - 48
                clip: true

                CalendarSidebar {
                    id: sidebar
                    anchors.left: parent.left
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    width: window.isCompactMode ? parent.width : implicitWidth
                    visible: window.isCompactMode ? window.menuVisible : true
                    currentView: window.currentView
                    selectedDate: window.selectedDate
                    onViewChanged: view => {
                        window.currentView = view;
                        if (window.isCompactMode)
                            window.menuVisible = false;
                    }
                    onTodayRequested: window.goToToday()
                    onCreateEventRequested: window.openCreateEvent()
                    onAddAccountRequested: window.openAddAccount()
                }

                Item {
                    anchors.left: window.isCompactMode ? (window.menuVisible ? sidebar.right : parent.left) : sidebar.right
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    clip: true

                    CalendarContent {
                        anchors.fill: parent
                        currentView: window.currentView
                        displayDate: window.displayDate
                        selectedDate: window.selectedDate
                        today: window.today
                        onTodayRequested: window.goToToday()
                        onPreviousRequested: window.shiftDisplayDate(-1)
                        onNextRequested: window.shiftDisplayDate(1)
                        onSettingsRequested: window.openSettings()
                        onSearchRequested: window.openSearch()
                        onEventClicked: ev => window.openEventDetails(ev)
                    }
                }
            }
        }
    }

    FloatingWindowControls {
        id: windowControls
        targetWindow: window
    }

    Loader {
        id: settingsLoader
        active: false
        sourceComponent: Component {
            SettingsModal {
                parentWindow: window
                onAddAccountRequested: window.openAddAccount()
            }
        }
    }

    Loader {
        id: accountLoader
        active: false
        sourceComponent: Component {
            AccountAddModal {
                parentWindow: window
            }
        }
    }

    Loader {
        id: eventLoader
        active: false
        sourceComponent: Component {
            EventDetailsModal {
                parentWindow: window
            }
        }
    }

    Loader {
        id: searchLoader
        active: false
        sourceComponent: Component {
            SearchModal {
                onEventSelected: ev => window.goToEvent(ev)
            }
        }
    }
}
