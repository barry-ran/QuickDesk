// Remote Device Selector with History Dropdown
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import QtQuick.Effects
import "../component"

Item {
    id: root
    
    // ============ Public Properties ============
    
    property string deviceId: ""
    property string placeholderText: qsTr("Enter device ID")
    property var deviceList: []
    
    // ============ Signals ============
    
    signal deviceSelected(string deviceId)
    signal deviceDeleted(string deviceId)
    
    // ============ Size ============
    
    implicitWidth: 200
    implicitHeight: Theme.buttonHeightMedium
    
    // ============ Main Layout ============
    
    // Device ID Input with embedded dropdown button
    QDTextField {
        id: deviceInput
        anchors.fill: parent
        
        text: root.deviceId
        placeholderText: root.placeholderText
        
        // Validator: 9 digits
        validator: RegularExpressionValidator { 
            regularExpression: /^\d{0,9}$/
        }
        
        onTextChanged: {
            root.deviceId = text
        }
        
        // Watch for external changes to root.deviceId
        Connections {
            target: root
            function onDeviceIdChanged() {
                if (deviceInput.text !== root.deviceId) {
                    deviceInput.text = root.deviceId
                }
            }
        }
        
        Keys.onReturnPressed: {
            if (text.length === 9) {
                root.deviceSelected(text)
            }
        }
        
        Keys.onEnterPressed: {
            if (text.length === 9) {
                root.deviceSelected(text)
            }
        }
        
        // Embedded Dropdown Button (like password show/hide button)
        Row {
            anchors.right: parent.right
            anchors.rightMargin: Theme.spacingSmall
            anchors.verticalCenter: parent.verticalCenter
            spacing: 4
            
            QDIconButton {
                id: dropdownButton
                
                buttonSize: QDIconButton.Size.Small
                buttonStyle: QDIconButton.Style.Transparent
                
                iconSource: FluentIconGlyph.chevronDownGlyph
                iconColor: Theme.textSecondary
                iconHoverColor: Theme.primary
                
                // Rotate icon when popup is visible
                rotation: popup.visible ? 180 : 0
                
                Behavior on rotation {
                    NumberAnimation {
                        duration: Theme.animationDurationFast
                        easing.type: Easing.OutCubic
                    }
                }
                
                onClicked: {
                    if (root.deviceList && root.deviceList.length > 0) {
                        popup.visible = !popup.visible
                    }
                }
                
                // Tooltip
                QDToolTip {
                    visible: parent.hovered && root.deviceList && root.deviceList.length > 0
                    text: qsTr("Show history (%1)").arg(root.deviceList ? root.deviceList.length : 0)
                }
            }
        }
    }
    
    // ============ History Popup ============
    
    Controls.Popup {
        id: popup
        
        y: root.height + Theme.spacingSmall
        width: root.width
        height: Math.min(listView.contentHeight + Theme.spacingSmall * 2, 300)
        
        padding: Theme.spacingSmall
        
        closePolicy: Controls.Popup.CloseOnEscape | Controls.Popup.CloseOnPressOutsideParent
        
        // Popup Background
        background: Rectangle {
            color: Theme.surface
            border.width: Theme.borderWidthThin
            border.color: Theme.border
            radius: Theme.radiusMedium
            
            // Shadow effect
            layer.enabled: true
            layer.effect: MultiEffect {
                shadowEnabled: true
                shadowHorizontalOffset: 0
                shadowVerticalOffset: 4
                shadowBlur: 1.0
                shadowColor: Qt.rgba(0, 0, 0, 0.2)
            }
        }
        
        // History ListView
        contentItem: QDListView {
            id: listView
            
            implicitHeight: contentHeight
            
            model: root.deviceList
            
            delegate: Controls.ItemDelegate {
                id: delegate
                
                width: listView.width
                height: Theme.buttonHeightMedium
                padding: 0  // Remove padding to avoid layout shift
                
                contentItem: Item {
                    implicitHeight: Theme.buttonHeightMedium
                    
                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacingSmall
                        anchors.rightMargin: Theme.spacingSmall
                        spacing: Theme.spacingMedium
                        
                        // Device ID Text
                        Text {
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            text: modelData.deviceId || modelData
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            color: delegate.hovered ? Theme.primary : Theme.text
                            verticalAlignment: Text.AlignVCenter
                            elide: Text.ElideRight
                            
                            Behavior on color {
                                ColorAnimation { duration: Theme.animationDurationFast }
                            }
                        }
                        
                        // Delete Button (always take space, just change visibility)
                        Item {
                            Layout.preferredWidth: 28  // Fixed width for button
                            Layout.fillHeight: true
                            
                            QDIconButton {
                                anchors.centerIn: parent
                                visible: delegate.hovered
                                
                                buttonSize: QDIconButton.Size.Small
                                buttonStyle: QDIconButton.Style.Transparent
                                circular: true
                                
                                iconSource: FluentIconGlyph.deleteGlyph
                                iconColor: Theme.textSecondary
                                iconHoverColor: Theme.error
                                
                                onClicked: {
                                    popup.close()
                                    root.deviceDeleted(modelData.deviceId || modelData)
                                }
                                
                                QDToolTip {
                                    visible: parent.hovered
                                    text: qsTr("Delete from history")
                                }
                            }
                        }
                    }
                }
                
                background: Rectangle {
                    color: delegate.pressed ? Theme.surfacePressed :
                           delegate.hovered ? Theme.surfaceHover : "transparent"
                    radius: Theme.radiusSmall
                    
                    Behavior on color {
                        ColorAnimation { duration: Theme.animationDurationFast }
                    }
                }
                
                onClicked: {
                    var selectedDeviceId = modelData.deviceId || modelData
                    root.deviceId = selectedDeviceId
                    deviceInput.text = selectedDeviceId
                    popup.close()
                    root.deviceSelected(selectedDeviceId)
                }
                
                HoverHandler {
                    cursorShape: Qt.PointingHandCursor
                }
            }
            
            // Empty State
            Controls.Label {
                visible: listView.count === 0
                anchors.centerIn: parent
                text: qsTr("No history")
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.textSecondary
            }
        }
    }
}
