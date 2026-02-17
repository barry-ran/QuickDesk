// Shadow effect component using MultiEffect (Qt6)
// Applies shadow as layer.effect on the target item.
import QtQuick
import QtQuick.Effects

Item {
    id: root
    visible: false

    property Item target
    property int shadowSize: 8
    property color shadowColor: Qt.rgba(0, 0, 0, 0.3)
    property int radius: 8

    Component.onCompleted: _applyEffect()
    onTargetChanged: _applyEffect()

    function _applyEffect() {
        if (target) {
            target.layer.enabled = true
            target.layer.effect = effectComponent.createObject(root)
        }
    }

    Component {
        id: effectComponent
        MultiEffect {
            shadowEnabled: true
            shadowHorizontalOffset: 0
            shadowVerticalOffset: 2
            shadowBlur: Math.min(root.shadowSize / 16.0, 1.0)
            shadowColor: root.shadowColor
        }
    }
}
