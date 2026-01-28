// Remote Tab Bar Component - Tab bar for remote desktop windows
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"
import "."

Rectangle {
    id: control
    
    // Properties
    property var connections: [] // Array of connection objects
    property int currentIndex: 0
    
    // Signals
    signal tabClicked(int index)
    signal tabCloseRequested(int index)
    signal newTabRequested()
    
    // Style
    color: Theme.surface
    border.width: Theme.borderWidthThin
    border.color: Theme.border
    
    implicitHeight: 50
    
    RowLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacingSmall
        spacing: Theme.spacingSmall
        
        // Scrollable tab area
        ListView {
            id: tabListView
            Layout.fillWidth: true
            Layout.fillHeight: true
            
            orientation: ListView.Horizontal
            spacing: Theme.spacingSmall
            clip: true
            
            model: control.connections
            
            delegate: RemoteTab {
                required property int index
                required property var modelData
                
                connectionId: modelData.id || ""
                deviceName: modelData.name || modelData.deviceId || ""
                ping: modelData.ping || 0
                connectionState: modelData.state || "connected"
                isActive: index === control.currentIndex
                
                onClicked: {
                    control.tabClicked(index)
                }
                
                onCloseRequested: {
                    control.tabCloseRequested(index)
                }
            }
            
            // Scroll buttons (if needed)
            ScrollBar.horizontal: ScrollBar {
                policy: ScrollBar.AsNeeded
            }
        }
        
        // Separator
        Rectangle {
            width: 1
            Layout.fillHeight: true
            color: Theme.border
        }
        
        // New Tab Button
        Rectangle {
            width: 40
            height: 40
            radius: Theme.radiusMedium
            color: newTabArea.containsMouse ? Theme.surfaceHover : "transparent"
            
            Behavior on color {
                ColorAnimation { duration: Theme.animationDurationFast }
            }
            
            Text {
                anchors.centerIn: parent
                text: FluentIconGlyph.addGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 16
                color: Theme.text
            }
            
            MouseArea {
                id: newTabArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                
                onClicked: {
                    control.newTabRequested()
                }
            }
            
            QDToolTip {
                visible: newTabArea.containsMouse
                text: qsTr("New Connection")
            }
        }
    }
}
