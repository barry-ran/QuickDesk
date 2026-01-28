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
    title: qsTr("QuickDesk - New UI")
    
    // 关闭窗口时退出程序
    onClosing: function(close) {
        Qt.quit()
    }
    
    // Remote window management
    property var remoteWindows: ({}) // Map of connectionId -> Window
    
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
        
        // Check if window already exists for this connection
        if (remoteWindows[connectionId]) {
            // Window exists, activate it
            var existingWindow = remoteWindows[connectionId]
            existingWindow.raise()
            existingWindow.requestActivate()
            console.log("Activated existing remote window for:", connectionId)
            return
        }
        
        // Create new remote window
        var component = Qt.createComponent("RemoteWindow.qml")
        if (component.status === Component.Error) {
            console.error("Error creating RemoteWindow:", component.errorString())
            return
        }
        
        if (component.status === Component.Ready) {
            var window = component.createObject(root, {
                clientManager: mainController.clientManager
            })
            
            if (window) {
                window.addConnection(connectionId, deviceId)
                
                // Handle window closing
                window.closing.connect(function() {
                    console.log("Remote window closing for:", connectionId)
                    delete remoteWindows[connectionId]
                })
                
                remoteWindows[connectionId] = window
                window.show()
                console.log("Created new remote window for:", connectionId)
            } else {
                console.error("Failed to create remote window object")
            }
        } else {
            console.error("RemoteWindow component not ready:", component.status)
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
                spacing: Theme.spacingSmall
                
                Rectangle {
                    width: 8
                    height: 8
                    radius: 4
                    anchors.verticalCenter: parent.verticalCenter
                    color: root.mainController && root.mainController.signalingState === "connected" ? 
                           Theme.success : Theme.textDisabled
                }
                
                Text {
                    text: root.mainController && root.mainController.signalingState === "connected" ? 
                          qsTr("Online") : qsTr("Offline")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    anchors.verticalCenter: parent.verticalCenter
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
            
            // If connection is established and no window exists, create one
            if (state === "已连接" && !remoteWindows[connectionId]) {
                root.showRemoteWindow(connectionId, connectionId)
            }
        }
    }
}
