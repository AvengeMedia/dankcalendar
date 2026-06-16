import QtQuick
import qs.Common

Text {
    property bool isMonospace: false

    color: Theme.surfaceText
    font.pixelSize: Theme.fontSizeMedium
    font.family: isMonospace ? Theme.monoFontFamily : Theme.fontFamily
    font.weight: Theme.fontWeight
    wrapMode: Text.WordWrap
    elide: Text.ElideRight
    horizontalAlignment: Text.AlignLeft
    verticalAlignment: Text.AlignVCenter

    Behavior on opacity {
        NumberAnimation {
            duration: Theme.shortDuration
            easing.type: Theme.standardEasing
        }
    }
}
