// VideoStatsOverlay - Semi-transparent overlay showing detailed video statistics
// Similar to game FPS overlays or Chrome's webrtc-internals
import QtQuick
import QtQuick.Layouts
import "../component"

Rectangle {
    id: root

    // Input: performance stats object from RemoteWindow.performanceStatsMap
    property var stats: null

    width: 320
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

    function routeTypeLabel(type) {
        if (type === "direct") return qsTr("P2P (Direct)")
        if (type === "stun")   return qsTr("P2P (STUN)")
        if (type === "relay")  return qsTr("Relay (TURN)")
        return "—"
    }

    function routeTypeColor(type) {
        if (type === "direct" || type === "stun") return "#66BB6A"
        if (type === "relay") return "#FFA726"
        return root._overlayValue
    }

    function ipVersionTag(isIpv6) {
        return isIpv6 ? "IPv6" : "IPv4"
    }

    function formatAddress(addr) {
        if (!addr) return "—"
        if (addr.indexOf("redacted-ip.invalid") >= 0) {
            var port = addr.split(":").pop()
            return qsTr("hidden:%1").arg(port)
        }
        return addr
    }

    readonly property double maxBarMs: {
        if (!stats) return 50
        return Math.max(stats.captureMs || 0, stats.encodeMs || 0,
                        stats.networkDelayMs || 0,
                        stats.decodeMs || 0, stats.paintMs || 0, 50)
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
            text: qsTr("Video Stats")
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
            text: qsTr("LATENCY")
            type: QDText.Type.Caption
            font.weight: Font.Bold
            font.letterSpacing: 1.5
            colorRole: QDText.ColorRole.Custom
            customColor: root._overlaySection
        }

        Repeater {
            model: [
                { label: qsTr("Capture"),  key: "captureMs",      barColor: "#4FC3F7" },
                { label: qsTr("Encode"),   key: "encodeMs",       barColor: "#81C784" },
                { label: qsTr("Network"),  key: "networkDelayMs", barColor: "#FFA726" },
                { label: qsTr("Decode"),   key: "decodeMs",       barColor: "#FFB74D" },
                { label: qsTr("Paint"),    key: "paintMs",        barColor: "#E57373" }
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
            label: qsTr("Total")
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
            text: qsTr("CONNECTION")
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
                label: qsTr("Resolution")
                value: root.stats && root.stats.frameWidth > 0
                       ? root.stats.frameWidth + " x " + root.stats.frameHeight : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Codec")
                value: root.stats && root.stats.codec ? root.stats.codec : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Quality")
                value: root.stats && root.stats.frameQuality >= 0 ? root.stats.frameQuality : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Frame Rate")
                value: root.stats ? Math.round(root.stats.frameRate || 0) + " fps" : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Bandwidth")
                value: root.fmtBandwidth(root.stats ? root.stats.bandwidthKbps : 0)
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Input RTT")
                value: root.fmtMs(root.stats ? root.stats.inputRoundtripMs : 0)
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Packet Rate")
                value: root.stats ? Math.round(root.stats.packetRate || 0) + " /s" : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Encoded Rect")
                value: root.stats && root.stats.encodedRectWidth > 0
                       ? root.stats.encodedRectWidth + " x " + root.stats.encodedRectHeight : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }
        }

        // ── ICE / Route section ──
        QDSeparator {
            orientation: QDSeparator.Orientation.Horizontal
            Layout.fillWidth: true
            separatorColor: Qt.rgba(1, 1, 1, 0.15)
        }

        QDText {
            text: qsTr("ICE / ROUTE")
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
                label: qsTr("Type")
                value: root.stats ? root.routeTypeLabel(root.stats.routeType) : "—"
                labelColor: root._overlayLabel
                valueColor: root.stats ? root.routeTypeColor(root.stats.routeType)
                                       : root._overlayValue
                monoValue: true
                valueWeight: Font.DemiBold
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Protocol")
                value: root.stats && root.stats.transportProtocol
                       ? root.stats.transportProtocol : "—"
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Local")
                value: {
                    if (!root.stats || !root.stats.localAddress) return "—"
                    var t = root.stats.localCandidateType || ""
                    return root.stats.localAddress + " (" + t + ")"
                }
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }

            QDLabel {
                Layout.fillWidth: true
                label: qsTr("Remote")
                value: {
                    if (!root.stats || !root.stats.remoteAddress) return "—"
                    var t = root.stats.remoteCandidateType || ""
                    return root.formatAddress(root.stats.remoteAddress) + " (" + t + ")"
                }
                labelColor: root._overlayLabel; valueColor: root._overlayValue; monoValue: true
            }
        }

        // ── Client Candidates ──
        QDText {
            text: qsTr("CLIENT CANDIDATES")
            type: QDText.Type.Caption
            font.weight: Font.Bold
            font.letterSpacing: 1.5
            colorRole: QDText.ColorRole.Custom
            customColor: root._overlaySection
            visible: !!(root.stats && root.stats.localCandidates
                       && root.stats.localCandidates.length > 0)
        }

        Repeater {
            model: root.stats && root.stats.localCandidates
                   ? root.stats.localCandidates : []

            RowLayout {
                Layout.fillWidth: true
                spacing: 4

                QDText {
                    Layout.preferredWidth: 38
                    text: modelData.type || ""
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: {
                        var t = modelData.type
                        if (t === "host") return "#66BB6A"
                        if (t === "srflx" || t === "prflx") return "#4FC3F7"
                        if (t === "relay") return "#FFA726"
                        return root._overlayLabel
                    }
                }

                QDText {
                    Layout.preferredWidth: 22
                    text: modelData.protocol || ""
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: root._overlayLabel
                }

                QDText {
                    Layout.preferredWidth: 28
                    text: root.ipVersionTag(modelData.isIpv6)
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: modelData.isIpv6 ? "#CE93D8" : "#90CAF9"
                }

                QDText {
                    Layout.fillWidth: true
                    text: root.formatAddress(modelData.address)
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: root._overlayValue
                    elide: Text.ElideRight
                }
            }
        }

        // ── Host Candidates ──
        QDText {
            text: qsTr("HOST CANDIDATES")
            type: QDText.Type.Caption
            font.weight: Font.Bold
            font.letterSpacing: 1.5
            colorRole: QDText.ColorRole.Custom
            customColor: root._overlaySection
            visible: !!(root.stats && root.stats.remoteCandidates
                       && root.stats.remoteCandidates.length > 0)
        }

        Repeater {
            model: root.stats && root.stats.remoteCandidates
                   ? root.stats.remoteCandidates : []

            RowLayout {
                Layout.fillWidth: true
                spacing: 4

                QDText {
                    Layout.preferredWidth: 38
                    text: modelData.type || ""
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: {
                        var t = modelData.type
                        if (t === "host") return "#66BB6A"
                        if (t === "srflx" || t === "prflx") return "#4FC3F7"
                        if (t === "relay") return "#FFA726"
                        return root._overlayLabel
                    }
                }

                QDText {
                    Layout.preferredWidth: 22
                    text: modelData.protocol || ""
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: root._overlayLabel
                }

                QDText {
                    Layout.preferredWidth: 28
                    text: root.ipVersionTag(modelData.isIpv6)
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: modelData.isIpv6 ? "#CE93D8" : "#90CAF9"
                }

                QDText {
                    Layout.fillWidth: true
                    text: root.formatAddress(modelData.address)
                    type: QDText.Type.Caption
                    mono: true
                    colorRole: QDText.ColorRole.Custom
                    customColor: root._overlayValue
                    elide: Text.ElideRight
                }
            }
        }
    } // ColumnLayout
}
