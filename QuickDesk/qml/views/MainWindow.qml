import QtQuick
import QtQuick.Window
import QtQuick.Controls
import QtQuick.Layouts
import QuickDesk 1.0

import "../"
import "../component"
import "../pages"
import "../quickdeskcomponent"

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
    color: Theme.background
    
    onClosing: function(close) {
        close.accepted = false
        root.hide()
        SystemTrayManager.minimizeToTray()
    }

    // Example {}
    
    // Remote window management - single window for all connections
    property var remoteWindow: null
    
    // Temporary storage for device credentials during connection
    property var pendingDeviceCredentials: ({})
    
    // Connections aborted due to window creation failure (suppress state change toasts)
    property var abortedConnections: ({})
    
    property string initErrorMessage: ""
    
    // Main controller - reuse existing controller
    property MainController mainController: MainController {
        id: mainControllerObj
        
        onInitializationFailed: function(error) {
            console.error("Initialization failed:", error)
            root.initErrorMessage = qsTr("Initialization failed: ") + error
        }
        
        onDeviceIdChanged: {
            console.log("Device ID changed:", mainControllerObj.deviceId)
        }
        
        onAccessCodeChanged: {
            console.log("Access Code changed:", mainControllerObj.accessCode)
        }
        
        onPresetLoadFailed: function(error) {
            presetFailedMessageBox.message = error
            presetFailedMessageBox.show()
        }
        
        onForceUpgradeRequired: function(minVersion) {
            forceUpgradeMessageBox.message = qsTr("Current version is too old. Minimum required version: %1. Please upgrade to continue using.").arg(minVersion)
            forceUpgradeMessageBox.show()
        }
    }
    
    // Listen to API requests to show remote window (e.g. from WebSocket API with showWindow=true)
    Connections {
        target: mainController

        function onRequestShowRemoteWindow(deviceId) {
            console.log("API requested showRemoteWindow:", deviceId)
            root.showRemoteWindow(deviceId)
        }
    }

    // §2.25 — surface native-messaging protocol mismatch to the user so
    // they know to upgrade. Fires once per helloResponse when the host's
    // protocol_version disagrees with Qt's (currently 2).
    Connections {
        target: mainController ? mainController.hostManager : null
        enabled: mainController != null && mainController.hostManager != null

        function onNativeMessagingProtocolMismatch(hostVersion, qtVersion) {
            console.warn("Native-messaging protocol mismatch: host=" + hostVersion
                         + " qt=" + qtVersion)
            toast.show(qsTr("Host version mismatch — please upgrade QuickDesk."),
                       QDToast.Type.Warning)
        }
    }

    // Listen to connection state changes to save device credentials
    Connections {
        target: mainController.clientManager
        
        function onConnectionStateChanged(deviceId, state, hostInfo) {
            console.log("MainWindow: Connection state changed:", deviceId, "->", state)
            
            // Handle aborted connections (window creation failed)
            if (root.abortedConnections[deviceId]) {
                if (state === "connected") {
                    // Initial disconnect was too early (race condition) — retry now
                    console.log("Retrying disconnect for aborted connection:", deviceId)
                    mainController.clientManager.disconnectFromHost(deviceId)
                } else if (state === "disconnected" || state === "failed") {
                    var cleaned = Object.assign({}, root.abortedConnections)
                    delete cleaned[deviceId]
                    root.abortedConnections = cleaned
                }
                return
            }

            if (state === "connected" && root.pendingDeviceCredentials[deviceId]) {
                var credentials = root.pendingDeviceCredentials[deviceId]
                console.log("Saving device to history:", credentials.deviceId)
                
                mainController.remoteDeviceManager.saveDevice(
                    credentials.deviceId,
                    credentials.password
                )
                
                // Clean up pending credentials
                delete root.pendingDeviceCredentials[deviceId]
            }
            
            // Clean up pending credentials on failure
            if (state === "failed" && root.pendingDeviceCredentials[deviceId]) {
                delete root.pendingDeviceCredentials[deviceId]
                remoteControlPage.resetConnectingState()
            }
        }

        function onErrorOccurred(deviceId, code, message) {
            if (deviceId && root.abortedConnections[deviceId]) {
                var cleaned = Object.assign({}, root.abortedConnections)
                delete cleaned[deviceId]
                root.abortedConnections = cleaned
            }

            if (deviceId && root.pendingDeviceCredentials[deviceId]) {
                delete root.pendingDeviceCredentials[deviceId]
                root.closeRemoteWindowTab(deviceId, false)
                remoteControlPage.resetConnectingState()
            }
        }
    }
    
    Connections {
        target: SystemTrayManager
        function onShowWindowRequested() {
            root.show()
            root.raise()
            root.requestActivate()
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
    function showOrSwitchToDevice(deviceId) {
        console.log("showOrSwitchToDevice called - deviceId:", deviceId)
        
        // Step 1: Check if RemoteWindow exists and has a connection to this device
        if (remoteWindow) {
            var idx = remoteWindow.connectionModel.indexOf(deviceId)
            if (idx >= 0) {
                console.log("Already connected to device:", deviceId, "- switching to existing tab:", idx)
                remoteWindow.currentTabIndex = idx
                remoteWindow.show()
                remoteWindow.raise()
                remoteWindow.requestActivate()
                return true
            }
        }
        
        // Step 2: Show remote window for this device if a connection already exists
        if (connectionExists(deviceId)) {
            return showRemoteWindow(deviceId)
        }

        return false
    }

    function connectionExists(deviceId) {
        var connectedDeviceIds = mainController.clientManager.connectedDeviceIds
        for (var i = 0; i < connectedDeviceIds.length; i++) {
            if (connectedDeviceIds[i] === deviceId) {
                return true
            }
        }
        return false
    }

    function closeRemoteWindowTab(deviceId, needDisconnect) {
        if (!remoteWindow) return

        var idx = remoteWindow.connectionModel.indexOf(deviceId)
        if (idx >= 0) {
            remoteWindow.closeConnection(idx, needDisconnect)
        }
    }
    
    // Function to create or show remote window
    function showRemoteWindow(deviceId, allowPendingConnection) {
        console.log("showRemoteWindow called:", deviceId)

        if (allowPendingConnection === undefined) allowPendingConnection = false

        if (!allowPendingConnection && !connectionExists(deviceId)) {
            console.log("Connection not found (may be connecting or failed quickly):", deviceId)
            return false
        }
        
        // If RemoteWindow exists, check if connection is already there
        if (remoteWindow) {
            var idx = remoteWindow.connectionModel.indexOf(deviceId)
            if (idx >= 0) {
                console.log("Connection already in RemoteWindow, switching to tab:", idx)
                remoteWindow.currentTabIndex = idx
                remoteWindow.show()
                remoteWindow.raise()
                remoteWindow.requestActivate()
                return true
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
                    clientManager: mainController.clientManager,
                    localDeviceId: mainController.deviceId
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
            remoteWindow.addConnection(deviceId)
            remoteWindow.show()
            remoteWindow.raise()
            remoteWindow.requestActivate()
            console.log("Added connection to remote window:", deviceId)
            return true
        }
        
        return false
    }
    
    // Unified function to disconnect from remote host
    // This ensures consistent behavior across all disconnect actions:
    // - Clicking disconnect in connection list
    // - Clicking tab close in RemoteWindow
    // - Clicking disconnect in floating tool button
    function disconnectFromRemoteHost(deviceId) {
        console.log("MainWindow: Disconnect requested for:", deviceId)
        
        // If RemoteWindow exists, find the connection and close it properly
        if (remoteWindow) {
            var idx = remoteWindow.connectionModel.indexOf(deviceId)
            if (idx >= 0) {
                console.log("Found connection at index:", idx, "- calling RemoteWindow.closeConnection()")
                remoteWindow.closeConnection(idx)
                return
            }
        }
        
        // Fallback: directly disconnect when RemoteWindow is null or connection not found
        console.log("Disconnecting directly via clientManager:", deviceId)
        if (mainController && mainController.clientManager) {
            mainController.clientManager.disconnectFromHost(deviceId)
        }
    }
    
    // Main layout with navigation and status bar
    ColumnLayout {
        anchors.fill: parent
        spacing: 0
        
        QDAnnouncementBar {
            id: announcementBar
            Layout.fillWidth: true
            text: mainController.presetManager ? mainController.presetManager.announcement : ""
            onLinkActivated: function(link) {
                Qt.openUrlExternally(link)
            }
        }
        
        // Navigation and content area
        QDNavigationView {
            id: navigationView
            Layout.fillWidth: true
            Layout.fillHeight: true
            
            isExpanded: true
            collapsedWidth: 48
            expandedWidth: 200
            
            menuItems: [
                { icon: FluentIconGlyph.remoteGlyph, text: qsTr("Remote Control") },
                { icon: FluentIconGlyph.contactInfoGlyph, text: qsTr("Device List") },
                { icon: FluentIconGlyph.settingsGlyph, text: qsTr("Settings") },
                { icon: FluentIconGlyph.infoGlyph, text: qsTr("About") }
            ]
            
            property var footerLinks: mainController.presetManager ? mainController.presetManager.links : []
            
            showFooter: footerLinks.length > 0
            
            header: Item {
                width: parent.width
                height: 72

                // User login area (replaces QuickDesk title)
                MouseArea {
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor

                    Rectangle {
                        anchors.fill: parent
                        anchors.margins: Theme.spacingXSmall
                        radius: Theme.radiusSmall
                        color: parent.containsMouse ? Theme.surfaceHover : "transparent"

                        Behavior on color {
                            ColorAnimation { duration: Theme.animationDurationFast }
                        }
                    }

                    onClicked: {
                        var loggedIn = root.mainController && root.mainController.authManager
                                       && root.mainController.authManager.isLoggedIn
                        if (loggedIn) {
                            userMenu.open()
                        } else {
                            loginDialog.open()
                        }
                    }

                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacingMedium
                        anchors.rightMargin: Theme.spacingMedium
                        spacing: Theme.spacingSmall

                        // User avatar / icon
                        Rectangle {
                            width: 36
                            height: 36
                            radius: 18
                            color: root.mainController && root.mainController.authManager
                                   && root.mainController.authManager.isLoggedIn
                                   ? Theme.primary : Theme.surfaceVariant

                            Text {
                                anchors.centerIn: parent
                                text: {
                                    if (root.mainController && root.mainController.authManager
                                        && root.mainController.authManager.isLoggedIn) {
                                        var name = root.mainController.authManager.username
                                        return name.length > 0 ? name.charAt(0).toUpperCase() : "U"
                                    }
                                    return FluentIconGlyph.peopleGlyph
                                }
                                font.family: root.mainController && root.mainController.authManager
                                             && root.mainController.authManager.isLoggedIn
                                             ? Theme.fontFamily : "Segoe Fluent Icons"
                                font.pixelSize: root.mainController && root.mainController.authManager
                                                && root.mainController.authManager.isLoggedIn
                                                ? 16 : 18
                                font.weight: Font.Bold
                                color: root.mainController && root.mainController.authManager
                                       && root.mainController.authManager.isLoggedIn
                                       ? "white" : Theme.textSecondary
                            }
                        }

                        // Username or "Login" text (only when expanded)
                        ColumnLayout {
                            visible: navigationView.isExpanded
                            Layout.fillWidth: true
                            spacing: 0

                            Text {
                                text: {
                                    if (root.mainController && root.mainController.authManager
                                        && root.mainController.authManager.isLoggedIn) {
                                        return root.mainController.authManager.username
                                    }
                                    return qsTr("Login")
                                }
                                font.pixelSize: Theme.fontSizeMedium
                                font.weight: Font.DemiBold
                                color: Theme.text
                                elide: Text.ElideRight
                                Layout.fillWidth: true
                            }

                            Text {
                                visible: root.mainController && root.mainController.authManager
                                         && root.mainController.authManager.isLoggedIn
                                text: "QuickDesk v" + APP_VERSION
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            Text {
                                visible: !(root.mainController && root.mainController.authManager
                                           && root.mainController.authManager.isLoggedIn)
                                text: qsTr("Click to login")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }
                        }
                    }
                }

                // User menu (when logged in)
                QDMenu {
                    id: userMenu
                    x: parent.width
                    y: parent.height

                    QDMenuItem {
                        text: qsTr("Account Settings")
                        iconText: FluentIconGlyph.contactGlyph
                        onTriggered: {
                            var webclientUrl = root.mainController.presetManager.webclientUrl
                            if (!webclientUrl) {
                                toast.show(qsTr("WebClient URL not configured on the server"), QDToast.Type.Warning)
                                return
                            }
                            var token = root.mainController.authManager.token
                            Qt.openUrlExternally(webclientUrl + "/#/account?token=" + token)
                        }
                    }

                    QDMenuItem {
                        text: qsTr("Logout")
                        iconText: FluentIconGlyph.signOutGlyph
                        isDestructive: true
                        onTriggered: root.mainController.authManager.logout(root.mainController.deviceId)
                    }
                }
            }
            
            footer: Column {
                width: parent.width
                padding: Theme.spacingXSmall
                spacing: 2
                
                Repeater {
                    model: navigationView.footerLinks
                    
                    delegate: Rectangle {
                        required property var modelData
                        
                        width: parent.width - parent.padding * 2
                        height: 36
                        radius: Theme.radiusSmall
                        color: linkMouseArea.containsMouse ? Theme.surfaceHover : "transparent"
                        
                        Behavior on color {
                            ColorAnimation { duration: Theme.animationDurationFast }
                        }
                        
                        RowLayout {
                            anchors.fill: parent
                            anchors.leftMargin: Theme.spacingMedium
                            anchors.rightMargin: Theme.spacingMedium
                            spacing: Theme.spacingMedium
                            
                            Text {
                                visible: modelData.icon !== undefined && modelData.icon !== ""
                                text: modelData.icon ? String.fromCharCode(parseInt(modelData.icon, 16)) : ""
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: 14
                                color: Theme.primary
                            }
                            
                            Text {
                                visible: navigationView.isExpanded
                                text: modelData.text || ""
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                Layout.fillWidth: true
                                elide: Text.ElideRight
                            }
                        }
                        
                        MouseArea {
                            id: linkMouseArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: {
                                if (modelData.url) {
                                    Qt.openUrlExternally(modelData.url)
                                }
                            }
                        }
                        
                        QDToolTip {
                            visible: linkMouseArea.containsMouse && modelData.url
                            text: modelData.url || ""
                        }
                    }
                }
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
                    abortedConnections: root.abortedConnections
                    
                    onShowToast: function(message, toastType) {
                        toast.show(message, toastType)
                    }
                    
                    onConnectRequested: function(deviceId, password) {
                        console.log("Connect requested:", deviceId)
                        
                        // Check if already connected to this device
                        if (root.showOrSwitchToDevice(deviceId)) {
                            // Already connected, reset connecting state and show toast
                            remoteControlPage.resetConnectingState()
                            toast.show(qsTr("Already connected, switched to existing window"), QDToast.Type.Info)
                            return
                        }
                        
                        // Not connected yet, create new connection
                        toast.show(qsTr("Connecting..."), QDToast.Type.Info)
                        var connectedDeviceId = root.mainController.connectToRemoteHost(deviceId, password)
                        if (connectedDeviceId) {
                            if (!root.showRemoteWindow(connectedDeviceId, true)) {
                                console.error("Remote window creation failed, aborting:", connectedDeviceId)
                                var newAborted = Object.assign({}, root.abortedConnections)
                                newAborted[connectedDeviceId] = true
                                root.abortedConnections = newAborted
                                root.mainController.disconnectFromRemoteHost(connectedDeviceId)
                                remoteControlPage.resetConnectingState()
                                return
                            }

                            root.pendingDeviceCredentials[connectedDeviceId] = {
                                deviceId: deviceId,
                                password: password
                            }
                        }
                    }
                    onViewConnectionRequested: function(deviceId) {
                        console.log("View connection requested:", deviceId)
                        root.showOrSwitchToDevice(deviceId)
                    }
                    onDisconnectRequested: function(deviceId) {
                        console.log("Disconnect requested from RemoteControlPage:", deviceId)
                        // Use unified disconnect function
                        root.disconnectFromRemoteHost(deviceId)
                    }
                }

                // Device List Page
                DeviceListPage {
                    id: deviceListPage
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    mainController: root.mainController

                    onConnectToDevice: function(deviceId, accessCode) {
                        // Switch to Remote Control tab
                        navigationView.currentIndex = 0
                        // Initiate connection and show the remote window immediately.
                        toast.show(qsTr("Connecting..."), QDToast.Type.Info)
                        var connectedDeviceId = root.mainController.connectToRemoteHost(deviceId, accessCode)
                        if (connectedDeviceId) {
                            if (!root.showRemoteWindow(connectedDeviceId, true)) {
                                console.error("Remote window creation failed, aborting:", connectedDeviceId)
                                var newAborted = Object.assign({}, root.abortedConnections)
                                newAborted[connectedDeviceId] = true
                                root.abortedConnections = newAborted
                                root.mainController.disconnectFromRemoteHost(connectedDeviceId)
                                remoteControlPage.resetConnectingState()
                                return
                            }

                            root.pendingDeviceCredentials[connectedDeviceId] = {
                                deviceId: deviceId,
                                password: accessCode
                            }
                        }
                    }

                    onShowToast: function(message, toastType) {
                        toast.show(message, toastType)
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
                
                // About Page
                AboutPage {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                }
            }
        }
        
        // Status Bar - spans entire window width including navigation pane
        QDStatusBar {
            id: statusBar
            Layout.fillWidth: true
            
            // Host Status
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
                        
                        if (processStatus === ProcessStatus.Running) {
                            if (serverStatus === ServerStatus.Connected) return Theme.success
                            if (serverStatus === ServerStatus.Connecting || serverStatus === ServerStatus.Reconnecting) return Theme.warning
                            if (serverStatus === ServerStatus.Failed) return Theme.error
                            return Theme.textDisabled
                        }
                        
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
                        
                        var statusText = ""
                        if (processStatus === ProcessStatus.Running) {
                            if (serverStatus === ServerStatus.Disconnected) statusText = qsTr("Disconnected")
                            else if (serverStatus === ServerStatus.Connecting) statusText = qsTr("Connecting")
                            else if (serverStatus === ServerStatus.Connected) statusText = qsTr("Connected")
                            else if (serverStatus === ServerStatus.Failed) statusText = qsTr("Connection Failed")
                            else if (serverStatus === ServerStatus.Reconnecting) statusText = qsTr("Reconnecting")
                            else statusText = qsTr("Unknown")
                        } else if (processStatus === ProcessStatus.NotStarted) {
                            statusText = qsTr("Not Started")
                        } else if (processStatus === ProcessStatus.Starting) {
                            statusText = qsTr("Starting")
                        } else if (processStatus === ProcessStatus.Failed) {
                            statusText = qsTr("Start Failed")
                        } else if (processStatus === ProcessStatus.Restarting) {
                            statusText = qsTr("Restarting")
                        } else {
                            statusText = qsTr("Unknown")
                        }

                        var modeTag = ""
                        var launchMode = root.mainController.hostLaunchMode
                        if (launchMode === HostLaunchMode.Service)
                            modeTag = " [Service]"
                        else if (launchMode === HostLaunchMode.ChildProcess)
                            modeTag = " [Process]"

                        return qsTr("Host") + ": " + statusText + modeTag
                    }
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    anchors.verticalCenter: parent.verticalCenter
                }
            }
            
            // Client Status
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
                        
                        if (processStatus === ProcessStatus.Running) {
                            if (serverStatus === ServerStatus.Connected) return Theme.success
                            if (serverStatus === ServerStatus.Connecting || serverStatus === ServerStatus.Reconnecting) return Theme.warning
                            if (serverStatus === ServerStatus.Failed) return Theme.error
                            return Theme.textDisabled
                        }
                        
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
                        
                        if (processStatus === ProcessStatus.Running) {
                            if (serverStatus === ServerStatus.Disconnected) return qsTr("Client") + ": " + qsTr("Disconnected")
                            if (serverStatus === ServerStatus.Connecting) return qsTr("Client") + ": " + qsTr("Connecting")
                            if (serverStatus === ServerStatus.Connected) return qsTr("Client") + ": " + qsTr("Connected")
                            if (serverStatus === ServerStatus.Failed) return qsTr("Client") + ": " + qsTr("Connection Failed")
                            if (serverStatus === ServerStatus.Reconnecting) return qsTr("Client") + ": " + qsTr("Reconnecting")
                            return qsTr("Client") + ": " + qsTr("Unknown")
                        }
                        
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
            
            QDSeparator {}
            
            // MCP (AI) Service Status - clickable indicator
            Rectangle {
                id: mcpIndicator
                width: mcpRow.width + Theme.spacingSmall * 2
                height: parent.height
                color: mcpMouseArea.containsMouse ? Theme.surfaceHover : "transparent"
                radius: Theme.radiusSmall
                
                Behavior on color {
                    ColorAnimation { duration: Theme.animationDurationFast }
                }
                
                Row {
                    id: mcpRow
                    anchors.centerIn: parent
                    spacing: Theme.spacingXSmall
                    
                    Text {
                        text: FluentIconGlyph.robotGlyph
                        font.family: "Segoe Fluent Icons"
                        font.pixelSize: 12
                        color: root.mainController && root.mainController.mcpServiceRunning
                               ? Theme.primary : Theme.textDisabled
                        anchors.verticalCenter: parent.verticalCenter
                        
                        SequentialAnimation on opacity {
                            running: root.mainController && root.mainController.mcpServiceRunning
                                     && root.mainController.mcpConnectedClients > 0
                            loops: Animation.Infinite
                            NumberAnimation { to: 0.4; duration: 800; easing.type: Easing.InOutSine }
                            NumberAnimation { to: 1.0; duration: 800; easing.type: Easing.InOutSine }
                        }
                    }
                    
                    Text {
                        text: {
                            if (!root.mainController) return "MCP"
                            var mode = root.mainController.mcpTransportMode === "http"
                                       ? "HTTP" : "stdio"
                            if (!root.mainController.mcpServiceRunning) return qsTr("MCP [%1]: Off").arg(mode)
                            var clients = root.mainController.mcpConnectedClients
                            if (clients > 0) return qsTr("MCP [%1]: %2 client(s)").arg(mode).arg(clients)
                            return qsTr("MCP [%1]: Ready").arg(mode)
                        }
                        font.pixelSize: Theme.fontSizeSmall
                        color: root.mainController && root.mainController.mcpServiceRunning
                               ? Theme.primary : Theme.textSecondary
                        font.weight: root.mainController && root.mainController.mcpServiceRunning
                                     ? Font.DemiBold : Font.Normal
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }
                
                MouseArea {
                    id: mcpMouseArea
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: mcpConfigPopup.open()
                }
                
                QDToolTip {
                    visible: mcpMouseArea.containsMouse
                    text: qsTr("Click to configure MCP integration")
                }
            }
            
            // Spacer
            Item {
                Layout.fillWidth: true
            }
            
            // Initialization error message (right side)
            Text {
                visible: root.initErrorMessage !== ""
                text: root.initErrorMessage
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.error
                elide: Text.ElideRight
            }
        }
    }
    
    // Monitor client connection state changes
    Connections {
        target: root.mainController ? root.mainController.clientManager : null
        
        function onConnectionStateChanged(deviceId, state, hostInfo) {
            // Skip aborted connections
            if (root.abortedConnections[deviceId]) return
            
            // If connection is established, ensure the window is shown
            if (state === "connected") {
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
    
    QDMessageBox {
        id: presetFailedMessageBox
        messageType: QDMessageBox.Type.Error
        title: qsTr("Server Connection Error")
        buttons: QDMessageBox.Buttons.Ok
        onClosed: Qt.quit()
    }
    
    QDMessageBox {
        id: forceUpgradeMessageBox
        messageType: QDMessageBox.Type.Warning
        title: qsTr("Upgrade Required")
        buttons: QDMessageBox.Buttons.Ok
        onClosed: Qt.quit()
    }
    
    McpConfigPopup {
        id: mcpConfigPopup
        mainController: root.mainController
    }

    LoginDialog {
        id: loginDialog
        mainController: root.mainController
    }
}
