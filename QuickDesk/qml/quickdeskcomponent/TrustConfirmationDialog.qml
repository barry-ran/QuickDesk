import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

import "../component"

Item {
    id: root

    property string confirmationId: ""
    property string connectionId: ""
    property string toolName: ""
    property var toolArguments: ({})
    property string riskLevel: "medium"
    property var reasons: []
    property int timeoutSecs: 60

    property bool showing: false

    signal approved(string confirmationId, string reason)
    signal rejected(string confirmationId, string reason)

    anchors.fill: parent
    visible: showing
    z: Theme.zIndexModal + 10

    function showConfirmation(confId, connId, tool, args, risk, reasonList, timeout) {
        confirmationId = confId
        connectionId = connId
        toolName = tool
        toolArguments = args
        riskLevel = risk
        reasons = reasonList
        timeoutSecs = timeout
        countdownTimer.remaining = timeout
        countdownTimer.start()
        showing = true
    }

    function hide() {
        showing = false
        countdownTimer.stop()
    }

    Timer {
        id: countdownTimer
        property int remaining: 60
        interval: 1000
        repeat: true
        onTriggered: {
            remaining--
            if (remaining <= 0) {
                root.rejected(root.confirmationId, "timeout")
                root.hide()
            }
        }
    }

    Rectangle {
        anchors.fill: parent
        color: Theme.overlay
        opacity: root.showing ? 1 : 0
        Behavior on opacity {
            NumberAnimation { duration: Theme.animationDurationMedium }
        }
        MouseArea {
            anchors.fill: parent
            onClicked: {}
        }
    }

    Item {
        anchors.centerIn: parent
        width: 480
        height: contentCol.implicitHeight + 32

        opacity: root.showing ? 1 : 0
        scale: root.showing ? 1 : 0.8
        Behavior on opacity {
            NumberAnimation { duration: Theme.animationDurationMedium }
        }
        Behavior on scale {
            NumberAnimation { duration: Theme.animationDurationMedium; easing.type: Easing.OutBack }
        }

        QDShadow {
            anchors.fill: dialogBg
            target: dialogBg
            shadowSize: 20
            shadowColor: Qt.rgba(0, 0, 0, 0.5)
        }

        Rectangle {
            id: dialogBg
            anchors.fill: parent
            radius: Theme.radiusLarge
            color: Theme.surface
            border.width: 2
            border.color: root.riskLevel === "critical" ? Theme.error :
                          root.riskLevel === "high" ? "#F59E0B" : Theme.warning
        }

        ColumnLayout {
            id: contentCol
            anchors.fill: parent
            anchors.margins: 0
            spacing: 0

            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 56
                color: root.riskLevel === "critical" ? Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.15) :
                       root.riskLevel === "high" ? Qt.rgba(0.96, 0.62, 0.04, 0.15) :
                       Qt.rgba(Theme.warning.r, Theme.warning.g, Theme.warning.b, 0.15)
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
                    spacing: Theme.spacingSmall

                    Text {
                        text: FluentIconGlyph.warningGlyph
                        font.family: "Segoe Fluent Icons"
                        font.pixelSize: 24
                        color: root.riskLevel === "critical" ? Theme.error : "#F59E0B"
                    }

                    Text {
                        Layout.fillWidth: true
                        text: qsTr("Operation Confirmation Required")
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeTitle
                        font.weight: Font.DemiBold
                        color: Theme.text
                    }

                    Text {
                        text: countdownTimer.remaining + "s"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeLarge
                        font.weight: Font.Bold
                        color: countdownTimer.remaining <= 10 ? Theme.error : Theme.textSecondary
                    }
                }
            }

            Item {
                Layout.fillWidth: true
                Layout.preferredHeight: infoCol.implicitHeight + Theme.spacingLarge * 2

                ColumnLayout {
                    id: infoCol
                    anchors.fill: parent
                    anchors.margins: Theme.spacingLarge
                    spacing: Theme.spacingMedium

                    RowLayout {
                        spacing: Theme.spacingSmall
                        Text {
                            text: qsTr("Tool:")
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            font.weight: Font.DemiBold
                            color: Theme.textSecondary
                        }
                        Text {
                            text: root.toolName
                            font.family: Theme.fontFamilyMono ? Theme.fontFamilyMono : "Consolas"
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                        }
                    }

                    RowLayout {
                        spacing: Theme.spacingSmall
                        Text {
                            text: qsTr("Risk:")
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            font.weight: Font.DemiBold
                            color: Theme.textSecondary
                        }
                        Rectangle {
                            width: riskText.implicitWidth + 12
                            height: riskText.implicitHeight + 4
                            radius: Theme.radiusSmall
                            color: root.riskLevel === "critical" ? Theme.error :
                                   root.riskLevel === "high" ? "#F59E0B" : Theme.warning
                            Text {
                                id: riskText
                                anchors.centerIn: parent
                                text: root.riskLevel.toUpperCase()
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeSmall
                                font.weight: Font.Bold
                                color: "white"
                            }
                        }
                    }

                    Rectangle {
                        visible: root.reasons.length > 0
                        Layout.fillWidth: true
                        Layout.preferredHeight: reasonCol.implicitHeight + Theme.spacingMedium
                        color: Theme.surfaceVariant
                        radius: Theme.radiusSmall

                        ColumnLayout {
                            id: reasonCol
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.top: parent.top
                            anchors.margins: Theme.spacingSmall
                            spacing: 2

                            Repeater {
                                model: root.reasons
                                Text {
                                    Layout.fillWidth: true
                                    text: "• " + modelData
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.Wrap
                                }
                            }
                        }
                    }

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.preferredHeight: Math.min(argsText.implicitHeight + Theme.spacingMedium, 120)
                        color: Theme.surfaceVariant
                        radius: Theme.radiusSmall
                        clip: true

                        ScrollView {
                            anchors.fill: parent
                            anchors.margins: Theme.spacingSmall

                            Text {
                                id: argsText
                                width: parent.width
                                text: JSON.stringify(root.toolArguments, null, 2)
                                font.family: Theme.fontFamilyMono ? Theme.fontFamilyMono : "Consolas"
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                wrapMode: Text.Wrap
                            }
                        }
                    }
                }
            }

            Rectangle {
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
                    spacing: Theme.spacingLarge

                    QDButton {
                        text: qsTr("Reject")
                        onClicked: {
                            root.rejected(root.confirmationId, "user_rejected")
                            root.hide()
                        }
                    }

                    QDButton {
                        text: qsTr("Approve")
                        highlighted: true
                        onClicked: {
                            root.approved(root.confirmationId, "user_approved")
                            root.hide()
                        }
                    }
                }
            }
        }
    }
}
