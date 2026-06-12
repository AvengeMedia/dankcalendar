import QtQuick
import QtQuick.Controls
import qs.Widgets
import "ScrollConstants.js" as Scroll

GridView {
    id: gridView

    property real momentumVelocity: 0
    property bool isMomentumActive: false
    property real friction: Scroll.friction

    flickDeceleration: Scroll.flickDeceleration
    maximumFlickVelocity: Scroll.maximumFlickVelocity
    boundsBehavior: Flickable.StopAtBounds
    boundsMovement: Flickable.FollowBoundsBehavior
    pressDelay: 0
    flickableDirection: Flickable.VerticalFlick

    onMovementStarted: {
        vbar._scrollBarActive = true;
        vbar.hideTimer.stop();
    }
    onMovementEnded: vbar.hideTimer.restart()

    WheelHandler {
        id: wheelHandler

        property real mouseWheelSpeed: Scroll.mouseWheelSpeed
        property real touchpadSpeed: Scroll.touchpadSpeed
        property real momentumRetention: Scroll.momentumRetention
        property real lastWheelTime: 0
        property real momentum: 0
        property var velocitySamples: []
        property bool sessionUsedMouseWheel: false

        function startMomentum() {
            isMomentumActive = true;
            momentumAnim.running = true;
        }

        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        onWheel: event => {
            vbar._scrollBarActive = true;
            vbar.hideTimer.restart();

            const currentTime = Date.now();
            const timeDelta = currentTime - lastWheelTime;
            lastWheelTime = currentTime;

            const hasPixel = event.pixelDelta && event.pixelDelta.y !== 0;
            const deltaY = event.angleDelta.y;
            const isTraditionalMouse = !hasPixel && Math.abs(deltaY) >= 120 && (Math.abs(deltaY) % 120) === 0;
            const isHighDpiMouse = !hasPixel && !isTraditionalMouse && deltaY !== 0;
            const isTouchpad = hasPixel;

            if (isTraditionalMouse) {
                sessionUsedMouseWheel = true;
                momentumAnim.running = false;
                isMomentumActive = false;
                velocitySamples = [];
                momentum = 0;
                momentumVelocity = 0;

                const lines = Math.round(Math.abs(deltaY) / 120);
                const scrollAmount = (deltaY > 0 ? -lines : lines) * cellHeight * 0.35;
                let newY = contentY + scrollAmount;
                newY = Math.max(0, Math.min(contentHeight - height, newY));

                if (flicking) {
                    cancelFlick();
                }

                contentY = newY;
            } else if (isHighDpiMouse) {
                sessionUsedMouseWheel = true;
                momentumAnim.running = false;
                isMomentumActive = false;
                velocitySamples = [];
                momentum = 0;
                momentumVelocity = 0;

                let delta = deltaY / 120 * cellHeight * 1.2;
                let newY = contentY - delta;
                newY = Math.max(0, Math.min(contentHeight - height, newY));

                if (flicking) {
                    cancelFlick();
                }

                contentY = newY;
            } else if (isTouchpad) {
                sessionUsedMouseWheel = false;
                momentumAnim.running = false;
                isMomentumActive = false;

                let delta = event.pixelDelta.y * touchpadSpeed;

                velocitySamples.push({
                    "delta": delta,
                    "time": currentTime
                });
                velocitySamples = velocitySamples.filter(s => currentTime - s.time < Scroll.velocitySampleWindowMs);

                if (velocitySamples.length > 1) {
                    const totalDelta = velocitySamples.reduce((sum, s) => sum + s.delta, 0);
                    const timeSpan = currentTime - velocitySamples[0].time;
                    if (timeSpan > 0) {
                        momentumVelocity = Math.max(-Scroll.maxMomentumVelocity, Math.min(Scroll.maxMomentumVelocity, totalDelta / timeSpan * 1000));
                    }
                }

                if (timeDelta < Scroll.momentumTimeThreshold) {
                    momentum = momentum * momentumRetention + delta * Scroll.momentumDeltaFactor;
                    delta += momentum;
                } else {
                    momentum = 0;
                }

                let newY = contentY - delta;
                newY = Math.max(0, Math.min(contentHeight - height, newY));

                if (flicking) {
                    cancelFlick();
                }

                contentY = newY;
            }

            event.accepted = true;
        }
        onActiveChanged: {
            if (!active) {
                if (!sessionUsedMouseWheel && Math.abs(momentumVelocity) >= Scroll.minMomentumVelocity) {
                    startMomentum();
                } else {
                    velocitySamples = [];
                    momentumVelocity = 0;
                }
            }
        }
    }

    FrameAnimation {
        id: momentumAnim
        running: false

        onTriggered: {
            const dt = frameTime;
            const newY = contentY - momentumVelocity * dt;
            const maxY = Math.max(0, contentHeight - height);

            if (newY < 0 || newY > maxY) {
                contentY = newY < 0 ? 0 : maxY;
                running = false;
                isMomentumActive = false;
                momentumVelocity = 0;
                return;
            }

            contentY = newY;
            momentumVelocity *= Math.pow(friction, dt / 0.016);

            if (Math.abs(momentumVelocity) < Scroll.momentumStopThreshold) {
                running = false;
                isMomentumActive = false;
                momentumVelocity = 0;
            }
        }
    }

    ScrollBar.vertical: DankScrollbar {
        id: vbar
    }
}
