// Fluent Design ComboBox Component
import QtQuick
import QtQuick.Controls as Controls
import Qt5Compat.GraphicalEffects

Controls.ComboBox {
    id: control
    
    // ============ Size & Style ============
    
    implicitWidth: 200
    implicitHeight: Theme.buttonHeightMedium
    
    leftPadding: Theme.spacingMedium
    rightPadding: Theme.spacingMedium + indicator.width + Theme.spacingSmall
    
    font.family: Theme.fontFamily
    font.pixelSize: Theme.fontSizeMedium
    
    // ============ Background ============
    
    background: Rectangle {
        radius: Theme.radiusMedium
        color: control.enabled ? Theme.surface : Theme.surfaceVariant
        border.width: Theme.borderWidthMedium
        border.color: {
            if (control.down || control.popup.visible) {
                return Theme.borderFocus
            }
            if (control.hovered) {
                return Theme.borderHover
            }
            return Theme.border
        }
        
        Behavior on border.color {
            ColorAnimation {
                duration: Theme.animationDurationFast
                easing.type: Theme.animationEasingType
            }
        }
    }
    
    // ============ Content Item ============
    
    contentItem: Text {
        leftPadding: 0
        rightPadding: control.indicator.width + control.spacing
        text: control.displayText
        font: control.font
        color: control.enabled ? Theme.text : Theme.textDisabled
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideRight
    }
    
    // ============ Indicator (Arrow) ============
    
    indicator: Text {
        x: control.width - width - control.rightPadding + Theme.spacingSmall
        y: control.topPadding + (control.availableHeight - height) / 2
        text: FluentIconGlyph.chevronDownGlyph
        font.family: "Segoe Fluent Icons"
        font.pixelSize: Theme.iconSizeSmall
        color: control.enabled ? Theme.textSecondary : Theme.textDisabled
        
        rotation: control.popup.visible ? 180 : 0
        
        Behavior on rotation {
            NumberAnimation {
                duration: Theme.animationDurationFast
                easing.type: Easing.OutCubic
            }
        }
    }
    
    // ============ Popup ============
    
    popup: Controls.Popup {
        y: control.height + Theme.spacingSmall
        width: control.width
        implicitHeight: Math.min(contentItem.implicitHeight + padding * 2, 300)
        padding: Theme.spacingSmall
        
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
        
        contentItem: ListView {
            clip: true
            implicitHeight: contentHeight
            model: control.popup.visible ? control.delegateModel : null
            currentIndex: control.highlightedIndex
            
            Controls.ScrollIndicator.vertical: Controls.ScrollIndicator { }
        }
    }
    
    // ============ Delegate ============
    
    delegate: Controls.ItemDelegate {
        width: ListView.view.width
        height: Theme.buttonHeightMedium
        
        contentItem: Text {
            text: control.textRole ? 
                  (Array.isArray(control.model) ? modelData[control.textRole] : model[control.textRole])
                  : modelData
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeMedium
            color: highlighted ? Theme.textOnPrimary : Theme.text
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
        }
        
        background: Rectangle {
            color: highlighted ? Theme.primary : 
                   hovered ? Theme.surfaceHover : "transparent"
            radius: Theme.radiusSmall
            
            Behavior on color {
                ColorAnimation { duration: Theme.animationDurationFast }
            }
        }
        
        highlighted: control.highlightedIndex === index
    }
    
    // Hover cursor
    HoverHandler {
        cursorShape: control.enabled ? Qt.PointingHandCursor : Qt.ForbiddenCursor
    }
}
