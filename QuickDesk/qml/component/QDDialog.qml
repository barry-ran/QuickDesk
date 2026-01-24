// Fluent Design Dialog Component
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Item {
    id: root
    
    // ============ Custom Properties ============
    
    property string title: "Dialog"
    property bool showing: false
    property bool closeOnOverlay: true
    property bool showCloseButton: true
    property int dialogWidth: 400
    property int dialogHeight: 300
    
    default property alias content: contentContainer.data
    property list<Item> footer
    
    signal accepted()
    signal rejected()
    signal closed()
    
    // ============ Functions ============
    
    function show() {
        showing = true
    }
    
    function hide() {
        showing = false
        closed()
    }
    
    function accept() {
        showing = false
        accepted()
    }
    
    function reject() {
        showing = false
        rejected()
    }
    
    // ============ Layout ============
    
    anchors.fill: parent
    visible: showing
    z: Theme.zIndexModal
    
    // ============ Overlay ============
    
    Rectangle {
        id: overlay
        anchors.fill: parent
        color: Theme.overlay
        opacity: root.showing ? 1 : 0
        
        Behavior on opacity {
            NumberAnimation {
                duration: Theme.animationDurationMedium
                easing.type: Easing.OutCubic
            }
        }
        
        MouseArea {
            anchors.fill: parent
            onClicked: {
                if (root.closeOnOverlay) {
                    root.hide()
                }
            }
        }
    }
    
    // ============ Dialog Container ============
    
    Item {
        anchors.centerIn: parent
        width: dialogWidth
        height: dialogHeight
        
        opacity: root.showing ? 1 : 0
        scale: root.showing ? 1 : 0.7
        
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
            anchors.fill: dialogContainer
            target: dialogContainer
            shadowSize: 16
            shadowColor: Qt.rgba(0, 0, 0, 0.4)
        }
        
        Rectangle {
            id: dialogContainer
            anchors.fill: parent
            radius: Theme.radiusLarge
            color: Theme.surface
            border.width: Theme.borderWidthThin
            border.color: Theme.border
        
        // Prevent click through
        MouseArea {
            anchors.fill: parent
            onClicked: {} // Consume clicks
        }
        
        // Layout
        ColumnLayout {
            anchors.fill: parent
            spacing: 0
            
            // ============ Title Bar ============
            
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 56
                color: Theme.surfaceVariant
                radius: Theme.radiusLarge
                
                Rectangle {
                    anchors.bottom: parent.bottom
                    width: parent.width
                    height: parent.radius
                    color: parent.color
                }
                
                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: Theme.spacingLarge
                    anchors.rightMargin: Theme.spacingLarge
                    spacing: Theme.spacingMedium
                    
                    // Title text
                    Text {
                        Layout.fillWidth: true
                        text: root.title
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeTitle
                        font.weight: Font.DemiBold
                        color: Theme.text
                        elide: Text.ElideRight
                    }
                    
                    // Close button
                    Rectangle {
                        visible: root.showCloseButton
                        Layout.preferredWidth: 32
                        Layout.preferredHeight: 32
                        radius: Theme.radiusMedium
                        color: closeMouseArea.containsMouse ? 
                               Theme.error : (closeMouseArea.pressed ? 
                               Theme.errorHover : "transparent")
                        
                        Behavior on color {
                            ColorAnimation { duration: Theme.animationDurationFast }
                        }
                        
                        Text {
                            anchors.centerIn: parent
                            text: FluentIconGlyph.cancelGlyph
                            font.family: "Segoe Fluent Icons"
                            font.pixelSize: Theme.iconSizeMedium
                            color: closeMouseArea.containsMouse ? 
                                   Theme.textOnPrimary : Theme.text
                        }
                        
                        MouseArea {
                            id: closeMouseArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: root.hide()
                        }
                    }
                }
            }
            
            // ============ Content Area ============
            
            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true
                
                Item {
                    id: contentContainer
                    anchors.fill: parent
                    anchors.margins: Theme.spacingLarge
                }
            }
            
            // ============ Footer ============
            
            Rectangle {
                visible: footer.length > 0
                Layout.fillWidth: true
                Layout.preferredHeight: 72
                color: Theme.surfaceVariant
                radius: Theme.radiusLarge
                
                Rectangle {
                    anchors.top: parent.top
                    width: parent.width
                    height: parent.radius
                    color: parent.color
                }
                
                RowLayout {
                    anchors.centerIn: parent
                    anchors.margins: Theme.spacingLarge
                    spacing: Theme.spacingMedium
                    
                    Repeater {
                        model: footer
                        delegate: Item {
                            width: modelData.width
                            height: modelData.height
                            Component.onCompleted: {
                                modelData.parent = this
                            }
                        }
                    }
                }
            }
        }
    }
}
}
