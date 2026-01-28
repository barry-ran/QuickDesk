// Remote Tab Component - Single tab in the tab bar
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"

Rectangle {
    id: control
    
    // Properties
    property string connectionId: ""
    property string deviceName: ""
    property int ping: 0
    property string connectionState: "connected" // connected, connecting, disconnected
    property bool isActive: false
    
    // Signals
    signal clicked()
    signal closeRequested()
    
    // Size
    implicitWidth: 200
    implicitHeight: 40
    
    // Style
    color: {
        if (isActive) return Theme.surfaceVariant
        if (mouseArea.containsMouse) return Theme.surfaceHover
        return Theme.surface
    }
    
    border.width: Theme.borderWidthThin
    border.color: isActive ? Theme.primary : Theme.border
    radius: Theme.radiusSmall
    
    Behavior on color {
        ColorAnimation { duration: Theme.animationDurationFast }
    }
    
    RowLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacingSmall
        spacing: Theme.spacingSmall
        
        // Device Icon
        Text {
            text: FluentIconGlyph.devicesGlyph
            font.family: "Segoe Fluent Icons"
            font.pixelSize: 16
            color: Theme.textSecondary
        }
        
        // Device Name and Status
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2
            
            Text {
                text: deviceName || connectionId
                font.pixelSize: Theme.fontSizeSmall
                font.weight: isActive ? Font.DemiBold : Font.Normal
                color: Theme.text
                elide: Text.ElideRight
                Layout.fillWidth: true
            }
            
            RowLayout {
                spacing: Theme.spacingXSmall
                
                // Status Indicator
                Rectangle {
                    width: 8
                    height: 8
                    radius: 4
                    color: {
                        if (connectionState !== "connected") return Theme.textDisabled
                        if (ping < 50) return Theme.success
                        if (ping < 100) return Theme.warning
                        return Theme.error
                    }
                    
                    Behavior on color {
                        ColorAnimation { duration: Theme.animationDurationFast }
                    }
                }
                
                // Ping Display
                Text {
                    text: ping + " ms"
                    font.pixelSize: Theme.fontSizeSmall
                    font.family: Theme.fontFamilyMono
                    color: Theme.textSecondary
                    visible: connectionState === "connected"
                }
                
                // Connecting Text
                Text {
                    text: qsTr("Connecting...")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    visible: connectionState === "connecting"
                }
            }
        }
        
        // Close Button
        Rectangle {
            width: 20
            height: 20
            radius: 10
            color: closeArea.containsMouse ? Theme.error : "transparent"
            
            Behavior on color {
                ColorAnimation { duration: Theme.animationDurationFast }
            }
            
            Text {
                anchors.centerIn: parent
                text: FluentIconGlyph.cancelGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 10
                color: closeArea.containsMouse ? Theme.textOnPrimary : Theme.textSecondary
            }
            
            MouseArea {
                id: closeArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                
                onClicked: {
                    control.closeRequested()
                }
            }
        }
    }
    
    // Tab Click Area
    MouseArea {
        id: mouseArea
        anchors.fill: parent
        anchors.rightMargin: 24 // Exclude close button
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        
        onClicked: {
            control.clicked()
        }
    }
}
