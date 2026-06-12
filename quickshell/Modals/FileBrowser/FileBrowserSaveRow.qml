import QtQuick
import qs.Common
import qs.Widgets

Row {
    id: saveRow

    property bool saveMode: false
    property bool folderMode: false
    property string defaultFileName: ""
    property string currentPath: ""

    signal saveRequested(string filePath)
    signal folderSelected(string folderPath)

    height: (saveMode || folderMode) ? 40 : 0
    visible: saveMode || folderMode
    spacing: Theme.spacingM

    DankTextField {
        id: fileNameInput

        visible: saveRow.saveMode
        width: parent.width - saveButton.width - Theme.spacingM
        height: 40
        text: defaultFileName
        placeholderText: I18n.tr("Enter filename...", "file browser save filename input placeholder")
        ignoreLeftRightKeys: false
        focus: saveMode
        topPadding: Theme.spacingS
        bottomPadding: Theme.spacingS
        Component.onCompleted: {
            if (saveMode)
                Qt.callLater(() => {
                    forceActiveFocus();
                });
        }
        onAccepted: {
            if (text.trim() !== "") {
                var basePath = currentPath.replace(/^file:\/\//, '');
                var fullPath = basePath + "/" + text.trim();
                fullPath = fullPath.replace(/\/+/g, '/');
                saveRequested(fullPath);
            }
        }
    }

    StyledRect {
        id: saveButton

        visible: saveRow.saveMode
        width: 80
        height: 40
        color: fileNameInput.text.trim() !== "" ? Theme.primary : Theme.surfaceVariant
        radius: Theme.cornerRadius

        StyledText {
            anchors.centerIn: parent
            text: I18n.tr("Save", "file browser save button")
            color: fileNameInput.text.trim() !== "" ? Theme.primaryText : Theme.surfaceVariantText
            font.pixelSize: Theme.fontSizeMedium
        }

        StateLayer {
            stateColor: Theme.primary
            cornerRadius: Theme.cornerRadius
            enabled: fileNameInput.text.trim() !== ""
            onClicked: {
                if (fileNameInput.text.trim() !== "") {
                    var basePath = currentPath.replace(/^file:\/\//, '');
                    var fullPath = basePath + "/" + fileNameInput.text.trim();
                    fullPath = fullPath.replace(/\/+/g, '/');
                    saveRequested(fullPath);
                }
            }
        }
    }

    StyledRect {
        id: useFolderButton

        visible: saveRow.folderMode
        width: parent.width
        height: 40
        color: Theme.primary
        radius: Theme.cornerRadius

        Row {
            anchors.centerIn: parent
            spacing: Theme.spacingS

            DankIcon {
                name: "check"
                size: Theme.iconSize - 4
                color: Theme.primaryText
                anchors.verticalCenter: parent.verticalCenter
            }

            StyledText {
                text: I18n.tr("Use this folder", "file browser folder selection confirm button")
                color: Theme.primaryText
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.Medium
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        StateLayer {
            stateColor: Theme.primaryText
            cornerRadius: Theme.cornerRadius
            onClicked: {
                var basePath = currentPath.replace(/^file:\/\//, '');
                saveRow.folderSelected(basePath);
            }
        }
    }
}
