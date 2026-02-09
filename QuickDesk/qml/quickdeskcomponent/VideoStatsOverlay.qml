// VideoStatsOverlay - Semi-transparent overlay showing detailed video statistics
// Similar to game FPS overlays or Chrome's webrtc-internals
import QtQuick
import QtQuick.Layouts
import "../component"

Rectangle {
    id: root

    // Input: performance stats object from RemoteWindow.performanceStatsMap
    property var stats: null

    width: 280
    height: contentColumn.implicitHeight + Theme.spacingMedium * 2
    radius: Theme.radiusMedium
    color: Qt.rgba(0, 0, 0, 0.78)
    border.width: Theme.borderWidthThin
    border.color: Qt.rgba(1, 1, 1, 0.1)

    // ── Helpers ──

    function fmtMs(val) {
        if (val === undefined || val === null) return "—"
        return val.toFixed(1) + " ms"
    }

    function fmtBandwidth(kbps) {
        if (kbps === undefined || kbps === null || kbps <= 0) return "—"
        if (kbps >= 1024) return (kbps / 1024).toFixed(1) + " Mbps"
        return kbps.toFixed(0) + " Kbps"
    }

    readonly property double maxBarMs: {
        if (!stats) return 1
        return Math.max(stats.captureMs || 0, stats.encodeMs || 0,
                        stats.decodeMs || 0, stats.paintMs || 0, 1)
    }

    // Overlay-specific colors (fixed light-on-dark, not theme-dependent)
    readonly property color _overlayLabel: Qt.rgba(1, 1, 1, 0.7)
    readonly property color _overlayValue: "#FFFFFF"
    readonly property color _overlaySection: Qt.rgba(1, 1, 1, 0.45)

    // Dark-overlay progress bar background (overrides QDProgressBar's Theme.surface)
    component OverlayProgressBackground: Rectangle {
        radius: height / 2
        color: Qt.rgba(1, 1, 1, 0.08)
    }

    ColumnLayout {
        id: contentColumn
        anchors.fill: parent
        anchors.margins: Theme.spacingMedium
        spacing: Theme.spacingXSmall

        // ── Title ──
        QDText {
            text: "Video Stats"
            type: QDText.Type.Body
            font.weight: Font.DemiBold
            colorRole: QDText.ColorRole.Custom
            customColor: root._overlayValue
        }

        QDSeparator {
            orientation: QDSeparator.Orientation.Horizontal
            Layout.fillWidth: true
            separatorColor: Qt.rgba(1, 1, 1, 0.15)
        }

        // ── Latency Breakdown ──
        QDText {
            text: "LATENCY"
            type: QDText.Type.Caption
            font.weight: Font.Bold
            font.letterSpacing: 1.5
            colorRole: QDText.ColorRole.Custom
            customColor: root._overlaySection
        }

        Repeater {
            model: [
                { label: "Capture",  key: "captureMs", barColor: "#4FC3F7" },
                { label: "Encode",   key: "encodeMs",  barColor: "#81C784" },
                { label: "Decode",   key: "decodeMs",  barColor: "#FFB74D" },
                { label: "Paint",    key: "paintMs",   barColor: "#E57373" }
            ]

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingSmall

                QDText {
                    Layout.preferredWidth: 55
                    text: modelData.label
                    type: QDText.Type.Caption
                    colorRole: QDText.ColorRole.Custom
                    customColor: root._overlayLabel
                }

                QDProgressBar {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 10
                    value: {
                        var val = root.stats ? (root.stats[modelData.key] || 0) : 0
                        return Math.min(val / root.maxBarMs, 1.0)
                    }
                    progressColor: modelData.barColor
                    animated: true
                    background: OverlayProgressBackground {
                        implicitWidth: parent.width
                        implicitHeight: parent.height
                    }
                }

                QDText {
                    Layout.preferredWidth: 55
                    text: root.fmtMs(root.stats ? root.stats[modelData.key] : 0)
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: root._overlayValue
                    horizontalAlignment: Text.AlignRight
                }
            }
        }

        // Total latency
        QDLabel {
            Layout.fillWidth: true
            label: "Total"
            value: root.fmtMs(root.stats ? root.stats.totalLatencyMs : 0)
            labelColor: root._overlayValue
            valueColor: root._overlayValue
            monoValue: true
            valueWeight: Font.DemiBold
            labelFontSize: Theme.fontSizeSmall
            valueFontSize: Theme.fontSizeSmall
        }

        QDSeparator {
            orientation: QDSeparator.Orientation.Horizontal
            Layout.fillWidth: true
            separatorColor: Qt.rgba(1, 1, 1, 0.15)
        }

        // ── Connection Info ──
        QDText {
            text: "CONNECTION"
            type: QDText.Type.Caption
            font.weight: Font.Bold
            font.letterSpacing: 1.5
            colorRole: QDText.ColorRole.Custom
            customColor: root._overlaySection
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            QDLabel {
                Layout.fillWidth: true
                label: "Resolution"
                value: root.stats && root.stats.frameWidth > 0
                       ? root.stats.frameWidth + " x " + root.stats.frameHeight : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "Codec"
                value: root.stats && root.stats.codec ? root.stats.codec : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "Quality"
                value: root.stats && root.stats.frameQuality >= 0 ? root.stats.frameQuality : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "Frame Rate"
                value: root.stats ? Math.round(root.stats.frameRate || 0) + " fps" : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "Bandwidth"
                value: root.fmtBandwidth(root.stats ? root.stats.bandwidthKbps : 0)
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "RTT"
                value: root.fmtMs(root.stats ? root.stats.roundTripMs : 0)
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "Packet Rate"
                value: root.stats ? Math.round(root.stats.packetRate || 0) + " /s" : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: "Encoded Rect"
                value: root.stats && root.stats.encodedRectWidth > 0
                       ? root.stats.encodedRectWidth + " x " + root.stats.encodedRectHeight : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }
        }
    }
}
