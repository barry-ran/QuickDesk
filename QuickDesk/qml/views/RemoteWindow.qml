// Remote Desktop Window - Independent window for remote desktop connections
import QtQuick
import QtQuick.Window
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Dialogs
import QuickDesk 1.0

import "../component"
import "../quickdeskcomponent"

Window {
    id: remoteWindow
    width: 1280
    height: 720
    visible: true
    title: qsTr("QuickDesk - Remote Desktop")
    
    // Properties
    property var clientManager: null
    property string localDeviceId: ""  // Local device ID — used to detect self-connection
    property alias connectionModel: connectionModelObj  // C++ model for incremental updates
    property int currentTabIndex: 0
    property bool hasAutoResized: false  // Only auto-resize once on first frame
    property bool showVideoStats: false  // Toggle video stats overlay
    property var closingConnections: ({})  // Guard against re-entrant closeConnection calls
    property bool emergencyStopActive: false

    // Multi-monitor: display list per device. Map: deviceId -> { displays: [], activeIndex: 0 }
    property var displayListMap: ({})
    property int displayListVersion: 0  // Increment to notify changes

    // Virtual displays per device. Map: deviceId -> [{index, width, height, refreshRate}]
    property var virtualDisplayMap: ({})
    property int virtualDisplayVersion: 0
    
    // C++ ConnectionListModel — only affected delegates are created/destroyed
    ConnectionListModel {
        id: connectionModelObj
    }
    
    // Performance stats stored separately to avoid triggering model rebuild
    // Map: deviceId -> { frameWidth, frameHeight, frameRate, ping,
    //   originalWidth, originalHeight, captureMs, encodeMs, networkDelayMs,
    //   decodeMs, paintMs, totalLatencyMs, inputRoundtripMs, bandwidthKbps,
    //   packetRate, codec, frameQuality, encodedRectWidth, encodedRectHeight }
    property var performanceStatsMap: ({})
    property int statsVersion: 0  // Increment to notify changes
    
    // Get performance stats for a connection
    function getPerformanceStats(deviceId) {
        return performanceStatsMap[deviceId] || {
            frameWidth: 0, frameHeight: 0, frameRate: 0, ping: 0,
            originalWidth: 0, originalHeight: 0,
            captureMs: 0, encodeMs: 0, networkDelayMs: 0, decodeMs: 0, paintMs: 0,
            totalLatencyMs: 0, inputRoundtripMs: 0,
            bandwidthKbps: 0, packetRate: 0,
            codec: "", frameQuality: -1,
            encodedRectWidth: 0, encodedRectHeight: 0
        }
    }
    
    // Update performance stats without modifying connections model
    function updatePerformanceStats(deviceId, width, height, fps, ping) {
        var stats = performanceStatsMap[deviceId]
        
        // Handle video size update
        if (width !== undefined && height !== undefined && width > 0 && height > 0) {
            // Ensure fps is non-negative and round to integer for comparison
            if (fps !== undefined) {
                fps = Math.max(0, Math.round(fps))
            }
            
            // Check if there's any actual change
            if (stats && stats.frameWidth === width && stats.frameHeight === height && 
                (fps === undefined || stats.frameRate === fps)) {
                // No video change, but might need to update ping
                if (ping === undefined) {
                    return  // Nothing to update
                }
            }
            
            // Record original resolution on first valid frame
            var originalWidth = stats ? stats.originalWidth : 0
            var originalHeight = stats ? stats.originalHeight : 0
            
            if (!stats || (stats.originalWidth === 0 && width > 0 && height > 0)) {
                originalWidth = width
                originalHeight = height
                console.log("✓ Recorded original resolution for", deviceId, ":", width + "x" + height)
            }
            
            // Merge into existing stats to preserve route data etc.
            var newStatsMap = Object.assign({}, performanceStatsMap)
            newStatsMap[deviceId] = Object.assign({}, stats || {}, {
                frameWidth: width,
                frameHeight: height,
                frameRate: fps !== undefined ? fps : (stats ? stats.frameRate : 0),
                ping: ping !== undefined ? ping : (stats ? stats.ping : 0),
                originalWidth: originalWidth,
                originalHeight: originalHeight
            })
            performanceStatsMap = newStatsMap
            
            // Only increment version if width or height changed (affects layout)
            if (!stats || stats.frameWidth !== width || stats.frameHeight !== height) {
                statsVersion++
            }
        } 
        // Handle ping-only update
        else if (ping !== undefined && stats) {
            var newStatsMap = Object.assign({}, performanceStatsMap)
            newStatsMap[deviceId] = Object.assign({}, stats, {ping: ping})
            performanceStatsMap = newStatsMap
        }
    }
    
    // Add connection to this window
    function addConnection(deviceId) {
        var existingIdx = connectionModel.indexOf(deviceId)
        if (existingIdx >= 0) {
            console.log("Connection already exists in window:", deviceId)
            currentTabIndex = existingIdx
            return
        }
        
        connectionModel.addConnection(deviceId)
        
        // Initialize performance stats
        var newStatsMap = Object.assign({}, performanceStatsMap)
        newStatsMap[deviceId] = {
            frameWidth: 0, frameHeight: 0, frameRate: 0, ping: 0,
            originalWidth: 0, originalHeight: 0,
            captureMs: 0, encodeMs: 0, networkDelayMs: 0, decodeMs: 0, paintMs: 0,
            totalLatencyMs: 0, inputRoundtripMs: 0,
            bandwidthKbps: 0, packetRate: 0,
            codec: "", frameQuality: -1,
            encodedRectWidth: 0, encodedRectHeight: 0
        }
        performanceStatsMap = newStatsMap
        
        currentTabIndex = connectionModel.count - 1
        console.log("Added connection to remote window:", deviceId, "Total tabs:", connectionModel.count)
    }
    
    // Close connection and remove tab (unified function for both scenarios)
    // needDisconnect=true when user initiates close; false when reacting to an already-disconnected signal
    function closeConnection(index, needDisconnect) {
        if (needDisconnect === undefined) needDisconnect = true

        if (index < 0 || index >= connectionModel.count) {
            return
        }
        
        var tabDeviceId = connectionModel.deviceIdAt(index)
        
        // Prevent re-entrant calls (disconnectFromHost may trigger onConnectionStateChanged synchronously)
        if (closingConnections[tabDeviceId]) {
            return
        }
        closingConnections[tabDeviceId] = true
        
        console.log("Closing connection:", tabDeviceId, "at index:", index, "needDisconnect:", needDisconnect)
        
        // 1. Remove the tab first (before disconnect to avoid re-entrant state change handler)
        removeConnection(index)
        
        // 2. Only call disconnectFromHost when the caller is initiating the disconnect
        //    (skip if we're reacting to a connectionStateChanged/connectionRemoved signal)
        if (needDisconnect && clientManager) {
            clientManager.disconnectFromHost(tabDeviceId)
        }
        
        delete closingConnections[tabDeviceId]
    }
    
    // Remove connection from this window (internal helper)
    function removeConnection(index) {
        if (index < 0 || index >= connectionModel.count) return
        
        var tabDeviceId = connectionModel.deviceIdAt(index)
        
        // Remove from performance stats map
        var newStatsMap = Object.assign({}, performanceStatsMap)
        delete newStatsMap[tabDeviceId]
        performanceStatsMap = newStatsMap
        
        // Remove from model — only destroys this one delegate
        connectionModel.removeConnection(index)
        
        // Update current tab index
        if (currentTabIndex >= connectionModel.count) {
            currentTabIndex = Math.max(0, connectionModel.count - 1)
        }
        
        // Close window if no connections left
        if (connectionModel.count === 0) {
            remoteWindow.close()
        }
        
        console.log("Removed connection from remote window:", tabDeviceId, "Remaining tabs:", connectionModel.count)
    }
    
    // Clean up all connections when window closes
    onClosing: function(close) {
        console.log("RemoteWindow closing, disconnecting all connections")
        for (var i = 0; i < connectionModel.count; i++) {
            if (clientManager) {
                console.log("Disconnecting:", connectionModel.deviceIdAt(i))
                clientManager.disconnectFromHost(connectionModel.deviceIdAt(i))
            }
        }
        connectionModel.clear()
    }
    
    // Core resize logic — resize window to best fit the given remote desktop resolution
    // Can be called manually at any time (e.g. from toolbar "Fit Window" button)
    function resizeToFit(fw, fh) {
        if (fw <= 0 || fh <= 0) return

        var scr = remoteWindow.screen
        if (!scr) {
            console.warn("resizeToFit: screen not available")
            return false
        }

        // Available screen space (leave margin for taskbar, etc.)
        var maxWidth = scr.desktopAvailableWidth * 0.92
        var maxHeight = scr.desktopAvailableHeight * 0.92

        // Account for tab bar height
        var tabBarH = tabBar.height > 0 ? tabBar.height : 36
        var contentMaxHeight = maxHeight - tabBarH

        // Calculate scale factor (never upscale)
        var scale = Math.min(maxWidth / fw, contentMaxHeight / fh, 1.0)

        var newWidth  = Math.round(fw * scale)
        var newHeight = Math.round(fh * scale) + tabBarH

        // Center on screen
        remoteWindow.width  = newWidth
        remoteWindow.height = newHeight
        remoteWindow.x = Math.round((scr.width  - newWidth)  / 2) + scr.virtualX
        remoteWindow.y = Math.round((scr.height - newHeight) / 2) + scr.virtualY

        console.log("Resized window to", newWidth + "x" + newHeight,
                     "for remote desktop", fw + "x" + fh,
                     "(scale:", scale.toFixed(3) + ")",
                     "screen:", scr.width + "x" + scr.height,
                     "available:", scr.desktopAvailableWidth + "x" + scr.desktopAvailableHeight)
        return true
    }

    // Auto-resize window to best fit the remote desktop resolution (called once on first frame)
    // Triggered from onStatsVersionChanged (at window level, not inside Repeater delegate)
    function autoResizeToFit(fw, fh) {
        console.log("autoResizeToFit called:", fw + "x" + fh,
                     "hasAutoResized:", hasAutoResized,
                     "screen:", remoteWindow.screen ? "valid" : "null")

        if (fw <= 0 || fh <= 0) return
        if (hasAutoResized) return

        if (!remoteWindow.screen) {
            // Screen not ready — retry via Timer
            console.log("autoResizeToFit: screen not ready, scheduling retry")
            retryResizeTimer.pendingWidth = fw
            retryResizeTimer.pendingHeight = fh
            retryResizeTimer.retryCount = 0
            retryResizeTimer.start()
            return
        }

        // Mark AFTER screen check so retries work
        hasAutoResized = true
        resizeToFit(fw, fh)
    }

    // When frame dimensions change (statsVersion incremented), try auto-resize
    // Using Qt.callLater ensures execution on a clean call stack,
    // avoiding issues with nested signal handler / delegate destruction races.
    onStatsVersionChanged: {
        if (hasAutoResized) return
        if (connectionModel.count === 0) return

        var tabDeviceId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                     ? connectionModel.deviceIdAt(currentTabIndex) : ""
        if (!tabDeviceId) return

        var s = getPerformanceStats(tabDeviceId)
        if (s && s.frameWidth > 0 && s.frameHeight > 0) {
            var w = s.frameWidth
            var h = s.frameHeight
            Qt.callLater(function() {
                autoResizeToFit(w, h)
            })
        }
    }

    // Update connection state
    function updateConnectionState(deviceId, state, ping) {
        // Update state in model (only emits dataChanged for the affected row)
        if (state !== "") {
            connectionModel.updateState(deviceId, state)
        }
        
        // Update ping in performance stats map (doesn't trigger model rebuild)
        if (ping !== undefined) {
            updatePerformanceStats(deviceId, undefined, undefined, undefined, ping)
        }
    }
    
    ColumnLayout {
        anchors.fill: parent
        spacing: 0
        
        // Tab Bar
        RemoteTabBar {
            id: tabBar
            Layout.fillWidth: true
            connectionModel: remoteWindow.connectionModel
            currentIndex: remoteWindow.currentTabIndex
            performanceStatsMap: remoteWindow.performanceStatsMap
            statsVersion: remoteWindow.statsVersion
            
            onTabClicked: function(index) {
                remoteWindow.currentTabIndex = index
            }
            
            onTabCloseRequested: function(index) {
                remoteWindow.closeConnection(index)
            }
            
            onNewTabRequested: {
                // TODO: Show quick connect dialog
                console.log("New tab requested")
            }
        }
        
        // Remote Desktop View Stack
        StackLayout {
            id: desktopStack
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: remoteWindow.currentTabIndex
            
            Repeater {
                model: connectionModel
                
                Item {
                    id: delegateItem
                    required property int index
                    required property string deviceId
                    required property string name
                    required property string state
                    
                    // Detect self-connection: remote deviceId matches local deviceId
                    readonly property bool isSelfConnection: remoteWindow.localDeviceId !== "" && delegateItem.deviceId === remoteWindow.localDeviceId
                    
                    // Remote desktop video view (ONLY video, no overlay UI)
                    RemoteDesktopView {
                        id: desktopView
                        anchors.fill: parent
                        deviceId: delegateItem.deviceId
                        clientManager: remoteWindow.clientManager
                        active: delegateItem.index === remoteWindow.currentTabIndex
                        inputEnabled: !delegateItem.isSelfConnection  // Disable input for self-connection
                        
                        onFilesDropped: function(urls) {
                            var devId = delegateItem.deviceId
                            if (!devId || !remoteWindow.clientManager) return
                            for (var i = 0; i < urls.length; i++) {
                                remoteWindow.clientManager.startFileUpload(devId, urls[i])
                                var fname = urls[i].toString().split('/').pop()
                                transferModel.append({
                                    transferId: "",
                                    deviceId: devId,
                                    filename: decodeURIComponent(fname),
                                    progress: 0,
                                    status: "uploading",
                                    errorMessage: "",
                                    direction: "upload",
                                    savePath: ""
                                })
                            }
                            fileTransferDrawer.open()
                        }

                        // Monitor video size changes (frameRate and ping updated from PerformanceTracker)
                        onFrameWidthChanged: {
                            if (frameWidth > 0 && frameHeight > 0) {
                                var devId = delegateItem.deviceId
                                var stats = remoteWindow.getPerformanceStats(devId)
                                remoteWindow.updatePerformanceStats(devId, frameWidth, frameHeight, stats.frameRate, stats.ping)
                            }
                        }
                        onFrameHeightChanged: {
                            if (frameWidth > 0 && frameHeight > 0) {
                                var devId = delegateItem.deviceId
                                var stats = remoteWindow.getPerformanceStats(devId)
                                remoteWindow.updatePerformanceStats(devId, frameWidth, frameHeight, stats.frameRate, stats.ping)
                            }
                        }
                    }
                }
            }
        }
    }
        
    Item {
        anchors.fill: parent
        anchors.topMargin: tabBar.height  // Offset by tab bar height        
        
        // Single floating button bound to current active connection
        FloatingToolButton {
            x: parent.width - width - Theme.spacingXLarge
            y: Theme.spacingXLarge
            z: 1000
            visible: connectionModel.count > 0
            
            deviceId: currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                ? connectionModel.deviceIdAt(currentTabIndex) 
                : ""
            clientManager: remoteWindow.clientManager
            supportsSendAttentionSequence: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var stats = devId ? remoteWindow.getPerformanceStats(devId) : null
                return stats ? (stats.supportsSendAttentionSequence || false) : false
            }
            supportsLockWorkstation: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var stats = devId ? remoteWindow.getPerformanceStats(devId) : null
                return stats ? (stats.supportsLockWorkstation || false) : false
            }
            supportsFileTransfer: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var stats = devId ? remoteWindow.getPerformanceStats(devId) : null
                return stats ? (stats.supportsFileTransfer || false) : false
            }
            supportsPrivacyScreen: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var stats = devId ? remoteWindow.getPerformanceStats(devId) : null
                return stats ? (stats.supportsPrivacyScreen || false) : false
            }
            supportsVirtualDisplay: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var stats = devId ? remoteWindow.getPerformanceStats(devId) : null
                return stats ? (stats.supportsVirtualDisplay || false) : false
            }
            virtualDisplays: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var _ = remoteWindow.virtualDisplayVersion
                return devId ? (remoteWindow.virtualDisplayMap[devId] || []) : []
            }
            emergencyStopActive: remoteWindow.emergencyStopActive
            displayList: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var _ = remoteWindow.displayListVersion  // force re-evaluate on change
                var info = devId ? remoteWindow.displayListMap[devId] : null
                return info ? info.displays : []
            }
            activeDisplayIndex: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                var _ = remoteWindow.displayListVersion
                var info = devId ? remoteWindow.displayListMap[devId] : null
                return info ? info.activeIndex : -1
            }
            videoInfo: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count 
                    ? connectionModel.deviceIdAt(currentTabIndex) 
                    : ""
                return devId ? remoteWindow.getPerformanceStats(devId) : null
            }
            desktopView: {
                // Find the current desktop view
                if (remoteWindow.currentTabIndex >= 0) {
                    var stackItem = desktopStack.children[remoteWindow.currentTabIndex]
                    return stackItem ? stackItem.children[0] : null
                }
                return null
            }
            
            onDisconnectRequested: function(deviceId) {
                console.log("FloatingToolButton disconnect requested for:", deviceId)
                var idx = connectionModel.indexOf(deviceId)
                if (idx >= 0) {
                    remoteWindow.closeConnection(idx)
                }
            }

            onEmergencyStopRequested: {
                if (remoteWindow.emergencyStopActive) {
                    mainController.deactivateEmergencyStop()
                    remoteWindow.emergencyStopActive = false
                    toast.show(qsTr("Emergency stop deactivated"))
                } else {
                    mainController.activateEmergencyStop("user_manual")
                    remoteWindow.emergencyStopActive = true
                    toast.show(qsTr("Emergency stop activated - all AI operations halted"))
                }
            }
            
            onFitToRemoteDesktopRequested: {
                // Get current tab's frame dimensions and resize window
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                if (!devId) return

                var s = remoteWindow.getPerformanceStats(devId)
                if (s && s.frameWidth > 0 && s.frameHeight > 0) {
                    console.log("Manual fit window to remote desktop:", s.frameWidth + "x" + s.frameHeight)
                    remoteWindow.resizeToFit(s.frameWidth, s.frameHeight)
                }
            }
            
            onToggleVideoStats: {
                remoteWindow.showVideoStats = !remoteWindow.showVideoStats
            }
            
            onShowToast: function(message, toastType) {
                toast.show(message, toastType)
            }

            activeTransferCount: remoteWindow.activeTransferCount

            onUploadFileRequested: {
                fileDialog.open()
            }

            onDownloadFileRequested: {
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                if (devId && remoteWindow.clientManager) {
                    remoteWindow.clientManager.startFileDownload(devId)
                }
            }

            onShowTransferPanelRequested: {
                fileTransferDrawer.open()
            }
        }
        
        // Video Stats Overlay — semi-transparent panel with detailed stats
        VideoStatsOverlay {
            id: videoStatsOverlay
            anchors.left: parent.left
            anchors.top: parent.top
            anchors.margins: Theme.spacingMedium
            z: 999
            visible: remoteWindow.showVideoStats && connectionModel.count > 0
            
            stats: {
                // Force re-evaluation when statsVersion changes
                var _version = remoteWindow.statsVersion
                var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                    ? connectionModel.deviceIdAt(currentTabIndex) : ""
                return devId ? remoteWindow.getPerformanceStats(devId) : null
            }
        }
    }
    
    // Monitor connection state changes
    Connections {
        target: remoteWindow.clientManager
        
        function onConnectionStateChanged(deviceId, state, hostInfo) {
            console.log("Remote window: connection state changed:", deviceId, state)

            if (!remoteWindow || !remoteWindow.closingConnections) return

            // Skip if this connection is already being closed
            if (remoteWindow.closingConnections[deviceId]) {
                return
            }
            
            // Update connection state
            remoteWindow.updateConnectionState(deviceId, state, 0)
            
            // Auto-close tab when connection is disconnected or failed
            // needDisconnect=false: the connection is already gone, just remove the tab
            if (state === "disconnected" || state === "failed") {
                var deviceIdCopy = deviceId
                Qt.callLater(function() {
                    var idx = connectionModel.indexOf(deviceIdCopy)
                    if (idx >= 0) {
                        console.log("Auto-closing tab for", state, "connection:", deviceIdCopy, "at index:", idx)
                        remoteWindow.closeConnection(idx, false)
                    }
                })
            }
        }
    }
    
    // Fallback: clean up tab when connection is removed from ClientManager
    // This catches cases where connectionStateChanged might not fire (e.g. disconnectAll, process crash)
    Connections {
        target: remoteWindow.clientManager
        
        function onConnectionRemoved(deviceId) {
            if (!remoteWindow || !remoteWindow.closingConnections) return
            if (remoteWindow.closingConnections[deviceId]) {
                return
            }
            
            var deviceIdCopy = deviceId
            Qt.callLater(function() {
                var idx = connectionModel.indexOf(deviceIdCopy)
                if (idx >= 0) {
                    console.log("Fallback: removing orphan tab for removed connection:", deviceIdCopy)
                    remoteWindow.closeConnection(idx, false)
                }
            })
        }
    }
    
    // Monitor performance stats updates (detailed stats from C++ PerformanceTracker)
    Connections {
        target: remoteWindow.clientManager
        
        function onPerformanceStatsUpdated(deviceId, detailedStats) {
            var totalLatencyMs = detailedStats.totalLatencyMs || 0
            var frameRate = detailedStats.frameRate || 0
            
            // Update connection latency value (for tab bar display)
            remoteWindow.updateConnectionState(deviceId, "", totalLatencyMs)
            
            // Update frameRate and merge detailed stats
            var existing = remoteWindow.getPerformanceStats(deviceId)
            if (existing && existing.frameWidth > 0 && existing.frameHeight > 0) {
                remoteWindow.updatePerformanceStats(deviceId,
                    existing.frameWidth, existing.frameHeight, frameRate, totalLatencyMs)
            }
            
            // Merge detailed timing/codec stats into performanceStatsMap
            var current = remoteWindow.performanceStatsMap[deviceId]
            if (current) {
                var newStatsMap = Object.assign({}, remoteWindow.performanceStatsMap)
                newStatsMap[deviceId] = Object.assign({}, current, {
                    captureMs:         detailedStats.captureMs || 0,
                    encodeMs:          detailedStats.encodeMs || 0,
                    networkDelayMs:    detailedStats.networkDelayMs || 0,
                    decodeMs:          detailedStats.decodeMs || 0,
                    paintMs:           detailedStats.paintMs || 0,
                    totalLatencyMs:    totalLatencyMs,
                    inputRoundtripMs:  detailedStats.inputRoundtripMs || 0,
                    bandwidthKbps:     detailedStats.bandwidthKbps || 0,
                    packetRate:        detailedStats.packetRate || 0,
                    codec:             detailedStats.codec || "",
                    frameQuality:      detailedStats.frameQuality !== undefined ? detailedStats.frameQuality : -1,
                    encodedRectWidth:  detailedStats.encodedRectWidth || 0,
                    encodedRectHeight: detailedStats.encodedRectHeight || 0
                })
                remoteWindow.performanceStatsMap = newStatsMap
            }
        }
    }

    // Monitor host capabilities negotiation
    Connections {
        target: remoteWindow.clientManager

        function onHostCapabilitiesChanged(deviceId, supportsSendAttentionSequence, supportsLockWorkstation, supportsFileTransfer, supportsPrivacyScreen, supportsVirtualDisplay) {
            var current = remoteWindow.performanceStatsMap[deviceId] || {}
            var newStatsMap = Object.assign({}, remoteWindow.performanceStatsMap)
            newStatsMap[deviceId] = Object.assign({}, current, {
                supportsSendAttentionSequence: supportsSendAttentionSequence,
                supportsLockWorkstation: supportsLockWorkstation,
                supportsFileTransfer: supportsFileTransfer,
                supportsPrivacyScreen: supportsPrivacyScreen,
                supportsVirtualDisplay: supportsVirtualDisplay
            })
            remoteWindow.performanceStatsMap = newStatsMap
        }
    }

    // Monitor virtual display state changes
    Connections {
        target: remoteWindow.clientManager

        function onVirtualDisplayStateChanged(deviceId, state) {
            var type = state.type || ""
            if (type === "created" || type === "removed" || type === "removedAll") {
                // Re-query to get the latest list
                remoteWindow.clientManager.queryVirtualDisplays(deviceId)
            } else if (type === "state") {
                var displays = state.displays || []
                var newMap = Object.assign({}, remoteWindow.virtualDisplayMap)
                newMap[deviceId] = displays
                remoteWindow.virtualDisplayMap = newMap
                remoteWindow.virtualDisplayVersion++
            } else if (type === "error") {
                console.warn("Virtual display error:", state.message)
            }
        }
    }

    // Monitor display list changes (multi-monitor)
    Connections {
        target: remoteWindow.clientManager

        function onDisplayListChanged(deviceId, displays, activeDisplayIndex) {
            var newMap = Object.assign({}, remoteWindow.displayListMap)
            newMap[deviceId] = {
                displays: displays,
                activeIndex: activeDisplayIndex
            }
            remoteWindow.displayListMap = newMap
            remoteWindow.displayListVersion++
            console.log("Display list updated for", deviceId,
                        ": displays=", displays.length,
                        ", active=", activeDisplayIndex)
        }
    }

    // Monitor ICE route changes
    Connections {
        target: remoteWindow.clientManager

        function onRouteChanged(deviceId, routeInfo) {
            var current = remoteWindow.performanceStatsMap[deviceId]
            if (current) {
                var newStatsMap = Object.assign({}, remoteWindow.performanceStatsMap)
                newStatsMap[deviceId] = Object.assign({}, current, {
                    routeType: routeInfo.routeType || "",
                    transportProtocol: routeInfo.transportProtocol || "",
                    localCandidateType: routeInfo.localCandidateType || "",
                    remoteCandidateType: routeInfo.remoteCandidateType || "",
                    localAddress: routeInfo.localAddress || "",
                    remoteAddress: routeInfo.remoteAddress || "",
                    localCandidates: routeInfo.localCandidates || [],
                    remoteCandidates: routeInfo.remoteCandidates || []
                })
                remoteWindow.performanceStatsMap = newStatsMap
            }
        }
    }

    // File upload dialog (supports multiple file selection)
    FileDialog {
        id: fileDialog
        title: qsTr("Select Files to Upload")
        fileMode: FileDialog.OpenFiles
        onAccepted: {
            var devId = currentTabIndex >= 0 && currentTabIndex < connectionModel.count
                ? connectionModel.deviceIdAt(currentTabIndex) : ""
            if (devId && remoteWindow.clientManager) {
                for (var i = 0; i < selectedFiles.length; i++) {
                    remoteWindow.clientManager.startFileUpload(devId, selectedFiles[i])
                    var fname = selectedFiles[i].toString().split('/').pop()
                    transferModel.append({
                        transferId: "",
                        deviceId: devId,
                        filename: decodeURIComponent(fname),
                        progress: 0,
                        status: "uploading",
                        errorMessage: "",
                        direction: "upload",
                        savePath: ""
                    })
                }
                fileTransferDrawer.open()
            }
        }
    }

    // File transfer data model
    ListModel {
        id: transferModel
    }

    property int activeTransferCount: {
        var count = 0
        for (var i = 0; i < transferModel.count; i++) {
            var s = transferModel.get(i).status
            if (s === "uploading" || s === "downloading") count++
        }
        return count
    }

    function findTransferIndex(transferId) {
        for (var i = 0; i < transferModel.count; i++) {
            if (transferModel.get(i).transferId === transferId) return i
        }
        return -1
    }

    function findTransferByFilename(filename) {
        for (var i = transferModel.count - 1; i >= 0; i--) {
            var item = transferModel.get(i)
            if (item.filename === filename && item.transferId === "") return i
        }
        return -1
    }

    // File transfer event handlers
    Connections {
        target: remoteWindow.clientManager

        function onFileTransferProgress(deviceId, transferId, filename, bytesSent, totalBytes) {
            var idx = remoteWindow.findTransferIndex(transferId)
            if (idx < 0) {
                idx = remoteWindow.findTransferByFilename(filename)
                if (idx >= 0) {
                    transferModel.setProperty(idx, "transferId", transferId)
                } else {
                    transferModel.append({
                        transferId: transferId,
                        deviceId: deviceId,
                        filename: filename,
                        progress: 0,
                        status: "uploading",
                        errorMessage: "",
                        direction: "upload",
                        savePath: ""
                    })
                    idx = transferModel.count - 1
                }
            }
            var pct = totalBytes > 0 ? bytesSent / totalBytes : 0
            transferModel.setProperty(idx, "progress", pct)
        }

        function onFileTransferComplete(deviceId, transferId, filename) {
            var idx = remoteWindow.findTransferIndex(transferId)
            if (idx < 0) idx = remoteWindow.findTransferByFilename(filename)
            if (idx >= 0) {
                transferModel.setProperty(idx, "status", "complete")
                transferModel.setProperty(idx, "progress", 1)
                if (transferModel.get(idx).transferId === "")
                    transferModel.setProperty(idx, "transferId", transferId)
            }
            toast.show(qsTr("Upload complete: %1").arg(filename), QDToast.Type.Success)
        }

        function onFileTransferError(deviceId, transferId, errorMessage) {
            var idx = remoteWindow.findTransferIndex(transferId)
            if (idx < 0) {
                for (var i = transferModel.count - 1; i >= 0; i--) {
                    if (transferModel.get(i).status === "uploading" && transferModel.get(i).transferId === "") {
                        idx = i
                        break
                    }
                }
            }
            if (idx >= 0) {
                transferModel.setProperty(idx, "status", "error")
                transferModel.setProperty(idx, "errorMessage", errorMessage)
                if (transferId && transferModel.get(idx).transferId === "")
                    transferModel.setProperty(idx, "transferId", transferId)
            }
            toast.show(qsTr("Upload failed: %1").arg(errorMessage), QDToast.Type.Error)
        }

        function onFileDownloadStarted(deviceId, transferId, filename, totalBytes) {
            transferModel.append({
                transferId: transferId,
                deviceId: deviceId,
                filename: filename,
                progress: 0,
                status: "downloading",
                errorMessage: "",
                direction: "download",
                savePath: ""
            })
            fileTransferDrawer.open()
        }

        function onFileDownloadProgress(deviceId, transferId, filename, bytesReceived, totalBytes) {
            var idx = remoteWindow.findTransferIndex(transferId)
            if (idx >= 0) {
                var pct = totalBytes > 0 ? bytesReceived / totalBytes : 0
                transferModel.setProperty(idx, "progress", pct)
            }
        }

        function onFileDownloadComplete(deviceId, transferId, filename, savePath) {
            var idx = remoteWindow.findTransferIndex(transferId)
            if (idx >= 0) {
                transferModel.setProperty(idx, "status", "complete")
                transferModel.setProperty(idx, "progress", 1)
                transferModel.setProperty(idx, "savePath", savePath)
            }
            toast.show(qsTr("Download complete: %1").arg(filename), QDToast.Type.Success)
        }

        function onFileDownloadError(deviceId, transferId, errorMessage) {
            var idx = remoteWindow.findTransferIndex(transferId)
            if (idx >= 0) {
                transferModel.setProperty(idx, "status", "error")
                transferModel.setProperty(idx, "errorMessage", errorMessage)
            }
            toast.show(qsTr("Download failed: %1").arg(errorMessage), QDToast.Type.Error)
        }
    }

    // File Transfer Drawer (right side panel)
    QDDrawer {
        id: fileTransferDrawer
        edge: Qt.RightEdge
        width: 360
        title: qsTr("File Transfers") + (remoteWindow.activeTransferCount > 0
            ? " (" + remoteWindow.activeTransferCount + ")" : "")

        ColumnLayout {
            anchors.fill: parent
            spacing: 0

            // Empty state
            QDEmptyState {
                visible: transferModel.count === 0
                Layout.fillWidth: true
                Layout.fillHeight: true
                iconSource: FluentIconGlyph.statusDataTransferGlyph
                title: qsTr("No Transfers")
                description: qsTr("Use the menu to upload or download files")
            }

            // Transfer list
            ListView {
                id: transferListView
                Layout.fillWidth: true
                Layout.fillHeight: true
                visible: transferModel.count > 0
                model: transferModel
                clip: true
                spacing: 2

                delegate: Rectangle {
                    width: transferListView.width
                    height: transferItemLayout.implicitHeight + Theme.spacingMedium * 2
                    color: delegateHover.hovered ? Theme.surfaceHover : "transparent"
                    radius: Theme.radiusSmall

                    HoverHandler { id: delegateHover }

                    ColumnLayout {
                        id: transferItemLayout
                        anchors.fill: parent
                        anchors.margins: Theme.spacingMedium
                        spacing: Theme.spacingSmall

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingSmall

                            // Direction icon (upload/download)
                            Text {
                                text: model.direction === "download"
                                    ? FluentIconGlyph.downloadGlyph
                                    : FluentIconGlyph.uploadGlyph
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: Theme.iconSizeMedium
                                color: {
                                    if (model.status === "complete") return Theme.success
                                    if (model.status === "error" || model.status === "cancelled") return Theme.error
                                    return Theme.primary
                                }
                            }

                            // Filename
                            Text {
                                Layout.fillWidth: true
                                text: model.filename
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                elide: Text.ElideMiddle
                            }

                            // Status / action
                            Text {
                                visible: model.status === "uploading" || model.status === "downloading"
                                text: Math.round(model.progress * 100) + "%"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            // Cancel button (for uploading or downloading)
                            QDIconButton {
                                visible: model.status === "uploading" || model.status === "downloading"
                                iconSource: FluentIconGlyph.cancelGlyph
                                buttonSize: QDIconButton.Size.Small
                                buttonStyle: QDIconButton.Style.Transparent
                                onClicked: {
                                    var item = transferModel.get(index)
                                    if (item.transferId && remoteWindow.clientManager) {
                                        if (item.direction === "download") {
                                            remoteWindow.clientManager.cancelFileDownload(
                                                item.deviceId, item.transferId)
                                        } else {
                                            remoteWindow.clientManager.cancelFileUpload(
                                                item.deviceId, item.transferId)
                                        }
                                    }
                                    transferModel.setProperty(index, "status", "cancelled")
                                    transferModel.setProperty(index, "errorMessage", qsTr("Cancelled"))
                                }
                            }

                            // Completed download actions
                            Row {
                                visible: model.status === "complete" && model.direction === "download" && model.savePath !== ""
                                spacing: 2

                                QDIconButton {
                                    iconSource: FluentIconGlyph.openFileGlyph
                                    buttonSize: QDIconButton.Size.Small
                                    buttonStyle: QDIconButton.Style.Transparent
                                    ToolTip.visible: hovered
                                    ToolTip.text: qsTr("Open File")
                                    onClicked: remoteWindow.clientManager.openDownloadedFile(model.savePath)
                                }

                                QDIconButton {
                                    iconSource: FluentIconGlyph.folderOpenGlyph
                                    buttonSize: QDIconButton.Size.Small
                                    buttonStyle: QDIconButton.Style.Transparent
                                    ToolTip.visible: hovered
                                    ToolTip.text: qsTr("Open Folder")
                                    onClicked: remoteWindow.clientManager.openContainingFolder(model.savePath)
                                }

                                QDIconButton {
                                    iconSource: FluentIconGlyph.deleteGlyph
                                    buttonSize: QDIconButton.Size.Small
                                    buttonStyle: QDIconButton.Style.Transparent
                                    ToolTip.visible: hovered
                                    ToolTip.text: qsTr("Delete File")
                                    onClicked: {
                                        if (remoteWindow.clientManager.deleteDownloadedFile(model.savePath)) {
                                            transferModel.setProperty(index, "status", "deleted")
                                            transferModel.setProperty(index, "errorMessage", qsTr("File deleted"))
                                        }
                                    }
                                }
                            }

                            // Upload complete icon
                            Text {
                                visible: model.status === "complete" && (model.direction === "upload" || model.savePath === "")
                                text: FluentIconGlyph.checkMarkGlyph
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: 14
                                color: Theme.success
                            }

                            // Error icon
                            Text {
                                visible: model.status === "error" || model.status === "cancelled" || model.status === "deleted"
                                text: model.status === "deleted" ? FluentIconGlyph.deleteGlyph : FluentIconGlyph.errorGlyph
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: 14
                                color: model.status === "deleted" ? Theme.textSecondary : Theme.error
                            }
                        }

                        // Progress bar (for uploading or downloading)
                        QDProgressBar {
                            visible: model.status === "uploading" || model.status === "downloading"
                            Layout.fillWidth: true
                            value: model.progress
                            from: 0
                            to: 1
                        }

                        // Status message
                        Text {
                            visible: (model.status === "error" || model.status === "cancelled" || model.status === "deleted") && model.errorMessage !== ""
                            Layout.fillWidth: true
                            text: model.errorMessage
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeSmall
                            color: model.status === "deleted" ? Theme.textSecondary : Theme.error
                            elide: Text.ElideRight
                        }
                    }
                }
            }

            // Clear completed button
            Rectangle {
                visible: transferModel.count > 0
                Layout.fillWidth: true
                Layout.preferredHeight: 48
                color: Theme.surfaceVariant

                QDButton {
                    anchors.centerIn: parent
                    text: qsTr("Clear Completed")
                    onClicked: {
                        for (var i = transferModel.count - 1; i >= 0; i--) {
                            var s = transferModel.get(i).status
                            if (s === "complete" || s === "error" || s === "cancelled" || s === "deleted") {
                                transferModel.remove(i)
                            }
                        }
                    }
                }
            }
        }
    }

    // Trust confirmation dialog
    TrustConfirmationDialog {
        id: trustDialog
        onApproved: function(confirmationId, reason) {
            mainController.resolveConfirmation(confirmationId, true, reason)
        }
        onRejected: function(confirmationId, reason) {
            mainController.resolveConfirmation(confirmationId, false, reason)
        }
    }

    Connections {
        target: mainController
        function onTrustConfirmationRequested(confirmationId, deviceId, toolName,
                                               argumentsJson, riskLevel, reasons, timeoutSecs) {
            var parsedArgs = {}
            try { parsedArgs = JSON.parse(argumentsJson) } catch(e) {}
            trustDialog.showConfirmation(confirmationId, deviceId, toolName,
                                          parsedArgs, riskLevel, reasons, timeoutSecs)
        }
        function onTrustEmergencyStopActivated(reason) {
            remoteWindow.emergencyStopActive = true
            toast.show(qsTr("Emergency stop activated: ") + reason)
        }
        function onTrustEmergencyStopDeactivated() {
            remoteWindow.emergencyStopActive = false
            toast.show(qsTr("Emergency stop deactivated"))
        }
    }

    // Toast for notifications
    QDToast {
        id: toast
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 50
        z: 9999
    }
}
