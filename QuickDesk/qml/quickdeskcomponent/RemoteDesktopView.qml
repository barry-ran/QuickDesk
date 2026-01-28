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
 *       connectionId: "conn_1"
 *       clientManager: mainController.clientManager
 *       active: visible  // Only render when visible
 *       inputEnabled: true // Enable mouse/keyboard input
 *   }
 */
Rectangle {
    id: root
    
    // Required properties
    required property string connectionId
    required property ClientManager clientManager
    
    // Optional properties
    property bool active: true
    property bool inputEnabled: true  // Enable/disable input capture
    property alias fillMode: videoOutput.fillMode
    
    // Read-only properties
    readonly property int frameWidth: frameProvider.frameSize.width
    readonly property int frameHeight: frameProvider.frameSize.height
    readonly property int frameRate: frameProvider.frameRate
    readonly property bool hasVideo: frameWidth > 0 && frameHeight > 0
    
    color: "#1a1a1a"  // Dark background
    focus: inputEnabled  // Enable keyboard focus when input is enabled
    
    // Video output for GPU rendering
    VideoOutput {
        id: videoOutput
        anchors.fill: parent
        fillMode: VideoOutput.PreserveAspectFit
    }
    
    // Convert local mouse coordinates to remote desktop coordinates
    function mapToRemote(localX, localY) {
        if (frameWidth <= 0 || frameHeight <= 0) {
            return { x: 0, y: 0 };
        }
        
        // Calculate the actual video display area (considering aspect ratio)
        var viewAspect = width / height;
        var frameAspect = frameWidth / frameHeight;
        
        var displayWidth, displayHeight, offsetX, offsetY;
        
        if (viewAspect > frameAspect) {
            // View is wider than video, black bars on sides
            displayHeight = height;
            displayWidth = height * frameAspect;
            offsetX = (width - displayWidth) / 2;
            offsetY = 0;
        } else {
            // View is taller than video, black bars on top/bottom
            displayWidth = width;
            displayHeight = width / frameAspect;
            offsetX = 0;
            offsetY = (height - displayHeight) / 2;
        }
        
        // Adjust for offset and scale
        var relativeX = localX - offsetX;
        var relativeY = localY - offsetY;
        
        // Check if click is within video area
        if (relativeX < 0 || relativeX > displayWidth || 
            relativeY < 0 || relativeY > displayHeight) {
            return null;  // Outside video area
        }
        
        // Scale to remote coordinates
        var remoteX = Math.round(relativeX * frameWidth / displayWidth);
        var remoteY = Math.round(relativeY * frameHeight / displayHeight);
        
        // Clamp to valid range
        remoteX = Math.max(0, Math.min(frameWidth - 1, remoteX));
        remoteY = Math.max(0, Math.min(frameHeight - 1, remoteY));
        
        return { x: remoteX, y: remoteY };
    }
    
    // Remote cursor display
    Image {
        id: remoteCursor
        visible: root.hasVideo && frameProvider.hasCursor && mouseArea.containsMouse
        source: frameProvider.hasCursor ? "image://cursor/" + root.connectionId + "/" + cursorVersion : ""
        
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
        cursorShape: (root.hasVideo && frameProvider.hasCursor) ? Qt.BlankCursor : Qt.ArrowCursor
        
        property point lastPosition: Qt.point(0, 0)
        
        onPositionChanged: function(mouse) {
            if (!root.clientManager) return;
            var remote = root.mapToRemote(mouse.x, mouse.y);
            if (remote) {
                root.clientManager.sendMouseMove(root.connectionId, remote.x, remote.y);
                lastPosition = Qt.point(remote.x, remote.y);
            }
        }
        
        onPressed: function(mouse) {
            if (!root.clientManager) return;
            root.forceActiveFocus();  // Take keyboard focus
            var remote = root.mapToRemote(mouse.x, mouse.y);
            if (remote) {
                var button = qtButtonToProtocol(mouse.button);
                root.clientManager.sendMousePress(root.connectionId, remote.x, remote.y, button);
            }
        }
        
        onReleased: function(mouse) {
            if (!root.clientManager) return;
            var remote = root.mapToRemote(mouse.x, mouse.y);
            if (remote) {
                var button = qtButtonToProtocol(mouse.button);
                root.clientManager.sendMouseRelease(root.connectionId, remote.x, remote.y, button);
            }
        }
        
        onWheel: function(wheel) {
            if (!root.clientManager) return;
            var remote = root.mapToRemote(wheel.x, wheel.y);
            if (remote) {
                // Qt provides angleDelta in 1/8 of a degree
                var delta = wheel.angleDelta.y;
                root.clientManager.sendMouseWheel(root.connectionId, remote.x, remote.y, delta);
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
    
    // Keyboard event handling
    Keys.onPressed: function(event) {
        if (!root.inputEnabled || !root.clientManager) return;
        
        var usbKeycode = KeycodeMapper.qtKeyToUsb(event.key, event.modifiers);
        if (usbKeycode > 0) {
            root.clientManager.sendKeyPress(root.connectionId, usbKeycode, event.modifiers);
            event.accepted = true;
        }
    }
    
    Keys.onReleased: function(event) {
        if (!root.inputEnabled || !root.clientManager) return;
        
        var usbKeycode = KeycodeMapper.qtKeyToUsb(event.key, event.modifiers);
        if (usbKeycode > 0) {
            root.clientManager.sendKeyRelease(root.connectionId, usbKeycode, event.modifiers);
            event.accepted = true;
        }
    }
    
    // Frame provider connects shared memory to video sink
    VideoFrameProvider {
        id: frameProvider
        videoSink: videoOutput.videoSink
        connectionId: root.connectionId
        sharedMemoryManager: root.clientManager ? root.clientManager.sharedMemoryManager : null
        active: root.active && root.visible
    }
    
    // Connect to videoFrameReady signal from ClientManager
    Connections {
        target: root.clientManager
        
        function onVideoFrameReady(connId, frameIndex) {
            if (connId === root.connectionId) {
                frameProvider.onVideoFrameReady(frameIndex)
            }
        }
        
        function onCursorShapeChanged(connId, width, height, hotspotX, hotspotY, data) {
            if (connId === root.connectionId) {
                frameProvider.onCursorShapeChanged(width, height, hotspotX, hotspotY, data)
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
