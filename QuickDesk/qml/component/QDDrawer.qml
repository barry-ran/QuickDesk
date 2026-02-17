// Fluent Design Drawer Component
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Effects

Controls.Drawer {
    id: control
    
    // ============ Custom Properties ============
    
    property string title: ""
    property bool showCloseButton: true
    
    // ============ Size & Style ============
    
    implicitWidth: 320
    implicitHeight: parent ? parent.height : 0
    
    modal: true
    interactive: true
    
    // Drawer animation
    Behavior on position {
        NumberAnimation {
            duration: Theme.animationDurationMedium
            easing.type: Easing.OutCubic
        }
    }
    
    // ============ Background ============
    
    background: Rectangle {
        color: Theme.surface
        
        // Shadow effect
        layer.enabled: true
        layer.effect: MultiEffect {
            shadowEnabled: true
            shadowHorizontalOffset: control.edge === Qt.LeftEdge ? 4 : -4
            shadowVerticalOffset: 0
            shadowBlur: 1.0
            shadowColor: Qt.rgba(0, 0, 0, 0.3)
        }
        
        // Border
        Rectangle {
            anchors.fill: parent
            color: "transparent"
            border.width: Theme.borderWidthThin
            border.color: Theme.border
        }
    }
    
    // ============ Content ============
    
    contentItem: Item {
        
        // Header
        Rectangle {
            id: header
            anchors.top: parent.top
            anchors.left: parent.left
            anchors.right: parent.right
            height: Theme.buttonHeightLarge
            color: Theme.surfaceVariant
            
            Row {
                anchors.fill: parent
                anchors.leftMargin: Theme.spacingMedium
                anchors.rightMargin: Theme.spacingMedium
                spacing: Theme.spacingMedium
                
                Text {
                    anchors.verticalCenter: parent.verticalCenter
                    text: control.title
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeLarge
                    font.weight: Font.DemiBold
                    color: Theme.text
                    elide: Text.ElideRight
                    width: parent.width - closeButton.width - parent.spacing
                }
                
                QDIconButton {
                    id: closeButton
                    visible: control.showCloseButton
                    anchors.verticalCenter: parent.verticalCenter
                    iconSource: FluentIconGlyph.cancelGlyph
                    buttonStyle: QDIconButton.Style.Transparent
                    onClicked: control.close()
                }
            }
            
            // Bottom border
            Rectangle {
                anchors.bottom: parent.bottom
                width: parent.width
                height: Theme.borderWidthThin
                color: Theme.border
            }
        }
        
        // Content area (to be filled by user)
        Item {
            id: drawerContent
            anchors.top: header.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            anchors.margins: Theme.spacingMedium
            
            // This will be replaced by user's content through default property
            // or explicit contentData assignment
        }
    }
    
    // Make contentData point to the content area
    default property alias content: drawerContent.data
    
    // ============ Dim Overlay ============
    
    Controls.Overlay.modal: Rectangle {
        color: Qt.rgba(0, 0, 0, 0.5)
        
        Behavior on opacity {
            NumberAnimation {
                duration: Theme.animationDurationMedium
                easing.type: Easing.OutCubic
            }
        }
    }
}
