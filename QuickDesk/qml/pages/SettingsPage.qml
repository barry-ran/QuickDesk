import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"

Item {
    id: root
    
    // Controller reference passed from MainWindow
    property var mainController
    
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
                        
                        // Auto Accept Connections
                        Row {
                            width: parent.width
                            spacing: Theme.spacingMedium
                            
                            Column {
                                width: parent.width - autoAcceptSwitch.width - parent.spacing
                                spacing: Theme.spacingXSmall
                                
                                Text {
                                    text: qsTr("Auto Accept Connections")
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Text {
                                    text: qsTr("No confirmation needed for incoming connections")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                    width: parent.width
                                }
                            }
                            
                            QDSwitch {
                                id: autoAcceptSwitch
                                checked: false
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
                            
                            Text {
                                text: qsTr("Language")
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                width: parent.width - langCombo.width - parent.spacing
                            }
                            
                            QDComboBox {
                                id: langCombo
                                model: ["English", "简体中文"]
                                currentIndex: 0
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
                                model: [qsTr("Light"), qsTr("Dark"), qsTr("Auto")]
                                currentIndex: 1
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
