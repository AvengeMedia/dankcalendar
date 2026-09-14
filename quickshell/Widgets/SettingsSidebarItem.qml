import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Rectangle {
    id: root

    property string iconName: ""
    property string title: ""
    property string hint: ""
    property string tone: "primary"
    property bool active: false
    property bool highlighted: false
    property bool isFirstInGroup: true
    property bool isLastInGroup: true

    signal clicked

    activeFocusOnTab: enabled
    Accessible.role: Accessible.Button
    Accessible.name: title
    Accessible.description: hint
    Accessible.onPressAction: clicked()

    Keys.onPressed: event => {
        switch (event.key) {
        case Qt.Key_Space:
        case Qt.Key_Return:
        case Qt.Key_Enter:
            root.clicked();
            event.accepted = true;
            break;
        }
    }

    readonly property color toneColor: {
        switch (tone) {
        case "secondary":
            return Theme.secondary;
        case "tertiary":
            return Theme.tertiary;
        case "error":
            return Theme.error;
        default:
            return Theme.primary;
        }
    }
    readonly property real topRadius: isFirstInGroup ? Theme.groupedListOuterRadius : Theme.groupedListInnerRadius
    readonly property real bottomRadius: isLastInGroup ? Theme.groupedListOuterRadius : Theme.groupedListInnerRadius

    width: parent?.width ?? 0
    height: Math.max(SettingsMetrics.navItemMinHeight, textColumn.implicitHeight + Theme.spacingS * 2)
    topLeftRadius: topRadius
    topRightRadius: topRadius
    bottomLeftRadius: bottomRadius
    bottomRightRadius: bottomRadius
    color: {
        if (active)
            return Theme.primaryContainer;
        if (highlighted)
            return Theme.blend(SettingsMetrics.rowColor, Theme.primary, SettingsMetrics.highlightBlend);
        return SettingsMetrics.rowColor;
    }

    Behavior on color {
        enabled: Theme.animationsEnabled
        ColorAnimation {
            duration: SettingsMetrics.fadeDuration
            easing.type: Easing.BezierSpline
            easing.bezierCurve: Theme.expressiveCurves.expressiveEffects
        }
    }

    Rectangle {
        anchors.fill: parent
        topLeftRadius: root.topRadius
        topRightRadius: root.topRadius
        bottomLeftRadius: root.bottomRadius
        bottomRightRadius: root.bottomRadius
        color: Theme.surfaceText
        opacity: mouseArea.pressed ? Theme.stateLayerPressed : (mouseArea.containsMouse ? Theme.stateLayerHover : 0)

        Behavior on opacity {
            enabled: Theme.animationsEnabled
            NumberAnimation {
                duration: SettingsMetrics.fadeDuration
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Theme.expressiveCurves.expressiveEffects
            }
        }
    }

    DankRipple {
        id: ripple
        rippleColor: root.active ? Theme.onPrimaryContainer : Theme.surfaceText
        topLeftRadius: root.topRadius
        topRightRadius: root.topRadius
        bottomLeftRadius: root.bottomRadius
        bottomRightRadius: root.bottomRadius
    }

    Rectangle {
        anchors.fill: parent
        anchors.margins: Theme.focusRingWidth / 2
        topLeftRadius: Math.max(0, root.topRadius - Theme.focusRingWidth / 2)
        topRightRadius: topLeftRadius
        bottomLeftRadius: Math.max(0, root.bottomRadius - Theme.focusRingWidth / 2)
        bottomRightRadius: bottomLeftRadius
        color: "transparent"
        border.width: Theme.focusRingWidth
        border.color: Theme.focusRingColor
        visible: root.activeFocus
    }

    Rectangle {
        id: iconCircle
        width: SettingsMetrics.navIconSize
        height: SettingsMetrics.navIconSize
        radius: Theme.fullRadius(width, height)
        anchors.left: parent.left
        anchors.leftMargin: Theme.spacingL
        anchors.verticalCenter: parent.verticalCenter
        color: root.active ? Theme.primary : Theme.withAlpha(root.toneColor, Theme.tonalTintAlpha)

        Behavior on color {
            enabled: Theme.animationsEnabled
            ColorAnimation {
                duration: SettingsMetrics.fadeDuration
                easing.type: Easing.BezierSpline
                easing.bezierCurve: Theme.expressiveCurves.expressiveEffects
            }
        }

        DankIcon {
            anchors.centerIn: parent
            name: root.iconName
            size: Theme.iconSizeMedium
            color: root.active ? Theme.onPrimary : root.toneColor
        }
    }

    Column {
        id: textColumn
        anchors.left: iconCircle.right
        anchors.leftMargin: Theme.spacingM
        anchors.right: parent.right
        anchors.rightMargin: Theme.spacingL
        anchors.verticalCenter: parent.verticalCenter
        spacing: Theme.spacingXXS

        StyledText {
            width: parent.width
            text: root.title
            font.pixelSize: Theme.fontSizeMedium
            font.weight: Theme.fontWeightMedium
            color: root.active ? Theme.onPrimaryContainer : Theme.surfaceText
            elide: Text.ElideRight
            horizontalAlignment: Text.AlignLeft
        }

        StyledText {
            width: parent.width
            text: root.hint
            font.pixelSize: Theme.fontSizeSmall
            color: root.active ? Theme.withAlpha(Theme.onPrimaryContainer, SettingsMetrics.activeHintAlpha) : Theme.surfaceVariantText
            elide: Text.ElideRight
            visible: root.hint !== ""
            horizontalAlignment: Text.AlignLeft
        }
    }

    MouseArea {
        id: mouseArea
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onPressed: mouse => ripple.trigger(mouse.x, mouse.y)
        onClicked: root.clicked()
    }
}
