// Floating Tool Button - Draggable button overlay on remote desktop
import QtQuick
import QtQuick.Controls
import "../component"

Item {
    id: root
    
    // Properties
    property string connectionId: ""
    property var clientManager: null
    property var desktopView: null
    
    // Size - include extra space for shadow (shadow size is 12px on each side)
    width: 80
    height: 80
    clip: false  // Don't clip shadow
    
    // Draggable behavior
    MouseArea {
        id: dragArea
        anchors.fill: parent
        drag.target: root
        drag.axis: Drag.XAndYAxis
        drag.minimumX: 0
        drag.maximumX: root.parent ? root.parent.width - root.width : 0
        drag.minimumY: 0
        drag.maximumY: root.parent ? root.parent.height - root.height : 0
        
        cursorShape: pressed ? Qt.ClosedHandCursor : Qt.OpenHandCursor
        hoverEnabled: true
        
        // Track if this is a drag or click
        property bool isDragging: false
        property point pressPos: Qt.point(0, 0)
        
        onPressed: function(mouse) {
            isDragging = false
            pressPos = Qt.point(mouse.x, mouse.y)
        }
        
        onPositionChanged: function(mouse) {
            // If moved more than 5 pixels, consider it a drag
            var dx = Math.abs(mouse.x - pressPos.x)
            var dy = Math.abs(mouse.y - pressPos.y)
            if (dx > 5 || dy > 5) {
                isDragging = true
            }
        }
        
        onClicked: {
            if (!isDragging) {
                // Calculate smart menu position
                var menuX, menuY
                var spaceRight = root.parent.width - (root.x + root.width)
                var spaceBottom = root.parent.height - (root.y + root.height)
                
                // Prefer right side, but use left if not enough space
                if (spaceRight >= floatingMenu.width + Theme.spacingMedium) {
                    menuX = root.x + root.width + Theme.spacingMedium
                } else {
                    menuX = root.x - floatingMenu.width - Theme.spacingMedium
                }
                
                // Prefer bottom, but use top if not enough space
                if (spaceBottom >= floatingMenu.height + Theme.spacingMedium) {
                    menuY = root.y
                } else {
                    menuY = root.y + root.height - floatingMenu.height
                }
                
                // Make sure menu is within bounds
                menuX = Math.max(Theme.spacingSmall, Math.min(menuX, root.parent.width - floatingMenu.width - Theme.spacingSmall))
                menuY = Math.max(Theme.spacingSmall, Math.min(menuY, root.parent.height - floatingMenu.height - Theme.spacingSmall))
                
                floatingMenu.x = menuX
                floatingMenu.y = menuY
                floatingMenu.open()
            }
        }
    }
    
    // Circular button background
    Rectangle {
        id: buttonBackground
        anchors.centerIn: parent
        width: 50
        height: 50
        radius: 25
        color: dragArea.containsMouse ? Theme.primaryHover : Theme.primary
        opacity: 0.9
        
        Behavior on color {
            ColorAnimation { duration: Theme.animationDurationFast }
        }
        
        // Icon
        Text {
            id: iconText
            anchors.centerIn: parent
            text: floatingMenu.opened ? FluentIconGlyph.cancelGlyph : FluentIconGlyph.moreGlyph
            font.family: "Segoe Fluent Icons"
            font.pixelSize: 20
            color: Theme.textOnPrimary
            
            // Smooth transition when icon changes
            Behavior on text {
                SequentialAnimation {
                    NumberAnimation {
                        target: iconText
                        property: "opacity"
                        to: 0
                        duration: Theme.animationDurationFast
                    }
                    PropertyAction {
                        target: iconText
                        property: "text"
                    }
                    NumberAnimation {
                        target: iconText
                        property: "opacity"
                        to: 1
                        duration: Theme.animationDurationFast
                    }
                }
            }
        }
    }

    // Shadow effect (outside of button, with margins for shadow space)
    QDShadow {
        anchors.fill: root
        target: root
        shadowSize: 12
        shadowColor: Qt.rgba(0, 0, 0, 0.4)
        z: -1
    }
    
    // Floating Menu using QDMenu
    QDMenu {
        id: floatingMenu
        parent: root.parent
        width: 220
        
        QDMenuItem {
            text: qsTr("Send Ctrl+Alt+Del")
            iconText: FluentIconGlyph.keyboardFullGlyph
            onTriggered: {
                console.log("Send Ctrl+Alt+Del to:", root.connectionId)
            }
        }
        
        QDMenuSeparator { }
        
        QDMenuItem {
            text: qsTr("Performance")
            iconText: FluentIconGlyph.areaChartGlyph
            onTriggered: {
                console.log("Show performance for:", root.connectionId)
            }
        }
        
        QDMenuItem {
            text: qsTr("Resolution")
            iconText: FluentIconGlyph.resizeMouseMediumGlyph
            onTriggered: {
                console.log("Show resolution for:", root.connectionId)
            }
        }
        
        QDMenuItem {
            text: qsTr("Quality")
            iconText: FluentIconGlyph.colorGlyph
            onTriggered: {
                console.log("Show quality for:", root.connectionId)
            }
        }
        
        QDMenuSeparator { }
        
        QDMenuItem {
            text: qsTr("Clipboard")
            iconText: FluentIconGlyph.clipboardListGlyph
            onTriggered: {
                console.log("Show clipboard for:", root.connectionId)
            }
        }
        
        QDMenuItem {
            text: qsTr("File Transfer")
            iconText: FluentIconGlyph.folderGlyph
            onTriggered: {
                console.log("Open file transfer for:", root.connectionId)
            }
        }
        
        QDMenuSeparator { }
        
        QDMenuItem {
            text: qsTr("Disconnect")
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                console.log("Disconnect connection:", root.connectionId)
                if (root.clientManager) {
                    root.clientManager.disconnectFromHost(root.connectionId)
                }
            }
        }
    }
}
