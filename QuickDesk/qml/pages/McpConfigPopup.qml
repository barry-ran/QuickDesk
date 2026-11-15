import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"

Popup {
    id: popup
    
    property var mainController

    function _showToast(message, toastType) {
        internalToast.show(message, toastType)
    }
    
    width: 400
    height: contentColumn.implicitHeight + padding * 2
    padding: Theme.spacingLarge
    
    // Anchor position set once on open so only the bottom moves on tab switch
    property real _anchorY: 0
    x: (parent.width - width) / 2
    y: _anchorY
    onAboutToShow: _anchorY = (parent.height - height) / 2
    
    modal: true
    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside
    
    background: Rectangle {
        color: Theme.surface
        radius: Theme.radiusMedium
        border.width: Theme.borderWidthThin
        border.color: Theme.border
        
        QDShadow { anchors.fill: parent; radius: parent.radius }
    }
    
    enter: Transition {
        NumberAnimation { property: "opacity"; from: 0.0; to: 1.0; duration: 150 }
        NumberAnimation { property: "scale"; from: 0.95; to: 1.0; duration: 150; easing.type: Easing.OutCubic }
    }
    
    exit: Transition {
        NumberAnimation { property: "opacity"; from: 1.0; to: 0.0; duration: 100 }
    }

    // Internal state
    property bool isHttpMode: mainController ? mainController.mcpTransportMode === "http" : false

    ColumnLayout {
        id: contentColumn
        width: parent.width
        spacing: Theme.spacingMedium
        
        // Header
        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall
            
            Text {
                text: FluentIconGlyph.robotGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 20
                color: Theme.primary
            }
            
            Text {
                text: qsTr("AI Integration (MCP)")
                font.pixelSize: Theme.fontSizeLarge
                font.weight: Font.DemiBold
                color: Theme.text
                Layout.fillWidth: true
            }
        }

        QDDivider { Layout.fillWidth: true }

        // ========== Connection Mode ==========
        Text {
            text: qsTr("Connection Mode")
            font.pixelSize: Theme.fontSizeMedium
            font.weight: Font.DemiBold
            color: Theme.text
        }

        QDToggleButtonGroup {
            Layout.fillWidth: true
            buttonSize: QDToggleButtonGroup.Size.Small
            options: [
                { text: qsTr("stdio"), value: "stdio" },
                { text: qsTr("HTTP/SSE"), value: "http" }
            ]
            currentIndex: popup.isHttpMode ? 1 : 0
            onValueChanged: function(value) {
                if (mainController) {
                    mainController.mcpTransportMode = value
                }
            }
        }

        Text {
            text: popup.isHttpMode
                  ? qsTr("HTTP/SSE mode: QuickDesk runs the MCP server. AI clients connect via network.")
                  : qsTr("stdio mode: AI client launches the MCP process automatically.")
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.textSecondary
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
            Layout.minimumHeight: implicitHeight
            Layout.preferredHeight: 32
        }

        QDDivider { Layout.fillWidth: true }



        // ========== Configure AI Client ==========
        // Configure AI Client section
        Text {
            text: qsTr("Configure AI Client")
            font.pixelSize: Theme.fontSizeMedium
            font.weight: Font.DemiBold
            color: Theme.text
        }
        
        Text {
            text: qsTr("Auto-configure or copy the config for your AI client.")
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.textSecondary
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
        }
        
        // Client config buttons - 2 column grid
        GridLayout {
            Layout.fillWidth: true
            columns: 2
            columnSpacing: Theme.spacingSmall
            rowSpacing: Theme.spacingSmall
            
            Repeater {
                model: [
                    { name: "Cursor", type: "cursor", icon: FluentIconGlyph.codeGlyph },
                    { name: "Claude Desktop", type: "claude", icon: FluentIconGlyph.chatBubblesGlyph },
                    { name: "Windsurf", type: "windsurf", icon: FluentIconGlyph.codeGlyph },
                    { name: "VS Code", type: "vscode", icon: FluentIconGlyph.codeGlyph }
                ]
                
                delegate: Rectangle {
                    id: configBtn
                    required property var modelData
                    required property int index
                    
                    Layout.fillWidth: true
                    height: 44
                    radius: Theme.radiusSmall
                    color: Theme.surfaceVariant
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border
                    
                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacingMedium
                        anchors.rightMargin: Theme.spacingSmall
                        spacing: Theme.spacingSmall
                        
                        Text {
                            text: configBtn.modelData.icon
                            font.family: "Segoe Fluent Icons"
                            font.pixelSize: 14
                            color: Theme.primary
                        }
                        
                        Text {
                            text: configBtn.modelData.name
                            font.pixelSize: Theme.fontSizeSmall
                            font.weight: Font.DemiBold
                            color: Theme.text
                            Layout.fillWidth: true
                        }
                        
                        // Auto-configure button
                        QDIconButton {
                            iconSource: FluentIconGlyph.downloadGlyph
                            buttonSize: QDIconButton.Size.Small
                            visible: Qt.platform.os === "windows" || Qt.platform.os === "osx" || Qt.platform.os === "linux"
                            onClicked: {
                                var result = mainController.writeMcpConfig(configBtn.modelData.type)
                                if (result === 0) {
                                    popup._showToast(
                                        qsTr("%1 configured! Restart to apply.").arg(configBtn.modelData.name), 0)
                                } else {
                                    popup._showToast(
                                        qsTr("Failed to write config. Copy manually."), 2)
                                }
                            }
                            QDToolTip {
                                visible: parent.hovered
                                text: qsTr("Auto-configure %1").arg(configBtn.modelData.name)
                            }
                        }
                        
                        // Copy config button
                        QDIconButton {
                            iconSource: FluentIconGlyph.copyGlyph
                            buttonSize: QDIconButton.Size.Small
                            onClicked: {
                                mainController.copyMcpConfig(configBtn.modelData.type)
                                popup._showToast(
                                    qsTr("%1 config copied").arg(configBtn.modelData.name), 0)
                            }
                            QDToolTip {
                                visible: parent.hovered
                                text: qsTr("Copy %1 config").arg(configBtn.modelData.name)
                            }
                        }
                    }
                }
            }
        }
        
        QDDivider { Layout.fillWidth: true }
        
        // ========== MCP Binary Path (stdio mode) ==========
        Text {
            text: qsTr("MCP Binary Path")
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.textSecondary
            visible: !popup.isHttpMode
        }
        
        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall
            visible: !popup.isHttpMode
            
            Rectangle {
                Layout.fillWidth: true
                height: 32
                radius: Theme.radiusSmall
                color: Theme.surfaceVariant
                border.width: Theme.borderWidthThin
                border.color: Theme.border
                clip: true
                
                Text {
                    anchors.fill: parent
                    anchors.leftMargin: Theme.spacingSmall
                    anchors.rightMargin: Theme.spacingSmall
                    text: mainController ? mainController.getMcpBinaryPath() : ""
                    font.pixelSize: Theme.fontSizeSmall - 1
                    font.family: "Consolas"
                    color: Theme.text
                    elide: Text.ElideMiddle
                    verticalAlignment: Text.AlignVCenter
                }
            }
            
            QDIconButton {
                iconSource: FluentIconGlyph.copyGlyph
                buttonSize: QDIconButton.Size.Small
                onClicked: {
                    if (mainController) {
                        mainController.copyToClipboard(mainController.getMcpBinaryPath())
                        popup._showToast(qsTr("Path copied"), 0)
                    }
                }
                
                QDToolTip {
                    visible: parent.hovered
                    text: qsTr("Copy path")
                }
            }
        }

        // ========== MCP HTTP Service (HTTP mode) ==========
        ColumnLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall
            visible: popup.isHttpMode

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingMedium

                Text {
                    text: qsTr("MCP HTTP Service")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                Text {
                    text: {
                        if (!mainController) return ""
                        if (!mainController.mcpServiceRunning)
                            return qsTr("Starting...")
                        return qsTr("Running")
                    }
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                Item { Layout.fillWidth: true }

                Rectangle {
                    width: 10; height: 10
                    radius: 5
                    color: mainController && mainController.mcpServiceRunning
                           ? "#4CAF50" : Theme.textSecondary
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingSmall
                visible: mainController && mainController.mcpServiceRunning

                Repeater {
                    model: [
                        {
                            label: qsTr("Local"),
                            url: mainController ? mainController.mcpHttpUrl : "",
                            hint: qsTr("Use on this computer")
                        },
                        {
                            label: qsTr("LAN"),
                            url: mainController ? mainController.mcpHttpLanUrl : "",
                            hint: qsTr("Use from another device on the same network")
                        }
                    ]

                    delegate: RowLayout {
                        required property var modelData

                        Layout.fillWidth: true
                        spacing: Theme.spacingSmall
                        visible: modelData.url.length > 0

                        Text {
                            text: modelData.label
                            font.pixelSize: Theme.fontSizeSmall - 1
                            color: Theme.textSecondary
                            Layout.preferredWidth: 44
                        }

                        Rectangle {
                            Layout.fillWidth: true
                            height: 32
                            radius: Theme.radiusSmall
                            color: Theme.surfaceVariant
                            border.width: Theme.borderWidthThin
                            border.color: Theme.border
                            clip: true

                            Text {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingSmall
                                anchors.rightMargin: Theme.spacingSmall
                                text: modelData.url
                                font.pixelSize: Theme.fontSizeSmall
                                font.family: "Consolas"
                                color: Theme.primary
                                elide: Text.ElideRight
                                verticalAlignment: Text.AlignVCenter
                            }
                        }

                        QDIconButton {
                            iconSource: FluentIconGlyph.copyGlyph
                            buttonSize: QDIconButton.Size.Small
                            onClicked: {
                                if (mainController && modelData.url) {
                                    mainController.copyToClipboard(modelData.url)
                                    popup._showToast(qsTr("URL copied"), 0)
                                }
                            }
                            QDToolTip {
                                visible: parent.hovered
                                text: modelData.hint
                            }
                        }
                    }
                }

                Text {
                    Layout.fillWidth: true
                    text: qsTr("For remote access, use the LAN URL and allow the port through your firewall.")
                    font.pixelSize: Theme.fontSizeSmall - 1
                    color: Theme.textSecondary
                    wrapMode: Text.WordWrap
                    visible: mainController && mainController.mcpHttpLanUrl.length > 0
                }
            }
        }
    }

    // Internal toast (shown above the popup content)
    QDToast {
        id: internalToast
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: Theme.spacingMedium
    }
}
