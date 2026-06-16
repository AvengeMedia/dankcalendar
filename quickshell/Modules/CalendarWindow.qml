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

    function weekStart(date) {
        const d = new Date(date);
        d.setDate(d.getDate() - ((d.getDay() - SettingsData.effectiveFirstDayOfWeek + 7) % 7));
        d.setHours(0, 0, 0, 0);
        return d;
    }

    function ensureSelectionVisible() {
        switch (currentView) {
        case "day":
            displayDate = selectedDate;
            return;
        case "week":
            if (weekStart(selectedDate).getTime() !== weekStart(displayDate).getTime())
                displayDate = selectedDate;
            return;
        case "agenda":
            {
                const start = new Date(displayDate.getFullYear(), displayDate.getMonth(), displayDate.getDate());
                const diff = Math.round((selectedDate.getTime() - start.getTime()) / 86400000);
                if (diff < 0 || diff >= 14)
                    displayDate = selectedDate;
                return;
            }
        default:
            if (selectedDate.getMonth() !== displayDate.getMonth() || selectedDate.getFullYear() !== displayDate.getFullYear())
                displayDate = selectedDate;
            return;
        }
    }

    function moveSelection(days) {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() + days);
        selectedDate = d;
        ensureSelectionVisible();
    }

    function jumpToEventDay(direction) {
        for (let i = 1; i <= 366; i++) {
            const d = new Date(selectedDate);
            d.setDate(d.getDate() + direction * i);
            if (DankCalService.eventsForDay(d).length === 0)
                continue;
            selectedDate = d;
            ensureSelectionVisible();
            return;
        }
    }

    function activateSelection() {
        const evs = DankCalService.eventsForDay(selectedDate);
        if (evs.length > 0) {
            openEventDetails(evs[0]);
            return;
        }
        openCreateEvent();
    }

    function requestClose() {
        switch (SettingsData.closeBehavior) {
        case "quit":
            DankCalService.quit();
            return;
        default:
            window.hideRequested();
            return;
        }
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

    onClosed: window.requestClose()

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
        id: focusScope
        anchors.fill: parent
        focus: true

        Component.onCompleted: forceActiveFocus()

        Shortcut {
            sequences: [StandardKey.Find]
            onActivated: window.openSearch()
        }

        Keys.onPressed: event => {
            if (helpOverlay.visible) {
                switch (event.key) {
                case Qt.Key_Escape:
                case Qt.Key_Question:
                case Qt.Key_Slash:
                    helpOverlay.visible = false;
                    break;
                }
                event.accepted = true;
                return;
            }

            const ctrl = event.modifiers & Qt.ControlModifier;
            const shift = event.modifiers & Qt.ShiftModifier;

            switch (event.key) {
            case Qt.Key_T:
                window.goToToday();
                break;
            case Qt.Key_H:
            case Qt.Key_Left:
                window.moveSelection(-1);
                break;
            case Qt.Key_L:
            case Qt.Key_Right:
                window.moveSelection(1);
                break;
            case Qt.Key_J:
            case Qt.Key_Down:
                window.moveSelection(7);
                break;
            case Qt.Key_K:
            case Qt.Key_Up:
                window.moveSelection(-7);
                break;
            case Qt.Key_Tab:
                window.jumpToEventDay(1);
                break;
            case Qt.Key_Backtab:
                window.jumpToEventDay(-1);
                break;
            case Qt.Key_Return:
            case Qt.Key_Enter:
                window.activateSelection();
                break;
            case Qt.Key_M:
                window.currentView = "month";
                break;
            case Qt.Key_W:
                window.currentView = "week";
                break;
            case Qt.Key_D:
                window.currentView = "day";
                break;
            case Qt.Key_A:
                window.currentView = "agenda";
                break;
            case Qt.Key_C:
                window.openCreateEvent();
                break;
            case Qt.Key_Slash:
                if (shift) {
                    helpOverlay.visible = true;
                    break;
                }
                window.openSearch();
                break;
            case Qt.Key_Question:
                helpOverlay.visible = true;
                break;
            case Qt.Key_P:
                if (!ctrl) {
                    event.accepted = false;
                    return;
                }
                window.openSearch();
                break;
            case Qt.Key_Comma:
                if (!ctrl) {
                    event.accepted = false;
                    return;
                }
                window.openSettings();
                break;
            default:
                event.accepted = false;
                return;
            }
            event.accepted = true;
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
                        onClicked: window.requestClose()
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
                        onDaySelected: day => window.selectedDate = day
                        onEventClicked: ev => window.openEventDetails(ev)
                    }
                }
            }
        }

        KeyboardShortcutsOverlay {
            id: helpOverlay
            anchors.fill: parent
            visible: false
            z: 100
            onDismissed: visible = false
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
                onAddCalendarRequested: window.openAddAccount()
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
