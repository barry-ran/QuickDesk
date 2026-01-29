import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"
import QuickDesk 1.0

Item {
    id: root
    
    // Controller reference passed from MainWindow
    property var mainController
    
    // Config ViewModel
    ConfigViewModel {
        id: configViewModel
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
                    expanded: true
                    
                    Column {
                        width: parent.width
                        spacing: Theme.spacingMedium
                        
                        // Password Refresh Interval
                        Row {
                            width: parent.width
                            spacing: Theme.spacingMedium
                            
                            Column {
                                width: parent.width - passwordRefreshCombo.width - parent.spacing
                                spacing: Theme.spacingXSmall
                                
                                Text {
                                    text: qsTr("Temporary Password Auto-Refresh")
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Text {
                                    text: qsTr("Automatically refresh temporary password at intervals")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                    width: parent.width
                                }
                            }
                            
                            QDComboBox {
                                id: passwordRefreshCombo
                                model: ListModel {
                                    id: passwordRefreshModel
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
                                        passwordRefreshModel.append(options[i])
                                    }

                                    // Set current value
                                    currentIndex = indexOfValue(configViewModel.passwordRefreshInterval)
                                }

                                onActivated: {
                                    // Save to config (timer will be updated automatically via signal)
                                    configViewModel.passwordRefreshInterval = selectedValue
                                }

                                function indexOfValue(value) {
                                    for (let i = 0; i < passwordRefreshModel.count; i++) {
                                        if (passwordRefreshModel.get(i).value === value) {
                                            return i
                                        }
                                    }
                                    return 3  // Default to 2 hours if not found
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
                    expanded: true
                    
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
                                
                                Text {
                                    id: restartTipText
                                    text: qsTr("(Effective after restart)")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    visible: false
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
                                    restartTipText.visible = true
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
                
                // About Section
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    
                    Column {
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingLarge
                        
                        Text {
                            text: qsTr("About")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }
                        
                        Rectangle { width: parent.width; height: 1; color: Theme.border }
                        
                        Row {
                            width: parent.width
                            spacing: Theme.spacingLarge
                            
                            QDAvatar {
                                width: 60
                                height: 60
                                name: "QuickDesk"
                                backgroundColor: Theme.primary
                            }
                            
                            Column {
                                width: parent.width - 60 - parent.spacing
                                spacing: Theme.spacingXSmall
                                
                                Text {
                                    text: "QuickDesk"
                                    font.pixelSize: Theme.fontSizeXLarge
                                    font.weight: Font.Bold
                                    color: Theme.text
                                }
                                
                                Text {
                                    text: qsTr("Version") + " 1.0.0"
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.textSecondary
                                }
                                
                                Text {
                                    text: qsTr("Remote Desktop Software")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                            }
                        }
                        
                        QDButton {
                            text: qsTr("Check for Updates")
                            buttonType: QDButton.Type.Secondary
                            iconText: FluentIconGlyph.downloadGlyph
                            onClicked: {
                                // TODO: Check for updates
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
