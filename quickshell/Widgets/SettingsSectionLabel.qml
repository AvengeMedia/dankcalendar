import QtQuick
import qs.Common
import qs.DankCommon.Widgets

Item {
    id: root

    property string text: ""
    property alias actions: actionsRow.data

    Accessible.role: Accessible.StaticText
    Accessible.name: text

    width: parent?.width ?? 0
    height: Math.max(label.implicitHeight, actionsRow.implicitHeight) + SettingsMetrics.sectionLabelTopGap + SettingsMetrics.sectionLabelBottomGap

    StyledText {
        id: label
        anchors.left: parent.left
        anchors.right: actionsRow.left
        anchors.rightMargin: Theme.spacingS
        anchors.verticalCenter: parent.verticalCenter
        anchors.verticalCenterOffset: (SettingsMetrics.sectionLabelTopGap - SettingsMetrics.sectionLabelBottomGap) / 2
        text: root.text
        font.pixelSize: Theme.fontSizeMedium
        font.weight: Theme.fontWeightMedium
        color: Theme.primary
        wrapMode: Text.WordWrap
        horizontalAlignment: Text.AlignLeft
    }

    Row {
        id: actionsRow
        anchors.right: parent.right
        anchors.verticalCenter: label.verticalCenter
        spacing: Theme.spacingXS
    }
}
