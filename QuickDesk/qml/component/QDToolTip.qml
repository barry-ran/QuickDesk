// Fluent Design ToolTip Component
import QtQuick
import QtQuick.Controls as Controls

Controls.ToolTip {
    id: control
    
    // ============ Custom Properties ============
    
    property int tipDelay: 500
    
    // ============ Size & Style ============
    
    delay: tipDelay
    timeout: 5000
    
    padding: Theme.spacingSmall
    leftPadding: Theme.spacingMedium
    rightPadding: Theme.spacingMedium
    topPadding: Theme.spacingSmall - 2
    bottomPadding: Theme.spacingSmall - 2
    
    font.family: Theme.fontFamily
    font.pixelSize: Theme.fontSizeSmall
    
    // ============ Background ============
    
    background: Rectangle {
        id: bgRect
        color: Theme.surfaceVariant  // 使用Theme的surface变体色
        border.width: Theme.borderWidthThin
        border.color: Theme.border  // 使用Theme的边框色
        radius: Theme.radiusSmall
        
        // Shadow using QDShadow
        QDShadow {
            anchors.fill: parent
            target: bgRect
            shadowSize: 8
            shadowColor: Theme.shadowMedium  // 使用Theme的阴影色
        }
    }
    
    // ============ Content ============
    
    contentItem: Text {
        text: control.text
        font: control.font
        color: Theme.text  // 使用Theme的文字色
        wrapMode: Text.WordWrap
    }
    
    // ============ Animation ============
    
    enter: Transition {
        NumberAnimation { 
            property: "opacity"
            from: 0.0
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
            easing.type: Easing.InCubic
        }
    }
}
