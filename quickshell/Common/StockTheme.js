.pragma library

const DARK = {
    primary: "#D0BCFF",
    primaryText: "#381E72",
    primaryContainer: "#4F378B",
    secondary: "#CCC2DC",
    surface: "#141218",
    surfaceText: "#e6e0e9",
    surfaceVariant: "#49454e",
    surfaceVariantText: "#cac4cf",
    surfaceTint: "#D0BCFF",
    background: "#141218",
    backgroundText: "#e6e0e9",
    outline: "#948f99",
    surfaceContainer: "#211f24",
    surfaceContainerHigh: "#2b292f",
    surfaceContainerHighest: "#36343a",
    error: "#F2B8B5",
    warning: "#FF9800",
    info: "#2196F3",
    success: "#4CAF50"
};

const LIGHT = {
    primary: "#6750A4",
    primaryText: "#FFFFFF",
    primaryContainer: "#EADDFF",
    secondary: "#625B71",
    surface: "#FEF7FF",
    surfaceText: "#1D1B20",
    surfaceVariant: "#E7E0EC",
    surfaceVariantText: "#49454F",
    surfaceTint: "#6750A4",
    background: "#FEF7FF",
    backgroundText: "#1D1B20",
    outline: "#79747E",
    surfaceContainer: "#F3EDF7",
    surfaceContainerHigh: "#ECE6F0",
    surfaceContainerHighest: "#E6E0E9",
    error: "#B3261E",
    warning: "#E58A00",
    info: "#1565C0",
    success: "#2E7D32"
};

function getDefault(isLight) {
    return isLight ? LIGHT : DARK;
}
