// Fluent Design Menu Component
import QtQuick
import QtQuick.Controls as Controls
import Qt5Compat.GraphicalEffects

Controls.Popup {
    id: control
    
    // ============ Custom Properties ============
    
    property list<QtObject> menuItems
    
    // ============ Size & Style ============
    
    implicitWidth: 200
    implicitHeight: menuContent.implicitHeight + padding * 2
    
    padding: Theme.spacingSmall
    modal: true
    dim: false
    closePolicy: Controls.Popup.CloseOnEscape | Controls.Popup.CloseOnPressOutside
    
    // ============ Background ============
    
    background: Rectangle {
        color: Theme.surface
        border.width: Theme.borderWidthThin
        border.color: Theme.border
        radius: Theme.radiusMedium
        
        // DropShadow effect
        layer.enabled: true
        layer.effect: DropShadow {
            horizontalOffset: 0
            verticalOffset: 2
            radius: 12
            samples: 25
            color: Qt.rgba(0, 0, 0, 0.3)
            transparentBorder: true
        }
    }
    
    // ============ Content ============
    
    contentItem: Column {
        id: menuContent
        spacing: Theme.spacingXSmall
        
        Repeater {
            model: control.contentData
            delegate: Item {
                width: control.width - control.padding * 2
                height: {
                    if (!modelData || (modelData.visible !== undefined && !modelData.visible)) return 0
                    return (modelData.separator === true) ? 1 : Theme.buttonHeightMedium
                }
                visible: !modelData || (modelData.visible === undefined || modelData.visible)
                
                // Menu Item
                Rectangle {
                    id: menuItemBg
                    anchors.fill: parent
                    visible: modelData && modelData.separator !== true
                    color: menuItemArea.containsMouse ? Theme.primary : "transparent"
                    radius: Theme.radiusSmall
                    
                    Behavior on color {
                        ColorAnimation { duration: Theme.animationDurationFast }
                    }
                    
                    Row {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacingMedium
                        anchors.rightMargin: Theme.spacingMedium
                        spacing: Theme.spacingMedium
                        
                        // Checkbox indicator
                        Item {
                            width: Theme.iconSizeSmall
                            height: parent.height
                            visible: modelData && modelData.checkable === true
                            
                            Text {
                                anchors.centerIn: parent
                                visible: modelData && modelData.checked === true
                                text: FluentIconGlyph.checkMarkGlyph
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: Theme.iconSizeSmall
                                color: menuItemArea.containsMouse ? Theme.textOnPrimary : Theme.primary
                                
                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }
                            }
                        }
                        
                        // Text
                        Text {
                            width: parent.width - ((modelData && modelData.checkable === true) ? Theme.iconSizeSmall + Theme.spacingMedium : 0)
                            height: parent.height
                            text: (modelData && modelData.text) ? modelData.text : ""
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            color: {
                                if (!modelData || (modelData.enabled !== undefined && !modelData.enabled)) {
                                    return Theme.textDisabled
                                }
                                return menuItemArea.containsMouse ? Theme.textOnPrimary : Theme.text
                            }
                            verticalAlignment: Text.AlignVCenter
                            elide: Text.ElideRight
                            
                            Behavior on color {
                                ColorAnimation { duration: Theme.animationDurationFast }
                            }
                        }
                    }
                    
                    MouseArea {
                        id: menuItemArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: (!modelData || (modelData.enabled !== undefined && !modelData.enabled)) ? Qt.ForbiddenCursor : Qt.PointingHandCursor
                        enabled: !modelData || (modelData.enabled === undefined || modelData.enabled)
                        
                        onClicked: {
                            if (modelData && modelData.triggered) {
                                modelData.triggered()
                            }
                            control.close()
                        }
                    }
                }
                
                // Separator
                Rectangle {
                    anchors.centerIn: parent
                    width: parent.width - Theme.spacingMedium * 2
                    height: 1
                    visible: modelData && modelData.separator === true
                    color: Theme.border
                }
            }
        }
    }
    
    // ============ Animation ============
    
    enter: Transition {
        NumberAnimation {
            property: "opacity"
            from: 0.0
            to: 1.0
            duration: Theme.animationDurationFast
        }
        NumberAnimation {
            property: "scale"
            from: 0.95
            to: 1.0
            duration: Theme.animationDurationFast
            easing.type: Easing.OutCubic
        }
    }
    
    exit: Transition {
        NumberAnimation {
            property: "opacity"
            from: 1.0
            to: 0.0
            duration: Theme.animationDurationFast
        }
    }
}
