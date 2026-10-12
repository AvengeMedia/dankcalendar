import QtQuick
import qs.Common

Column {
    id: root

    property string title: ""
    default property alias content: contentGroup.data
    property alias headerActions: sectionLabel.actions
    readonly property bool hasHeader: title !== ""

    width: parent?.width ?? 0
    spacing: 0

    SettingsSectionLabel {
        id: sectionLabel
        width: parent.width
        text: root.title
        visible: root.hasHeader
    }

    SettingsGroup {
        id: contentGroup
        width: parent.width
    }
}
