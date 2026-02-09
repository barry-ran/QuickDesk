// Fluent Design Label Component
// A key-value display row for forms, settings, and stat panels.
//
// Usage:
//   QDLabel { label: "Resolution"; value: "1920 x 1080" }
//   QDLabel { label: "Status"; value: "Connected"; valueColor: Theme.success }
//   QDLabel { label: "Codec"; value: "H264"; monoValue: true }
import QtQuick
import QtQuick.Layouts

Item {
    id: control
    
    // ============ Custom Properties ============
    
    property string label: ""
    property string value: ""
    
    // Label style
    property color labelColor: Theme.textSecondary
    property int labelFontSize: Theme.fontSizeSmall
    
    // Value style
    property color valueColor: Theme.text
    property int valueFontSize: Theme.fontSizeSmall
    property bool monoValue: false  // Use monospace font for value
    property int valueWeight: Font.Normal
    
    // Layout
    property int labelWidth: -1  // Fixed label width (-1 = auto)
    property int spacing: Theme.spacingMedium
    
    // ============ Size ============
    
    implicitWidth: rowLayout.implicitWidth
    implicitHeight: Math.max(labelText.implicitHeight, valueText.implicitHeight)
    
    // ============ Content ============
    
    RowLayout {
        id: rowLayout
        anchors.fill: parent
        spacing: control.spacing
        
        Text {
            id: labelText
            text: control.label
            font.family: Theme.fontFamily
            font.pixelSize: control.labelFontSize
            color: control.labelColor
            Layout.preferredWidth: control.labelWidth >= 0 ? control.labelWidth : implicitWidth
        }
        
        Text {
            id: valueText
            text: control.value
            font.family: control.monoValue ? Theme.fontFamilyMono : Theme.fontFamily
            font.pixelSize: control.valueFontSize
            font.weight: control.valueWeight
            color: control.valueColor
            Layout.fillWidth: true
            elide: Text.ElideRight
        }
    }
}
