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
    
    Rectangle {
        anchors.fill: parent
        color: Theme.background
        
        // 居中容器，限制最大宽度
        Item {
            anchors.centerIn: parent
            width: Math.min(parent.width - Theme.spacingXLarge * 2, 300)
            height: parent.height
            
            Column {
                anchors.top: parent.top
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.topMargin: Theme.spacingXLarge
                width: parent.width
                spacing: Theme.spacingLarge
                
                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: qsTr("Host Information")
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.DemiBold
                    color: Theme.text
                }
                
                QDCard {
                    width: parent.width
                    implicitHeight: cardColumn.implicitHeight + Theme.spacingLarge * 2
                    
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
                            spacing: Theme.spacingSmall
                            
                            Text {
                                width: parent.width - copyIdBtn.width - parent.spacing
                                text: mainController.deviceId || qsTr("Loading...")
                                font.pixelSize: 20
                                font.weight: Font.Bold
                                font.family: "Consolas"
                                color: mainController.deviceId ? Theme.primary : Theme.textDisabled
                                elide: Text.ElideMiddle
                            }
                            
                            QDIconButton {
                                id: copyIdBtn
                                iconSource: FluentIconGlyph.copyGlyph
                                enabled: mainController.deviceId && mainController.deviceId.length > 0
                                onClicked: {
                                    mainController.copyToClipboard(mainController.deviceId)
                                    toast.show(qsTr("Device ID copied"), QDToast.Type.Success)
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
                        
                        // Temporary Password
                        Text {
                            width: parent.width
                            text: qsTr("Temporary Password")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }
                        
                        Row {
                            width: parent.width
                            spacing: Theme.spacingSmall
                            
                            Text {
                                width: parent.width - refreshBtn.width - copyPwdBtn.width - parent.spacing * 2
                                text: mainController.accessCode || qsTr("Loading...")
                                font.pixelSize: 18
                                font.weight: Font.Bold
                                font.family: "Consolas"
                                color: mainController.accessCode ? Theme.success : Theme.textDisabled
                            }
                            
                            QDIconButton {
                                id: refreshBtn
                                iconSource: FluentIconGlyph.refreshGlyph
                                enabled: mainController.signalingState === "connected"
                                onClicked: {
                                    console.log("Refresh access code clicked")
                                    mainController.refreshTempPassword()
                                }
                                
                                QDToolTip {
                                    visible: parent.hovered
                                    text: mainController.signalingState === "connected" ? 
                                          qsTr("Refresh Password") : qsTr("Connect to server first")
                                }
                            }
                            
                            QDIconButton {
                                id: copyPwdBtn
                                iconSource: FluentIconGlyph.copyGlyph
                                enabled: mainController.accessCode && mainController.accessCode.length > 0
                                onClicked: {
                                    mainController.copyToClipboard(mainController.accessCode)
                                    toast.show(qsTr("Password copied"), QDToast.Type.Success)
                                }
                                
                                QDToolTip {
                                    visible: parent.hovered
                                    text: qsTr("Copy Password")
                                }
                            }
                        }
                    }
                }
                
                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: qsTr("Connect to Remote")
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.DemiBold
                    color: Theme.text
                }
                
                QDCard {
                    width: parent.width
                    implicitHeight: connectCardColumn.implicitHeight + Theme.spacingLarge * 2
                    
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
                            width: parent.width
                            text: qsTr("Connect")
                            buttonType: QDButton.Type.Primary
                            enabled: remoteDeviceIdInput.text.length === 9 && 
                                     remotePasswordInput.text.length > 0 &&
                                     !connectingState
                            
                            property bool connectingState: false
                            
                            onClicked: {
                                console.log("Connect clicked, deviceId:", remoteDeviceIdInput.text)
                                
                                // Validate device ID (9 digits)
                                if (remoteDeviceIdInput.text.length !== 9) {
                                    toast.show(qsTr("Device ID must be 9 digits"), QDToast.Type.Error)
                                    return
                                }
                                
                                // Validate password
                                if (remotePasswordInput.text.length === 0) {
                                    toast.show(qsTr("Please enter access password"), QDToast.Type.Error)
                                    return
                                }
                                
                                // Start connecting
                                connectingState = true
                                
                                var connectionId = mainController.connectToRemoteHost(
                                    remoteDeviceIdInput.text,
                                    remotePasswordInput.text,
                                    "" // Use default server URL
                                )
                                
                                console.log("Connection initiated, connectionId:", connectionId)
                                toast.show(qsTr("Connecting..."), QDToast.Type.Info)
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
                toast.show(qsTr("Connected successfully"), QDToast.Type.Success)
                openRemoteWindow(connectionId, hostInfo)
            } else if (state === "failed") {
                var errorMsg = hostInfo.error || qsTr("Connection failed")
                toast.show(qsTr("Connection failed: ") + errorMsg, QDToast.Type.Error)
            } else if (state === "disconnected") {
                toast.show(qsTr("Disconnected"), QDToast.Type.Info)
            }
        }
        
        function onErrorOccurred(connectionId, code, message) {
            console.log("Connection error:", connectionId, code, message)
            toast.show(qsTr("Error: ") + message, QDToast.Type.Error)
            
            // Reset connecting state
            var connectBtn = remoteDeviceIdInput.parent.parent.children[3]
            if (connectBtn) {
                connectBtn.connectingState = false
            }
        }
    }
    
    // Function to open remote desktop window
    function openRemoteWindow(connectionId, hostInfo) {
        console.log("Opening remote window for connection:", connectionId)
        
        // Create RemoteWindow component dynamically
        var component = Qt.createComponent("../views/RemoteWindow.qml")
        
        if (component.status === Component.Ready) {
            var window = component.createObject(root, {
                clientManager: mainController.clientManager
            })
            
            if (window) {
                // Add the connection to the window
                window.addConnection(connectionId, hostInfo.deviceId || remoteDeviceIdInput.text)
                window.show()
                console.log("Remote window created and shown")
                
                // Clear input fields after successful connection
                remoteDeviceIdInput.text = ""
                remotePasswordInput.text = ""
            } else {
                console.error("Failed to create remote window object")
                toast.show(qsTr("Failed to create remote window"), QDToast.Type.Error)
            }
        } else if (component.status === Component.Error) {
            console.error("Error loading RemoteWindow.qml:", component.errorString())
            toast.show(qsTr("Failed to load remote window"), QDToast.Type.Error)
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
    
    // Toast notification
    QDToast {
        id: toast
    }
    
    // Connect to refresh password result
    Connections {
        target: mainController.hostManager
        function onRefreshTempPasswordResult(success, errorCode, errorMessage) {
            if (success) {
                toast.show(qsTr("Password refreshed successfully"), QDToast.Type.Success)
            } else {
                toast.show(qsTr("Refresh failed: ") + errorMessage, QDToast.Type.Error)
            }
        }
    }
}
