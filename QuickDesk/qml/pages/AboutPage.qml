import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"

Item {
    id: root

    Rectangle {
        anchors.fill: parent
        color: Theme.background

        Flickable {
            anchors.fill: parent
            contentWidth: width
            contentHeight: contentColumn.implicitHeight
            clip: true

            ScrollBar.vertical: QDScrollBar {}

            Column {
                id: contentColumn
                width: parent.width
                spacing: Theme.spacingLarge

                Item { width: 1; height: Theme.spacingMedium }

                Text {
                    x: Theme.spacingXLarge
                    text: qsTr("About")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                // App Info Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: appInfoColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: appInfoColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingLarge

                        Row {
                            width: parent.width
                            spacing: Theme.spacingLarge

                            Image {
                                width: 64
                                height: 64
                                source: "qrc:/image/tray/logo.png"
                                fillMode: Image.PreserveAspectFit
                                smooth: true
                            }

                            Column {
                                width: parent.width - 64 - parent.spacing
                                spacing: Theme.spacingXSmall
                                anchors.verticalCenter: parent.verticalCenter

                                Text {
                                    text: "QuickDesk"
                                    font.pixelSize: Theme.fontSizeXLarge
                                    font.weight: Font.Bold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("Version") + " " + APP_VERSION
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.textSecondary
                                }
                            }
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        Text {
                            width: parent.width
                            text: qsTr("QuickDesk is an open-source remote desktop application built on Chromium Remoting technology, providing secure, high-performance remote control capabilities.")
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                            wrapMode: Text.WordWrap
                            lineHeight: 1.4
                        }
                    }
                }

                // Links Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: linksColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: linksColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingMedium

                        Text {
                            text: qsTr("Links")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        // GitHub link
                        Rectangle {
                            width: parent.width
                            height: 48
                            color: githubMouseArea.containsMouse ? Theme.surfaceHover : "transparent"
                            radius: Theme.radiusSmall

                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingMedium
                                anchors.rightMargin: Theme.spacingMedium
                                spacing: Theme.spacingMedium

                                Text {
                                    text: FluentIconGlyph.globeGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 16
                                    color: Theme.primary
                                }

                                Column {
                                    Layout.fillWidth: true
                                    spacing: 2

                                    Text {
                                        text: qsTr("Source Code")
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }

                                    Text {
                                        text: "https://github.com/barry-ran/QuickDesk"
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                }

                                Text {
                                    text: FluentIconGlyph.goGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 12
                                    color: Theme.textSecondary
                                }
                            }

                            MouseArea {
                                id: githubMouseArea
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: Qt.openUrlExternally("https://github.com/barry-ran/QuickDesk")
                            }
                        }

                        // Chromium Remoting link
                        Rectangle {
                            width: parent.width
                            height: 48
                            color: chromiumMouseArea.containsMouse ? Theme.surfaceHover : "transparent"
                            radius: Theme.radiusSmall

                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingMedium
                                anchors.rightMargin: Theme.spacingMedium
                                spacing: Theme.spacingMedium

                                Text {
                                    text: FluentIconGlyph.linkGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 16
                                    color: Theme.primary
                                }

                                Column {
                                    Layout.fillWidth: true
                                    spacing: 2

                                    Text {
                                        text: qsTr("Based on Chromium Remoting")
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }

                                    Text {
                                        text: "https://chromium.googlesource.com/chromium/src/+/refs/tags/140.0.7339.249/remoting/"
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        elide: Text.ElideRight
                                        width: parent.width
                                    }
                                }

                                Text {
                                    text: FluentIconGlyph.goGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 12
                                    color: Theme.textSecondary
                                }
                            }

                            MouseArea {
                                id: chromiumMouseArea
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: Qt.openUrlExternally("https://chromium.googlesource.com/chromium/src/+/refs/tags/140.0.7339.249/remoting/")
                            }
                        }
                    }
                }

                // License Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: licenseColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: licenseColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingMedium

                        Text {
                            text: qsTr("License")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        Text {
                            width: parent.width
                            text: qsTr("This project is licensed under the BSD 3-Clause License, same as the Chromium project.")
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                            wrapMode: Text.WordWrap
                            lineHeight: 1.4
                        }
                    }
                }

                Item { width: 1; height: Theme.spacingXLarge }
            }
        }
    }
}
