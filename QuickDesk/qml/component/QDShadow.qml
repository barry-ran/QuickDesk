// Shadow effect component using DropShadow
import QtQuick
import Qt5Compat.GraphicalEffects

Item {
    id: root
    
    property alias target: dropShadow.source
    property int shadowSize: 8
    property color shadowColor: Qt.rgba(0, 0, 0, 0.3)
    property int radius: 8
    
    DropShadow {
        id: dropShadow
        anchors.fill: parent
        horizontalOffset: 0
        verticalOffset: 2
        radius: root.shadowSize
        samples: root.shadowSize * 2 + 1
        color: root.shadowColor
        transparentBorder: true
    }
}
