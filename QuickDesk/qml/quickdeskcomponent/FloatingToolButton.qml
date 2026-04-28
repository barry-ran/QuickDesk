// Floating Tool Button - Draggable button overlay on remote desktop
import QtQuick
import QtQuick.Controls
import "../component"

Item {
    id: root
    
    // Properties
    property string deviceId: ""
    property var clientManager: null
    property var desktopView: null
    property var videoInfo: null  // Video info including original resolution
    
    // Audio state
    property bool audioEnabled: true  // Default: audio enabled
    
    // Framerate boost modes
    readonly property int boostModeOff: 0      // 关闭
    readonly property int boostModeOffice: 1   // 办公模式（默认）
    readonly property int boostModeGaming: 2   // 游戏模式
    
    // Current boost mode (per connection)
    property int framerateBoostMode: boostModeOffice  // 默认办公模式
    
    // Target framerate options
    property int targetFramerate: 30  // 默认30 FPS
    
    // Bitrate options (in bps, using 1024 as unit: 1 MiB = 1024*1024)
    property int preferredMinBitrate: 10485760  // 默认10 MiB (10 * 1024 * 1024)
    
    // Host capabilities (negotiated with host, updated via hostCapabilitiesChanged)
    property bool supportsSendAttentionSequence: false
    property bool supportsLockWorkstation: false
    property bool supportsFileTransfer: false
    property bool supportsPrivacyScreen: false
    property bool privacyScreenActive: false
    property bool supportsVirtualDisplay: false
    property var virtualDisplays: []  // Active virtual displays [{index, width, height, refreshRate}]
    property bool emergencyStopActive: false

    // Multi-monitor display list
    property var displayList: []        // Array of display objects from displayListChanged
    property int activeDisplayIndex: -1 // Currently active display index

    // Active file transfer count (for badge display)
    property int activeTransferCount: 0

    // Signals
    signal disconnectRequested(string deviceId)
    signal fitToRemoteDesktopRequested()
    signal emergencyStopRequested()
    signal toggleVideoStats()
    signal showToast(string message, var toastType)
    signal uploadFileRequested()
    signal downloadFileRequested()
    signal showTransferPanelRequested()
    
    // Apply framerate boost mode
    function applyFramerateBoostMode(mode) {
        if (!clientManager || !deviceId) {
            return
        }
        
        switch(mode) {
            case boostModeOff:
                console.log("FramerateBoost: Disabled for:", deviceId)
                clientManager.setFramerateBoost(deviceId, false, 30, 300)
                break
            case boostModeOffice:
                console.log("FramerateBoost: Office mode (30ms/300ms) for:", deviceId)
                clientManager.setFramerateBoost(deviceId, true, 30, 300)
                break
            case boostModeGaming:
                console.log("FramerateBoost: Gaming mode (15ms/500ms) for:", deviceId)
                clientManager.setFramerateBoost(deviceId, true, 15, 500)
                break
        }
    }
    
    // Size - include extra space for shadow (shadow size is 12px on each side)
    width: 80
    height: 80
    clip: false  // Don't clip shadow
    
    // Draggable behavior
    MouseArea {
        id: dragArea
        anchors.fill: parent
        drag.target: root
        drag.axis: Drag.XAndYAxis
        drag.minimumX: 0
        drag.maximumX: root.parent ? root.parent.width - root.width : 0
        drag.minimumY: 0
        drag.maximumY: root.parent ? root.parent.height - root.height : 0
        
        cursorShape: pressed ? Qt.ClosedHandCursor : Qt.OpenHandCursor
        hoverEnabled: true
        
        // Track if this is a drag or click
        property bool isDragging: false
        property point pressPos: Qt.point(0, 0)
        
        onPressed: function(mouse) {
            isDragging = false
            pressPos = Qt.point(mouse.x, mouse.y)
        }
        
        onPositionChanged: function(mouse) {
            // If moved more than 5 pixels, consider it a drag
            var dx = Math.abs(mouse.x - pressPos.x)
            var dy = Math.abs(mouse.y - pressPos.y)
            if (dx > 5 || dy > 5) {
                isDragging = true
            }
        }
        
        onClicked: {
            if (!isDragging) {
                // Calculate smart menu position
                var menuX, menuY
                var spaceRight = root.parent.width - (root.x + root.width)
                var spaceBottom = root.parent.height - (root.y + root.height)
                
                // Prefer right side, but use left if not enough space
                if (spaceRight >= floatingMenu.width + Theme.spacingMedium) {
                    menuX = root.x + root.width + Theme.spacingMedium
                } else {
                    menuX = root.x - floatingMenu.width - Theme.spacingMedium
                }
                
                // Prefer bottom, but use top if not enough space
                if (spaceBottom >= floatingMenu.height + Theme.spacingMedium) {
                    menuY = root.y
                } else {
                    menuY = root.y + root.height - floatingMenu.height
                }
                
                // Make sure menu is within bounds
                menuX = Math.max(Theme.spacingSmall, Math.min(menuX, root.parent.width - floatingMenu.width - Theme.spacingSmall))
                menuY = Math.max(Theme.spacingSmall, Math.min(menuY, root.parent.height - floatingMenu.height - Theme.spacingSmall))
                
                floatingMenu.x = menuX
                floatingMenu.y = menuY
                floatingMenu.open()
            }
        }
    }
    
    // Circular button background
    Rectangle {
        id: buttonBackground
        anchors.centerIn: parent
        width: 50
        height: 50
        radius: 25
        color: dragArea.containsMouse ? Theme.primaryHover : Theme.primary
        opacity: 0.9
        
        Behavior on color {
            ColorAnimation { duration: Theme.animationDurationFast }
        }
        
        // Icon
        Text {
            id: iconText
            anchors.centerIn: parent
            text: floatingMenu.opened ? FluentIconGlyph.cancelGlyph : FluentIconGlyph.moreGlyph
            font.family: "Segoe Fluent Icons"
            font.pixelSize: 20
            color: Theme.textOnPrimary
            
            // Smooth transition when icon changes
            Behavior on text {
                SequentialAnimation {
                    NumberAnimation {
                        target: iconText
                        property: "opacity"
                        to: 0
                        duration: Theme.animationDurationFast
                    }
                    PropertyAction {
                        target: iconText
                        property: "text"
                    }
                    NumberAnimation {
                        target: iconText
                        property: "opacity"
                        to: 1
                        duration: Theme.animationDurationFast
                    }
                }
            }
        }
    }

    // Active transfer count badge
    QDBadge {
        visible: root.activeTransferCount > 0
        anchors.top: buttonBackground.top
        anchors.right: buttonBackground.right
        anchors.topMargin: -4
        anchors.rightMargin: -4
        z: 1
        count: root.activeTransferCount
        badgeType: QDBadge.Type.Info
    }

    // Shadow effect (outside of button, with margins for shadow space)
    QDShadow {
        anchors.fill: root
        target: root
        shadowSize: 12
        shadowColor: Qt.rgba(0, 0, 0, 0.4)
        z: -1
    }
    
    // Floating Menu using QDMenu
    QDMenu {
        id: floatingMenu
        parent: root.parent
        width: 220

        // Display switcher header (multi-monitor) — shown above menu items
        header: Component {
            Column {
                visible: root.displayList && root.displayList.length > 1
                width: parent ? parent.width : 0
                spacing: Theme.spacingXSmall

                Text {
                    text: qsTr("Displays")
                    font.pixelSize: 11
                    color: Theme.textSecondary
                    leftPadding: Theme.spacingSmall
                }

                Row {
                    leftPadding: Theme.spacingSmall
                    spacing: Theme.spacingXSmall

                    Repeater {
                        model: (root.displayList && root.displayList.length > 1)
                               ? root.displayList.length : 0

                        Rectangle {
                            required property int index
                            readonly property bool isActive: index === root.activeDisplayIndex
                            readonly property bool isHovered: displayMouseArea.containsMouse

                            width: 36
                            height: 30
                            radius: Theme.radiusSmall
                            color: isActive ? Theme.accent
                                   : isHovered ? Theme.surfaceHover
                                   : "transparent"
                            border.color: isActive ? Theme.accent : Theme.border
                            border.width: 1

                            Text {
                                anchors.centerIn: parent
                                text: (parent.index + 1).toString()
                                font.pixelSize: 13
                                font.bold: parent.isActive
                                color: parent.isActive
                                       ? Theme.textOnPrimary : Theme.text
                            }

                            MouseArea {
                                id: displayMouseArea
                                anchors.fill: parent
                                cursorShape: Qt.PointingHandCursor
                                hoverEnabled: true
                                onClicked: {
                                    if (parent.index !== root.activeDisplayIndex &&
                                        root.clientManager && root.deviceId) {
                                        root.clientManager.selectDisplay(
                                            root.deviceId, parent.index)
                                        var d = root.displayList ? root.displayList[parent.index] : null
                                        var label = qsTr("Display") + " " + (parent.index + 1)
                                        if (d && d.displayName) label += " (" + d.displayName + ")"
                                        root.showToast(qsTr("Switched to %1").arg(label),
                                                       QDToast.Type.Success)
                                    }
                                }
                            }

                            ToolTip.visible: displayHoverHandler.hovered
                            ToolTip.text: {
                                var label = qsTr("Display") + " " + (index + 1)
                                var d = root.displayList ? root.displayList[index] : null
                                if (!d) return label
                                if (d.displayName) label += " - " + d.displayName
                                label += "\n" + (d.width || 0) + "×" + (d.height || 0)
                                return label
                            }
                            ToolTip.delay: 400

                            HoverHandler { id: displayHoverHandler }
                        }
                    }
                }

                // Separator
                Rectangle {
                    width: parent ? parent.width - Theme.spacingSmall * 2 : 0
                    height: 1
                    anchors.horizontalCenter: parent.horizontalCenter
                    color: Theme.border
                    visible: root.displayList && root.displayList.length > 1
                }
            }
        }

        // Smart Boost submenu (帧率提升)
        QDMenuItem {
            id: smartBoostMenuItem
            text: qsTr("Smart Boost")
            iconText: FluentIconGlyph.lightningBoltGlyph
            hasSubmenu: true
            onTriggered: {
                // Calculate smart submenu position
                var parentMenu = floatingMenu
                var submenu = smartBoostMenu
                var windowWidth = root.parent ? root.parent.width : 1920
                var windowHeight = root.parent ? root.parent.height : 1080
                
                // Estimate submenu height (3 items + padding)
                var itemHeight = Theme.buttonHeightMedium
                var menuPadding = Theme.spacingSmall
                var estimatedSubmenuHeight = (3 * itemHeight) + (menuPadding * 2) + (Theme.spacingXSmall * 3)
                
                // Calculate vertical position - Smart Boost is the first menu item
                var itemOffsetInMenu = menuPadding
                var menuY = parentMenu.y + itemOffsetInMenu
                
                // Check if submenu would go off bottom
                var spaceBottom = windowHeight - menuY
                if (spaceBottom < estimatedSubmenuHeight) {
                    menuY = Math.max(Theme.spacingSmall, Math.min(menuY, windowHeight - estimatedSubmenuHeight - Theme.spacingSmall))
                }
                
                // Calculate horizontal position
                var rightX = parentMenu.x + parentMenu.width + Theme.spacingSmall
                var spaceRight = windowWidth - rightX
                
                var menuX
                if (spaceRight >= submenu.width + Theme.spacingSmall) {
                    menuX = rightX
                } else {
                    menuX = parentMenu.x - submenu.width - Theme.spacingSmall
                    if (menuX < Theme.spacingSmall) {
                        menuX = Theme.spacingSmall
                    }
                }
                
                smartBoostMenu.x = menuX
                smartBoostMenu.y = menuY
                smartBoostMenu.open()
            }
        }
        
        // Target Framerate submenu (基础帧率)
        QDMenuItem {
            id: framerateMenuItem
            text: qsTr("Target Framerate")
            iconText: FluentIconGlyph.speedHighGlyph
            hasSubmenu: true
            onTriggered: {
                // Calculate smart submenu position
                var parentMenu = floatingMenu
                var submenu = framerateMenu
                var windowWidth = root.parent ? root.parent.width : 1920
                var windowHeight = root.parent ? root.parent.height : 1080
                
                // Estimate submenu height (4 items + padding)
                var itemHeight = Theme.buttonHeightMedium
                var menuPadding = Theme.spacingSmall
                var estimatedSubmenuHeight = (4 * itemHeight) + (menuPadding * 2) + (Theme.spacingXSmall * 4)
                
                // Calculate vertical position - Framerate is the first menu item
                var itemOffsetInMenu = menuPadding
                var menuY = parentMenu.y + itemOffsetInMenu
                
                // Check if submenu would go off bottom
                var spaceBottom = windowHeight - menuY
                if (spaceBottom < estimatedSubmenuHeight) {
                    menuY = Math.max(Theme.spacingSmall, Math.min(menuY, windowHeight - estimatedSubmenuHeight - Theme.spacingSmall))
                }
                
                // Calculate horizontal position
                var rightX = parentMenu.x + parentMenu.width + Theme.spacingSmall
                var spaceRight = windowWidth - rightX
                
                var menuX
                if (spaceRight >= submenu.width + Theme.spacingSmall) {
                    menuX = rightX
                } else {
                    menuX = parentMenu.x - submenu.width - Theme.spacingSmall
                    if (menuX < Theme.spacingSmall) {
                        menuX = Theme.spacingSmall
                    }
                }
                
                framerateMenu.x = menuX
                framerateMenu.y = menuY
                framerateMenu.open()
            }
        }
        
        // Resolution submenu (分辨率)
        QDMenuItem {
            id: resolutionMenuItem
            text: qsTr("Resolution")
            iconText: FluentIconGlyph.resizeMouseMediumGlyph
            hasSubmenu: true
            onTriggered: {
                // Calculate smart submenu position
                var parentMenu = floatingMenu
                var submenu = resolutionMenu
                var windowWidth = root.parent ? root.parent.width : 1920
                var windowHeight = root.parent ? root.parent.height : 1080
                
                // Estimate submenu height (8 items + 1 separator + padding)
                var itemHeight = Theme.buttonHeightMedium
                var separatorHeight = 1 + Theme.spacingXSmall * 2
                var menuPadding = Theme.spacingSmall
                var estimatedSubmenuHeight = (8 * itemHeight) + separatorHeight + (menuPadding * 2) + (Theme.spacingXSmall * 8)
                
                // Calculate vertical position - Resolution is the 3rd menu item
                var itemOffsetInMenu = menuPadding + itemHeight * 2
                var menuY = parentMenu.y + itemOffsetInMenu
                
                // Check if submenu would go off bottom
                var spaceBottom = windowHeight - menuY
                if (spaceBottom < estimatedSubmenuHeight) {
                    // Adjust upward to fit, but align with parent menu if possible
                    menuY = Math.max(Theme.spacingSmall, Math.min(menuY, windowHeight - estimatedSubmenuHeight - Theme.spacingSmall))
                }
                
                // Calculate horizontal position
                var rightX = parentMenu.x + parentMenu.width + Theme.spacingSmall
                var spaceRight = windowWidth - rightX
                
                var menuX
                if (spaceRight >= submenu.width + Theme.spacingSmall) {
                    // Enough space on right - show on right side
                    menuX = rightX
                } else {
                    // Not enough space on right - show on left side
                    menuX = parentMenu.x - submenu.width - Theme.spacingSmall
                    // Make sure it doesn't go off left edge
                    if (menuX < Theme.spacingSmall) {
                        menuX = Theme.spacingSmall
                    }
                }
                
                resolutionMenu.x = menuX
                resolutionMenu.y = menuY
                resolutionMenu.open()
            }
        }
        
        // Bitrate submenu (码率)
        QDMenuItem {
            id: bitrateMenuItem
            text: qsTr("Bitrate")
            iconText: FluentIconGlyph.speedHighGlyph
            hasSubmenu: true
            onTriggered: {
                // Calculate smart submenu position
                var parentMenu = floatingMenu
                var submenu = bitrateMenu
                var windowWidth = root.parent ? root.parent.width : 1920
                var windowHeight = root.parent ? root.parent.height : 1080
                
                // Estimate submenu height (5 items + padding)
                var itemHeight = Theme.buttonHeightMedium
                var menuPadding = Theme.spacingSmall
                var estimatedSubmenuHeight = (5 * itemHeight) + (menuPadding * 2) + (Theme.spacingXSmall * 5)
                
                // Calculate vertical position - Bitrate is the 4th menu item
                var itemOffsetInMenu = menuPadding + itemHeight * 3
                var menuY = parentMenu.y + itemOffsetInMenu
                
                // Check if submenu would go off bottom
                var spaceBottom = windowHeight - menuY
                if (spaceBottom < estimatedSubmenuHeight) {
                    menuY = Math.max(Theme.spacingSmall, Math.min(menuY, windowHeight - estimatedSubmenuHeight - Theme.spacingSmall))
                }
                
                // Calculate horizontal position
                var rightX = parentMenu.x + parentMenu.width + Theme.spacingSmall
                var spaceRight = windowWidth - rightX
                
                var menuX
                if (spaceRight >= submenu.width + Theme.spacingSmall) {
                    menuX = rightX
                } else {
                    menuX = parentMenu.x - submenu.width - Theme.spacingSmall
                    if (menuX < Theme.spacingSmall) {
                        menuX = Theme.spacingSmall
                    }
                }
                
                bitrateMenu.x = menuX
                bitrateMenu.y = menuY
                bitrateMenu.open()
            }
        }
        
        // Fit window to remote desktop resolution
        QDMenuItem {
            text: qsTr("Fit Window")
            iconText: FluentIconGlyph.fullScreenGlyph
            enabled: root.videoInfo && root.videoInfo.frameWidth > 0 && root.videoInfo.frameHeight > 0
            onTriggered: {
                console.log("Fit window to remote desktop requested for:", root.deviceId)
                root.fitToRemoteDesktopRequested()
                root.showToast(qsTr("Window resized to fit remote desktop"), QDToast.Type.Success)
            }
        }
        
        // Toggle video stats overlay
        QDMenuItem {
            text: qsTr("Video Stats")
            iconText: FluentIconGlyph.diagnosticGlyph
            onTriggered: {
                root.toggleVideoStats()
            }
        }
        
        // Toggle audio (mute/unmute)
        QDMenuItem {
            text: root.audioEnabled ? qsTr("Mute Audio") : qsTr("Unmute Audio")
            iconText: root.audioEnabled ? FluentIconGlyph.volumeGlyph : FluentIconGlyph.muteGlyph
            onTriggered: {
                root.audioEnabled = !root.audioEnabled
                console.log("Audio toggled:", root.audioEnabled ? "enabled" : "muted", "for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setAudioEnabled(root.deviceId, root.audioEnabled)
                    root.showToast(
                        root.audioEnabled ? qsTr("Audio: Enabled") : qsTr("Audio: Muted"),
                        QDToast.Type.Success
                    )
                }
            }
        }
        
        // Remote actions (only visible when host supports them)
        QDMenuSeparator {
            visible: root.supportsSendAttentionSequence || root.supportsLockWorkstation
        }
        
        QDMenuItem {
            visible: root.supportsSendAttentionSequence
            text: qsTr("Send Ctrl+Alt+Del")
            iconText: FluentIconGlyph.keyboardShortcutGlyph
            onTriggered: {
                console.log("Send Ctrl+Alt+Del for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.sendAction(root.deviceId, "sendAttentionSequenceAction")
                    root.showToast(qsTr("Ctrl+Alt+Del sent"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            visible: root.supportsLockWorkstation
            text: qsTr("Lock Screen")
            iconText: FluentIconGlyph.lockGlyph
            onTriggered: {
                console.log("Lock screen for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.sendAction(root.deviceId, "lockWorkstationAction")
                    root.showToast(qsTr("Lock screen sent"), QDToast.Type.Success)
                }
            }
        }

        QDMenuItem {
            visible: root.supportsPrivacyScreen
            text: root.privacyScreenActive ? qsTr("Disable Privacy Screen") : qsTr("Privacy Screen")
            iconText: root.privacyScreenActive ? FluentIconGlyph.redEyeGlyph : FluentIconGlyph.hideGlyph
            onTriggered: {
                var newState = !root.privacyScreenActive
                console.log("Toggle privacy screen:", newState, "for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.togglePrivacyScreen(root.deviceId, newState)
                    root.privacyScreenActive = newState
                    root.showToast(
                        newState ? qsTr("Privacy Screen: On") : qsTr("Privacy Screen: Off"),
                        QDToast.Type.Success
                    )
                }
            }
        }

        QDMenuItem {
            visible: root.supportsVirtualDisplay
            text: qsTr("Virtual Display")
            iconText: FluentIconGlyph.tVMonitorGlyph
            hasSubmenu: true
            onTriggered: {
                var parentMenu = floatingMenu
                var submenu = virtualDisplayMenu
                var windowWidth = root.parent ? root.parent.width : 1920
                var windowHeight = root.parent ? root.parent.height : 1080

                var itemHeight = Theme.buttonHeightMedium
                var menuPadding = Theme.spacingSmall
                var estimatedSubmenuHeight = (7 * itemHeight) + (menuPadding * 2) + (Theme.spacingXSmall * 7)

                var menuY = parentMenu.y + menuPadding
                var spaceBottom = windowHeight - menuY
                if (spaceBottom < estimatedSubmenuHeight) {
                    menuY = Math.max(Theme.spacingSmall, Math.min(menuY, windowHeight - estimatedSubmenuHeight - Theme.spacingSmall))
                }

                var rightX = parentMenu.x + parentMenu.width + Theme.spacingSmall
                var spaceRight = windowWidth - rightX
                var menuX
                if (spaceRight >= submenu.width + Theme.spacingSmall) {
                    menuX = rightX
                } else {
                    menuX = parentMenu.x - submenu.width - Theme.spacingSmall
                    if (menuX < Theme.spacingSmall) {
                        menuX = Theme.spacingSmall
                    }
                }

                virtualDisplayMenu.x = menuX
                virtualDisplayMenu.y = menuY
                virtualDisplayMenu.open()
            }
        }

        QDMenuSeparator {
            visible: root.supportsFileTransfer
        }

        QDMenuItem {
            visible: root.supportsFileTransfer
            text: qsTr("Upload File")
            iconText: FluentIconGlyph.uploadGlyph
            onTriggered: {
                console.log("Upload file for:", root.deviceId)
                root.uploadFileRequested()
            }
        }

        QDMenuItem {
            visible: root.supportsFileTransfer
            text: qsTr("Download File")
            iconText: FluentIconGlyph.downloadGlyph
            onTriggered: {
                console.log("Download file for:", root.deviceId)
                root.downloadFileRequested()
            }
        }

        QDMenuItem {
            visible: root.supportsFileTransfer && root.activeTransferCount > 0
            text: qsTr("Transfers") + " (" + root.activeTransferCount + ")"
            iconText: FluentIconGlyph.statusDataTransferGlyph
            onTriggered: {
                root.showTransferPanelRequested()
            }
        }
        
        QDMenuSeparator { }
        
        QDMenuSeparator {}

        QDMenuItem {
            text: root.emergencyStopActive ? qsTr("Deactivate Emergency Stop") : qsTr("Emergency Stop")
            iconText: FluentIconGlyph.importantGlyph
            isDestructive: !root.emergencyStopActive
            onTriggered: {
                root.emergencyStopRequested()
            }
        }

        QDMenuSeparator {}

        QDMenuItem {
            text: qsTr("Disconnect")
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                console.log("Disconnect connection:", root.deviceId)
                root.disconnectRequested(root.deviceId)
            }
        }
    }
    
    // Smart Boost submenu (帧率提升模式)
    QDMenu {
        id: smartBoostMenu
        parent: root.parent
        width: 180
        
        // Close both menus when submenu closes
        onClosed: {
            if (floatingMenu.opened) {
                floatingMenu.close()
            }
        }
        
        // Smart framerate boost modes
        QDMenuItem {
            text: qsTr("Off") + (root.framerateBoostMode === root.boostModeOff ? " ✓" : "")
            iconText: FluentIconGlyph.cancelGlyph
            onTriggered: {
                root.framerateBoostMode = root.boostModeOff
                root.applyFramerateBoostMode(root.boostModeOff)
                root.showToast(qsTr("Smart Boost: Off"), QDToast.Type.Success)
            }
        }
        
        QDMenuItem {
            text: qsTr("Office") + (root.framerateBoostMode === root.boostModeOffice ? " ✓" : "")
            iconText: FluentIconGlyph.editGlyph
            onTriggered: {
                root.framerateBoostMode = root.boostModeOffice
                root.applyFramerateBoostMode(root.boostModeOffice)
                root.showToast(qsTr("Smart Boost: Office Mode"), QDToast.Type.Success)
            }
        }
        
        QDMenuItem {
            text: qsTr("Gaming") + (root.framerateBoostMode === root.boostModeGaming ? " ✓" : "")
            iconText: FluentIconGlyph.gameGlyph
            onTriggered: {
                root.framerateBoostMode = root.boostModeGaming
                root.applyFramerateBoostMode(root.boostModeGaming)
                root.showToast(qsTr("Smart Boost: Gaming Mode"), QDToast.Type.Success)
            }
        }
    }
    
    // Target Framerate submenu (基础帧率设置)
    QDMenu {
        id: framerateMenu
        parent: root.parent
        width: 150
        
        // Close both menus when submenu closes
        onClosed: {
            if (floatingMenu.opened) {
                floatingMenu.close()
            }
        }
        
        QDMenuItem {
            text: "60 FPS" + (root.targetFramerate === 60 ? " ✓" : "")
            onTriggered: {
                console.log("Set target framerate 60 FPS for:", root.deviceId)
                root.targetFramerate = 60
                if (root.clientManager) {
                    root.clientManager.setTargetFramerate(root.deviceId, 60)
                    root.showToast(qsTr("Target Framerate: 60 FPS"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "30 FPS" + (root.targetFramerate === 30 ? " ✓" : "")
            onTriggered: {
                console.log("Set target framerate 30 FPS for:", root.deviceId)
                root.targetFramerate = 30
                if (root.clientManager) {
                    root.clientManager.setTargetFramerate(root.deviceId, 30)
                    root.showToast(qsTr("Target Framerate: 30 FPS"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "15 FPS" + (root.targetFramerate === 15 ? " ✓" : "")
            onTriggered: {
                console.log("Set target framerate 15 FPS for:", root.deviceId)
                root.targetFramerate = 15
                if (root.clientManager) {
                    root.clientManager.setTargetFramerate(root.deviceId, 15)
                    root.showToast(qsTr("Target Framerate: 15 FPS"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "5 FPS" + (root.targetFramerate === 5 ? " ✓" : "")
            onTriggered: {
                console.log("Set target framerate 5 FPS for:", root.deviceId)
                root.targetFramerate = 5
                if (root.clientManager) {
                    root.clientManager.setTargetFramerate(root.deviceId, 5)
                    root.showToast(qsTr("Target Framerate: 5 FPS"), QDToast.Type.Success)
                }
            }
        }
    }
    
    // Resolution submenu
    QDMenu {
        id: resolutionMenu
        parent: root.parent
        width: 190
        
        // Close both menus when submenu closes
        onClosed: {
            if (floatingMenu.opened) {
                floatingMenu.close()
            }
        }
        
        QDMenuItem {
            text: {
                if (root.videoInfo && root.videoInfo.originalWidth > 0 && root.videoInfo.originalHeight > 0) {
                    return qsTr("Original") + " (" + root.videoInfo.originalWidth + "x" + root.videoInfo.originalHeight + ")"
                }
                return qsTr("Original")
            }
            enabled: root.videoInfo && root.videoInfo.originalWidth > 0 && root.videoInfo.originalHeight > 0
            onTriggered: {
                // Restore to original resolution (first frame resolution)
                if (root.videoInfo && root.videoInfo.originalWidth > 0 && root.videoInfo.originalHeight > 0 && root.clientManager) {
                    console.log("Restore to original resolution:", root.videoInfo.originalWidth + "x" + root.videoInfo.originalHeight)
                    root.clientManager.setResolution(
                        root.deviceId, 
                        root.videoInfo.originalWidth, 
                        root.videoInfo.originalHeight, 
                        96
                    )
                    root.showToast(qsTr("Resolution: ") + root.videoInfo.originalWidth + "x" + root.videoInfo.originalHeight + " (" + qsTr("Original") + ")", QDToast.Type.Success)
                } else {
                    console.log("Cannot restore to original: invalid resolution data. Width:", root.videoInfo ? root.videoInfo.originalWidth : "null", "Height:", root.videoInfo ? root.videoInfo.originalHeight : "null")
                }
            }
        }
        
        QDMenuSeparator { }
        
        QDMenuItem {
            text: "3840 x 2160 (4K)"
            onTriggered: {
                console.log("Set resolution 3840x2160 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 3840, 2160, 96)
                    root.showToast(qsTr("Resolution: 3840x2160 (4K)"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "2560 x 1440 (2K)"
            onTriggered: {
                console.log("Set resolution 2560x1440 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 2560, 1440, 96)
                    root.showToast(qsTr("Resolution: 2560x1440 (2K)"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "1920 x 1080 (FHD)"
            onTriggered: {
                console.log("Set resolution 1920x1080 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 1920, 1080, 96)
                    root.showToast(qsTr("Resolution: 1920x1080 (FHD)"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "1600 x 900"
            onTriggered: {
                console.log("Set resolution 1600x900 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 1600, 900, 96)
                    root.showToast(qsTr("Resolution: 1600x900"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "1366 x 768"
            onTriggered: {
                console.log("Set resolution 1366x768 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 1366, 768, 96)
                    root.showToast(qsTr("Resolution: 1366x768"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "1280 x 720"
            onTriggered: {
                console.log("Set resolution 1280x720 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 1280, 720, 96)
                    root.showToast(qsTr("Resolution: 1280x720"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "1024 x 768"
            onTriggered: {
                console.log("Set resolution 1024x768 for:", root.deviceId)
                if (root.clientManager) {
                    root.clientManager.setResolution(root.deviceId, 1024, 768, 96)
                    root.showToast(qsTr("Resolution: 1024x768"), QDToast.Type.Success)
                }
            }
        }
    }
    
    // Bitrate submenu
    QDMenu {
        id: bitrateMenu
        parent: root.parent
        width: 150
        
        // Close both menus when submenu closes
        onClosed: {
            if (floatingMenu.opened) {
                floatingMenu.close()
            }
        }
        
        QDMenuItem {
            text: "100 MiB" + (root.preferredMinBitrate === 104857600 ? " ✓" : "")
            onTriggered: {
                console.log("Set bitrate 100 MiB for:", root.deviceId)
                root.preferredMinBitrate = 104857600  // 100 * 1024 * 1024
                if (root.clientManager) {
                    root.clientManager.setBitrate(root.deviceId, 104857600)
                    root.showToast(qsTr("Bitrate: 100 MiB"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "50 MiB" + (root.preferredMinBitrate === 52428800 ? " ✓" : "")
            onTriggered: {
                console.log("Set bitrate 50 MiB for:", root.deviceId)
                root.preferredMinBitrate = 52428800  // 50 * 1024 * 1024
                if (root.clientManager) {
                    root.clientManager.setBitrate(root.deviceId, 52428800)
                    root.showToast(qsTr("Bitrate: 50 MiB"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "10 MiB" + (root.preferredMinBitrate === 10485760 ? " ✓" : "")
            onTriggered: {
                console.log("Set bitrate 10 MiB for:", root.deviceId)
                root.preferredMinBitrate = 10485760  // 10 * 1024 * 1024
                if (root.clientManager) {
                    root.clientManager.setBitrate(root.deviceId, 10485760)
                    root.showToast(qsTr("Bitrate: 10 MiB"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "5 MiB" + (root.preferredMinBitrate === 5242880 ? " ✓" : "")
            onTriggered: {
                console.log("Set bitrate 5 MiB for:", root.deviceId)
                root.preferredMinBitrate = 5242880  // 5 * 1024 * 1024
                if (root.clientManager) {
                    root.clientManager.setBitrate(root.deviceId, 5242880)
                    root.showToast(qsTr("Bitrate: 5 MiB"), QDToast.Type.Success)
                }
            }
        }
        
        QDMenuItem {
            text: "2 MiB" + (root.preferredMinBitrate === 2097152 ? " ✓" : "")
            onTriggered: {
                console.log("Set bitrate 2 MiB for:", root.deviceId)
                root.preferredMinBitrate = 2097152  // 2 * 1024 * 1024
                if (root.clientManager) {
                    root.clientManager.setBitrate(root.deviceId, 2097152)
                    root.showToast(qsTr("Bitrate: 2 MiB"), QDToast.Type.Success)
                }
            }
        }
    }

    // Virtual Display submenu
    QDMenu {
        id: virtualDisplayMenu
        parent: root.parent
        width: 220

        onClosed: {
            if (floatingMenu.opened) {
                floatingMenu.close()
            }
        }

        QDMenuItem {
            text: qsTr("Add 3840×2160 @60Hz")
            iconText: FluentIconGlyph.addGlyph
            onTriggered: {
                if (root.clientManager) {
                    root.clientManager.createVirtualDisplay(root.deviceId, 3840, 2160, 60)
                    root.showToast(qsTr("Creating virtual display: 3840×2160"), QDToast.Type.Info)
                }
            }
        }

        QDMenuItem {
            text: qsTr("Add 2560×1440 @60Hz")
            iconText: FluentIconGlyph.addGlyph
            onTriggered: {
                if (root.clientManager) {
                    root.clientManager.createVirtualDisplay(root.deviceId, 2560, 1440, 60)
                    root.showToast(qsTr("Creating virtual display: 2560×1440"), QDToast.Type.Info)
                }
            }
        }

        QDMenuItem {
            text: qsTr("Add 1920×1080 @60Hz")
            iconText: FluentIconGlyph.addGlyph
            onTriggered: {
                if (root.clientManager) {
                    root.clientManager.createVirtualDisplay(root.deviceId, 1920, 1080, 60)
                    root.showToast(qsTr("Creating virtual display: 1920×1080"), QDToast.Type.Info)
                }
            }
        }

        QDMenuItem {
            text: qsTr("Add 1280×720 @60Hz")
            iconText: FluentIconGlyph.addGlyph
            onTriggered: {
                if (root.clientManager) {
                    root.clientManager.createVirtualDisplay(root.deviceId, 1280, 720, 60)
                    root.showToast(qsTr("Creating virtual display: 1280×720"), QDToast.Type.Info)
                }
            }
        }

        QDMenuSeparator {
            visible: root.virtualDisplays && root.virtualDisplays.length > 0
        }

        QDMenuItem {
            visible: root.virtualDisplays && root.virtualDisplays.length > 0
            readonly property var vd: (root.virtualDisplays && root.virtualDisplays.length > 0) ? root.virtualDisplays[0] : null
            text: vd ? (qsTr("Remove #%1").arg(vd.index) + (vd.width > 0 ? (" (" + vd.width + "×" + vd.height + ")") : "")) : ""
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                if (root.clientManager && vd) {
                    root.clientManager.removeVirtualDisplay(root.deviceId, vd.index)
                    root.showToast(qsTr("Removing virtual display #%1").arg(vd.index), QDToast.Type.Info)
                }
            }
        }

        QDMenuItem {
            visible: root.virtualDisplays && root.virtualDisplays.length > 1
            readonly property var vd: (root.virtualDisplays && root.virtualDisplays.length > 1) ? root.virtualDisplays[1] : null
            text: vd ? (qsTr("Remove #%1").arg(vd.index) + (vd.width > 0 ? (" (" + vd.width + "×" + vd.height + ")") : "")) : ""
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                if (root.clientManager && vd) {
                    root.clientManager.removeVirtualDisplay(root.deviceId, vd.index)
                    root.showToast(qsTr("Removing virtual display #%1").arg(vd.index), QDToast.Type.Info)
                }
            }
        }

        QDMenuItem {
            visible: root.virtualDisplays && root.virtualDisplays.length > 2
            readonly property var vd: (root.virtualDisplays && root.virtualDisplays.length > 2) ? root.virtualDisplays[2] : null
            text: vd ? (qsTr("Remove #%1").arg(vd.index) + (vd.width > 0 ? (" (" + vd.width + "×" + vd.height + ")") : "")) : ""
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                if (root.clientManager && vd) {
                    root.clientManager.removeVirtualDisplay(root.deviceId, vd.index)
                    root.showToast(qsTr("Removing virtual display #%1").arg(vd.index), QDToast.Type.Info)
                }
            }
        }

        QDMenuItem {
            visible: root.virtualDisplays && root.virtualDisplays.length > 3
            readonly property var vd: (root.virtualDisplays && root.virtualDisplays.length > 3) ? root.virtualDisplays[3] : null
            text: vd ? (qsTr("Remove #%1").arg(vd.index) + (vd.width > 0 ? (" (" + vd.width + "×" + vd.height + ")") : "")) : ""
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                if (root.clientManager && vd) {
                    root.clientManager.removeVirtualDisplay(root.deviceId, vd.index)
                    root.showToast(qsTr("Removing virtual display #%1").arg(vd.index), QDToast.Type.Info)
                }
            }
        }

        QDMenuSeparator {
            visible: root.virtualDisplays && root.virtualDisplays.length > 0
        }

        QDMenuItem {
            visible: root.virtualDisplays && root.virtualDisplays.length > 0
            text: qsTr("Remove All")
            iconText: FluentIconGlyph.cancelGlyph
            isDestructive: true
            onTriggered: {
                if (root.clientManager) {
                    root.clientManager.removeAllVirtualDisplays(root.deviceId)
                    root.showToast(qsTr("Removing all virtual displays"), QDToast.Type.Info)
                }
            }
        }
    }
}
