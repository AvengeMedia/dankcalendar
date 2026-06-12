import QtQuick

Item {
    id: root

    readonly property real edgeSize: 8
    required property var targetWindow
    property bool supported: typeof targetWindow.startSystemMove === "function"
    readonly property bool canMaximize: targetWindow.minimumSize.width !== targetWindow.maximumSize.width || targetWindow.minimumSize.height !== targetWindow.maximumSize.height

    anchors.fill: parent

    function tryStartMove() {
        if (!supported)
            return;
        targetWindow.startSystemMove();
    }

    function tryStartResize(edges) {
        if (!supported || !canMaximize)
            return;
        targetWindow.startSystemResize(edges);
    }

    function tryToggleMaximize() {
        if (!supported || !canMaximize)
            return;
        targetWindow.maximized = !targetWindow.maximized;
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        height: root.edgeSize
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.leftMargin: 6
        anchors.rightMargin: 6
        cursorShape: Qt.SizeVerCursor
        onPressed: root.tryStartResize(Qt.TopEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        width: root.edgeSize
        anchors.left: parent.left
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        anchors.topMargin: 6
        anchors.bottomMargin: 6
        cursorShape: Qt.SizeHorCursor
        onPressed: root.tryStartResize(Qt.LeftEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        width: root.edgeSize
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        anchors.topMargin: 6
        anchors.bottomMargin: 6
        cursorShape: Qt.SizeHorCursor
        onPressed: root.tryStartResize(Qt.RightEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        width: root.edgeSize
        height: root.edgeSize
        anchors.left: parent.left
        anchors.top: parent.top
        cursorShape: Qt.SizeFDiagCursor
        onPressed: root.tryStartResize(Qt.LeftEdge | Qt.TopEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        width: root.edgeSize
        height: root.edgeSize
        anchors.right: parent.right
        anchors.top: parent.top
        cursorShape: Qt.SizeBDiagCursor
        onPressed: root.tryStartResize(Qt.RightEdge | Qt.TopEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        height: root.edgeSize
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.leftMargin: 6
        anchors.rightMargin: 6
        cursorShape: Qt.SizeVerCursor
        onPressed: root.tryStartResize(Qt.BottomEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        width: root.edgeSize
        height: root.edgeSize
        anchors.left: parent.left
        anchors.bottom: parent.bottom
        cursorShape: Qt.SizeBDiagCursor
        onPressed: root.tryStartResize(Qt.LeftEdge | Qt.BottomEdge)
    }

    MouseArea {
        visible: root.supported && root.canMaximize
        width: root.edgeSize
        height: root.edgeSize
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        cursorShape: Qt.SizeFDiagCursor
        onPressed: root.tryStartResize(Qt.RightEdge | Qt.BottomEdge)
    }
}
