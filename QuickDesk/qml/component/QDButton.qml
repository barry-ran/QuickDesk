// Fluent Design Button Component
import QtQuick
import QtQuick.Controls as Controls

Controls.Button {
    id: control
    
    // ============ Custom Properties ============
    
    // Button variants
    enum Type {
        Primary,
        Secondary,
        Danger,
        Success,
        Ghost
    }
    
    property int buttonType: QDButton.Type.Primary
    property bool loading: false
    property string iconText: ""
    property int iconSize: Theme.iconSizeMedium
    
    // ============ Size & Style ============
    
    implicitWidth: Math.max(100, contentItem.implicitWidth + leftPadding + rightPadding)
    implicitHeight: Theme.buttonHeightMedium
    
    leftPadding: Theme.spacingLarge
    rightPadding: Theme.spacingLarge
    topPadding: Theme.spacingSmall
    bottomPadding: Theme.spacingSmall
    
    // ============ Background ============
    
    background: Rectangle {
        id: buttonBackground
        radius: Theme.radiusMedium
        border.width: buttonType === QDButton.Type.Ghost ? Theme.borderWidthThin : 0
        border.color: control.enabled ? Theme.border : Theme.borderHover
        
        color: {
            if (!control.enabled) {
                return Theme.primaryDisabled
            }
            
            switch (buttonType) {
                case QDButton.Type.Primary:
                    return control.down ? Theme.primaryPressed : 
                           control.hovered ? Theme.primaryHover : Theme.primary
                case QDButton.Type.Danger:
                    return control.down ? Theme.errorHover : 
                           control.hovered ? Theme.error : Theme.error
                case QDButton.Type.Success:
                    return control.down ? Theme.successHover : 
                           control.hovered ? Theme.success : Theme.success
                case QDButton.Type.Secondary:
                    return control.down ? Theme.surfaceVariant : 
                           control.hovered ? Theme.surfaceHover : Theme.surface
                case QDButton.Type.Ghost:
                    return control.down ? Theme.surfaceVariant : 
                           control.hovered ? Theme.surfaceHover : "transparent"
                default:
                    return Theme.primary
            }
        }
        
        Behavior on color {
            ColorAnimation {
                duration: Theme.animationDurationFast
                easing.type: Theme.animationEasingType
            }
        }
        
        // Ripple effect container
        clip: true
        
        Rectangle {
            id: ripple
            anchors.centerIn: parent
            width: 0
            height: width
            radius: width / 2
            color: Qt.rgba(1, 1, 1, 0.3)
            opacity: 0
            
            ParallelAnimation {
                id: rippleAnimation
                NumberAnimation {
                    target: ripple
                    property: "width"
                    from: 0
                    to: buttonBackground.width * 2
                    duration: 500
                    easing.type: Easing.OutCubic
                }
                NumberAnimation {
                    target: ripple
                    property: "opacity"
                    from: 0.5
                    to: 0
                    duration: 500
                    easing.type: Easing.OutCubic
                }
            }
        }
    }
    
    // ============ Content ============
    
    contentItem: Item {
        implicitWidth: contentRow.implicitWidth
        implicitHeight: contentRow.implicitHeight
        
        Row {
            id: contentRow
            anchors.centerIn: parent
            spacing: Theme.spacingSmall
            
            // Loading spinner
            Item {
                visible: control.loading
                width: visible ? control.iconSize : 0
                height: control.iconSize
                anchors.verticalCenter: parent.verticalCenter
                
                Rectangle {
                    id: spinner
                    width: control.iconSize
                    height: control.iconSize
                    color: "transparent"
                    border.width: 2
                    border.color: {
                        if (control.buttonType === QDButton.Type.Secondary || control.buttonType === QDButton.Type.Ghost) {
                            return Theme.primary
                        }
                        return Theme.textOnPrimary
                    }
                    radius: control.iconSize / 2
                    
                    Rectangle {
                        width: control.iconSize / 4
                        height: control.iconSize / 4
                        radius: width / 2
                        color: parent.border.color
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.verticalCenter
                    }
                    
                    RotationAnimation on rotation {
                        running: control.loading
                        loops: Animation.Infinite
                        from: 0
                        to: 360
                        duration: 1000
                    }
                }
            }
            
            // Icon
            Text {
                visible: control.iconText !== "" && !control.loading
                text: control.iconText
                font.family: "Segoe Fluent Icons"
                font.pixelSize: control.iconSize
                color: {
                    if (!control.enabled) {
                        return Theme.textDisabled
                    }
                    if (control.buttonType === QDButton.Type.Secondary || control.buttonType === QDButton.Type.Ghost) {
                        return Theme.text
                    }
                    return Theme.textOnPrimary
                }
                anchors.verticalCenter: parent.verticalCenter
            }
            
            // Text
            Text {
                visible: control.text !== ""
                text: control.text
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.DemiBold
                color: {
                    if (!control.enabled) {
                        return Theme.textDisabled
                    }
                    if (control.buttonType === QDButton.Type.Secondary || control.buttonType === QDButton.Type.Ghost) {
                        return Theme.text
                    }
                    return Theme.textOnPrimary
                }
                anchors.verticalCenter: parent.verticalCenter
            }
        }
    }
    
    // ============ Interactions ============
    
    onPressed: {
        if (enabled) {
            rippleAnimation.start()
        }
    }
    
    // Hover cursor
    HoverHandler {
        cursorShape: control.enabled ? Qt.PointingHandCursor : Qt.ForbiddenCursor
    }
}
