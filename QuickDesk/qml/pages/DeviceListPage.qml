import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

import "../component"
import "../quickdeskcomponent"

Item {
    id: root

    required property var mainController

    signal connectToDevice(string deviceId, string accessCode)
    signal showToast(string message, int toastType)

    property bool isLoggedIn: mainController && mainController.authManager ? mainController.authManager.isLoggedIn : false

    property var myDevices: mainController && mainController.cloudDeviceManager
                            ? mainController.cloudDeviceManager.myDevices : []

    // Not logged in prompt
    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.contactInfoGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to view device list")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.textSecondary
                Layout.alignment: Qt.AlignHCenter
            }

            QDButton {
                text: qsTr("Login")
                highlighted: true
                Layout.alignment: Qt.AlignHCenter
                onClicked: loginDialog.open()
            }
        }
    }

    // Logged in content
    ColumnLayout {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        anchors.margins: Theme.spacingLarge
        visible: root.isLoggedIn
        spacing: Theme.spacingSmall

        // ---- My Devices Section ----
        QDAccordion {
            Layout.fillWidth: true
            title: qsTr("My Devices") + (deviceRepeater.count > 0 ? " (" + deviceRepeater.count + ")" : "")
            iconSource: FluentIconGlyph.contactInfoGlyph
            expanded: true

            Item {
                width: parent.width
                height: deviceRepeater.count === 0 ? emptyDevicesText.height
                                                   : Math.min(deviceRepeater.count, 4) * 44

                Text {
                    id: emptyDevicesText
                    visible: deviceRepeater.count === 0
                    text: qsTr("No devices bound yet")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                ScrollView {
                    anchors.fill: parent
                    visible: deviceRepeater.count > 0
                    clip: true
                    background: Rectangle { color: "transparent" }
                    ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                    ScrollBar.vertical: QDScrollBar {}

                    Column {
                        width: parent.width
                        spacing: 2

                        Repeater {
                            id: deviceRepeater
                            model: root.mainController && root.mainController.cloudDeviceManager
                                   ? root.mainController.cloudDeviceManager.myDevices : []

                            delegate: Rectangle {
                                required property var modelData
                                required property int index

                                width: parent.width
                                height: 40
                                radius: Theme.radiusSmall
                                color: deviceRowHover.hovered ? Theme.surfaceHover : "transparent"

                                HoverHandler { id: deviceRowHover }

                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }

                                RowLayout {
                                    anchors.fill: parent
                                    anchors.leftMargin: Theme.spacingSmall
                                    anchors.rightMargin: Theme.spacingSmall
                                    spacing: Theme.spacingSmall

                                    Rectangle {
                                        width: 8; height: 8; radius: 4
                                        color: modelData.online ? Theme.success : Theme.textDisabled
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: {
                                            var name = modelData.remark || modelData.device_name || qsTr("Device")
                                            return name + " (" + (modelData.device_id || "") + ")"
                                        }
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                        elide: Text.ElideRight
                                    }

                                    QDIconButton {
                                        visible: modelData.online === true
                                        iconSource: FluentIconGlyph.remoteGlyph
                                        buttonSize: QDIconButton.Size.Small

                                        QDToolTip { visible: parent.hovered; text: qsTr("Connect") }

                                        onClicked: {
                                            var accessCode = root.mainController.cloudDeviceManager.getDeviceAccessCode(modelData.device_id)
                                            if (accessCode)
                                                root.connectToDevice(modelData.device_id, accessCode)
                                            else
                                                root.showToast(qsTr("Access code not available"), 2)
                                        }
                                    }
                                }

                                MouseArea {
                                    anchors.fill: parent
                                    hoverEnabled: true
                                    acceptedButtons: Qt.RightButton
                                    z: -1
                                    onClicked: function(mouse) {
                                        if (mouse.button === Qt.RightButton) {
                                            deviceContextMenu.deviceId = modelData.device_id
                                            deviceContextMenu.deviceRemark = modelData.remark || ""
                                            var pos = mapToItem(root, mouse.x, mouse.y)
                                            deviceContextMenu.x = pos.x
                                            deviceContextMenu.y = pos.y
                                            deviceContextMenu.open()
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        // ---- My Favorites Section ----
        QDAccordion {
            Layout.fillWidth: true
            title: qsTr("My Favorites") + (favoriteRepeater.count > 0 ? " (" + favoriteRepeater.count + ")" : "")
            iconSource: FluentIconGlyph.favoriteStarGlyph
            expanded: true

            Item {
                width: parent.width
                height: favoriteRepeater.count === 0 ? emptyFavText.height
                                                     : Math.min(favoriteRepeater.count, 4) * 44

                Text {
                    id: emptyFavText
                    visible: favoriteRepeater.count === 0
                    text: qsTr("No favorites yet")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                ScrollView {
                    anchors.fill: parent
                    visible: favoriteRepeater.count > 0
                    clip: true
                    background: Rectangle { color: "transparent" }
                    ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                    ScrollBar.vertical: QDScrollBar {}

                    Column {
                        width: parent.width
                        spacing: 2

                        Repeater {
                            id: favoriteRepeater
                            model: root.mainController && root.mainController.cloudDeviceManager
                                   ? root.mainController.cloudDeviceManager.myFavorites : []

                            delegate: Rectangle {
                                required property var modelData
                                required property int index

                                width: parent.width
                                height: 40
                                radius: Theme.radiusSmall
                                color: favRowHover.hovered ? Theme.surfaceHover : "transparent"

                                HoverHandler { id: favRowHover }

                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }

                                RowLayout {
                                    anchors.fill: parent
                                    anchors.leftMargin: Theme.spacingSmall
                                    anchors.rightMargin: Theme.spacingSmall
                                    spacing: Theme.spacingSmall

                                    Text {
                                        text: FluentIconGlyph.favoriteStarFillGlyph
                                        font.family: "Segoe Fluent Icons"
                                        font.pixelSize: 12
                                        color: Theme.warning
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: {
                                            var name = modelData.device_name || modelData.device_id || qsTr("Device")
                                            return name + " (" + (modelData.device_id || "") + ")"
                                        }
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                        elide: Text.ElideRight
                                    }

                                    QDIconButton {
                                        iconSource: FluentIconGlyph.remoteGlyph
                                        buttonSize: QDIconButton.Size.Small

                                        QDToolTip { visible: parent.hovered; text: qsTr("Connect") }

                                        onClicked: {
                                            var password = modelData.access_password || ""
                                            if (password)
                                                root.connectToDevice(modelData.device_id, password)
                                            else
                                                root.showToast(qsTr("No password saved for this device"), 2)
                                        }
                                    }
                                }

                                MouseArea {
                                    anchors.fill: parent
                                    hoverEnabled: true
                                    acceptedButtons: Qt.RightButton
                                    z: -1
                                    onClicked: function(mouse) {
                                        if (mouse.button === Qt.RightButton) {
                                            favContextMenu.deviceId = modelData.device_id
                                            favContextMenu.deviceName = modelData.device_name || ""
                                            var pos = mapToItem(root, mouse.x, mouse.y)
                                            favContextMenu.x = pos.x
                                            favContextMenu.y = pos.y
                                            favContextMenu.open()
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        // ---- Connection Logs Section ----
        QDAccordion {
            Layout.fillWidth: true
            Layout.fillHeight: true
            title: qsTr("Connection Logs") + (logsRepeater.count > 0 ? " (" + logsRepeater.count + ")" : "")
            iconSource: FluentIconGlyph.historyGlyph
            expanded: false

            Item {
                width: parent.width
                height: logsRepeater.count === 0 ? emptyLogsText.height : Math.min(logsRepeater.count, 6) * 40

                Text {
                    id: emptyLogsText
                    visible: logsRepeater.count === 0
                    text: qsTr("No connection logs")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                ScrollView {
                    anchors.fill: parent
                    visible: logsRepeater.count > 0
                    clip: true
                    background: Rectangle { color: "transparent" }
                    ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                    ScrollBar.vertical: QDScrollBar {}

                    Column {
                        width: parent.width
                        spacing: 2

                        Repeater {
                            id: logsRepeater
                            model: {
                                var logs = root.mainController && root.mainController.cloudDeviceManager
                                          ? root.mainController.cloudDeviceManager.connectionLogs : []
                                return logs.length > 20 ? logs.slice(0, 20) : logs
                            }

                            delegate: Rectangle {
                                required property var modelData
                                required property int index

                                width: parent.width
                                height: 40
                                radius: Theme.radiusSmall
                                color: "transparent"

                                RowLayout {
                                    anchors.fill: parent
                                    anchors.leftMargin: Theme.spacingSmall
                                    anchors.rightMargin: Theme.spacingSmall
                                    spacing: Theme.spacingSmall

                                    Rectangle {
                                        width: 8; height: 8; radius: 4
                                        color: modelData.status === "success" ? Theme.success : Theme.error
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: {
                                            var parts = [modelData.device_id || ""]
                                            if (modelData.created_at) parts.push(new Date(modelData.created_at).toLocaleString())
                                            if (modelData.duration > 0) {
                                                var m = Math.floor(modelData.duration / 60)
                                                var s = modelData.duration % 60
                                                parts.push(m + "m" + s + "s")
                                            }
                                            if (modelData.error_msg) parts.push(modelData.error_msg)
                                            return parts.join(" · ")
                                        }
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        elide: Text.ElideRight
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    // Context menu for My Devices
    QDMenu {
        id: deviceContextMenu
        property string deviceId: ""
        property string deviceRemark: ""

        QDMenuItem {
            text: qsTr("Set Remark")
            iconText: FluentIconGlyph.editGlyph
            onTriggered: {
                remarkDialog.deviceId = deviceContextMenu.deviceId
                remarkDialog.isFavorite = false
                remarkDialog.currentRemark = deviceContextMenu.deviceRemark
                remarkDialog.open()
            }
        }
        QDMenuItem {
            text: qsTr("Remove")
            iconText: FluentIconGlyph.deleteGlyph
            isDestructive: true
            onTriggered: root.mainController.cloudDeviceManager.unbindDevice(deviceContextMenu.deviceId)
        }
    }

    // Context menu for Favorites
    QDMenu {
        id: favContextMenu
        property string deviceId: ""
        property string deviceName: ""

        QDMenuItem {
            text: qsTr("Edit Remark")
            iconText: FluentIconGlyph.editGlyph
            onTriggered: {
                remarkDialog.deviceId = favContextMenu.deviceId
                remarkDialog.isFavorite = true
                remarkDialog.currentRemark = favContextMenu.deviceName
                remarkDialog.open()
            }
        }
        QDMenuItem {
            text: qsTr("Remove Favorite")
            iconText: FluentIconGlyph.favoriteStarGlyph
            isDestructive: true
            onTriggered: root.mainController.cloudDeviceManager.removeFavorite(favContextMenu.deviceId)
        }
    }

    // Remark edit dialog
    Popup {
        id: remarkDialog
        modal: true
        anchors.centerIn: parent
        width: 300
        padding: Theme.spacingLarge

        property string deviceId: ""
        property bool isFavorite: false
        property string currentRemark: ""

        background: Rectangle {
            color: Theme.surface
            radius: Theme.radiusLarge
            border.width: Theme.borderWidthThin
            border.color: Theme.border
        }

        onOpened: remarkField.text = currentRemark

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingMedium

            Text {
                text: qsTr("Set Remark")
                font.pixelSize: Theme.fontSizeLarge
                font.weight: Font.Bold
                color: Theme.text
            }

            QDTextField {
                id: remarkField
                Layout.fillWidth: true
                placeholderText: qsTr("Enter remark name")
            }

            RowLayout {
                Layout.alignment: Qt.AlignRight
                spacing: Theme.spacingSmall

                QDButton {
                    text: qsTr("Cancel")
                    onClicked: remarkDialog.close()
                }

                QDButton {
                    text: qsTr("Save")
                    highlighted: true
                    enabled: remarkField.text.length > 0
                    onClicked: {
                        var id = remarkDialog.deviceId
                        var text = remarkField.text
                        var isOwnDevice = root.myDevices.some(function(d) { return d.device_id === id })
                        var isFav = root.mainController.cloudDeviceManager.myFavorites.some(function(f) { return f.device_id === id })
                        if (isOwnDevice)
                            root.mainController.cloudDeviceManager.setDeviceRemark(id, text)
                        if (isFav)
                            root.mainController.cloudDeviceManager.updateFavorite(id, text, "")
                        remarkDialog.close()
                    }
                }
            }
        }
    }

    // Login dialog
    LoginDialog {
        id: loginDialog
        mainController: root.mainController
    }
}
