// Fluent Design MessageBox Component
import QtQuick
import QtQuick.Layouts

Item {
    id: root
    
    // ============ MessageBox Types ============
    
    enum Type {
        Information,
        Warning,
        Error,
        Question,
        Success
    }
    
    enum Buttons {
        Ok,
        OkCancel,
        YesNo,
        YesNoCancel
    }
    
    // ============ Custom Properties ============
    
    property int messageType: QDMessageBox.Type.Information
    property int buttons: QDMessageBox.Buttons.Ok
    property string title: "提示"
    property string message: ""
    property string detailMessage: ""
    property bool showing: false
    
    // ============ Signals ============
    
    signal accepted()
    signal rejected()
    signal yesClicked()
    signal noClicked()
    signal closed()
    
    // ============ Functions ============
    
    function show() {
        showing = true
    }
    
    function hide() {
        showing = false
        closed()
    }
    
    function showMessage(msg, type, btns) {
        message = msg
        messageType = type !== undefined ? type : QDMessageBox.Type.Information
        buttons = btns !== undefined ? btns : QDMessageBox.Buttons.Ok
        show()
    }
    
    // ============ Layout ============
    
    anchors.fill: parent
    visible: showing
    z: Theme.zIndexModal
    
    // ============ Overlay ============
    
    Rectangle {
        anchors.fill: parent
        color: Theme.overlay
        opacity: root.showing ? 1 : 0
        
        Behavior on opacity {
            NumberAnimation {
                duration: Theme.animationDurationMedium
                easing.type: Theme.animationEasingType
            }
        }
        
        MouseArea {
            anchors.fill: parent
            onClicked: {
                // Prevent click through
            }
        }
    }
    
    // ============ MessageBox Container ============
    
    Item {
        anchors.centerIn: parent
        width: 450
        height: messageBoxContent.implicitHeight + Theme.spacingXXLarge * 2
        
        opacity: root.showing ? 1 : 0
        scale: root.showing ? 1 : 0.9
        
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
            anchors.fill: messageBoxContainer
            target: messageBoxContainer
            shadowSize: 16
            shadowColor: Qt.rgba(0, 0, 0, 0.4)
        }
        
        Rectangle {
            id: messageBoxContainer
            anchors.fill: parent
            radius: Theme.radiusMedium
            color: Theme.surface
            border.width: Theme.borderWidthThin
            border.color: Theme.border
        
        ColumnLayout {
            id: messageBoxContent
            anchors.fill: parent
            anchors.margins: Theme.spacingXXLarge
            spacing: Theme.spacingLarge
            
            // Header with icon and title
            RowLayout {
                spacing: Theme.spacingMedium
                Layout.fillWidth: true
                
                // Icon
                Text {
                    text: {
                        switch(root.messageType) {
                            case QDMessageBox.Type.Information: return FluentIconGlyph.infoGlyph
                            case QDMessageBox.Type.Warning: return FluentIconGlyph.warningGlyph
                            case QDMessageBox.Type.Error: return FluentIconGlyph.errorGlyph
                            case QDMessageBox.Type.Question: return FluentIconGlyph.helpGlyph
                            case QDMessageBox.Type.Success: return FluentIconGlyph.checkMarkGlyph
                            default: return FluentIconGlyph.infoGlyph
                        }
                    }
                    font.family: "Segoe Fluent Icons"
                    font.pixelSize: 32
                    color: {
                        switch(root.messageType) {
                            case QDMessageBox.Type.Information: return Theme.info
                            case QDMessageBox.Type.Warning: return Theme.warning
                            case QDMessageBox.Type.Error: return Theme.error
                            case QDMessageBox.Type.Question: return Theme.primary
                            case QDMessageBox.Type.Success: return Theme.success
                            default: return Theme.info
                        }
                    }
                }
                
                // Title
                Text {
                    text: root.title
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.DemiBold
                    color: Theme.text
                    Layout.fillWidth: true
                }
            }
            
            // Message
            Text {
                text: root.message
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.text
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
            
            // Detail message (if provided)
            Text {
                visible: root.detailMessage !== ""
                text: root.detailMessage
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.textSecondary
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
            
            Item { Layout.fillHeight: true }
            
            // Buttons
            RowLayout {
                spacing: Theme.spacingMedium
                Layout.alignment: Qt.AlignRight
                
                // Yes button
                QDButton {
                    visible: root.buttons === QDMessageBox.Buttons.YesNo || 
                            root.buttons === QDMessageBox.Buttons.YesNoCancel
                    text: "是"
                    buttonType: QDButton.Type.Primary
                    onClicked: {
                        root.yesClicked()
                        root.accepted()
                        root.hide()
                    }
                }
                
                // No button
                QDButton {
                    visible: root.buttons === QDMessageBox.Buttons.YesNo || 
                            root.buttons === QDMessageBox.Buttons.YesNoCancel
                    text: "否"
                    buttonType: QDButton.Type.Secondary
                    onClicked: {
                        root.noClicked()
                        root.hide()
                    }
                }
                
                // Ok button
                QDButton {
                    visible: root.buttons === QDMessageBox.Buttons.Ok || 
                            root.buttons === QDMessageBox.Buttons.OkCancel
                    text: "确定"
                    buttonType: QDButton.Type.Primary
                    onClicked: {
                        root.accepted()
                        root.hide()
                    }
                }
                
                // Cancel button
                QDButton {
                    visible: root.buttons === QDMessageBox.Buttons.OkCancel || 
                            root.buttons === QDMessageBox.Buttons.YesNoCancel
                    text: "取消"
                    buttonType: QDButton.Type.Secondary
                    onClicked: {
                        root.rejected()
                        root.hide()
                    }
                }
            }
        }
    }
}
}
