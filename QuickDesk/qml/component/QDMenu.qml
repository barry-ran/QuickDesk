// Fluent Design Menu Component
import QtQuick
import QtQuick.Controls as Controls

Controls.Popup {
    id: control
    
    // ============ Custom Properties ============
    
    property list<QtObject> menuItems
    default property list<QtObject> contentData  // 改为 QtObject 以接受 Component
    
    // ============ Size & Style ============
    
    implicitWidth: 200
    implicitHeight: menuContent.implicitHeight + padding * 2
    
    padding: Theme.spacingSmall
    modal: true
    dim: false
    closePolicy: Controls.Popup.CloseOnEscape | Controls.Popup.CloseOnPressOutside
    
    // ============ Background ============
    
    background: Item {
        Rectangle {
            id: menuBg
            anchors.fill: parent
            color: Theme.surfaceVariant
            border.width: 1
            border.color: Theme.borderHover
            radius: Theme.radiusMedium
        }
        
        // Shadow using QDShadow
        QDShadow {
            anchors.fill: menuBg
            target: menuBg
            shadowSize: 12
            shadowColor: Qt.rgba(0, 0, 0, 0.3)
            z: -1
            visible: true
        }
    }
    
    // ============ Content ============
    
    contentItem: Column {
        id: menuContent
        spacing: Theme.spacingXSmall
        
        Component.onCompleted: {
            // 如果 menuItems 为空，但有 contentData，将它们添加到 menuItems
            if (control.menuItems.length === 0 && control.contentData.length > 0) {
                var items = []
                for (var i = 0; i < control.contentData.length; i++) {
                    var item = control.contentData[i]
                    var itemType = item.toString()
                    
                    // 排除 Component 类型
                    if (item && itemType.indexOf("Component") === -1) {
                        items.push(item)
                    }
                }
                control.menuItems = items
            }
        }
        
        // 如果使用 menuItems 属性（数组方式）
        Repeater {
            model: control.menuItems.length > 0 ? control.menuItems : []
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
                    color: {
                        if (!menuItemArea.containsMouse) return "transparent"
                        if (modelData && modelData.isDestructive === true) {
                            return Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.1)
                        }
                        return Theme.surfaceHover
                    }
                    radius: Theme.radiusSmall
                    
                    Behavior on color {
                        ColorAnimation { duration: Theme.animationDurationFast }
                    }
                    
                    Row {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacingMedium
                        anchors.rightMargin: Theme.spacingMedium
                        spacing: Theme.spacingMedium
                        
                        // Icon (if provided)
                        Item {
                            width: Theme.iconSizeSmall
                            height: parent.height
                            visible: modelData && modelData.iconText !== undefined && modelData.iconText !== ""
                            
                            Text {
                                anchors.centerIn: parent
                                text: modelData ? (modelData.iconText || "") : ""
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: Theme.iconSizeSmall
                                color: {
                                    if (!modelData || (modelData.enabled !== undefined && !modelData.enabled)) {
                                        return Theme.textDisabled
                                    }
                                    if (modelData && modelData.isDestructive === true) {
                                        return Theme.error
                                    }
                                    return menuItemArea.containsMouse ? Theme.primary : Theme.text
                                }
                                
                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }
                            }
                        }
                        
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
                                color: menuItemArea.containsMouse ? Theme.primary : Theme.primary
                                
                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }
                            }
                        }
                        
                        // Text
                        Text {
                            width: {
                                var w = parent.width
                                if (modelData && modelData.iconText !== undefined && modelData.iconText !== "") {
                                    w -= Theme.iconSizeSmall + Theme.spacingMedium
                                }
                                if (modelData && modelData.checkable === true) {
                                    w -= Theme.iconSizeSmall + Theme.spacingMedium
                                }
                                if (modelData && modelData.hasSubmenu === true) {
                                    w -= Theme.iconSizeSmall + Theme.spacingMedium
                                }
                                return w
                            }
                            height: parent.height
                            text: (modelData && modelData.text) ? modelData.text : ""
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            color: {
                                if (!modelData || (modelData.enabled !== undefined && !modelData.enabled)) {
                                    return Theme.textDisabled
                                }
                                if (modelData && modelData.isDestructive === true) {
                                    return Theme.error
                                }
                                return Theme.text
                            }
                            verticalAlignment: Text.AlignVCenter
                            elide: Text.ElideRight
                            
                            Behavior on color {
                                ColorAnimation { duration: Theme.animationDurationFast }
                            }
                        }
                        
                        // Submenu indicator (right arrow)
                        Item {
                            width: Theme.iconSizeSmall
                            height: parent.height
                            visible: modelData && modelData.hasSubmenu === true
                            
                            Text {
                                anchors.centerIn: parent
                                text: FluentIconGlyph.chevronRightGlyph
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: Theme.iconSizeSmall
                                color: menuItemArea.containsMouse ? Theme.primary : Theme.textSecondary
                                
                                Behavior on color {
                                    ColorAnimation { duration: Theme.animationDurationFast }
                                }
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
                            // Only close menu if item doesn't have submenu
                            if (!modelData || !modelData.hasSubmenu) {
                                control.close()
                            }
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
        
        // 声明式子元素会自动添加到这里（通过 default property）
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
