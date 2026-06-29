import QtQuick
import QtQuick.Controls
import QtQuick.Window
import qs.Common
import qs.Widgets

Item {
    id: root

    property int minutes: 600
    property bool use24Hour: false
    property int stepMinutes: 30
    property string iconName: "schedule"
    property bool openUpwards: false

    signal timeSelected(int value)

    // updateDirection flips the list above the field when it would clip off the
    // bottom of the window.
    function updateDirection() {
        const winH = Window.height;
        if (winH <= 0) {
            openUpwards = false;
            return;
        }
        const topInWindow = root.mapToItem(null, 0, 0).y;
        const spaceBelow = winH - (topInWindow + root.height);
        openUpwards = spaceBelow < popup.height + Theme.spacingXS && topInWindow > spaceBelow;
    }

    readonly property int slotCount: Math.ceil(1440 / stepMinutes)

    function formatMinutes(value) {
        const d = new Date(2000, 0, 1, Math.floor(value / 60), value % 60);
        return Qt.formatTime(d, use24Hour ? "HH:mm" : "h:mm AP");
    }

    // Accepts free-form input: "9", "9:23", "923", "9.23 pm", "21:23", "9p".
    // Returns minutes since midnight, or -1 when unparseable.
    function parseTime(text) {
        const t = text.trim().toLowerCase().replace(/\s+/g, " ");
        if (t === "")
            return -1;

        let m = t.match(/^(\d{3,4})\s*(a|am|p|pm)?$/);
        let hours, mins, suffix;
        if (m) {
            hours = parseInt(m[1].slice(0, -2), 10);
            mins = parseInt(m[1].slice(-2), 10);
            suffix = m[2] || "";
        } else {
            m = t.match(/^(\d{1,2})(?:[:.h](\d{1,2}))?\s*(a|am|p|pm)?$/);
            if (!m)
                return -1;
            hours = parseInt(m[1], 10);
            mins = m[2] !== undefined ? parseInt(m[2], 10) : 0;
            suffix = m[3] || "";
        }

        if (mins > 59)
            return -1;
        if (suffix !== "") {
            if (hours < 1 || hours > 12)
                return -1;
            if (suffix.charAt(0) === "p" && hours !== 12)
                hours += 12;
            if (suffix.charAt(0) === "a" && hours === 12)
                hours = 0;
        }
        if (hours > 23)
            return -1;
        return hours * 60 + mins;
    }

    function _syncText() {
        input.text = formatMinutes(minutes);
    }

    function _commit() {
        const parsed = parseTime(input.text);
        if (parsed >= 0 && parsed !== minutes)
            timeSelected(parsed);
        _syncText();
    }

    onMinutesChanged: {
        if (!input.activeFocus)
            _syncText();
    }
    onUse24HourChanged: _syncText()
    Component.onCompleted: _syncText()

    height: 48

    Rectangle {
        id: field

        anchors.fill: parent
        radius: Theme.cornerRadius
        color: Theme.surfaceContainer
        border.width: 1
        border.color: (popup.visible || input.activeFocus) ? Theme.primary : Theme.outlineLight

        MouseArea {
            anchors.fill: parent
            onClicked: {
                input.forceActiveFocus();
                popup.open();
            }
        }

        Row {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.leftMargin: Theme.spacingM
            anchors.rightMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS

            DankIcon {
                name: root.iconName
                size: Theme.iconSize - 6
                color: (popup.visible || input.activeFocus) ? Theme.primary : Theme.surfaceVariantText
                anchors.verticalCenter: parent.verticalCenter
            }

            TextInput {
                id: input

                width: parent.width - Theme.iconSize - 6 - Theme.spacingS
                anchors.verticalCenter: parent.verticalCenter
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceText
                selectionColor: Theme.primarySelected
                selectedTextColor: Theme.surfaceText
                clip: true
                inputMethodHints: Qt.ImhTime

                onActiveFocusChanged: {
                    if (activeFocus) {
                        selectAll();
                        popup.open();
                    } else {
                        root._commit();
                        popup.close();
                    }
                }
                onAccepted: {
                    root._commit();
                    popup.close();
                    focus = false;
                }
                Keys.onEscapePressed: {
                    root._syncText();
                    popup.close();
                    focus = false;
                }
            }
        }
    }

    Popup {
        id: popup

        y: root.openUpwards ? -(height + Theme.spacingXS) : (field.height + Theme.spacingXS)
        width: root.width
        height: 240
        padding: Theme.spacingXS
        closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutsideParent
        onAboutToShow: root.updateDirection()
        onOpened: list.positionViewAtIndex(Math.min(Math.floor(root.minutes / root.stepMinutes), root.slotCount - 1), ListView.Center)

        background: Rectangle {
            color: Theme.surfaceContainerHigh
            radius: Theme.cornerRadius
            border.width: 1
            border.color: Theme.outlineMedium
        }

        contentItem: DankListView {
            id: list

            clip: true
            spacing: 1
            model: root.slotCount

            LayoutMirroring.enabled: I18n.isRtl
            LayoutMirroring.childrenInherit: true

            delegate: Rectangle {
                id: slot

                required property int index
                readonly property int slotMinutes: index * root.stepMinutes
                readonly property bool active: slotMinutes === root.minutes

                width: list.width
                height: 32
                radius: Theme.cornerRadiusSmall
                color: active ? Theme.primaryHover : "transparent"

                StyledText {
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingM
                    anchors.verticalCenter: parent.verticalCenter
                    text: root.formatMinutes(slot.slotMinutes)
                    font.pixelSize: Theme.fontSizeMedium
                    color: slot.active ? Theme.primary : Theme.surfaceText
                }

                StateLayer {
                    stateColor: Theme.primary
                    cornerRadius: parent.radius
                    onClicked: {
                        root.timeSelected(slot.slotMinutes);
                        root._syncText();
                        popup.close();
                        input.focus = false;
                    }
                }
            }
        }
    }
}
