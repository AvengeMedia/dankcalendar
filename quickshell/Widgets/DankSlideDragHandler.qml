import QtQuick

DragHandler {
    id: root

    required property Item slideArea
    property real lastX: 0

    target: null
    xAxis.enabled: true
    yAxis.enabled: false

    onActiveChanged: {
        if (active) {
            lastX = 0;
            slideArea.interrupt();
            return;
        }
        slideArea.finish();
    }
    onTranslationChanged: {
        if (!active)
            return;
        slideArea.feed(activeTranslation.x - lastX);
        lastX = activeTranslation.x;
    }
}
