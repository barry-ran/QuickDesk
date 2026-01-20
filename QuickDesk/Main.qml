import QtQuick
import QtQuick.Window
import QtQuick.Controls
import QtQuick.Layouts
import QuickDesk 1.0

ApplicationWindow {
    id: root
    width: 900
    height: 600
    visible: true
    title: "QuickDesk - 远程桌面"

    // Main controller
    MainController {
        id: mainController
        
        onInitializationFailed: (error) => {
            console.error("Initialization failed:", error)
            statusText.text = "初始化失败: " + error
        }
        
        onDeviceIdChanged: {
            console.log("Device ID changed:", mainController.deviceId)
        }
        
        onAccessCodeChanged: {
            console.log("Access Code changed:", mainController.accessCode)
        }
    }

    Component.onCompleted: {
        console.log("Main.qml loaded, initializing...")
        mainController.initialize()
    }

    Component.onDestruction: {
        console.log("Main.qml unloading, shutting down...")
        mainController.shutdown()
    }

    // Main layout
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 20

        // Header
        Text {
            text: "QuickDesk"
            font.pixelSize: 28
            font.bold: true
            color: "#2196F3"
        }

        // Status bar
        Rectangle {
            Layout.fillWidth: true
            height: 40
            color: mainController.isInitialized ? "#E8F5E9" : "#FFF3E0"
            radius: 5
            
            RowLayout {
                anchors.fill: parent
                anchors.margins: 10
                
                Text {
                    id: statusText
                    text: mainController.initStatus
                    color: mainController.isInitialized ? "#2E7D32" : "#E65100"
                }
                
                Item { Layout.fillWidth: true }
                
                Text {
                    text: mainController.serverManager ? 
                          "服务器: " + mainController.serverManager.serverUrl : ""
                    color: "#666"
                    font.pixelSize: 12
                }
            }
        }

        // Main content
        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 20

            // Left panel - Host info
            Rectangle {
                Layout.preferredWidth: 300
                Layout.fillHeight: true
                color: "#F5F5F5"
                radius: 10

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 15
                    spacing: 15

                    Text {
                        text: "我的设备"
                        font.pixelSize: 18
                        font.bold: true
                        color: "#333"
                    }

                    // Device ID card
                    Rectangle {
                        Layout.fillWidth: true
                        height: 100
                        color: "white"
                        radius: 8
                        border.color: "#E0E0E0"
                        border.width: 1

                        ColumnLayout {
                            anchors.fill: parent
                            anchors.margins: 12
                            spacing: 8

                            Text {
                                text: "设备 ID"
                                font.pixelSize: 12
                                color: "#666"
                            }

                            Text {
                                text: mainController.deviceId || "获取中..."
                                font.pixelSize: 24
                                font.bold: true
                                font.family: "Consolas"
                                color: mainController.deviceId ? "#1976D2" : "#999"
                            }
                        }
                    }

                    // Access code card
                    Rectangle {
                        Layout.fillWidth: true
                        height: 120
                        color: "white"
                        radius: 8
                        border.color: "#E0E0E0"
                        border.width: 1

                        ColumnLayout {
                            anchors.fill: parent
                            anchors.margins: 12
                            spacing: 8

                            Text {
                                text: "临时访问码"
                                font.pixelSize: 12
                                color: "#666"
                            }

                            RowLayout {
                                Text {
                                    text: mainController.accessCode || "获取中..."
                                    font.pixelSize: 24
                                    font.bold: true
                                    font.family: "Consolas"
                                    color: mainController.accessCode ? "#4CAF50" : "#999"
                                }
                                
                                Item { Layout.fillWidth: true }
                                
                                // Refresh button
                                Rectangle {
                                    width: 30
                                    height: 30
                                    radius: 15
                                    color: refreshArea.containsMouse ? "#E3F2FD" : "transparent"
                                    enabled: mainController.signalingState === "connected"
                                    opacity: enabled ? 1.0 : 0.5
                                    
                                    Text {
                                        anchors.centerIn: parent
                                        text: "🔄"
                                        font.pixelSize: 16
                                    }
                                    
                                    MouseArea {
                                        id: refreshArea
                                        anchors.fill: parent
                                        hoverEnabled: true
                                        enabled: parent.enabled
                                        cursorShape: enabled ? Qt.PointingHandCursor : Qt.ForbiddenCursor
                                        onClicked: {
                                            console.log("Refresh access code clicked")
                                            mainController.refreshTempPassword()
                                        }
                                    }
                                    
                                    ToolTip.visible: refreshArea.containsMouse
                                    ToolTip.text: mainController.signalingState === "connected" ? 
                                                  "刷新临时密码" : "需要先连接信令服务器"
                                }
                            }
                            
                            // Copy buttons row
                            RowLayout {
                                Layout.fillWidth: true
                                spacing: 8
                                
                                Button {
                                    text: "复制设备ID"
                                    font.pixelSize: 11
                                    enabled: mainController.deviceId.length > 0
                                    onClicked: {
                                        mainController.copyToClipboard(mainController.deviceId)
                                        copyIdTip.visible = true
                                        copyIdTimer.start()
                                    }
                                    
                                    ToolTip {
                                        id: copyIdTip
                                        text: "已复制!"
                                        visible: false
                                    }
                                    
                                    Timer {
                                        id: copyIdTimer
                                        interval: 1500
                                        onTriggered: copyIdTip.visible = false
                                    }
                                }
                                
                                Button {
                                    text: "复制访问码"
                                    font.pixelSize: 11
                                    enabled: mainController.accessCode.length > 0
                                    onClicked: {
                                        mainController.copyToClipboard(mainController.accessCode)
                                        copyCodeTip.visible = true
                                        copyCodeTimer.start()
                                    }
                                    
                                    ToolTip {
                                        id: copyCodeTip
                                        text: "已复制!"
                                        visible: false
                                    }
                                    
                                    Timer {
                                        id: copyCodeTimer
                                        interval: 1500
                                        onTriggered: copyCodeTip.visible = false
                                    }
                                }
                                
                                Button {
                                    text: "复制全部"
                                    font.pixelSize: 11
                                    enabled: mainController.deviceId.length > 0 && mainController.accessCode.length > 0
                                    onClicked: {
                                        mainController.copyDeviceInfo()
                                        copyAllTip.visible = true
                                        copyAllTimer.start()
                                    }
                                    
                                    ToolTip {
                                        id: copyAllTip
                                        text: "已复制!"
                                        visible: false
                                    }
                                    
                                    Timer {
                                        id: copyAllTimer
                                        interval: 1500
                                        onTriggered: copyAllTip.visible = false
                                    }
                                }
                            }
                        }
                    }

                    // Connection status
                    Rectangle {
                        Layout.fillWidth: true
                        height: 60
                        color: {
                            var state = mainController.signalingState
                            if (state === "connected") return "#E8F5E9"
                            if (state === "connecting" || state === "reconnecting") return "#FFF8E1"
                            return "#FFEBEE"
                        }
                        radius: 8

                        ColumnLayout {
                            anchors.fill: parent
                            anchors.margins: 10
                            spacing: 4

                            RowLayout {
                                Layout.fillWidth: true

                                // Status indicator
                                Rectangle {
                                    width: 12
                                    height: 12
                                    radius: 6
                                    color: {
                                        var state = mainController.signalingState
                                        if (state === "connected") return "#4CAF50"
                                        if (state === "connecting" || state === "reconnecting") return "#FF9800"
                                        return "#F44336"
                                    }
                                    
                                    // Blinking animation for connecting/reconnecting
                                    SequentialAnimation on opacity {
                                        loops: Animation.Infinite
                                        running: mainController.signalingState === "connecting" || 
                                                 mainController.signalingState === "reconnecting"
                                        NumberAnimation { to: 0.3; duration: 500 }
                                        NumberAnimation { to: 1; duration: 500 }
                                    }
                                }

                                Text {
                                    text: mainController.signalingStatusText
                                    color: {
                                        var state = mainController.signalingState
                                        if (state === "connected") return "#2E7D32"
                                        if (state === "connecting" || state === "reconnecting") return "#E65100"
                                        return "#C62828"
                                    }
                                    font.pixelSize: 13
                                }
                            }

                            // Show error message if any
                            Text {
                                visible: mainController.signalingError.length > 0 && 
                                         mainController.signalingState !== "connected"
                                text: mainController.signalingError
                                color: "#999"
                                font.pixelSize: 11
                                elide: Text.ElideRight
                                Layout.fillWidth: true
                            }
                        }
                    }

                    Item { Layout.fillHeight: true }

                    // Connected clients section
                    Text {
                        text: "已连接的客户端"
                        font.pixelSize: 14
                        font.bold: true
                        color: "#333"
                    }

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        Layout.minimumHeight: 100
                        color: "white"
                        radius: 8
                        border.color: "#E0E0E0"
                        border.width: 1
                        clip: true

                        // Empty state
                        Text {
                            anchors.centerIn: parent
                            visible: !mainController.hostManager || 
                                     mainController.hostManager.clientCount === 0
                            text: mainController.hostManager ? "暂无连接" : "加载中..."
                            color: "#999"
                        }

                        // Client list
                        ListView {
                            id: clientList
                            anchors.fill: parent
                            anchors.margins: 8
                            visible: mainController.hostManager && 
                                     mainController.hostManager.clientCount > 0
                            model: mainController.hostManager ? 
                                   mainController.hostManager.clientIds : []
                            spacing: 8
                            
                            delegate: Rectangle {
                                width: clientList.width
                                height: 50
                                color: "#F5F5F5"
                                radius: 6
                                
                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: 10
                                    spacing: 10
                                    
                                    // Client info
                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        spacing: 2
                                        
                                        Text {
                                            text: mainController.hostManager.getClientUsername(modelData) || modelData
                                            font.pixelSize: 13
                                            font.bold: true
                                            color: "#333"
                                            elide: Text.ElideMiddle
                                            Layout.fillWidth: true
                                        }
                                        
                                        Text {
                                            text: mainController.hostManager.getClientState(modelData) || "已连接"
                                            font.pixelSize: 11
                                            color: "#4CAF50"
                                        }
                                    }
                                    
                                    // Disconnect button
                                    Button {
                                        text: "断开"
                                        font.pixelSize: 11
                                        onClicked: {
                                            console.log("Kicking client:", modelData)
                                            mainController.hostManager.kickClient(modelData)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }

            // Right panel - Client controls
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: "#FAFAFA"
                radius: 10

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 15
                    spacing: 15

                    Text {
                        text: "连接远程主机"
                        font.pixelSize: 18
                        font.bold: true
                        color: "#333"
                    }

                    // Connection input
                    Rectangle {
                        Layout.fillWidth: true
                        height: 180
                        color: "white"
                        radius: 8
                        border.color: "#E0E0E0"
                        border.width: 1

                        ColumnLayout {
                            anchors.fill: parent
                            anchors.margins: 15
                            spacing: 12

                            // Device ID input
                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 4

                                Text {
                                    text: "设备 ID"
                                    font.pixelSize: 12
                                    color: "#666"
                                }

                                TextField {
                                    id: remoteDeviceId
                                    Layout.fillWidth: true
                                    placeholderText: "输入远程设备 ID"
                                    font.pixelSize: 16
                                }
                            }

                            // Access code input
                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 4

                                Text {
                                    text: "访问码"
                                    font.pixelSize: 12
                                    color: "#666"
                                }

                                TextField {
                                    id: remoteAccessCode
                                    Layout.fillWidth: true
                                    placeholderText: "输入访问码"
                                    font.pixelSize: 16
                                    echoMode: TextInput.Password
                                }
                            }

                            // Connect button
                            Button {
                                Layout.fillWidth: true
                                text: "连接"
                                enabled: remoteDeviceId.text.length > 0 && 
                                         remoteAccessCode.text.length > 0 &&
                                         mainController.isInitialized
                                
                                onClicked: {
                                    console.log("Connecting to:", remoteDeviceId.text)
                                    var connId = mainController.connectToRemoteHost(
                                        remoteDeviceId.text,
                                        remoteAccessCode.text
                                    )
                                    console.log("Connection ID:", connId)
                                }
                            }
                        }
                    }

                    // Active connections
                    Text {
                        text: "我的远程连接"
                        font.pixelSize: 14
                        font.bold: true
                        color: "#333"
                    }

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        color: "white"
                        radius: 8
                        border.color: "#E0E0E0"
                        border.width: 1
                        clip: true

                        // Empty state
                        Text {
                            anchors.centerIn: parent
                            visible: !mainController.clientManager || 
                                     mainController.clientManager.connectionCount === 0
                            text: mainController.clientManager ? "暂无远程连接" : "加载中..."
                            color: "#999"
                        }

                        // Connection list
                        ListView {
                            id: connectionList
                            anchors.fill: parent
                            anchors.margins: 8
                            visible: mainController.clientManager && 
                                     mainController.clientManager.connectionCount > 0
                            model: mainController.clientManager ? 
                                   mainController.clientManager.connectionIds : []
                            spacing: 8
                            
                            delegate: Rectangle {
                                width: connectionList.width
                                height: 50
                                color: "#F5F5F5"
                                radius: 6
                                
                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: 10
                                    spacing: 10
                                    
                                    // Connection info
                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        spacing: 2
                                        
                                        Text {
                                            text: "设备: " + modelData
                                            font.pixelSize: 13
                                            font.bold: true
                                            color: "#333"
                                        }
                                        
                                        Text {
                                            text: mainController.clientManager.getConnectionState(modelData) || "连接中..."
                                            font.pixelSize: 11
                                            color: "#666"
                                        }
                                    }
                                    
                                    // Disconnect button
                                    Button {
                                        text: "断开"
                                        font.pixelSize: 11
                                        onClicked: {
                                            console.log("Disconnecting:", modelData)
                                            mainController.disconnectFromRemoteHost(modelData)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    
    // Toast notification for errors
    Rectangle {
        id: toast
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 50
        width: toastText.width + 40
        height: 40
        radius: 20
        color: "#333"
        opacity: 0
        visible: opacity > 0
        
        Text {
            id: toastText
            anchors.centerIn: parent
            color: "white"
            font.pixelSize: 14
        }
        
        Behavior on opacity {
            NumberAnimation { duration: 200 }
        }
        
        Timer {
            id: toastTimer
            interval: 3000
            onTriggered: toast.opacity = 0
        }
        
        function show(message) {
            toastText.text = message
            toast.opacity = 1
            toastTimer.restart()
        }
    }
    
    // Connect to refresh password result
    Connections {
        target: mainController.hostManager
        function onRefreshTempPasswordResult(success, errorCode, errorMessage) {
            if (!success) {
                toast.show("刷新失败: " + errorMessage)
            }
        }
    }
}
