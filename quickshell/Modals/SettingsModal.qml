import QtQuick
import Quickshell
import qs.Common
import qs.Widgets

FloatingWindow {
    id: settingsModal

    property int currentTabIndex: 0
    property bool isCompactMode: width < 640
    property bool menuVisible: !isCompactMode

    signal addAccountRequested

    function show() {
        visible = true;
    }

    function hide() {
        visible = false;
    }

    title: I18n.tr("Calendar Settings", "settings window title")
    minimumSize: Qt.size(520, 420)
    implicitWidth: 880
    implicitHeight: screen ? Math.min(720, screen.height - 120) : 720
    color: Theme.surface
    visible: false

    onIsCompactModeChanged: {
        if (!isCompactMode)
            menuVisible = true;
    }

    FocusScope {
        anchors.fill: parent
        focus: true

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
                        visible: settingsModal.isCompactMode
                        circular: false
                        iconName: "menu"
                        iconColor: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                        onClicked: settingsModal.menuVisible = !settingsModal.menuVisible
                    }

                    DankIcon {
                        name: "settings"
                        size: Theme.iconSize
                        color: Theme.primary
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("Settings", "settings window header title")
                        font.pixelSize: Theme.fontSizeXLarge
                        font.weight: Font.Medium
                        color: Theme.surfaceText
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
                        iconName: settingsModal.maximized ? "fullscreen_exit" : "fullscreen"
                        iconColor: Theme.surfaceText
                        onClicked: windowControls.tryToggleMaximize()
                    }

                    DankActionButton {
                        circular: false
                        iconName: "close"
                        iconColor: Theme.surfaceText
                        onClicked: settingsModal.hide()
                    }
                }
            }

            Item {
                width: parent.width
                height: parent.height - 48
                clip: true

                SettingsSidebar {
                    id: sidebar
                    anchors.left: parent.left
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    width: settingsModal.isCompactMode ? parent.width : implicitWidth
                    visible: settingsModal.isCompactMode ? settingsModal.menuVisible : true
                    currentIndex: settingsModal.currentTabIndex
                    onTabSelected: index => {
                        settingsModal.currentTabIndex = index;
                        if (settingsModal.isCompactMode)
                            settingsModal.menuVisible = false;
                    }
                }

                Item {
                    anchors.left: settingsModal.isCompactMode ? (settingsModal.menuVisible ? sidebar.right : parent.left) : sidebar.right
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    clip: true

                    SettingsContent {
                        anchors.fill: parent
                        currentIndex: settingsModal.currentTabIndex
                        onAddAccountRequested: settingsModal.addAccountRequested()
                    }
                }
            }
        }
    }

    Shortcut {
        sequence: "Escape"
        onActivated: settingsModal.hide()
    }

    FloatingWindowControls {
        id: windowControls
        targetWindow: settingsModal
    }
}
