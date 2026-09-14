import QtQuick
import qs.Common

Column {
    readonly property bool isSettingsGroup: true

    width: parent?.width ?? 0
    spacing: Theme.groupedListGap
}
