// Fluent Design NavigationView Component
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts

Item {
    id: control
    
    // ============ Custom Properties ============
    
    property bool isExpanded: true
    property int collapsedWidth: 48
    property int expandedWidth: 320
    property alias menuItems: menuRepeater.model
    property int currentIndex: 0
    property alias header: headerContent.data
    property alias footer: footerContent.data
    property alias content: contentArea.data
    property bool showFooter: true
    
    // ============ Signals ============
    
    signal itemClicked(int index, var item)
    signal toggleRequested()
    
    // ============ Layout ============
    
    implicitWidth: 800
    implicitHeight: 600
    
    RowLayout {
        anchors.fill: parent
        spacing: 0
        
        // ============ Navigation Pane ============
        
        Rectangle {
            id: navigationPane
            Layout.preferredWidth: control.isExpanded ? control.expandedWidth : control.collapsedWidth
            Layout.fillHeight: true
            color: Theme.surface
            border.width: Theme.borderWidthThin
            border.color: Theme.border
            
            Behavior on Layout.preferredWidth {
                NumberAnimation { duration: Theme.animationDurationMedium; easing.type: Easing.OutCubic }
            }
            
            ColumnLayout {
                anchors.fill: parent
                spacing: 0
                
                // Header Area
                Item {
                    id: headerContent
                    Layout.fillWidth: true
                    Layout.preferredHeight: 72
                }
                
                // Separator after header
                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 1
                    color: Theme.border
                }
                
                // Menu Items ScrollView
                Controls.ScrollView {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    clip: true
                    
                    Controls.ScrollBar.horizontal.policy: Controls.ScrollBar.AlwaysOff
                    Controls.ScrollBar.vertical: QDScrollBar {}
                    
                    ColumnLayout {
                        width: navigationPane.width
                        spacing: Theme.spacingXSmall
                        
                        Repeater {
                            id: menuRepeater
                            
                            delegate: Rectangle {
                                required property int index
                                required property var modelData
                                
                                Layout.fillWidth: true
                                Layout.preferredHeight: 48
                                Layout.leftMargin: Theme.spacingSmall
                                Layout.rightMargin: Theme.spacingSmall
                                
                                property bool isSelected: control.currentIndex === index
                                property bool isHovered: false
                                
                                color: {
                                    if (isSelected) return Theme.surfaceVariant
                                    if (isHovered) return Theme.surfaceHover
                                    return "transparent"
                                }
                                radius: Theme.radiusSmall
                                
                                // Selection Indicator
                                Rectangle {
                                    visible: isSelected
                                    width: 3
                                    height: parent.height - 8
                                    anchors.left: parent.left
                                    anchors.leftMargin: 0
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: Theme.primary
                                    radius: 2
                                }
                                
                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }
                                
                                RowLayout {
                                    anchors.fill: parent
                                    anchors.leftMargin: Theme.spacingMedium
                                    anchors.rightMargin: Theme.spacingMedium
                                    spacing: Theme.spacingMedium
                                    
                                    Text {
                                        text: modelData.icon || FluentIconGlyph.favoriteStarGlyph
                                        font.family: "Segoe Fluent Icons"
                                        font.pixelSize: 16
                                        color: isSelected ? Theme.primary : Theme.text
                                        
                                        Behavior on color {
                                            ColorAnimation { duration: Theme.animationDurationFast }
                                        }
                                    }
                                    
                                    Text {
                                        visible: control.isExpanded
                                        text: modelData.text || "Item " + (index + 1)
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: isSelected ? Theme.primary : Theme.text
                                        Layout.fillWidth: true
                                        elide: Text.ElideRight
                                        
                                        Behavior on color {
                                            ColorAnimation { duration: Theme.animationDurationFast }
                                        }
                                    }
                                }
                                
                                MouseArea {
                                    anchors.fill: parent
                                    hoverEnabled: true
                                    cursorShape: Qt.PointingHandCursor
                                    
                                    onEntered: parent.isHovered = true
                                    onExited: parent.isHovered = false
                                    
                                    onClicked: {
                                        control.currentIndex = index
                                        control.itemClicked(index, modelData)
                                    }
                                }
                            }
                        }
                    }
                }
                
                Item { Layout.fillHeight: true }
                
                // Separator before footer
                Rectangle {
                    visible: control.showFooter
                    Layout.fillWidth: true
                    Layout.preferredHeight: 1
                    color: Theme.border
                }
                
                // Footer Area (auto-sizes to content)
                Item {
                    id: footerContent
                    visible: control.showFooter
                    Layout.fillWidth: true
                    Layout.preferredHeight: childrenRect.height
                }
            }
        }
        
        // ============ Content Area ============
        
        Item {
            id: contentArea
            Layout.fillWidth: true
            Layout.fillHeight: true
        }
    }
}
