pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.Services
import "StockTheme.js" as StockTheme

Singleton {
    id: root

    readonly property string defaultFontFamily: "Inter Variable"
    readonly property string defaultMonoFontFamily: "Fira Code"

    // "auto" follows the desktop portal color-scheme; no preference falls back to dark
    readonly property bool isLightMode: {
        switch (SettingsData.themeMode) {
        case "light":
            return true;
        case "dark":
            return false;
        default:
            return PortalService.systemPrefersLight;
        }
    }

    readonly property string xdgCacheDir: {
        const xdg = Quickshell.env("XDG_CACHE_HOME");
        if (xdg && xdg !== "")
            return xdg;
        const home = Quickshell.env("HOME");
        return home + "/.cache";
    }

    readonly property string dmsColorsPath: xdgCacheDir + "/DankMaterialShell/dms-colors.json"

    property var matugenColors: ({})
    property bool colorsLoaded: false

    function getMatugenColor(path, fallback) {
        const colorMode = isLightMode ? "light" : "dark";
        let cur = matugenColors && matugenColors.colors && matugenColors.colors[colorMode];
        if (!cur)
            return fallback;
        const parts = path.split(".");
        for (let i = 0; i < parts.length; i++) {
            if (!cur || typeof cur !== "object" || !(parts[i] in cur))
                return fallback;
            cur = cur[parts[i]];
        }
        return cur || fallback;
    }

    readonly property var currentThemeData: {
        const stock = StockTheme.getDefault(isLightMode);
        if (!colorsLoaded)
            return stock;
        return {
            "primary": getMatugenColor("primary", stock.primary),
            "primaryText": getMatugenColor("on_primary", stock.primaryText),
            "primaryContainer": getMatugenColor("primary_container", stock.primaryContainer),
            "secondary": getMatugenColor("secondary", stock.secondary),
            "surface": getMatugenColor("surface", stock.surface),
            "surfaceText": getMatugenColor("on_surface", stock.surfaceText),
            "surfaceVariant": getMatugenColor("surface_variant", stock.surfaceVariant),
            "surfaceVariantText": getMatugenColor("on_surface_variant", stock.surfaceVariantText),
            "surfaceTint": getMatugenColor("surface_tint", stock.surfaceTint),
            "background": getMatugenColor("background", stock.background),
            "backgroundText": getMatugenColor("on_background", stock.backgroundText),
            "outline": getMatugenColor("outline", stock.outline),
            "surfaceContainer": getMatugenColor("surface_container", stock.surfaceContainer),
            "surfaceContainerHigh": getMatugenColor("surface_container_high", stock.surfaceContainerHigh),
            "surfaceContainerHighest": getMatugenColor("surface_container_highest", stock.surfaceContainerHighest),
            "error": stock.error,
            "warning": stock.warning,
            "info": stock.info,
            "success": stock.success
        };
    }

    property color primary: currentThemeData.primary
    property color primaryText: currentThemeData.primaryText
    property color primaryContainer: currentThemeData.primaryContainer
    property color secondary: currentThemeData.secondary
    property color surface: currentThemeData.surface
    property color surfaceText: currentThemeData.surfaceText
    property color surfaceVariant: currentThemeData.surfaceVariant
    property color surfaceVariantText: currentThemeData.surfaceVariantText
    property color surfaceTint: currentThemeData.surfaceTint
    property color background: currentThemeData.background
    property color backgroundText: currentThemeData.backgroundText
    property color outline: currentThemeData.outline
    property color outlineVariant: Qt.rgba(outline.r, outline.g, outline.b, 0.6)
    property color surfaceContainer: currentThemeData.surfaceContainer
    property color surfaceContainerHigh: currentThemeData.surfaceContainerHigh
    property color surfaceContainerHighest: currentThemeData.surfaceContainerHighest

    property color onSurface: surfaceText
    property color onSurfaceVariant: surfaceVariantText
    property color onPrimary: primaryText

    property color error: currentThemeData.error
    property color warning: currentThemeData.warning
    property color info: currentThemeData.info
    property color success: currentThemeData.success

    property color primaryHover: Qt.rgba(primary.r, primary.g, primary.b, 0.12)
    property color primaryHoverLight: Qt.rgba(primary.r, primary.g, primary.b, 0.08)
    property color primaryPressed: Qt.rgba(primary.r, primary.g, primary.b, 0.16)
    property color primarySelected: Qt.rgba(primary.r, primary.g, primary.b, 0.3)
    property color primaryBackground: Qt.rgba(primary.r, primary.g, primary.b, 0.04)

    property color secondaryHover: Qt.rgba(secondary.r, secondary.g, secondary.b, 0.08)

    property color surfaceHover: Qt.rgba(surfaceVariant.r, surfaceVariant.g, surfaceVariant.b, 0.08)
    property color surfaceVariantHover: Qt.lighter(surfaceVariant, 1.2)
    property color surfacePressed: Qt.rgba(surfaceVariant.r, surfaceVariant.g, surfaceVariant.b, 0.12)
    property color surfaceSelected: Qt.rgba(surfaceVariant.r, surfaceVariant.g, surfaceVariant.b, 0.15)
    property color surfaceLight: Qt.rgba(surfaceVariant.r, surfaceVariant.g, surfaceVariant.b, 0.1)
    property color surfaceVariantAlpha: Qt.rgba(surfaceVariant.r, surfaceVariant.g, surfaceVariant.b, 0.2)

    property color surfaceTextHover: Qt.rgba(surfaceText.r, surfaceText.g, surfaceText.b, 0.08)
    property color surfaceTextAlpha: Qt.rgba(surfaceText.r, surfaceText.g, surfaceText.b, 0.3)
    property color surfaceTextLight: Qt.rgba(surfaceText.r, surfaceText.g, surfaceText.b, 0.06)
    property color surfaceTextMedium: Qt.rgba(surfaceText.r, surfaceText.g, surfaceText.b, 0.7)

    property color outlineButton: Qt.rgba(outline.r, outline.g, outline.b, 0.5)
    property color outlineLight: Qt.rgba(outline.r, outline.g, outline.b, 0.05)
    property color outlineMedium: Qt.rgba(outline.r, outline.g, outline.b, 0.12)
    property color outlineStrong: Qt.rgba(outline.r, outline.g, outline.b, 0.18)

    property color errorHover: Qt.rgba(error.r, error.g, error.b, 0.12)
    property color errorPressed: Qt.rgba(error.r, error.g, error.b, 0.16)

    property color shadowMedium: Qt.rgba(0, 0, 0, 0.08)
    property color shadowStrong: Qt.rgba(0, 0, 0, 0.3)

    property color buttonBg: primary
    property color buttonText: primaryText
    property color buttonHover: primaryHover
    property color buttonPressed: primaryPressed

    property real spacingXS: 4
    property real spacingS: 8
    property real spacingM: 12
    property real spacingL: 16
    property real spacingXL: 24

    property real fontScale: 1.0
    property real fontSizeSmall: Math.round(fontScale * 12)
    property real fontSizeMedium: Math.round(fontScale * 14)
    property real fontSizeLarge: Math.round(fontScale * 16)
    property real fontSizeXLarge: Math.round(fontScale * 20)

    property real iconSize: 24
    property real iconSizeSmall: 16
    property real iconSizeLarge: 32

    property real cornerRadius: 12
    property real cornerRadiusSmall: 8
    property real cornerRadiusLarge: 16

    property string fontFamily: defaultFontFamily
    property string monoFontFamily: defaultMonoFontFamily
    property int fontWeight: Font.Normal

    property real popupTransparency: 1.0

    readonly property color floatingSurface: withAlpha(surfaceContainer, popupTransparency)

    property color widgetBaseHoverColor: {
        const blended = blend(surfaceContainerHigh, primary, 0.1);
        return withAlpha(blended, Math.max(0.3, blended.a));
    }

    property int shorterDuration: 100
    property int shortDuration: 200
    property int mediumDuration: 400
    property int longDuration: 600
    property int standardEasing: Easing.OutCubic
    property int emphasizedEasing: Easing.OutQuart

    function withAlpha(c, a) {
        return Qt.rgba(c.r, c.g, c.b, a);
    }

    function blend(c1, c2, r) {
        return Qt.rgba(c1.r * (1 - r) + c2.r * r, c1.g * (1 - r) + c2.g * r, c1.b * (1 - r) + c2.b * r, c1.a * (1 - r) + c2.a * r);
    }

    FileView {
        id: dmsColorsView
        path: root.dmsColorsPath
        blockLoading: false
        watchChanges: true

        onLoaded: {
            try {
                const text = dmsColorsView.text();
                if (!text)
                    return;
                root.matugenColors = JSON.parse(text);
                root.colorsLoaded = true;
            } catch (e) {
                root.colorsLoaded = false;
            }
        }

        onFileChanged: dmsColorsView.reload()

        onLoadFailed: function (error) {
            root.colorsLoaded = false;
        }
    }
}
