import QtQuick
import QtQuick.Window
import QtQuick.Controls
import QtQuick.Layouts
import QuickDesk 1.0

import "../"
import "../component"
import "../pages"

ApplicationWindow {
    id: root
    width: 800
    height: 600
    minimumWidth: 800
    maximumWidth: 800
    minimumHeight: 600
    maximumHeight: 600
    visible: true
    title: qsTr("QuickDesk")
    
    // 关闭窗口时退出程序
    onClosing: function(close) {
        Qt.quit()
    }
    
    // Remote window management - single window for all connections
    property var remoteWindow: null
    
    // Main controller - reuse existing controller
    property MainController mainController: MainController {
        id: mainControllerObj
        
        onInitializationFailed: function(error) {
            console.error("Initialization failed:", error)
            statusBar.message = qsTr("Initialization failed: ") + error
        }
        
        onDeviceIdChanged: {
            console.log("Device ID changed:", mainControllerObj.deviceId)
        }
        
        onAccessCodeChanged: {
            console.log("Access Code changed:", mainControllerObj.accessCode)
        }
    }
    
    Component.onCompleted: {
        console.log("MainWindow.qml loaded, initializing...")
        mainController.initialize()
    }
    
    Component.onDestruction: {
        console.log("MainWindow.qml unloading, shutting down...")
        mainController.shutdown()
    }
    
    // Function to create or show remote window
    function showRemoteWindow(connectionId, deviceId) {
        console.log("showRemoteWindow called:", connectionId, deviceId)
        
        // Create RemoteWindow if not exists
        if (!remoteWindow) {
            var component = Qt.createComponent("RemoteWindow.qml")
            
            if (component.status === Component.Error) {
                console.error("Error creating RemoteWindow:", component.errorString())
                return
            }
            
            if (component.status === Component.Ready) {
                remoteWindow = component.createObject(root, {
                    clientManager: mainController.clientManager
                })
                
                if (!remoteWindow) {
                    console.error("Failed to create RemoteWindow object")
                    return
                }
                
                // Handle window destruction
                remoteWindow.closing.connect(function() {
                    console.log("RemoteWindow destroyed")
                    remoteWindow = null
                })
            } else {
                console.error("RemoteWindow component not ready:", component.status)
                return
            }
        }
        
        // Add connection to window
        if (remoteWindow) {
            remoteWindow.addConnection(connectionId, deviceId)
            remoteWindow.show()
            remoteWindow.raise()
            remoteWindow.requestActivate()
            console.log("Added connection to remote window:", connectionId)
        }
    }
    
    // Main content with navigation
    QDNavigationView {
        id: navigationView
        anchors.fill: parent
        
        isExpanded: true
        collapsedWidth: 48
        expandedWidth: 200
        
        menuItems: [
            { icon: FluentIconGlyph.remoteGlyph, text: qsTr("Remote Control") },
            { icon: FluentIconGlyph.settingsGlyph, text: qsTr("Settings") }
        ]
        
        header: Item {
            width: parent.width
            height: 72
            
            Column {
                anchors.centerIn: parent
                spacing: Theme.spacingXSmall
                
                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "QuickDesk"
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.Bold
                    color: Theme.text
                }
                
                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "v" + APP_VERSION
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }
            }
        }
        
        footer: Rectangle {
            anchors.fill: parent
            color: Theme.surfaceVariant
            
            Row {
                anchors.centerIn: parent
                spacing: Theme.spacingLarge
                
                // Host Status (show server status if running, otherwise show process status)
                Row {
                    spacing: Theme.spacingXSmall
                    
                    Rectangle {
                        width: 8
                        height: 8
                        radius: 4
                        anchors.verticalCenter: parent.verticalCenter
                        color: {
                            if (!root.mainController) return Theme.textDisabled
                            var processStatus = root.mainController.hostProcessStatus
                            var serverStatus = root.mainController.hostServerStatus
                            
                            // If process is running, show server status color
                            if (processStatus === ProcessStatus.Running) {
                                if (serverStatus === ServerStatus.Connected) return Theme.success
                                if (serverStatus === ServerStatus.Connecting || serverStatus === ServerStatus.Reconnecting) return Theme.warning
                                if (serverStatus === ServerStatus.Failed) return Theme.error
                                return Theme.textDisabled
                            }
                            
                            // Otherwise show process status color
                            if (processStatus === ProcessStatus.Starting || processStatus === ProcessStatus.Restarting) return Theme.warning
                            if (processStatus === ProcessStatus.Failed) return Theme.error
                            return Theme.textDisabled
                        }
                    }
                    
                    Text {
                        text: {
                            if (!root.mainController) return qsTr("Host") + ": " + qsTr("Unknown")
                            var processStatus = root.mainController.hostProcessStatus
                            var serverStatus = root.mainController.hostServerStatus
                            
                            // If process is running, show server status
                            if (processStatus === ProcessStatus.Running) {
                                if (serverStatus === ServerStatus.Disconnected) return qsTr("Host") + ": " + qsTr("Disconnected")
                                if (serverStatus === ServerStatus.Connecting) return qsTr("Host") + ": " + qsTr("Connecting")
                                if (serverStatus === ServerStatus.Connected) return qsTr("Host") + ": " + qsTr("Connected")
                                if (serverStatus === ServerStatus.Failed) return qsTr("Host") + ": " + qsTr("Connection Failed")
                                if (serverStatus === ServerStatus.Reconnecting) return qsTr("Host") + ": " + qsTr("Reconnecting")
                                return qsTr("Host") + ": " + qsTr("Unknown")
                            }
                            
                            // Otherwise show process status
                            if (processStatus === ProcessStatus.NotStarted) return qsTr("Host") + ": " + qsTr("Not Started")
                            if (processStatus === ProcessStatus.Starting) return qsTr("Host") + ": " + qsTr("Starting")
                            if (processStatus === ProcessStatus.Failed) return qsTr("Host") + ": " + qsTr("Start Failed")
                            if (processStatus === ProcessStatus.Restarting) return qsTr("Host") + ": " + qsTr("Restarting")
                            return qsTr("Host") + ": " + qsTr("Unknown")
                        }
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }
                
                // Client Status (show server status if running, otherwise show process status)
                Row {
                    spacing: Theme.spacingXSmall
                    
                    Rectangle {
                        width: 8
                        height: 8
                        radius: 4
                        anchors.verticalCenter: parent.verticalCenter
                        color: {
                            if (!root.mainController) return Theme.textDisabled
                            var processStatus = root.mainController.clientProcessStatus
                            var serverStatus = root.mainController.clientServerStatus
                            
                            // If process is running, show server status color
                            if (processStatus === ProcessStatus.Running) {
                                if (serverStatus === ServerStatus.Connected) return Theme.success
                                if (serverStatus === ServerStatus.Connecting || serverStatus === ServerStatus.Reconnecting) return Theme.warning
                                if (serverStatus === ServerStatus.Failed) return Theme.error
                                return Theme.textDisabled
                            }
                            
                            // Otherwise show process status color
                            if (processStatus === ProcessStatus.Starting || processStatus === ProcessStatus.Restarting) return Theme.warning
                            if (processStatus === ProcessStatus.Failed) return Theme.error
                            return Theme.textDisabled
                        }
                    }
                    
                    Text {
                        text: {
                            if (!root.mainController) return qsTr("Client") + ": " + qsTr("Unknown")
                            var processStatus = root.mainController.clientProcessStatus
                            var serverStatus = root.mainController.clientServerStatus
                            
                            // If process is running, show server status
                            if (processStatus === ProcessStatus.Running) {
                                if (serverStatus === ServerStatus.Disconnected) return qsTr("Client") + ": " + qsTr("Disconnected")
                                if (serverStatus === ServerStatus.Connecting) return qsTr("Client") + ": " + qsTr("Connecting")
                                if (serverStatus === ServerStatus.Connected) return qsTr("Client") + ": " + qsTr("Connected")
                                if (serverStatus === ServerStatus.Failed) return qsTr("Client") + ": " + qsTr("Connection Failed")
                                if (serverStatus === ServerStatus.Reconnecting) return qsTr("Client") + ": " + qsTr("Reconnecting")
                                return qsTr("Client") + ": " + qsTr("Unknown")
                            }
                            
                            // Otherwise show process status
                            if (processStatus === ProcessStatus.NotStarted) return qsTr("Client") + ": " + qsTr("Not Started")
                            if (processStatus === ProcessStatus.Starting) return qsTr("Client") + ": " + qsTr("Starting")
                            if (processStatus === ProcessStatus.Failed) return qsTr("Client") + ": " + qsTr("Start Failed")
                            if (processStatus === ProcessStatus.Restarting) return qsTr("Client") + ": " + qsTr("Restarting")
                            return qsTr("Client") + ": " + qsTr("Unknown")
                        }
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }
            }
        }
        
        content: ColumnLayout {
            anchors.fill: parent
            spacing: 0
            
            StackLayout {
                Layout.fillWidth: true
                Layout.fillHeight: true
                currentIndex: navigationView.currentIndex
                
                // Remote Control Page
                RemoteControlPage {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    mainController: root.mainController
                    onConnectRequested: function(deviceId, password) {
                        console.log("Connect requested:", deviceId)
                        var connId = root.mainController.connectToRemoteHost(deviceId, password)
                        if (connId) {
                            // Create remote window after a short delay to allow connection to establish
                            Qt.callLater(function() {
                                root.showRemoteWindow(connId, deviceId)
                            })
                        }
                    }
                }
                
                // Settings Page
                SettingsPage {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    mainController: root.mainController
                }
            }
            
            // Status Bar
            QDStatusBar {
                id: statusBar
                Layout.fillWidth: true
                leftText: ""
                message: ""
                rightText: ""
            }
        }
    }
    
    // Monitor client connection state changes
    Connections {
        target: root.mainController ? root.mainController.clientManager : null
        
        function onConnectionStateChanged(connectionId, state, hostInfo) {
            console.log("MainWindow: Connection state changed:", connectionId, "->", state)
            
            // If connection is established, ensure the window is shown
            if (state === "connected") {
                // Window should already exist from onConnectRequested, 
                // but we'll ensure it's visible and raised
                if (remoteWindow) {
                    remoteWindow.show()
                    remoteWindow.raise()
                    remoteWindow.requestActivate()
                }
            }
        }
    }
}
