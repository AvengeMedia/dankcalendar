import QtQuick
import qs.Common
import qs.DankCommon.Widgets

SettingsRow {
    id: root

    property string text: ""
    property string description: ""
    property string currentValue: ""
    property alias options: dropdown.options
    property alias optionIcons: dropdown.optionIcons
    property alias optionIconMap: dropdown.optionIconMap
    property alias optionColorMap: dropdown.optionColorMap
    property alias enableFuzzySearch: dropdown.enableFuzzySearch
    property alias maxPopupHeight: dropdown.maxPopupHeight
    property alias dropdownWidth: dropdown.dropdownWidth
    property alias emptyText: dropdown.emptyText

    signal valueChanged(string value)

    title: text
    subtitle: description
    onCurrentValueChanged: dropdown.currentValue = currentValue

    DankDropdown {
        id: dropdown
        enabled: root.enabled
        Accessible.name: root.text
        Accessible.description: root.description + (root.description ? " · " : "") + currentValue
        width: Math.min(dropdownWidth, root.width - SettingsMetrics.rowPaddingH * 2)
        Component.onCompleted: currentValue = root.currentValue
        onValueChanged: value => root.valueChanged(value)
    }
}
