import QtQuick

// Diagonal stripes marking a tentatively accepted invitation, tiled from an
// inline SVG so the stripes can follow the event's calendar color.
Image {
    property color stripeColor: "transparent"

    readonly property string stripeHex: Qt.rgba(stripeColor.r, stripeColor.g, stripeColor.b, 1).toString()

    anchors.fill: parent
    fillMode: Image.Tile
    opacity: 0.3
    source: "data:image/svg+xml;utf8," + encodeURIComponent("<svg xmlns='http://www.w3.org/2000/svg' width='8' height='8'><path d='M-2 2 4 -4 M0 8 8 0 M6 10 12 4' stroke='" + stripeHex + "' stroke-width='2'/></svg>")
}
