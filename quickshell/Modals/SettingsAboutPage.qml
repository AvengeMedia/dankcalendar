import QtQuick
import QtQuick.Effects
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: aboutPage

    LayoutMirroring.enabled: I18n.isRtl
    LayoutMirroring.childrenInherit: true

    readonly property string githubUrl: "https://github.com/AvengeMedia/dankcalendar"
    readonly property string docsUrl: "https://danklinux.com/docs"
    readonly property string discordUrl: "https://discord.gg/ppWTpKmPgT"
    readonly property string kofiUrl: "https://ko-fi.com/danklinux"
    readonly property string licenseUrl: githubUrl + "/blob/master/LICENSE"

    readonly property string versionText: {
        const version = DankCalService.daemonVersion;
        if (!version)
            return "dankcal";
        if (version === "dev")
            return "dankcal (dev)";
        if (/^[\d.]+$/.test(version))
            return `dankcal v${version}`;
        return `dankcal ${version}`;
    }

    DankFlickable {
        anchors.fill: parent
        clip: true
        contentHeight: mainColumn.height + Theme.spacingXL
        contentWidth: width

        Column {
            id: mainColumn
            topPadding: 4

            width: Math.min(550, parent.width - Theme.spacingL * 2)
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: Theme.spacingXL

            StyledRect {
                width: parent.width
                height: headerSection.implicitHeight + Theme.spacingL * 2
                radius: Theme.cornerRadius
                color: Theme.surfaceContainerHigh

                Column {
                    id: headerSection

                    anchors.fill: parent
                    anchors.margins: Theme.spacingL
                    spacing: Theme.spacingM

                    Row {
                        anchors.horizontalCenter: parent.horizontalCenter
                        spacing: parent.width < 350 ? Theme.spacingM : Theme.spacingL

                        property bool compactLogo: parent.width < 400
                        property bool hideLogo: parent.width < 280

                        Image {
                            visible: !parent.hideLogo
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.compactLogo ? 80 : 120
                            height: width * (569.94629 / 506.50931)
                            fillMode: Image.PreserveAspectFit
                            smooth: true
                            mipmap: true
                            asynchronous: true
                            source: Qt.resolvedUrl("../assets/danklogonormal.svg")
                            layer.enabled: true
                            layer.smooth: true
                            layer.mipmap: true
                            layer.effect: MultiEffect {
                                saturation: 0
                                colorization: 1
                                colorizationColor: Theme.primary
                            }
                        }

                        StyledText {
                            anchors.verticalCenter: parent.verticalCenter
                            text: "DANK CALENDAR"
                            font.pixelSize: parent.compactLogo ? 26 : 38
                            font.weight: Font.Bold
                            font.family: interFont.name
                            color: Theme.surfaceText
                            antialiasing: true

                            FontLoader {
                                id: interFont
                                source: Qt.resolvedUrl("../assets/fonts/inter/InterVariable.ttf")
                            }
                        }
                    }

                    StyledText {
                        text: aboutPage.versionText
                        font.pixelSize: Theme.fontSizeXLarge
                        font.weight: Font.Bold
                        color: Theme.surfaceText
                        horizontalAlignment: Text.AlignHCenter
                        width: parent.width
                    }

                    StyledText {
                        text: I18n.tr("Part of the Dank Linux suite.", "app tagline on about page")
                        font.pixelSize: Theme.fontSizeMedium
                        font.italic: true
                        color: Theme.surfaceVariantText
                        horizontalAlignment: Text.AlignHCenter
                        width: parent.width
                    }

                    Row {
                        id: resourceButtonsRow
                        anchors.horizontalCenter: parent.horizontalCenter
                        spacing: Theme.spacingS

                        property bool compactMode: parent.width < 450

                        DankButton {
                            id: docsButton
                            text: resourceButtonsRow.compactMode ? "" : I18n.tr("Docs", "about page resource button")
                            iconName: "menu_book"
                            iconSize: 18
                            backgroundColor: Qt.rgba(Theme.surfaceText.r, Theme.surfaceText.g, Theme.surfaceText.b, 0.08)
                            textColor: Theme.surfaceText
                            onClicked: Qt.openUrlExternally(aboutPage.docsUrl)
                            onHoveredChanged: {
                                if (hovered) {
                                    resourceTooltip.show(resourceButtonsRow.compactMode ? I18n.tr("Docs", "about page resource button") + " - danklinux.com/docs" : "danklinux.com/docs", docsButton, 0, 0, "bottom");
                                    return;
                                }
                                resourceTooltip.hide();
                            }
                        }

                        DankButton {
                            id: githubButton
                            text: resourceButtonsRow.compactMode ? "" : I18n.tr("GitHub", "about page resource button")
                            iconName: "code"
                            iconSize: 18
                            backgroundColor: Qt.rgba(Theme.surfaceText.r, Theme.surfaceText.g, Theme.surfaceText.b, 0.08)
                            textColor: Theme.surfaceText
                            onClicked: Qt.openUrlExternally(aboutPage.githubUrl)
                            onHoveredChanged: {
                                if (hovered) {
                                    resourceTooltip.show(resourceButtonsRow.compactMode ? "GitHub - AvengeMedia/dankcalendar" : "github.com/AvengeMedia/dankcalendar", githubButton, 0, 0, "bottom");
                                    return;
                                }
                                resourceTooltip.hide();
                            }
                        }

                        DankButton {
                            id: kofiButton
                            text: resourceButtonsRow.compactMode ? "" : I18n.tr("Ko-fi", "about page resource button")
                            iconName: "favorite"
                            iconSize: 18
                            backgroundColor: Qt.rgba(Theme.primary.r, Theme.primary.g, Theme.primary.b, 0.12)
                            textColor: Theme.primary
                            onClicked: Qt.openUrlExternally(aboutPage.kofiUrl)
                            onHoveredChanged: {
                                if (hovered) {
                                    resourceTooltip.show(resourceButtonsRow.compactMode ? I18n.tr("Ko-fi", "about page resource button") + " - ko-fi.com/danklinux" : "ko-fi.com/danklinux", kofiButton, 0, 0, "bottom");
                                    return;
                                }
                                resourceTooltip.hide();
                            }
                        }
                    }

                    DankTooltipV2 {
                        id: resourceTooltip
                    }

                    Item {
                        id: discordButton
                        anchors.horizontalCenter: parent.horizontalCenter
                        width: 20
                        height: 20

                        Image {
                            anchors.fill: parent
                            source: Qt.resolvedUrl("../assets/discord.svg")
                            sourceSize: Qt.size(20, 20)
                            smooth: true
                            fillMode: Image.PreserveAspectFit
                        }

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            hoverEnabled: true
                            onEntered: resourceTooltip.show(I18n.tr("Dank Linux Discord", "about page community icon tooltip"), discordButton, 0, 0, "bottom")
                            onExited: resourceTooltip.hide()
                            onClicked: Qt.openUrlExternally(aboutPage.discordUrl)
                        }
                    }
                }
            }

            StyledRect {
                width: parent.width
                height: projectSection.implicitHeight + Theme.spacingL * 2
                radius: Theme.cornerRadius
                color: Theme.surfaceContainerHigh

                Column {
                    id: projectSection

                    anchors.fill: parent
                    anchors.margins: Theme.spacingL
                    spacing: Theme.spacingM

                    Row {
                        width: parent.width
                        spacing: Theme.spacingM

                        DankIcon {
                            name: "info"
                            size: Theme.iconSize
                            color: Theme.primary
                            anchors.verticalCenter: parent.verticalCenter
                        }

                        StyledText {
                            text: I18n.tr("About", "about page project section header")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.Medium
                            color: Theme.surfaceText
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }

                    StyledText {
                        text: I18n.tr('Dank Calendar is a calendar for the modern Linux desktop with a <a href="https://m3.material.io/" style="text-decoration:none; color:%1;">material 3 inspired</a> design, plus a hackable CLI and socket API for integrations.<br /><br/>It is built with <a href="https://quickshell.org" style="text-decoration:none; color:%1;">Quickshell</a>, a QT6 framework for building desktop shells, and <a href="https://go.dev" style="text-decoration:none; color:%1;">Go</a>, a statically typed, compiled programming language.', 'about page project description with embedded links').arg(Theme.primary)
                        textFormat: Text.RichText
                        font.pixelSize: Theme.fontSizeMedium
                        linkColor: Theme.primary
                        onLinkActivated: url => Qt.openUrlExternally(url)
                        color: Theme.surfaceVariantText
                        width: parent.width
                        wrapMode: Text.WordWrap

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: parent.hoveredLink ? Qt.PointingHandCursor : Qt.ArrowCursor
                            acceptedButtons: Qt.NoButton
                            propagateComposedEvents: true
                        }
                    }
                }
            }

            StyledRect {
                width: parent.width
                height: backendSection.implicitHeight + Theme.spacingL * 2
                radius: Theme.cornerRadius
                color: Theme.surfaceContainerHigh

                Column {
                    id: backendSection

                    anchors.fill: parent
                    anchors.margins: Theme.spacingL
                    spacing: Theme.spacingM

                    Row {
                        width: parent.width
                        spacing: Theme.spacingM

                        DankIcon {
                            name: "dns"
                            size: Theme.iconSize
                            color: Theme.primary
                            anchors.verticalCenter: parent.verticalCenter
                        }

                        StyledText {
                            text: I18n.tr("Backend", "about page daemon section header")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.Medium
                            color: Theme.surfaceText
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }

                    Row {
                        anchors.left: parent.left
                        spacing: Theme.spacingL

                        Column {
                            spacing: 2

                            StyledText {
                                text: I18n.tr("Version", "about page daemon version column label")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.surfaceVariantText
                            }

                            StyledText {
                                text: DankCalService.daemonVersion || "—"
                                font.pixelSize: Theme.fontSizeMedium
                                font.weight: Font.Medium
                                color: Theme.surfaceText
                            }
                        }

                        Rectangle {
                            width: 1
                            height: 32
                            color: Theme.outlineVariant
                        }

                        Column {
                            spacing: 2

                            StyledText {
                                text: I18n.tr("API", "about page daemon api version column label")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.surfaceVariantText
                            }

                            StyledText {
                                text: DankCalService.apiVersion > 0 ? `v${DankCalService.apiVersion}` : "—"
                                font.pixelSize: Theme.fontSizeMedium
                                font.weight: Font.Medium
                                color: Theme.surfaceText
                            }
                        }

                        Rectangle {
                            width: 1
                            height: 32
                            color: Theme.outlineVariant
                        }

                        Column {
                            spacing: 2

                            StyledText {
                                text: I18n.tr("Status", "about page daemon status column label")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.surfaceVariantText
                            }

                            Row {
                                spacing: 4

                                Rectangle {
                                    width: 8
                                    height: 8
                                    radius: 4
                                    color: DankCalService.connected ? Theme.success : Theme.error
                                    anchors.verticalCenter: parent.verticalCenter
                                }

                                StyledText {
                                    text: DankCalService.connected ? I18n.tr("Connected", "about page daemon status value") : I18n.tr("Offline", "about page daemon status value")
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.Medium
                                    color: Theme.surfaceText
                                }
                            }
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: Theme.spacingS
                        visible: DankCalService.capabilities.length > 0

                        StyledText {
                            text: I18n.tr("Capabilities", "about page daemon capabilities label")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.surfaceVariantText
                            width: parent.width
                        }

                        Flow {
                            width: parent.width
                            spacing: 6

                            Repeater {
                                model: ScriptModel {
                                    values: DankCalService.capabilities
                                }

                                Rectangle {
                                    required property var modelData
                                    width: capText.implicitWidth + 16
                                    height: 26
                                    radius: 13
                                    color: Qt.rgba(Theme.primary.r, Theme.primary.g, Theme.primary.b, 0.12)

                                    StyledText {
                                        id: capText
                                        anchors.centerIn: parent
                                        text: parent.modelData
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.primary
                                    }
                                }
                            }
                        }
                    }
                }
            }

            StyledText {
                anchors.horizontalCenter: parent.horizontalCenter
                text: I18n.tr('<a href="%2" style="text-decoration:none; color:%1;">MIT License</a>', 'about page license footer link, %2 is the license url').arg(Theme.surfaceVariantText).arg(aboutPage.licenseUrl)
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceVariantText
                textFormat: Text.RichText
                wrapMode: Text.NoWrap
                onLinkActivated: url => Qt.openUrlExternally(url)

                MouseArea {
                    anchors.fill: parent
                    cursorShape: parent.hoveredLink ? Qt.PointingHandCursor : Qt.ArrowCursor
                    acceptedButtons: Qt.NoButton
                    propagateComposedEvents: true
                }
            }
        }
    }
}
