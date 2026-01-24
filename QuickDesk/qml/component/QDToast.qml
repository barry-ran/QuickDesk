// Fluent Design Toast Component
import QtQuick

Item {
    id: root
    
    // ============ Toast Types ============
    
    enum Type {
        Success,
        Error,
        Warning,
        Info
    }
    
    // ============ Custom Properties ============
    
    property int toastType: QDToast.Type.Info
    property string message: ""
    property int duration: 3000
    property bool autoHide: true
    
    // ============ Private Properties ============
    
    property bool _showing: false
    
    // ============ Functions ============
    
    function show(msg, type, dur) {
        message = msg
        toastType = type !== undefined ? type : QDToast.Type.Info
        duration = dur !== undefined ? dur : 3000
        _showing = true
        
        if (autoHide) {
            hideTimer.restart()
        }
    }
    
    function hide() {
        _showing = false
    }
    
    // ============ Layout ============
    
    anchors.horizontalCenter: parent ? parent.horizontalCenter : undefined
    anchors.bottom: parent ? parent.bottom : undefined
    anchors.bottomMargin: 50
    width: toastContainer.width
    height: toastContainer.height
    z: Theme.zIndexToast
    
    // ============ Toast Container ============
    
    Item {
        width: Math.min(500, toastContent.implicitWidth + Theme.spacingXLarge * 2)
        height: toastContent.implicitHeight + Theme.spacingMedium * 2
        
        opacity: _showing ? 1 : 0
        scale: _showing ? 1 : 0.8
        visible: opacity > 0
        
        Behavior on opacity {
            NumberAnimation {
                duration: Theme.animationDurationMedium
                easing.type: Easing.OutCubic
            }
        }
        
        Behavior on scale {
            NumberAnimation {
                duration: Theme.animationDurationMedium
                easing.type: Easing.OutBack
            }
        }
        
        // Shadow
        QDShadow {
            anchors.fill: toastContainer
            target: toastContainer
            shadowSize: 12
            shadowColor: Qt.rgba(0, 0, 0, 0.3)
        }
        
        Rectangle {
            id: toastContainer
            anchors.fill: parent
            radius: Theme.radiusMedium
        
            color: {
                switch(toastType) {
                    case QDToast.Type.Success: return Theme.success
                    case QDToast.Type.Error: return Theme.error
                    case QDToast.Type.Warning: return Theme.warning
                    case QDToast.Type.Info: return Theme.info
                    default: return Theme.surface
                }
            }
        
            // Content
            Row {
            id: toastContent
            anchors.centerIn: parent
            spacing: Theme.spacingMedium
            
            // Icon
            Text {
                text: {
                    switch(toastType) {
                        case QDToast.Type.Success: return FluentIconGlyph.checkMarkGlyph
                        case QDToast.Type.Error: return FluentIconGlyph.errorGlyph
                        case QDToast.Type.Warning: return FluentIconGlyph.warningGlyph
                        case QDToast.Type.Info: return FluentIconGlyph.infoGlyph
                        default: return FluentIconGlyph.infoGlyph
                    }
                }
                font.family: "Segoe Fluent Icons"
                font.pixelSize: Theme.iconSizeMedium
                color: Theme.textOnPrimary
                anchors.verticalCenter: parent.verticalCenter
            }
            
            // Message text
            Text {
                text: root.message
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.textOnPrimary
                wrapMode: Text.WordWrap
                maximumLineCount: 3
                elide: Text.ElideRight
                anchors.verticalCenter: parent.verticalCenter
            }
        }
        
        // Progress bar container (用于裁剪进度条不超出圆角)
        Item {
            visible: root.autoHide && _showing
            anchors.bottom: parent.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: toastContainer.radius
            height: 3
            clip: true
            
            Rectangle {
                anchors.left: parent.left
                anchors.top: parent.top
                height: parent.height
                width: (toastContainer.width - toastContainer.radius * 2) * (1 - hideTimer.progress)
                color: Qt.rgba(1, 1, 1, 0.3)
                radius: 1.5
                }
            }
        }
    }
    
    // ============ Auto Hide Timer ============
    
    Timer {
        id: hideTimer
        interval: root.duration
        running: false
        repeat: false
        
        property real progress: 0
        
        onTriggered: {
            root.hide()
            progress = 0
        }
        
        onRunningChanged: {
            if (running) {
                progressAnimation.start()
            } else {
                progressAnimation.stop()
                progress = 0
            }
        }
    }
    
    NumberAnimation {
        id: progressAnimation
        target: hideTimer
        property: "progress"
        from: 0
        to: 1
        duration: root.duration
        easing.type: Easing.Linear
    }
}
