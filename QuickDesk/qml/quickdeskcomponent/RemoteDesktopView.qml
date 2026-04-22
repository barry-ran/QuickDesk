// Copyright 2026 QuickDesk Authors
// Remote desktop video display component with input event support

import QtQuick
import QtMultimedia
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
    property alias fillMode: videoOutput.fillMode
    
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
    
    // Video output for GPU rendering
    VideoOutput {
        id: videoOutput
        anchors.fill: parent
        fillMode: VideoOutput.PreserveAspectFit
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

        // VideoOutput.contentRect gives the exact rectangle where the video
        // is actually rendered (excludes black bars from PreserveAspectFit)
        var rect = videoOutput.contentRect;
        if (rect.width <= 0 || rect.height <= 0) {
            return null;
        }

        // Calculate position relative to the video content area
        var relativeX = localX - rect.x;
        var relativeY = localY - rect.y;

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
        source: frameProvider.hasCursor ? "image://cursor/" + root.deviceId + "/" + cursorVersion : ""
        
        // Track cursor version for image refresh
        property int cursorVersion: 0
        
        // Position follows mouse, offset by hotspot
        x: mouseArea.mouseX - frameProvider.cursorHotspot.x
        y: mouseArea.mouseY - frameProvider.cursorHotspot.y
        
        // Update when cursor changes
        Connections {
            target: frameProvider
            function onCursorChanged() {
                remoteCursor.cursorVersion++
            }
        }
    }
    
    // Mouse event capture
    MouseArea {
        id: mouseArea
        anchors.fill: parent
        enabled: root.inputEnabled && root.hasVideo
        hoverEnabled: true
        acceptedButtons: Qt.AllButtons
        cursorShape: (root.inputEnabled && root.hasVideo && frameProvider.hasCursor) ? Qt.BlankCursor : Qt.ArrowCursor
        
        property point lastPosition: Qt.point(0, 0)
        
        onPositionChanged: function(mouse) {
            if (!root.clientManager) return;
            var remote = root.mapToRemote(mouse.x, mouse.y);
            if (remote) {
                root.clientManager.sendMouseMove(root.deviceId, remote.x, remote.y);
                lastPosition = Qt.point(remote.x, remote.y);
            }
        }
        
        onPressed: function(mouse) {
            if (!root.clientManager) return;
            root.forceActiveFocus();  // Take keyboard focus
            var remote = root.mapToRemote(mouse.x, mouse.y);
            if (remote) {
                var button = qtButtonToProtocol(mouse.button);
                root.clientManager.sendMousePress(root.deviceId, remote.x, remote.y, button);
            }
        }
        
        onReleased: function(mouse) {
            if (!root.clientManager) return;
            var remote = root.mapToRemote(mouse.x, mouse.y);
            if (remote) {
                var button = qtButtonToProtocol(mouse.button);
                root.clientManager.sendMouseRelease(root.deviceId, remote.x, remote.y, button);
            }
        }
        
        onWheel: function(wheel) {
            if (!root.clientManager) return;
            var remote = root.mapToRemote(wheel.x, wheel.y);
            if (remote) {
                root.clientManager.sendMouseWheel(
                    root.deviceId, remote.x, remote.y,
                    wheel.angleDelta.x, wheel.angleDelta.y);
            }
        }
        
        onDoubleClicked: function(mouse) {
            // Double click is handled as two rapid press/release events
            // by the individual press/release handlers
        }
        
        // Convert Qt button to protocol button value
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

        // Intercept Ctrl+V: if clipboard contains files, upload them instead
        if (event.key === Qt.Key_V && (event.modifiers & Qt.ControlModifier)) {
            if (root.clientManager.pasteFilesFromClipboard(root.deviceId)) {
                event.accepted = true
                return
            }
        }

        root.clientManager.sendKeyPress(
            root.deviceId, KeyboardStateTracker.getLastNativeKeycode(),
            KeyboardStateTracker.getLockStates());
        event.accepted = true;
    }

    Keys.onReleased: function(event) {
        if (!root.inputEnabled || !root.clientManager) return;

        root.clientManager.sendKeyRelease(
            root.deviceId, KeyboardStateTracker.getLastNativeKeycode(),
            KeyboardStateTracker.getLockStates());
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
