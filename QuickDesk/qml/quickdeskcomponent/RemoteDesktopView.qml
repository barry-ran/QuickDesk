// Copyright 2026 QuickDesk Authors
// Remote desktop video display component with input event support

import QtQuick
import QtQuick.Controls as Controls
import QtMultimedia
import QtQuick.Window
import QuickDesk 1.0

/**
 * RemoteDesktopView - Displays remote desktop video stream with input support
 * 
 * Usage:
 *   RemoteDesktopView {
 *       deviceId: "123456789"
 *       clientManager: mainController.clientManager
 *       active: visible  // Only render when visible
 *       inputEnabled: true // Enable mouse/keyboard input
 *   }
 */
Rectangle {
    id: root
    
    // Required properties
    required property string deviceId
    required property ClientManager clientManager
    
    // Optional properties
    property bool active: true
    property bool inputEnabled: true  // Enable/disable input capture
    property bool filePasteKeyHeld: false
    property var forwardedKeys: ({})
    // Set by an overlapping local control surface (for example, the tab bar).
    // While active, show the native cursor instead of the remote cursor.
    property bool suppressRemoteCursor: false

    readonly property int fitToScreenMode: 0
    readonly property int originalSizeMode: 1
    readonly property int stretchToFillMode: 2
    property int displayMode: fitToScreenMode
    property bool showMiniMap: true
    
    signal filesDropped(var urls)
    
    // Read-only properties
    readonly property int frameWidth: frameProvider.frameSize.width
    readonly property int frameHeight: frameProvider.frameSize.height
    readonly property int frameRate: frameProvider.frameRate
    readonly property bool hasVideo: frameWidth > 0 && frameHeight > 0

    // DIP dimensions from host VideoLayout (logical pixels / points).
    // Used for mouse coordinate mapping. Falls back to frame pixel size
    // when VideoLayout has not been received yet.
    property int remoteDipWidth: 0
    property int remoteDipHeight: 0

    // Display offset in the global desktop coordinate space (DIPs).
    // Added to mouse coordinates so the host maps input to the correct monitor.
    property int remoteOffsetX: 0
    property int remoteOffsetY: 0
    
    color: "#1a1a1a"  // Dark background
    focus: inputEnabled  // Enable keyboard focus when input is enabled

    // System shortcuts and focus changes can prevent QML from delivering the
    // matching key-up event. Tell the host to clear its authoritative state.
    function releaseRemoteInput() {
        filePasteKeyHeld = false
        forwardedKeys = ({})
        if (clientManager && deviceId.length > 0) {
            clientManager.releaseAllInput(deviceId)
        }
    }

    onInputEnabledChanged: {
        if (!inputEnabled) releaseRemoteInput()
    }

    onActiveFocusChanged: {
        if (!activeFocus) {
            releaseRemoteInput()
        }
    }

    onActiveChanged: {
        if (!active) {
            releaseRemoteInput()
        }
    }

    onVisibleChanged: {
        if (!visible) {
            releaseRemoteInput()
        }
    }

    Connections {
        target: root.Window.window
        function onActiveChanged() {
            if (target && !target.active) {
                root.releaseRemoteInput()
            }
        }
    }
    
    // The original-size mode uses a scrollable viewport. The other modes keep
    // the video output exactly viewport-sized and rely on VideoOutput scaling.
    Flickable {
        id: desktopViewport
        anchors.fill: parent
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        contentWidth: root.displayMode === root.originalSizeMode
                      ? Math.max(width, root.frameWidth) : width
        contentHeight: root.displayMode === root.originalSizeMode
                       ? Math.max(height, root.frameHeight) : height
        interactive: false

        VideoOutput {
            id: videoOutput
            width: root.displayMode === root.originalSizeMode
                   ? root.frameWidth : desktopViewport.width
            height: root.displayMode === root.originalSizeMode
                    ? root.frameHeight : desktopViewport.height
            x: root.displayMode === root.originalSizeMode
               ? Math.max(0, (desktopViewport.width - width) / 2) : 0
            y: root.displayMode === root.originalSizeMode
               ? Math.max(0, (desktopViewport.height - height) / 2) : 0
            fillMode: root.displayMode === root.stretchToFillMode
                      ? VideoOutput.Stretch : VideoOutput.PreserveAspectFit

            // Capture remote input inside the scrollable content so that the
            // viewport's scrollbars remain usable in original-size mode.
            MouseArea {
                id: mouseArea
                anchors.fill: parent
                enabled: root.inputEnabled && root.hasVideo
                hoverEnabled: true
                acceptedButtons: Qt.AllButtons
                cursorShape: (root.inputEnabled && root.hasVideo && frameProvider.hasCursor
                          && !root.suppressRemoteCursor
                          && !horizontalScrollBar.pressed
                          && !verticalScrollBar.pressed) ? Qt.BlankCursor : Qt.ArrowCursor

                property point lastPosition: Qt.point(0, 0)

                function rootPoint(mouse) {
                    return mouseArea.mapToItem(root, mouse.x, mouse.y)
                }

                onPositionChanged: function(mouse) {
                    if (!root.clientManager) return;
                    var point = rootPoint(mouse)
                    var remote = root.mapToRemote(point.x, point.y);
                    if (remote) {
                        root.clientManager.sendMouseMove(root.deviceId, remote.x, remote.y);
                        lastPosition = Qt.point(remote.x, remote.y);
                    }
                }

                onPressed: function(mouse) {
                    if (!root.clientManager) return;
                    root.forceActiveFocus();
                    var point = rootPoint(mouse)
                    var remote = root.mapToRemote(point.x, point.y);
                    if (remote) {
                        root.clientManager.sendMousePress(
                            root.deviceId, remote.x, remote.y, qtButtonToProtocol(mouse.button));
                    }
                }

                onReleased: function(mouse) {
                    if (!root.clientManager) return;
                    var point = rootPoint(mouse)
                    var remote = root.mapToRemote(point.x, point.y);
                    if (remote) {
                        root.clientManager.sendMouseRelease(
                            root.deviceId, remote.x, remote.y, qtButtonToProtocol(mouse.button));
                    }
                }

                onWheel: function(wheel) {
                    if (!root.clientManager) return;
                    var point = rootPoint(wheel)
                    var remote = root.mapToRemote(point.x, point.y);
                    if (remote) {
                        root.clientManager.sendMouseWheel(
                            root.deviceId, remote.x, remote.y,
                            wheel.angleDelta.x, wheel.angleDelta.y);
                    }
                }

                function qtButtonToProtocol(qtButton) {
                    switch (qtButton) {
                    case Qt.LeftButton: return 1;
                    case Qt.RightButton: return 2;
                    case Qt.MiddleButton: return 4;
                    case Qt.BackButton: return 8;
                    case Qt.ForwardButton: return 16;
                    default: return 0;
                    }
                }
            }
        }

        Controls.ScrollBar.horizontal: QDScrollBar {
            id: horizontalScrollBar
            z: 3
            policy: root.displayMode === root.originalSizeMode
                    ? Controls.ScrollBar.AlwaysOn : Controls.ScrollBar.AlwaysOff
            autoHide: false
            implicitHeight: 20
            padding: 4
            contentItem: Rectangle {
                implicitWidth: 100
                implicitHeight: 12
                radius: height / 2
                color: horizontalScrollBar.pressed ? Theme.primary
                       : horizontalScrollBar.hovered ? Qt.rgba(Theme.text.r, Theme.text.g, Theme.text.b, 0.6)
                       : Qt.rgba(Theme.text.r, Theme.text.g, Theme.text.b, 0.4)
            }
        }
        Controls.ScrollBar.vertical: QDScrollBar {
            id: verticalScrollBar
            z: 3
            policy: root.displayMode === root.originalSizeMode
                    ? Controls.ScrollBar.AlwaysOn : Controls.ScrollBar.AlwaysOff
            autoHide: false
            implicitWidth: 20
            padding: 4
            contentItem: Rectangle {
                implicitWidth: 12
                implicitHeight: 100
                radius: width / 2
                color: verticalScrollBar.pressed ? Theme.primary
                       : verticalScrollBar.hovered ? Qt.rgba(Theme.text.r, Theme.text.g, Theme.text.b, 0.6)
                       : Qt.rgba(Theme.text.r, Theme.text.g, Theme.text.b, 0.4)
            }
        }
    }

    // A full-desktop overview for navigating an oversized original-size view.
    Rectangle {
        id: miniMap
        readonly property real aspectRatio: root.frameWidth > 0 && root.frameHeight > 0
                                            ? root.frameWidth / root.frameHeight : 16 / 9
        readonly property real viewportWidthRatio: desktopViewport.width / desktopViewport.contentWidth
        readonly property real viewportHeightRatio: desktopViewport.height / desktopViewport.contentHeight
        width: Math.min(180, root.width * 0.22, root.height * 0.22 * aspectRatio)
        height: width / aspectRatio
        anchors.left: parent.left
        anchors.bottom: parent.bottom
        anchors.leftMargin: Theme.spacingMedium
        anchors.bottomMargin: Theme.spacingMedium
        visible: root.showMiniMap && root.displayMode === root.originalSizeMode && root.hasVideo
        z: 20
        color: "#d91a1a1a"
        border.color: "#99ffffff"
        border.width: 1
        radius: Theme.radiusSmall
        clip: true

        ShaderEffectSource {
            id: miniMapPreview
            anchors.fill: parent
            sourceItem: videoOutput
            live: true
            hideSource: false
        }

        Rectangle {
            id: viewportIndicator
            x: Math.max(0, Math.min(miniMap.width - width,
                                     desktopViewport.contentX / desktopViewport.contentWidth * miniMap.width))
            y: Math.max(0, Math.min(miniMap.height - height,
                                     desktopViewport.contentY / desktopViewport.contentHeight * miniMap.height))
            width: Math.min(miniMap.width, miniMap.viewportWidthRatio * miniMap.width)
            height: Math.min(miniMap.height, miniMap.viewportHeightRatio * miniMap.height)
            color: "#3373b8ff"
            border.color: "#e6ffffff"
            border.width: 1
        }

        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.OpenHandCursor
            acceptedButtons: Qt.LeftButton

            function moveViewport(mouse) {
                desktopViewport.contentX = Math.max(0, Math.min(
                    desktopViewport.contentWidth - desktopViewport.width,
                    mouse.x / miniMap.width * desktopViewport.contentWidth - desktopViewport.width / 2))
                desktopViewport.contentY = Math.max(0, Math.min(
                    desktopViewport.contentHeight - desktopViewport.height,
                    mouse.y / miniMap.height * desktopViewport.contentHeight - desktopViewport.height / 2))
            }

            onPressed: function(mouse) { moveViewport(mouse) }
            onPositionChanged: function(mouse) {
                if (pressed) moveViewport(mouse)
            }
        }
    }
    
    // Convert local mouse coordinates to remote desktop DIP coordinates.
    // Uses VideoOutput.contentRect to get the actual video display area,
    // ensuring perfect alignment with the rendered video region.
    // Outputs DIP coordinates that match what the host's InputInjector
    // expects (logical points on macOS, DIPs on Windows/Linux).
    function mapToRemote(localX, localY) {
        if (frameWidth <= 0 || frameHeight <= 0) {
            return null;
        }

        // Convert through the viewport because original-size mode can be
        // scrolled. contentRect remains local to VideoOutput.
        var videoPoint = root.mapToItem(videoOutput, localX, localY);
        var rect = videoOutput.contentRect;
        if (rect.width <= 0 || rect.height <= 0) {
            return null;
        }

        // Calculate position relative to the video content area
        var relativeX = videoPoint.x - rect.x;
        var relativeY = videoPoint.y - rect.y;

        // Check if the mouse is within the video area (not on black bars)
        if (relativeX < 0 || relativeX > rect.width ||
            relativeY < 0 || relativeY > rect.height) {
            return null;
        }

        // Use DIP dimensions from VideoLayout when available, otherwise
        // fall back to frame pixel dimensions (correct for non-HiDPI hosts).
        var targetWidth = remoteDipWidth > 0 ? remoteDipWidth : frameWidth;
        var targetHeight = remoteDipHeight > 0 ? remoteDipHeight : frameHeight;

        // Scale to remote desktop DIP coordinates
        var remoteX = Math.round(relativeX * targetWidth / rect.width);
        var remoteY = Math.round(relativeY * targetHeight / rect.height);

        // Clamp to valid range within this display
        remoteX = Math.max(0, Math.min(targetWidth - 1, remoteX));
        remoteY = Math.max(0, Math.min(targetHeight - 1, remoteY));

        // Add display offset for multi-monitor coordinate mapping
        remoteX += remoteOffsetX;
        remoteY += remoteOffsetY;

        return { x: remoteX, y: remoteY };
    }
    
    // Remote cursor display
    Image {
        id: remoteCursor
        visible: root.hasVideo && frameProvider.hasCursor && mouseArea.containsMouse
             && !root.suppressRemoteCursor
        source: frameProvider.hasCursor ? "image://cursor/" + root.deviceId + "/" + cursorVersion : ""
        
        // Track cursor version for image refresh
        property int cursorVersion: 0
        
        // Position follows mouse, offset by hotspot
        x: mouseArea.mapToItem(root, mouseArea.mouseX, mouseArea.mouseY).x
           - frameProvider.cursorHotspot.x
        y: mouseArea.mapToItem(root, mouseArea.mouseX, mouseArea.mouseY).y
           - frameProvider.cursorHotspot.y
        
        // Update when cursor changes
        Connections {
            target: frameProvider
            function onCursorChanged() {
                remoteCursor.cursorVersion++
            }
        }
    }
    
    // File drag-and-drop support
    DropArea {
        anchors.fill: parent
        keys: ["text/uri-list"]

        onEntered: function(drag) {
            if (drag.hasUrls) {
                drag.accepted = true
                dropOverlay.visible = true
            }
        }
        onExited: {
            dropOverlay.visible = false
        }
        onDropped: function(drop) {
            dropOverlay.visible = false
            if (drop.hasUrls && drop.urls.length > 0) {
                root.filesDropped(drop.urls)
            }
        }
    }

    Rectangle {
        id: dropOverlay
        anchors.fill: parent
        visible: false
        color: "#60000000"
        z: 50
        border.width: 3
        border.color: Theme.primary
        radius: 8

        Column {
            anchors.centerIn: parent
            spacing: 12
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: FluentIconGlyph.uploadGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.primary
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("Drop files here to upload")
                font.pixelSize: 16
                font.weight: Font.Medium
                color: "#ffffff"
            }
        }
    }

    // Keyboard event handling — passes nativeScanCode directly.
    // The C++ client converts it to USB HID keycode via Chromium's
    // KeycodeConverter::NativeKeycodeToUsbKeycode().
    Keys.onPressed: function(event) {
        if (!root.inputEnabled || !root.clientManager) return;

        // File paste is local queue submission, not a remote V key. Consume its
        // repeats and matching release even if the clipboard changes mid-press.
        if (event.key === Qt.Key_V && root.filePasteKeyHeld) {
            event.accepted = true
            return
        }
        if (!event.isAutoRepeat && event.key === Qt.Key_V
                && (event.modifiers & Qt.ControlModifier)) {
            if (root.clientManager.pasteFilesFromClipboard(root.deviceId)) {
                root.filePasteKeyHeld = true
                event.accepted = true
                return
            }
        }

        var nativeKey = KeyboardStateTracker.getLastNativeKeycode()
        if (event.isAutoRepeat && !root.forwardedKeys[nativeKey]) {
            event.accepted = true
            return
        }
        root.forwardedKeys[nativeKey] = true
        root.clientManager.sendKeyPress(
            root.deviceId, nativeKey, KeyboardStateTracker.getLockStates());
        event.accepted = true;
    }

    Keys.onReleased: function(event) {
        if (event.key === Qt.Key_V && root.filePasteKeyHeld) {
            if (!event.isAutoRepeat) root.filePasteKeyHeld = false
            event.accepted = true
            return
        }
        if (!root.inputEnabled || !root.clientManager) return;

        // A focus loss already released the host's authoritative input state.
        // Never synthesize key-up for an intercepted or no-longer-held key.
        var nativeKey = KeyboardStateTracker.getLastNativeKeycode()
        if (root.forwardedKeys[nativeKey]) {
            if (!event.isAutoRepeat) delete root.forwardedKeys[nativeKey]
            root.clientManager.sendKeyRelease(
                root.deviceId, nativeKey, KeyboardStateTracker.getLockStates());
        }
        event.accepted = true;
    }

    // Frame provider connects shared memory to video sink
    VideoFrameProvider {
        id: frameProvider
        videoSink: videoOutput.videoSink
        deviceId: root.deviceId
        sharedMemoryManager: root.clientManager ? root.clientManager.sharedMemoryManager : null
        active: root.active && root.visible
    }
    
    // Connect to videoFrameReady signal from ClientManager
    Connections {
        target: root.clientManager
        
        function onVideoFrameReady(deviceId, frameIndex) {
            if (deviceId === root.deviceId) {
                frameProvider.onVideoFrameReady(frameIndex)
            }
        }
        
        function onCursorShapeChanged(deviceId, width, height, hotspotX, hotspotY, data) {
            if (deviceId === root.deviceId) {
                frameProvider.onCursorShapeChanged(width, height, hotspotX, hotspotY, data)
            }
        }

        function onVideoLayoutChanged(deviceId, widthDips, heightDips) {
            if (deviceId === root.deviceId) {
                root.remoteDipWidth = widthDips
                root.remoteDipHeight = heightDips
            }
        }

        function onDisplayListChanged(deviceId, displays, activeDisplayIndex) {
            if (deviceId === root.deviceId && activeDisplayIndex >= 0 &&
                activeDisplayIndex < displays.length) {
                var d = displays[activeDisplayIndex]
                root.remoteDipWidth = d.width || 0
                root.remoteDipHeight = d.height || 0
                root.remoteOffsetX = d.positionX || 0
                root.remoteOffsetY = d.positionY || 0
            }
        }
    }
    
    // Loading indicator when no video
    Column {
        anchors.centerIn: parent
        spacing: 16
        visible: !root.hasVideo && root.active
        
        QDSpinner {
            anchors.horizontalCenter: parent.horizontalCenter
            size: 48
            running: visible
        }
        
        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: qsTr("Waiting for video...")
            color: "#888888"
            font.pixelSize: 14
        }
    }
    
    // Frame rate overlay (optional, for debugging)
    Rectangle {
        anchors.top: parent.top
        anchors.right: parent.right
        anchors.margins: 8
        width: fpsText.width + 12
        height: fpsText.height + 8
        radius: 4
        color: "#80000000"
        visible: root.hasVideo && false  // Set to true to show FPS
        
        Text {
            id: fpsText
            anchors.centerIn: parent
            text: root.frameRate + " FPS"
            color: root.frameRate >= 30 ? "#00ff00" : 
                   root.frameRate >= 15 ? "#ffff00" : "#ff0000"
            font.pixelSize: 12
            font.family: "Consolas"
        }
    }
}
