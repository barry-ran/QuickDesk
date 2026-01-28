// Fluent Design Menu Item Component
import QtQuick

QtObject {
    id: control
    
    // ============ Properties ============
    
    property string text: ""
    property string iconText: ""  // FluentIcon glyph
    property bool checkable: false
    property bool checked: false
    property bool enabled: true
    property bool separator: false
    property bool visible: true
    property bool isDestructive: false  // Red text for destructive actions
    
    // ============ Signals ============
    
    signal triggered()
}
