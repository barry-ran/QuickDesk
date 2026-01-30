import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"

Item {
    id: root
    
    // Controller reference passed from MainWindow
    property var mainController
    
    // Signal for connection request
    signal connectRequested(string deviceId, string password)
    
    // Signal for viewing existing connection
    signal viewConnectionRequested(string connectionId)
    
    // Signal for disconnecting from remote host (unified)
    signal disconnectRequested(string connectionId)
    
    // Signal for showing toast (unified toast in MainWindow)
    signal showToast(string message, int toastType)
    
    // Function to reset connecting state (called when duplicate connection detected)
    function resetConnectingState() {
        if (connectBtn) {
            connectBtn.resetConnectingState()
        }
    }
    
    Rectangle {
        anchors.fill: parent
        color: Theme.background
        
        // 2x2 Grid Layout
        GridLayout {
            anchors.fill: parent
            anchors.margins: Theme.spacingLarge
            columns: 2
            rows: 2
            columnSpacing: Theme.spacingLarge
            rowSpacing: Theme.spacingLarge
            
            // ========== Row 1: Left (Host Info) and Right (Connected Clients) ==========
            
            // Host Information Card (Row 1, Col 1)
            ColumnLayout {
                Layout.row: 0
                Layout.column: 0
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.preferredWidth: parent.width / 2
                Layout.minimumHeight: 200
                spacing: Theme.spacingSmall
                
                Text {
                    text: qsTr("Host Information")
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.DemiBold
                    color: Theme.text
                }
                
                QDCard {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    
                    Column {
                        id: cardColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium
                        
                        // Device ID
                        Text {
                            width: parent.width
                            text: qsTr("Device ID")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }
                        
                        Row {
                            width: parent.width
                            height: copyIdBtn.height
                            spacing: Theme.spacingSmall
                            
                            Text {
                                width: parent.width - copyIdBtn.width - parent.spacing
                                text: mainController.deviceId || qsTr("Loading...")
                                font.pixelSize: 26
                                font.weight: Font.Bold
                                font.family: "Consolas"
                                color: mainController.deviceId ? Theme.primary : Theme.textDisabled
                                elide: Text.ElideMiddle
                                verticalAlignment: Text.AlignVCenter
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            
                            QDIconButton {
                                id: copyIdBtn
                                iconSource: FluentIconGlyph.copyGlyph
                                enabled: mainController.deviceId && mainController.deviceId.length > 0
                                anchors.verticalCenter: parent.verticalCenter
                                onClicked: {
                                    mainController.copyToClipboard(mainController.deviceId)
                                    root.showToast(qsTr("Device ID copied"), QDToast.Type.Success)
                                }
                                
                                QDToolTip {
                                    visible: parent.hovered
                                    text: qsTr("Copy Device ID")
                                }
                            }
                        }
                        
                        Rectangle {
                            width: parent.width
                            height: 1
                            color: Theme.border
                        }
                        
                        // Temporary Password Header with Auto-refresh Info
                        Row {
                            width: parent.width
                            spacing: Theme.spacingXXLarge
                            
                            Text {
                                text: qsTr("Access Code")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            
                            Text {
                                text: qsTr("Auto-refresh: ") + (mainController.nextAccessCodeRefreshTime || qsTr("Never"))
                                font.pixelSize: Theme.fontSizeSmall - 1
                                color: Theme.textSecondary
                                anchors.verticalCenter: parent.verticalCenter
                                visible: mainController.accessCode
                            }
                        }
                        
                        Row {
                            width: parent.width
                            height: refreshBtn.height
                            spacing: Theme.spacingSmall
                            
                            Text {
                                width: parent.width - refreshBtn.width - copyPwdBtn.width - parent.spacing * 2
                                text: mainController.accessCode || qsTr("Loading...")
                                font.pixelSize: 24
                                font.weight: Font.Bold
                                font.family: "Consolas"
                                color: mainController.accessCode ? Theme.success : Theme.textDisabled
                                verticalAlignment: Text.AlignVCenter
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            
                            QDIconButton {
                                id: refreshBtn
                                iconSource: FluentIconGlyph.refreshGlyph
                                enabled: mainController.signalingState === "connected"
                                anchors.verticalCenter: parent.verticalCenter
                                onClicked: {
                                    console.log("Refresh access code clicked")
                                    mainController.refreshAccessCode()
                                }
                                
                                QDToolTip {
                                    visible: parent.hovered
                                    text: mainController.signalingState === "connected" ? 
                                          qsTr("Refresh Access Code") : qsTr("Connect to server first")
                                }
                            }
                            
                            QDIconButton {
                                id: copyPwdBtn
                                iconSource: FluentIconGlyph.copyGlyph
                                enabled: mainController.accessCode && mainController.accessCode.length > 0
                                anchors.verticalCenter: parent.verticalCenter
                                onClicked: {
                                    mainController.copyToClipboard(mainController.accessCode)
                                    root.showToast(qsTr("Access Code copied"), QDToast.Type.Success)
                                }
                                
                                QDToolTip {
                                    visible: parent.hovered
                                    text: qsTr("Copy Access Code")
                                }
                            }
                        }
                    }
                }
            }
            
            // Connected Clients List (Row 1, Col 2)
            ConnectionListPanel {
                Layout.row: 0
                Layout.column: 1
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.preferredWidth: parent.width / 2
                Layout.minimumWidth: 200
                Layout.minimumHeight: 200
                mainController: root.mainController
                panelType: "clients"
                
                onViewConnectionRequested: function(connectionId) {
                    console.log("View connection requested:", connectionId)
                    root.viewConnectionRequested(connectionId)
                }
            }
            
            // ========== Row 2: Left (Connect to Remote) and Right (My Remote Connections) ==========
            
            // Connect to Remote Card (Row 2, Col 1)
            ColumnLayout {
                Layout.row: 1
                Layout.column: 0
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.preferredWidth: parent.width / 2
                Layout.minimumHeight: 200
                spacing: Theme.spacingSmall
                
                Text {
                    text: qsTr("Connect to Remote")
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.DemiBold
                    color: Theme.text
                }
                
                QDCard {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    
                    Column {
                        id: connectCardColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium
                        
                        // Device ID input
                        Column {
                            width: parent.width
                            spacing: 4
                            
                            Text {
                                text: qsTr("Remote Device ID")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }
                            
                            QDTextField {
                                id: remoteDeviceIdInput
                                width: parent.width
                                placeholderText: qsTr("Enter 9-digit device ID")
                                validator: RegularExpressionValidator { 
                                    regularExpression: /^\d{0,9}$/
                                }
                                
                                // Error state
                                property bool hasError: text.length > 0 && text.length !== 9
                                
                                Rectangle {
                                    anchors.bottom: parent.bottom
                                    width: parent.width
                                    height: 1
                                    color: parent.hasError ? Theme.error : "transparent"
                                }
                            }
                            
                            Text {
                                visible: remoteDeviceIdInput.hasError
                                text: qsTr("Device ID must be 9 digits")
                                font.pixelSize: Theme.fontSizeSmall - 1
                                color: Theme.error
                            }
                        }
                        
                        // Password input
                        Column {
                            width: parent.width
                            spacing: 4
                            
                            Text {
                                text: qsTr("Access Password")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }
                            
                            QDTextField {
                                id: remotePasswordInput
                                width: parent.width
                                placeholderText: qsTr("Enter access password")
                                echoMode: showPasswordBtn.checked ? TextInput.Normal : TextInput.Password
                                
                                Row {
                                    anchors.right: parent.right
                                    anchors.rightMargin: Theme.spacingSmall
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: 4
                                    
                                    QDIconButton {
                                        id: showPasswordBtn
                                        iconSource: checked ? FluentIconGlyph.passwordKeyHideGlyph : FluentIconGlyph.passwordKeyShowGlyph
                                        buttonSize: QDIconButton.Size.Small
                                        buttonStyle: QDIconButton.Style.Transparent
                                        checkable: true
                                        
                                        QDToolTip {
                                            visible: parent.hovered
                                            text: showPasswordBtn.checked ? qsTr("Hide password") : qsTr("Show password")
                                        }
                                    }
                                }
                            }
                        }
                        
                        // Connect button
                        QDButton {
                            id: connectBtn
                            width: parent.width
                            text: qsTr("Connect")
                            buttonType: QDButton.Type.Primary
                            enabled: remoteDeviceIdInput.text.length === 9 && 
                                     remotePasswordInput.text.length > 0 &&
                                     !connectingState
                            
                            property bool connectingState: false
                            
                            // Function to reset connecting state
                            function resetConnectingState() {
                                connectingState = false
                            }
                            
                            onClicked: {
                                console.log("Connect clicked, deviceId:", remoteDeviceIdInput.text)
                                
                                // Validate device ID (9 digits)
                                if (remoteDeviceIdInput.text.length !== 9) {
                                    root.showToast(qsTr("Device ID must be 9 digits"), QDToast.Type.Error)
                                    return
                                }
                                
                                // Validate password
                                if (remotePasswordInput.text.length === 0) {
                                    root.showToast(qsTr("Please enter access password"), QDToast.Type.Error)
                                    return
                                }
                                
                                // Start connecting
                                connectingState = true
                                
                                // Emit signal to MainWindow which will create the remote window
                                console.log("Connection requested, waiting for window creation")
                                root.connectRequested(remoteDeviceIdInput.text, remotePasswordInput.text)
                            }
                            
                            // Reset connecting state after timeout
                            Timer {
                                id: connectTimeout
                                interval: 10000  // 10 seconds timeout
                                running: parent.connectingState
                                onTriggered: {
                                    parent.connectingState = false
                                }
                            }
                        }
                    }
                }
            }
            
            // My Remote Connections List (Row 2, Col 2)
            ConnectionListPanel {
                Layout.row: 1
                Layout.column: 1
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.preferredWidth: parent.width / 2
                Layout.minimumWidth: 200
                Layout.minimumHeight: 200
                mainController: root.mainController
                panelType: "connections"
                
                onViewConnectionRequested: function(connectionId) {
                    console.log("View connection requested:", connectionId)
                    // Emit signal to parent (MainWindow) to show remote window
                    // MainWindow will handle validation and error messages
                    root.viewConnectionRequested(connectionId)
                }
                
                onDisconnectRequested: function(connectionId) {
                    console.log("Disconnect requested:", connectionId)
                    // Forward to MainWindow for unified handling
                    root.disconnectRequested(connectionId)
                }
            }
        }
    }
    
    // Listen to connection state changes
    Connections {
        target: mainController.clientManager
        
        function onConnectionStateChanged(connectionId, state, hostInfo) {
            console.log("Connection state changed:", connectionId, state)
            
            // Reset connecting state
            var connectBtn = remoteDeviceIdInput.parent.parent.children[3]
            if (connectBtn) {
                connectBtn.connectingState = false
            }
            
            if (state === "connected") {
                root.showToast(qsTr("Connected successfully"), QDToast.Type.Success)
                // Window is already created by MainWindow.showRemoteWindow()
                
                // Clear input fields after successful connection
                remoteDeviceIdInput.text = ""
                remotePasswordInput.text = ""
            } else if (state === "failed") {
                var errorMsg = hostInfo.error || qsTr("Connection failed")
                root.showToast(qsTr("Connection failed: ") + errorMsg, QDToast.Type.Error)
            } else if (state === "disconnected") {
                root.showToast(qsTr("Disconnected"), QDToast.Type.Info)
            }
        }
        
        function onErrorOccurred(connectionId, code, message) {
            console.log("Connection error:", connectionId, code, message)
            root.showToast(qsTr("Error: ") + message, QDToast.Type.Error)
            
            // Reset connecting state
            var connectBtn = remoteDeviceIdInput.parent.parent.children[3]
            if (connectBtn) {
                connectBtn.connectingState = false
            }
        }
    }
    
    Component.onCompleted: {
        console.log("RemoteControlPage loaded, size:", width, "x", height)
        console.log("mainController:", mainController)
        if (mainController) {
            console.log("  deviceId:", mainController.deviceId)
            console.log("  accessCode:", mainController.accessCode)
        }
    }
    
    // Connect to refresh access code result
    Connections {
        target: mainController.hostManager
        function onRefreshAccessCodeResult(success, errorCode, errorMessage) {
            if (success) {
                root.showToast(qsTr("Access code refreshed successfully"), QDToast.Type.Success)
            } else {
                root.showToast(qsTr("Refresh failed: ") + errorMessage, QDToast.Type.Error)
            }
        }
    }
}
