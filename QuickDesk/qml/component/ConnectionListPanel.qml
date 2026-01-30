// Connection List Panel - Shows connected clients or remote connections
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    
    // ============ Custom Properties ============
    
    property var mainController
    property string panelType: "clients"  // "clients" or "connections"
    property string title: panelType === "clients" ? qsTr("Connected Clients") : qsTr("My Remote Connections")
    property var listModel: panelType === "clients" ? 
        (mainController && mainController.hostManager ? mainController.hostManager.clientIds : []) :
        (mainController && mainController.clientManager ? mainController.clientManager.connectionIds : [])
    
    signal viewConnectionRequested(string connectionId)
    signal disconnectRequested(string connectionId)  // 统一的断开信号
    
    spacing: Theme.spacingSmall
    
    // ============ Title (Outside Card) ============
    
    Text {
        text: root.title
        font.pixelSize: Theme.fontSizeLarge
        font.weight: Font.DemiBold
        color: Theme.text
    }
    
    // ============ Card with List ============
    
    QDCard {
        Layout.fillWidth: true
        Layout.fillHeight: true
        
        Item {
            anchors.fill: parent
            anchors.margins: Theme.spacingMedium
            
            // Empty state
            QDEmptyState {
                anchors.centerIn: parent
                visible: !root.listModel || root.listModel.length === 0
                iconSource: root.panelType === "clients" ? 
                    FluentIconGlyph.contactGlyph : 
                    FluentIconGlyph.remoteGlyph
                title: root.panelType === "clients" ? 
                    qsTr("No clients connected") : 
                    qsTr("No remote connections")
                description: ""
            }
            
            // List view
            ListView {
                id: listView
                anchors.fill: parent
                visible: root.listModel && root.listModel.length > 0
                model: root.listModel
                spacing: Theme.spacingSmall
                clip: true
                
                ScrollBar.vertical: QDScrollBar {
                    policy: ScrollBar.AsNeeded
                }
                
                delegate: Rectangle {
                    width: listView.width - (listView.ScrollBar.vertical.visible ? listView.ScrollBar.vertical.width : 0)
                    height: 56
                    radius: Theme.radiusMedium
                    color: itemMouseArea.containsMouse ? Theme.surfaceHover : Theme.surfaceVariant
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border
                    
                    Behavior on color {
                        ColorAnimation { 
                            duration: Theme.animationDurationFast 
                        }
                    }
                    
                    MouseArea {
                        id: itemMouseArea
                        anchors.fill: parent
                        hoverEnabled: true
                        
                        onDoubleClicked: {
                            if (root.panelType === "connections") {
                                var state = root.mainController.clientManager.getConnectionState(modelData)
                                if (state === "connected" || state === "已连接") {
                                    root.viewConnectionRequested(modelData)
                                }
                            }
                        }
                    }
                    
                    RowLayout {
                        anchors.fill: parent
                        anchors.margins: Theme.spacingMedium
                        spacing: Theme.spacingMedium
                        anchors.verticalCenter: parent.verticalCenter  // 确保整行垂直居中
                        
                        // Icon
                        Text {
                            text: root.panelType === "clients" ? 
                                FluentIconGlyph.contactGlyph :  // 客户端用联系人图标
                                FluentIconGlyph.remoteGlyph      // 远程连接用远程图标
                            font.family: "Segoe Fluent Icons"
                            font.pixelSize: 20
                            color: Theme.primary
                            verticalAlignment: Text.AlignVCenter
                            Layout.alignment: Qt.AlignVCenter  // Layout中垂直居中
                        }
                        
                        // Info column
                        Text {
                            Layout.fillWidth: true
                            text: {
                                if (root.panelType === "clients") {
                                    // 客户端：只显示用户名，去掉状态
                                    return root.mainController.hostManager.getClientUsername(modelData) || modelData
                                } else {
                                    // 远程连接：去掉 "Device: " 前缀
                                    return modelData
                                }
                            }
                            font.pixelSize: Theme.fontSizeMedium
                            font.weight: Font.DemiBold
                            color: Theme.text
                            elide: Text.ElideMiddle
                            verticalAlignment: Text.AlignVCenter  // 垂直居中
                            Layout.alignment: Qt.AlignVCenter  // Layout中垂直居中
                        }
                        
                        // View button (only for connections)
                        QDIconButton {
                            visible: root.panelType === "connections"
                            iconSource: FluentIconGlyph.fullScreenGlyph
                            buttonStyle: QDIconButton.Style.Subtle
                            Layout.alignment: Qt.AlignVCenter  // Layout中垂直居中
                            enabled: {
                                if (root.panelType === "connections") {
                                    var state = root.mainController.clientManager.getConnectionState(modelData)
                                    return state === "connected" || state === "已连接"
                                }
                                return false
                            }
                            onClicked: {
                                root.viewConnectionRequested(modelData)
                            }
                            
                            QDToolTip {
                                visible: parent.hovered
                                text: qsTr("View Remote Desktop")
                            }
                        }
                        
                        // Disconnect button
                        QDIconButton {
                            iconSource: FluentIconGlyph.cancelGlyph  // 使用 cancel 图标，类似 Main.qml 的"断开"
                            buttonStyle: QDIconButton.Style.Subtle
                            Layout.alignment: Qt.AlignVCenter  // Layout中垂直居中
                            onClicked: {
                                if (root.panelType === "clients") {
                                    // 踢出客户端 - 直接调用
                                    console.log("Kicking client:", modelData)
                                    root.mainController.hostManager.kickClient(modelData)
                                } else {
                                    // 断开远程连接 - 发送信号给父组件处理
                                    console.log("Disconnect requested for remote connection:", modelData)
                                    root.disconnectRequested(modelData)
                                }
                            }
                            
                            QDToolTip {
                                visible: parent.hovered
                                text: qsTr("Disconnect")
                            }
                        }
                    }
                }
            }
        }
    }
}
