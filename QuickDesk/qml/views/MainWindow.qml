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
    width: 900
    height: 600
    minimumWidth: 900
    maximumWidth: 900
    minimumHeight: 600
    maximumHeight: 600
    visible: true
    title: qsTr("QuickDesk")
    
    // 关闭窗口时退出程序
    onClosing: function(close) {
        Qt.quit()
    }

    // Example {}
    
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
        console.log("MainWindow.qml unload finish")
    }
    
    // Unified function to show or switch to a device connection
    // This handles both connecting to a new device and viewing existing connections
    function showOrSwitchToDevice(deviceId, connectionId) {
        console.log("showOrSwitchToDevice called - deviceId:", deviceId, "connectionId:", connectionId)
        
        // Step 1: Check if RemoteWindow exists and has a connection to this device
        if (remoteWindow) {
            for (var i = 0; i < remoteWindow.connections.length; i++) {
                if (remoteWindow.connections[i].deviceId === deviceId) {
                    console.log("Already connected to device:", deviceId, "- switching to existing tab:", i)
                    remoteWindow.currentTabIndex = i
                    remoteWindow.show()
                    remoteWindow.raise()
                    remoteWindow.requestActivate()
                    return true
                }
            }
        }
        
        // Step 2: If connectionId provided, show that specific connection
        if (connectionId) {
            return showRemoteWindow(connectionId, deviceId)
        }
        
        // Step 3: Device not connected and no connectionId provided
        console.warn("Device not connected and no connectionId provided:", deviceId)
        toast.show(qsTr("Device not connected: ") + deviceId, QDToast.Type.Error)
        return false
    }
    
    // Function to create or show remote window
    function showRemoteWindow(connectionId, deviceId) {
        console.log("showRemoteWindow called:", connectionId, deviceId)
        
        // Validate connection exists
        var connectionIds = mainController.clientManager.connectionIds
        var connectionExists = false
        for (var i = 0; i < connectionIds.length; i++) {
            if (connectionIds[i] === connectionId) {
                connectionExists = true
                break
            }
        }
        
        if (!connectionExists) {
            console.warn("Connection not found:", connectionId)
            toast.show(qsTr("Connection not found: ") + connectionId, QDToast.Type.Error)
            return false
        }
        
        // If RemoteWindow exists, check if connection is already there
        if (remoteWindow) {
            // Search for existing connection
            for (var j = 0; j < remoteWindow.connections.length; j++) {
                if (remoteWindow.connections[j].id === connectionId) {
                    console.log("Connection already in RemoteWindow, switching to tab:", j)
                    remoteWindow.currentTabIndex = j
                    remoteWindow.show()
                    remoteWindow.raise()
                    remoteWindow.requestActivate()
                    return true
                }
            }
        }
        
        // Create RemoteWindow if not exists
        if (!remoteWindow) {
            var component = Qt.createComponent("RemoteWindow.qml")
            
            if (component.status === Component.Error) {
                console.error("Error creating RemoteWindow:", component.errorString())
                toast.show(qsTr("Failed to create RemoteWindow"), QDToast.Type.Error)
                return false
            }
            
            if (component.status === Component.Ready) {
                remoteWindow = component.createObject(null, {
                    clientManager: mainController.clientManager
                })
                
                if (!remoteWindow) {
                    console.error("Failed to create RemoteWindow object")
                    toast.show(qsTr("Failed to create RemoteWindow"), QDToast.Type.Error)
                    return false
                }
                
                // Handle window destruction
                remoteWindow.closing.connect(function() {
                    console.log("RemoteWindow destroyed")
                    remoteWindow = null
                })
            } else {
                console.error("RemoteWindow component not ready:", component.status)
                toast.show(qsTr("RemoteWindow not ready"), QDToast.Type.Error)
                return false
            }
        }
        
        // Add connection to window
        if (remoteWindow) {
            remoteWindow.addConnection(connectionId, deviceId)
            remoteWindow.show()
            remoteWindow.raise()
            remoteWindow.requestActivate()
            console.log("Added connection to remote window:", connectionId)
            return true
        }
        
        return false
    }
    
    // Unified function to disconnect from remote host
    // This ensures consistent behavior across all disconnect actions:
    // - Clicking disconnect in connection list
    // - Clicking tab close in RemoteWindow
    // - Clicking disconnect in floating tool button
    function disconnectFromRemoteHost(connectionId) {
        console.log("MainWindow: Disconnect requested for:", connectionId)
        
        // If RemoteWindow exists, find the connection and close it properly
        if (remoteWindow) {
            // Find the connection index
            for (var i = 0; i < remoteWindow.connections.length; i++) {
                if (remoteWindow.connections[i].id === connectionId) {
                    console.log("Found connection at index:", i, "- calling RemoteWindow.closeConnection()")
                    remoteWindow.closeConnection(i)
                    return
                }
            }
            console.log("Connection not found in RemoteWindow, disconnecting directly")
        }        
    }
    
    // Main layout with navigation and status bar
    ColumnLayout {
        anchors.fill: parent
        spacing: 0
        
        // Navigation and content area
        QDNavigationView {
            id: navigationView
            Layout.fillWidth: true
            Layout.fillHeight: true
            
            isExpanded: true
            collapsedWidth: 48
            expandedWidth: 200
            showFooter: false  // Hide footer and separator
            
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
            
            footer: Item {
                // Empty footer - component will auto-hide separator and footer area
            }
            
            content: StackLayout {
                anchors.fill: parent
                currentIndex: navigationView.currentIndex
                
                // Remote Control Page
                RemoteControlPage {
                    id: remoteControlPage
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    mainController: root.mainController
                    
                    onShowToast: function(message, toastType) {
                        toast.show(message, toastType)
                    }
                    
                    onConnectRequested: function(deviceId, password) {
                        console.log("Connect requested:", deviceId)
                        
                        // Check if already connected to this device
                        if (root.showOrSwitchToDevice(deviceId, null)) {
                            // Already connected, reset connecting state and show toast
                            remoteControlPage.resetConnectingState()
                            toast.show(qsTr("Already connected, switched to existing window"), QDToast.Type.Info)
                            return
                        }
                        
                        // Not connected yet, create new connection
                        toast.show(qsTr("Connecting..."), QDToast.Type.Info)
                        var connId = root.mainController.connectToRemoteHost(deviceId, password)
                        if (connId) {
                            // Create remote window after a short delay to allow connection to establish
                            Qt.callLater(function() {
                                root.showRemoteWindow(connId, deviceId)
                            })
                        }
                    }
                    onViewConnectionRequested: function(connectionId) {
                        console.log("View connection requested:", connectionId)
                        
                        // Try to get deviceId from existing connection in RemoteWindow
                        var deviceId = null
                        if (root.remoteWindow) {
                            for (var i = 0; i < root.remoteWindow.connections.length; i++) {
                                if (root.remoteWindow.connections[i].id === connectionId) {
                                    deviceId = root.remoteWindow.connections[i].deviceId
                                    break
                                }
                            }
                        }
                        
                        // If not found, show error
                        if (!deviceId) {
                            console.error("Cannot find deviceId for connection:", connectionId)
                            toast.show(qsTr("Cannot find device for connection: ") + connectionId, QDToast.Type.Error)
                            return
                        }
                        
                        // Use unified function to show or switch to device
                        root.showOrSwitchToDevice(deviceId, connectionId)
                    }
                    onDisconnectRequested: function(connectionId) {
                        console.log("Disconnect requested from RemoteControlPage:", connectionId)
                        // Use unified disconnect function
                        root.disconnectFromRemoteHost(connectionId)
                    }
                }
                
                // Settings Page
                SettingsPage {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    mainController: root.mainController
                    
                    onShowToast: function(message, toastType) {
                        toast.show(message, toastType)
                    }
                }
            }
        }
        
        // Status Bar - spans entire window width including navigation pane
        QDStatusBar {
            id: statusBar
            Layout.fillWidth: true
            leftText: ""
            message: ""
            rightText: ""
            
            // Host and Client status indicators
            Row {
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
    
    // Toast for error messages
    QDToast {
        id: toast
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 50
        z: 9999
    }
}
