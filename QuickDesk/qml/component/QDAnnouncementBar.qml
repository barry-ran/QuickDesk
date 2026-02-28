// Fluent Design AnnouncementBar Component
// A top banner that displays announcements with auto-scrolling marquee effect.
// Supports rich text with clickable hyperlinks.
import QtQuick
import QtQuick.Layouts

Rectangle {
    id: control

    property string text: ""
    property string icon: FluentIconGlyph.megaphoneGlyph
    property real scrollSpeed: 60
    property bool showSeparator: true

    signal linkActivated(string link)

    visible: control.text !== ""
    implicitHeight: visible ? Theme.buttonHeightSmall : 0
    color: Theme.surfaceVariant

    Behavior on implicitHeight {
        NumberAnimation { duration: Theme.animationDurationMedium; easing.type: Easing.OutCubic }
    }

    QDSeparator {
        visible: control.showSeparator
        orientation: QDSeparator.Orientation.Horizontal
        anchors.bottom: parent.bottom
        width: parent.width
    }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingMedium
        anchors.rightMargin: Theme.spacingMedium
        spacing: Theme.spacingSmall

        QDText {
            visible: control.icon !== ""
            text: control.icon
            font.family: "Segoe Fluent Icons"
            font.pixelSize: 14
            colorRole: QDText.ColorRole.Custom
            customColor: Theme.info
        }

        Item {
            id: scrollArea
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true

            readonly property real overflow: Math.max(0, contentText.implicitWidth - scrollArea.width)
            readonly property bool needsScroll: overflow > 0
            readonly property bool hovered: scrollMouseArea.containsMouse

            QDText {
                id: contentText
                y: (parent.height - height) / 2
                text: control.text
                type: QDText.Type.Caption
                textFormat: Text.StyledText
                linkColor: Theme.primary

                onLinkActivated: function(link) {
                    control.linkActivated(link)
                }

                MouseArea {
                    id: scrollMouseArea
                    anchors.fill: parent
                    hoverEnabled: true
                    acceptedButtons: Qt.NoButton
                    cursorShape: contentText.hoveredLink ? Qt.PointingHandCursor : Qt.ArrowCursor
                }
            }

            SequentialAnimation {
                id: scrollAnimation
                running: scrollArea.needsScroll && control.visible
                paused: running && scrollArea.hovered
                loops: Animation.Infinite

                PauseAnimation { duration: 2000 }

                NumberAnimation {
                    target: contentText
                    property: "x"
                    from: 0
                    to: -scrollArea.overflow
                    duration: scrollArea.overflow / control.scrollSpeed * 1000
                    easing.type: Easing.Linear
                }

                PauseAnimation { duration: 2000 }

                ScriptAction {
                    script: contentText.x = 0
                }
            }

            Connections {
                target: scrollArea
                function onNeedsScrollChanged() {
                    if (!scrollArea.needsScroll) {
                        contentText.x = 0
                    }
                }
            }
        }
    }
}
