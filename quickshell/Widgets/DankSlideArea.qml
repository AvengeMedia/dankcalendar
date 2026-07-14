import QtQuick
import "../DankCommon/Widgets/ScrollConstants.js" as Scroll

Item {
    id: root

    property bool continuous: true
    property bool dragEnabled: true
    property real pageThresholdPx: 200
    property int stepCooldownMs: 100
    property real touchpadFactor: Scroll.touchpadSpeed
    property real highDpiFactor: Scroll.touchpadSpeed / 8
    property real momentumVelocity: 0

    signal moved(real dx)
    signal stepped(int direction)
    signal settled

    property real pageAccumulator: 0

    function feed(dx) {
        if (continuous) {
            moved(dx);
            return;
        }
        pageAccumulator += dx;
        if (Math.abs(pageAccumulator) < pageThresholdPx)
            return;
        emitStep(pageAccumulator > 0 ? 1 : -1);
        pageAccumulator = 0;
    }

    function emitStep(direction) {
        if (stepCooldown.running)
            return;
        stepCooldown.restart();
        stepped(direction);
    }

    function finish() {
        pageAccumulator = 0;
        momentumVelocity = 0;
        settled();
    }

    function interrupt() {
        momentumAnim.running = false;
    }

    Timer {
        id: stepCooldown
        interval: root.stepCooldownMs
    }

    WheelHandler {
        id: wheel
        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        orientation: Qt.Horizontal

        property real lastWheelTime: 0
        property real momentum: 0
        property var velocitySamples: []
        property bool engaged: false
        property bool kineticEligible: false

        onWheel: event => {
            if (!engaged && Math.abs(event.angleDelta.x) <= Math.abs(event.angleDelta.y)) {
                event.accepted = false;
                return;
            }

            engaged = true;
            momentumAnim.running = false;

            const currentTime = Date.now();
            const timeDelta = currentTime - lastWheelTime;
            lastWheelTime = currentTime;

            const hasPixel = event.pixelDelta && event.pixelDelta.x !== 0;
            const deltaX = event.angleDelta.x;
            const isTraditionalMouse = !hasPixel && Math.abs(deltaX) >= 120 && Math.abs(deltaX) % 120 === 0;

            if (isTraditionalMouse) {
                kineticEligible = false;
                momentum = 0;
                velocitySamples = [];
                root.momentumVelocity = 0;
                const notches = Math.round(Math.abs(deltaX) / 120);
                for (let i = 0; i < notches; i++)
                    root.emitStep(deltaX > 0 ? 1 : -1);
                return;
            }

            if (!hasPixel) {
                if (deltaX === 0)
                    return;
                kineticEligible = false;
                momentum = 0;
                velocitySamples = [];
                root.momentumVelocity = 0;
                root.feed(deltaX * root.highDpiFactor);
                return;
            }

            let delta = event.pixelDelta.x * root.touchpadFactor;

            velocitySamples.push({
                "delta": delta,
                "time": currentTime
            });
            velocitySamples = velocitySamples.filter(s => currentTime - s.time < Scroll.velocitySampleWindowMs);

            if (velocitySamples.length > 1) {
                const totalDelta = velocitySamples.reduce((sum, s) => sum + s.delta, 0);
                const timeSpan = currentTime - velocitySamples[0].time;
                if (timeSpan > 0)
                    root.momentumVelocity = Math.max(-Scroll.maxMomentumVelocity, Math.min(Scroll.maxMomentumVelocity, totalDelta / timeSpan * 1000));
            }

            if (timeDelta < Scroll.momentumTimeThreshold) {
                momentum = momentum * Scroll.momentumRetention + delta * Scroll.momentumDeltaFactor;
                delta += momentum;
            } else {
                momentum = 0;
            }

            kineticEligible = true;
            root.feed(delta);
        }

        onActiveChanged: {
            if (active)
                return;
            engaged = false;
            momentum = 0;
            velocitySamples = [];
            if (kineticEligible && root.continuous && Math.abs(root.momentumVelocity) >= Scroll.minMomentumVelocity) {
                momentumAnim.running = true;
                return;
            }
            root.finish();
        }
    }

    FrameAnimation {
        id: momentumAnim
        running: false

        onTriggered: {
            const dt = frameTime;
            root.feed(root.momentumVelocity * dt);
            root.momentumVelocity *= Math.pow(Scroll.friction, dt / 0.016);
            if (Math.abs(root.momentumVelocity) < Scroll.momentumStopThreshold) {
                running = false;
                root.finish();
            }
        }
    }

    DragHandler {
        enabled: root.dragEnabled
        acceptedDevices: PointerDevice.TouchScreen
        target: null
        xAxis.enabled: true
        yAxis.enabled: false

        property real lastX: 0

        onActiveChanged: {
            if (active) {
                lastX = 0;
                momentumAnim.running = false;
                return;
            }
            root.finish();
        }
        onTranslationChanged: {
            if (!active)
                return;
            root.feed(activeTranslation.x - lastX);
            lastX = activeTranslation.x;
        }
    }
}
