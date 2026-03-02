import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"
import QuickDesk 1.0

Item {
    id: root
    
    // Controller reference passed from MainWindow
    property var mainController
    
    // Signal for showing toast (unified toast in MainWindow)
    signal showToast(string message, int toastType)
    
    // Server list model for Repeater
    property var serverListModel: []
    
    // Config ViewModel
    ConfigViewModel {
        id: configViewModel
    }
    
    // Helper function for removing server (accessible from delegates)
    function removeServerAt(index, url) {
        if (mainController) {
            mainController.turnServerManager.removeServer(index)
            root.showToast(qsTr("Server removed. Restart to apply changes."), QDToast.Type.Info)
        }
    }
    
    // Update server list when servers change
    function updateServerList() {
        if (!mainController || !mainController.turnServerManager) {
            serverListModel = []
            return
        }
        
        let servers = mainController.turnServerManager.servers
        let newModel = []
        
        for (let i = 0; i < servers.length; i++) {
            let server = servers[i]
            let urls = server.urls || []
            let url = urls.length > 0 ? urls[0] : ""
            let isTurn = server.username ? true : false
            
            newModel.push({
                index: i,
                url: url,
                isTurn: isTurn
            })
        }
        
        serverListModel = newModel
    }
    
    // Listen to server changes
    Connections {
        target: mainController ? mainController.turnServerManager : null
        function onServersChanged() {
            updateServerList()
        }
    }
    
    // Watch mainController changes
    onMainControllerChanged: {
        if (mainController && mainController.turnServerManager) {
            Qt.callLater(updateServerList)
        }
    }
    
    Component.onCompleted: {
        Qt.callLater(updateServerList)
    }
    
    // Theme apply function
    function applyTheme(themeIndex) {
        if (themeIndex === 0) {
            // Light
            Theme.currentTheme = Theme.ThemeType.FluentLight
        } else if (themeIndex === 1) {
            // Dark
            Theme.currentTheme = Theme.ThemeType.FluentDark
        } else {
            // Auto - detect system theme (默认使用Dark)
            Theme.currentTheme = Theme.ThemeType.FluentDark
        }
    }
    
    Rectangle {
        anchors.fill: parent
        color: Theme.background
        
        Flickable {
            anchors.fill: parent
            contentWidth: width
            contentHeight: contentColumn.implicitHeight
            clip: true
            
            ScrollBar.vertical: QDScrollBar {}
            
            Column {
                id: contentColumn
                width: parent.width
                spacing: Theme.spacingLarge
                
                // Top padding
                Item { width: 1; height: Theme.spacingMedium }
                
                // Page Title
                Text {
                    x: Theme.spacingXLarge
                    text: qsTr("Settings")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }
                
                // Security Settings
                QDAccordion {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    title: qsTr("Security")
                    iconSource: FluentIconGlyph.lockGlyph
                    expanded: false
                    
                    Column {
                        width: parent.width
                        spacing: Theme.spacingMedium
                        
                        // Access Code Refresh Interval
                        Row {
                            width: parent.width
                            spacing: Theme.spacingMedium
                            
                            Column {
                                width: parent.width - accessCodeRefreshCombo.width - parent.spacing
                                spacing: Theme.spacingXSmall
                                
                                Text {
                                    text: qsTr("Access Code Auto-Refresh")
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Text {
                                    text: qsTr("Automatically refresh access code at intervals")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                    width: parent.width
                                }
                            }
                            
                            QDComboBox {
                                id: accessCodeRefreshCombo
                                model: ListModel {
                                    id: accessCodeRefreshModel
                                }
                                textRole: "text"
                                valueRole: "value"

                                Component.onCompleted: {
                                    // Build interval options based on debug mode
                                    let options = [
                                            {"text": qsTr("Never"), "value": -1}
                                        ]

                                    // Only add 1 minute option in debug builds
                                    if (APP_VERSION.includes("Debug") || APP_VERSION === "0.0.0.1") {
                                        options.push({"text": qsTr("1 Minute (Debug)"), "value": 1})
                                    }

                                    options.push(
                                                {"text": qsTr("30 Minutes"), "value": 30},
                                                {"text": qsTr("2 Hours"), "value": 120},
                                                {"text": qsTr("6 Hours"), "value": 360},
                                                {"text": qsTr("12 Hours"), "value": 720},
                                                {"text": qsTr("24 Hours"), "value": 1440}
                                                )

                                    for (let i = 0; i < options.length; i++) {
                                        accessCodeRefreshModel.append(options[i])
                                    }

                                    // Set current value
                                    currentIndex = indexOfValue(configViewModel.accessCodeRefreshInterval)
                                }

                                onActivated: {
                                    // Save to config (timer will be updated automatically via signal)
                                    configViewModel.accessCodeRefreshInterval = selectedValue
                                }

                                function indexOfValue(value) {
                                    for (let i = 0; i < accessCodeRefreshModel.count; i++) {
                                        if (accessCodeRefreshModel.get(i).value === value) {
                                            return i
                                        }
                                    }
                                    return 3  // Default to 2 hours if not found
                                }
                            }
                        }
                    }
                }
                
                // Network Settings (TURN/STUN servers)
                QDAccordion {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    title: qsTr("Network")
                    iconSource: FluentIconGlyph.networkGlyph
                    expanded: false
                    
                    Column {
                        width: parent.width
                        spacing: Theme.spacingMedium
                        
                        // Signaling Server Section
                        Column {
                            width: parent.width
                            spacing: Theme.spacingSmall
                            
                            Text {
                                text: qsTr("Signaling Server")
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            Text {
                                text: qsTr("Configure the signaling server address for remote connections.")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                wrapMode: Text.WordWrap
                                width: parent.width
                            }
                            
                            // Current server URL display
                            QDCard {
                                width: parent.width
                                height: 50
                                
                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: Theme.spacingMedium
                                    spacing: Theme.spacingMedium
                                    
                                    Column {
                                        Layout.fillWidth: true
                                        spacing: 2
                                        
                                        Text {
                                            text: qsTr("Current:")
                                            font.pixelSize: Theme.fontSizeSmall
                                            color: Theme.textSecondary
                                        }
                                        
                                        Text {
                                            text: mainController ? mainController.serverManager.serverUrl : ""
                                            font.pixelSize: Theme.fontSizeMedium
                                            color: Theme.text
                                            elide: Text.ElideMiddle
                                        }
                                    }
                                }
                            }
                            
                            // Update server URL
                            Row {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                QDTextField {
                                    id: signalingServerUrlField
                                    width: parent.width - updateServerBtn.width - parent.spacing
                                    placeholderText: qsTr("ws://your-server.com:8000")
                                }
                                
                                QDButton {
                                    id: updateServerBtn
                                    text: qsTr("Update")
                                    iconText: FluentIconGlyph.saveGlyph
                                    buttonType: QDButton.Type.Primary
                                    enabled: signalingServerUrlField.text.length > 0 && 
                                            signalingServerUrlField.text !== (mainController ? mainController.serverManager.serverUrl : "")
                                    
                                    onClicked: {
                                        if (!mainController) return
                                        
                                        let url = signalingServerUrlField.text.trim()
                                        
                                        // Basic validation
                                        if (!url.startsWith("ws://") && !url.startsWith("wss://")) {
                                            root.showToast(qsTr("Server URL must start with ws:// or wss://"), QDToast.Type.Error)
                                            return
                                        }
                                        
                                        mainController.serverManager.serverUrl = url
                                        root.showToast(qsTr("Server URL updated. Restart to apply changes."), QDToast.Type.Success)
                                    }
                                }
                            }
                        }
                        
                        // TURN/STUN Servers Section (hidden - ICE config is always fetched from signaling server)
                        Column {
                            visible: false
                            width: parent.width
                            spacing: Theme.spacingMedium

                            Rectangle { width: parent.width; height: 1; color: Theme.border }

                            Text {
                                text: qsTr("TURN/STUN Servers")
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }

                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall

                                Repeater {
                                    id: serverRepeater
                                    model: root.serverListModel

                                    delegate: QDCard {
                                        width: parent.width
                                        height: 60

                                        RowLayout {
                                            anchors.fill: parent
                                            anchors.margins: Theme.spacingMedium
                                            spacing: Theme.spacingMedium

                                            Column {
                                                Layout.fillWidth: true
                                                spacing: Theme.spacingXSmall

                                                Text {
                                                    text: modelData.url || ""
                                                    font.pixelSize: Theme.fontSizeMedium
                                                    font.weight: Font.DemiBold
                                                    color: Theme.text
                                                }

                                                Text {
                                                    text: modelData.isTurn ? qsTr("TURN Server") : qsTr("STUN Server")
                                                    font.pixelSize: Theme.fontSizeSmall
                                                    color: Theme.textSecondary
                                                }
                                            }

                                            QDIconButton {
                                                Layout.preferredWidth: 32
                                                Layout.preferredHeight: 32
                                                iconSource: FluentIconGlyph.deleteGlyph
                                                onClicked: {
                                                    root.removeServerAt(modelData.index, modelData.url)
                                                }
                                            }
                                        }
                                    }
                                }

                                Text {
                                    visible: serverRepeater.count === 0
                                    text: qsTr("No custom servers configured")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    horizontalAlignment: Text.AlignHCenter
                                    width: parent.width
                                }
                            }

                            Rectangle { width: parent.width; height: 1; color: Theme.border }

                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall

                                Text {
                                    text: qsTr("Add TURN Server")
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                QDTextField {
                                    id: turnUrlField
                                    width: parent.width
                                    placeholderText: qsTr("turn:your-server.com:3478")
                                }

                                Row {
                                    width: parent.width
                                    spacing: Theme.spacingSmall

                                    QDTextField {
                                        id: turnUsernameField
                                        width: (parent.width - parent.spacing) / 2
                                        placeholderText: qsTr("Username")
                                    }

                                    QDTextField {
                                        id: turnPasswordField
                                        width: (parent.width - parent.spacing) / 2
                                        placeholderText: qsTr("Password")
                                        echoMode: TextInput.Password
                                    }
                                }

                                QDButton {
                                    text: qsTr("Add TURN Server")
                                    iconText: FluentIconGlyph.addGlyph
                                    buttonType: QDButton.Type.Primary
                                    enabled: turnUrlField.text.length > 0 &&
                                            turnUsernameField.text.length > 0 &&
                                            turnPasswordField.text.length > 0

                                    onClicked: {
                                        if (!mainController) return
                                        if (mainController.turnServerManager.addTurnServer(
                                                turnUrlField.text,
                                                turnUsernameField.text,
                                                turnPasswordField.text)) {
                                            turnUrlField.text = ""
                                            turnUsernameField.text = ""
                                            turnPasswordField.text = ""
                                            toast.show(qsTr("TURN server added. Restart to apply changes."), QDToast.Type.Success)
                                        } else {
                                            root.showToast(qsTr("Invalid server URL format"), QDToast.Type.Error)
                                        }
                                    }
                                }
                            }

                            Rectangle { width: parent.width; height: 1; color: Theme.border }

                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall

                                Text {
                                    text: qsTr("Add STUN Server")
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                QDTextField {
                                    id: stunUrlField
                                    width: parent.width
                                    placeholderText: qsTr("stun:stun.l.google.com:19302")
                                }

                                QDButton {
                                    text: qsTr("Add STUN Server")
                                    iconText: FluentIconGlyph.addGlyph
                                    buttonType: QDButton.Type.Primary
                                    enabled: stunUrlField.text.length > 0

                                    onClicked: {
                                        if (!mainController) return
                                        if (mainController.turnServerManager.addStunServer(stunUrlField.text)) {
                                            stunUrlField.text = ""
                                            root.showToast(qsTr("STUN server added. Restart to apply changes."), QDToast.Type.Success)
                                        } else {
                                            root.showToast(qsTr("Invalid server URL format"), QDToast.Type.Error)
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
                
                // Application Settings
                QDAccordion {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    title: qsTr("Application")
                    iconSource: FluentIconGlyph.settingsGlyph
                    expanded: false
                    
                    Column {
                        width: parent.width
                        spacing: Theme.spacingMedium
                        
                        // Language
                        Row {
                            width: parent.width
                            spacing: Theme.spacingMedium
                            
                            Column {
                                width: parent.width - langCombo.width - parent.spacing
                                spacing: Theme.spacingXSmall
                                
                                Text {
                                    text: qsTr("Language")
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                            }
                            
                            QDComboBox {
                                id: langCombo
                                model: ListModel {
                                    id: languageModel
                                }
                                textRole: "text"
                                valueRole: "value"
                                
                                Component.onCompleted: {
                                    let languages = LanguageManage.getSupportLanguages()
                                    for (let i = 0; i < languages.length; ++i) {
                                        languageModel.append({
                                                                 "text": LanguageManage.getLanguageName(languages[i]),
                                                                 "value": languages[i]
                                                             })
                                    }
                                    currentIndex = indexOfValue(LanguageManage.getCurrentLanguage())
                                }
                                
                                onActivated: {
                                    LanguageManage.setCurrentLanguage(selectedValue)
                                    root.showToast(qsTr("Language updated. Restart to apply changes."), QDToast.Type.Info)
                                }
                                
                                function indexOfValue(value) {
                                    for (let i = 0; i < languageModel.count; i++) {
                                        if (languageModel.get(i).value === value) {
                                            return i
                                        }
                                    }
                                    return 0
                                }
                            }
                        }
                        
                        Rectangle { width: parent.width; height: 1; color: Theme.border }
                        
                        // Video Codec
                        Row {
                            width: parent.width
                            spacing: Theme.spacingMedium
                            
                            Column {
                                width: parent.width - codecCombo.width - parent.spacing
                                spacing: Theme.spacingXSmall
                                
                                Text {
                                    text: qsTr("Video Codec")
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Text {
                                    text: qsTr("Preferred video codec for remote connections")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                    width: parent.width
                                }
                            }
                            
                            QDComboBox {
                                id: codecCombo
                                model: ListModel {
                                    id: codecModel
                                    ListElement { text: "H264"; value: "H264" }
                                    ListElement { text: "VP8";  value: "VP8" }
                                    ListElement { text: "VP9";  value: "VP9" }
                                    ListElement { text: "AV1";  value: "AV1" }
                                }
                                textRole: "text"
                                valueRole: "value"
                                
                                Component.onCompleted: {
                                    currentIndex = indexOfValue(configViewModel.preferredVideoCodec)
                                }
                                
                                onActivated: {
                                    configViewModel.preferredVideoCodec = currentValue
                                    root.showToast(qsTr("Video codec updated. Effective on next connection."), QDToast.Type.Info)
                                }
                                
                                function indexOfValue(value) {
                                    for (let i = 0; i < codecModel.count; i++) {
                                        if (codecModel.get(i).value === value) {
                                            return i
                                        }
                                    }
                                    return 0
                                }
                            }
                        }
                        
                        Rectangle { width: parent.width; height: 1; color: Theme.border }
                        
                        // Theme
                        Row {
                            width: parent.width
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: qsTr("Theme")
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                width: parent.width - themeCombo.width - parent.spacing
                            }
                            
                            QDComboBox {
                                id: themeCombo
                                model: [qsTr("Light"), qsTr("Dark")]
                                currentIndex: configViewModel.darkTheme
                                
                                onActivated: {
                                    configViewModel.darkTheme = currentIndex
                                    applyTheme(currentIndex)
                                }
                                
                                Component.onCompleted: {
                                    applyTheme(configViewModel.darkTheme)
                                }
                            }
                        }
                    }
                }
                
                
                // Bottom padding
                Item { width: 1; height: Theme.spacingXLarge }
            }
        }
    }
}
