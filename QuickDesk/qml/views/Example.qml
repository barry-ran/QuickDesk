// Fluent Design Components Example
import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import "component"

Window {
    width: 1200
    height: 800
    visible: true
    
    Item {
        id: root
        anchors.fill: parent
    
    // ============ Global Toast ============
    
    QDToast {
        id: globalToast
    }
    
    // ============ Global MessageBox ============
    
    QDMessageBox {
        id: globalMessageBox
    }
    
    // ============ Global Drawers ============
    
    QDDrawer {
        id: leftDrawer
        width: 320
        height: parent.height
        edge: Qt.LeftEdge
        title: "左侧导航"
        
        Column {
            width: parent.width
            spacing: Theme.spacingSmall
            
            Repeater {
                model: ["主页", "设置", "关于", "帮助", "反馈"]
                
                Rectangle {
                    width: parent.width
                    height: Theme.buttonHeightMedium
                    color: menuItemArea.containsMouse ? Theme.surfaceHover : "transparent"
                    radius: Theme.radiusSmall
                    
                    Row {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacingMedium
                        spacing: Theme.spacingMedium
                        
                        Text {
                            text: FluentIconGlyph.homeGlyph
                            font.family: "Segoe Fluent Icons"
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                            verticalAlignment: Text.AlignVCenter
                            height: parent.height
                        }
                        
                        Text {
                            text: modelData
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                            verticalAlignment: Text.AlignVCenter
                            height: parent.height
                        }
                    }
                    
                    MouseArea {
                        id: menuItemArea
                        anchors.fill: parent
                        hoverEnabled: true
                        onClicked: {
                            globalToast.show("点击: " + modelData, QDToast.Type.Info)
                            leftDrawer.close()
                        }
                    }
                    
                    Behavior on color {
                        ColorAnimation { duration: Theme.animationDurationFast }
                    }
                }
            }
        }
    }
    
    QDDrawer {
        id: rightDrawer
        width: 320
        height: parent.height
        edge: Qt.RightEdge
        title: "设置面板"
        
        Column {
            width: parent.width
            spacing: Theme.spacingMedium
            
            Text {
                text: "显示设置"
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.DemiBold
                color: Theme.text
            }
            
            Row {
                width: parent.width
                spacing: Theme.spacingMedium
                
                Text {
                    text: "暗黑模式"
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.text
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width - darkModeSwitch.width - parent.spacing
                }
                
                QDSwitch {
                    id: darkModeSwitch
                    checked: true
                    onCheckedChanged: {
                        globalToast.show(checked ? "已开启暗黑模式" : "已关闭暗黑模式", QDToast.Type.Info)
                    }
                }
            }
            
            QDDivider {
                width: parent.width
            }
            
            Text {
                text: "通知设置"
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.DemiBold
                color: Theme.text
            }
            
            Column {
                width: parent.width
                spacing: Theme.spacingSmall
                
                QDCheckBox {
                    text: "桌面通知"
                    checked: true
                }
                
                QDCheckBox {
                    text: "声音提示"
                    checked: false
                }
                
                QDCheckBox {
                    text: "震动反馈"
                    checked: true
                }
            }
        }
    }
    
    // ============ Example Dialog ============
    
    QDDialog {
        id: exampleDialog
        title: "示例对话框"
        dialogWidth: 500
        dialogHeight: 400
        
        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingLarge
            
            Text {
                text: "这是一个 Fluent Design 风格的对话框示例"
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.text
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
            
            QDTextField {
                placeholderText: "输入一些内容..."
                Layout.fillWidth: true
            }
            
            QDCheckBox {
                text: "记住我的选择"
            }
            
            Item { Layout.fillHeight: true }
        }
        
        footer: [
            QDButton {
                text: "取消"
                buttonType: QDButton.Type.Secondary
                onClicked: exampleDialog.reject()
            },
            QDButton {
                text: "确认"
                buttonType: QDButton.Type.Primary
                onClicked: {
                    exampleDialog.accept()
                    globalToast.show("操作已确认", QDToast.Type.Success)
                }
            }
        ]
    }
    
    // ============ Main Content ============
    
    Rectangle {
        anchors.fill: parent
        color: Theme.background
    }
    
    QQC.ScrollView {
        anchors.fill: parent
        contentWidth: availableWidth
        
        ColumnLayout {
            width: parent.width
            spacing: Theme.spacingXXLarge
            
            // Header
            Item {
                Layout.fillWidth: true
                Layout.preferredHeight: 160
                
                Rectangle {
                    anchors.fill: parent
                    gradient: Gradient {
                        GradientStop { position: 0.0; color: Theme.primary }
                        GradientStop { position: 1.0; color: Theme.accent }
                    }
                    
                    ColumnLayout {
                        anchors.centerIn: parent
                        spacing: Theme.spacingMedium
                        
                        Text {
                            text: "Fluent Design 组件库"
                            font.family: Theme.fontFamily
                            font.pixelSize: 32
                            font.weight: Font.Bold
                            color: Theme.textOnPrimary
                            Layout.alignment: Qt.AlignHCenter
                        }
                        
                        Text {
                            text: "Modern Fluent Design System for QuickDesk"
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            color: Qt.rgba(1, 1, 1, 0.8)
                            Layout.alignment: Qt.AlignHCenter
                        }
                        
                        // Theme Switcher
                        Row {
                            spacing: Theme.spacingSmall
                            Layout.alignment: Qt.AlignHCenter
                            Layout.topMargin: Theme.spacingMedium
                            
                            Text {
                                text: "主题: "
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textOnPrimary
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            
                            Repeater {
                                model: [
                                    { name: "Fluent Dark", type: Theme.ThemeType.FluentDark },
                                    { name: "Fluent Light", type: Theme.ThemeType.FluentLight },
                                    { name: "Nord", type: Theme.ThemeType.NordDark },
                                    { name: "Dracula", type: Theme.ThemeType.DraculaDark },
                                    { name: "Monokai", type: Theme.ThemeType.MonokaiDark },
                                    { name: "Solarized", type: Theme.ThemeType.SolarizedLight }
                                ]
                                
                                Rectangle {
                                    width: 90
                                    height: 28
                                    radius: Theme.radiusSmall
                                    color: Theme.currentTheme === modelData.type ? 
                                           Qt.rgba(1, 1, 1, 0.3) : Qt.rgba(0, 0, 0, 0.2)
                                    border.width: Theme.currentTheme === modelData.type ? 2 : 1
                                    border.color: Qt.rgba(1, 1, 1, Theme.currentTheme === modelData.type ? 0.6 : 0.3)
                                    
                                    Text {
                                        anchors.centerIn: parent
                                        text: modelData.name
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        font.weight: Theme.currentTheme === modelData.type ? Font.DemiBold : Font.Normal
                                        color: Theme.textOnPrimary
                                    }
                                    
                                    MouseArea {
                                        anchors.fill: parent
                                        cursorShape: Qt.PointingHandCursor
                                        onClicked: {
                                            Theme.currentTheme = modelData.type
                                            globalToast.show("已切换到 " + modelData.name + " 主题", QDToast.Type.Success)
                                        }
                                    }
                                    
                                    Behavior on color {
                                        ColorAnimation { duration: 150 }
                                    }
                                    
                                    Behavior on border.color {
                                        ColorAnimation { duration: 150 }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            
            // Content sections
            ColumnLayout {
                Layout.fillWidth: true
                Layout.margins: Theme.spacingXXLarge
                spacing: Theme.spacingXXLarge
                
                // ============ Buttons Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "按钮 (Buttons)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: buttonContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: buttonContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            // Primary buttons
                            RowLayout {
                                spacing: Theme.spacingMedium
                                
                                QDButton {
                                    text: "Primary"
                                    buttonType: QDButton.Type.Primary
                                    onClicked: globalToast.show("Primary 按钮被点击", QDToast.Type.Info)
                                }
                                
                                QDButton {
                                    text: "带图标"
                                    buttonType: QDButton.Type.Primary
                                    iconText: FluentIconGlyph.checkMarkGlyph
                                    onClicked: globalToast.show("操作成功", QDToast.Type.Success)
                                }
                                
                                QDButton {
                                    text: "加载中..."
                                    buttonType: QDButton.Type.Primary
                                    loading: true
                                }
                                
                                QDButton {
                                    text: "禁用"
                                    buttonType: QDButton.Type.Primary
                                    enabled: false
                                }
                            }
                            
                            // Secondary & Other buttons
                            RowLayout {
                                spacing: Theme.spacingMedium
                                
                                QDButton {
                                    text: "Secondary"
                                    buttonType: QDButton.Type.Secondary
                                    onClicked: globalToast.show("Secondary 按钮", QDToast.Type.Info)
                                }
                                
                                QDButton {
                                    text: "Success"
                                    buttonType: QDButton.Type.Success
                                    iconText: FluentIconGlyph.acceptGlyph
                                    onClicked: globalToast.show("成功操作", QDToast.Type.Success)
                                }
                                
                                QDButton {
                                    text: "Danger"
                                    buttonType: QDButton.Type.Danger
                                    iconText: FluentIconGlyph.deleteGlyph
                                    onClicked: globalToast.show("危险操作", QDToast.Type.Error)
                                }
                                
                                QDButton {
                                    text: "Ghost"
                                    buttonType: QDButton.Type.Ghost
                                    onClicked: globalToast.show("Ghost 按钮", QDToast.Type.Info)
                                }
                            }
                        }
                    }
                }
                
                // ============ Input Fields Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "文本输入 (Text Fields)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: inputContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        GridLayout {
                            id: inputContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            columns: 2
                            rowSpacing: Theme.spacingLarge
                            columnSpacing: Theme.spacingXLarge
                            
                            QDTextField {
                                placeholderText: "普通输入框"
                                Layout.fillWidth: true
                            }
                            
                            QDTextField {
                                placeholderText: "带前缀图标"
                                prefixIcon: FluentIconGlyph.searchGlyph
                                Layout.fillWidth: true
                            }
                            
                            QDTextField {
                                placeholderText: "带后缀图标"
                                suffixIcon: FluentIconGlyph.settingsGlyph
                                Layout.fillWidth: true
                            }
                            
                            QDTextField {
                                placeholderText: "密码输入"
                                echoMode: TextInput.Password
                                prefixIcon: FluentIconGlyph.lockGlyph
                                Layout.fillWidth: true
                            }
                            
                            QDTextField {
                                text: "错误状态"
                                error: true
                                errorText: "输入不合法"
                                Layout.fillWidth: true
                            }
                            
                            QDTextField {
                                placeholderText: "禁用状态"
                                enabled: false
                                Layout.fillWidth: true
                            }
                        }
                    }
                }
                
                // ============ Checkboxes Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "复选框 (CheckBoxes)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: checkboxContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: checkboxContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingMedium
                            
                            QDCheckBox {
                                text: "默认复选框"
                            }
                            
                            QDCheckBox {
                                text: "已选中"
                                checked: true
                            }
                            
                            QDCheckBox {
                                text: "部分选中"
                                checkState: Qt.PartiallyChecked
                            }
                            
                            QDCheckBox {
                                text: "禁用状态"
                                enabled: false
                            }
                            
                            QDCheckBox {
                                text: "禁用且选中"
                                enabled: false
                                checked: true
                            }
                        }
                    }
                }
                
                // ============ Cards Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "卡片 (Cards)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    GridLayout {
                        Layout.fillWidth: true
                        columns: 3
                        rowSpacing: Theme.spacingLarge
                        columnSpacing: Theme.spacingLarge
                        
                        QDCard {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 150
                            elevation: 1
                            
                            ColumnLayout {
                                anchors.centerIn: parent
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: FluentIconGlyph.cloudGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 32
                                    color: Theme.accent
                                    Layout.alignment: Qt.AlignHCenter
                                }
                                
                                Text {
                                    text: "普通卡片"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                    Layout.alignment: Qt.AlignHCenter
                                }
                            }
                        }
                        
                        QDCard {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 150
                            elevation: 2
                            hoverable: true
                            
                            ColumnLayout {
                                anchors.centerIn: parent
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: FluentIconGlyph.heartGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 32
                                    color: Theme.error
                                    Layout.alignment: Qt.AlignHCenter
                                }
                                
                                Text {
                                    text: "可悬停卡片"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                    Layout.alignment: Qt.AlignHCenter
                                }
                            }
                        }
                        
                        QDCard {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 150
                            elevation: 3
                            hoverable: true
                            clickable: true
                            
                            onClicked: globalToast.show("卡片被点击了", QDToast.Type.Info)
                            
                            ColumnLayout {
                                anchors.centerIn: parent
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: FluentIconGlyph.clickGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 32
                                    color: Theme.success
                                    Layout.alignment: Qt.AlignHCenter
                                }
                                
                                Text {
                                    text: "可点击卡片"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                    Layout.alignment: Qt.AlignHCenter
                                }
                            }
                        }
                    }
                }
                
                // ============ Toast & Dialog Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "提示与对话框 (Toast & Dialog)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: toastContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: toastContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            Text {
                                text: "Toast 消息提示"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            RowLayout {
                                spacing: Theme.spacingMedium
                                
                                QDButton {
                                    text: "Success"
                                    buttonType: QDButton.Type.Success
                                    onClicked: globalToast.show("操作成功完成！", QDToast.Type.Success)
                                }
                                
                                QDButton {
                                    text: "Error"
                                    buttonType: QDButton.Type.Danger
                                    onClicked: globalToast.show("发生错误，请重试", QDToast.Type.Error)
                                }
                                
                                QDButton {
                                    text: "Warning"
                                    buttonType: QDButton.Type.Secondary
                                    iconText: FluentIconGlyph.warningGlyph
                                    onClicked: globalToast.show("这是一个警告消息", QDToast.Type.Warning)
                                }
                                
                                QDButton {
                                    text: "Info"
                                    buttonType: QDButton.Type.Primary
                                    onClicked: globalToast.show("这是一条普通信息", QDToast.Type.Info)
                                }
                            }
                            
                            Rectangle {
                                Layout.fillWidth: true
                                height: 1
                                color: Theme.border
                            }
                            
                            Text {
                                text: "Dialog 对话框"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            QDButton {
                                text: "打开对话框"
                                buttonType: QDButton.Type.Primary
                                iconText: FluentIconGlyph.openPaneGlyph
                                onClicked: exampleDialog.show()
                            }
                        }
                    }
                }
                
                // ============ Switch & Slider Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "开关与滑块 (Switch & Slider)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: switchContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: switchContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            Text {
                                text: "Switch 开关"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            RowLayout {
                                spacing: Theme.spacingXLarge
                                
                                QDSwitch {
                                    text: "默认开关"
                                }
                                
                                QDSwitch {
                                    text: "已开启"
                                    checked: true
                                }
                                
                                QDSwitch {
                                    text: "禁用"
                                    enabled: false
                                }
                            }
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                            
                            Text {
                                text: "Slider 滑块"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            ColumnLayout {
                                spacing: Theme.spacingMedium
                                
                                RowLayout {
                                    spacing: Theme.spacingMedium
                                    
                                    Text {
                                        text: "音量:"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }
                                    
                                    QDSlider {
                                        id: volumeSlider
                                        Layout.fillWidth: true
                                        from: 0
                                        to: 100
                                        value: 50
                                        showValue: true
                                        decimals: 0
                                    }
                                    
                                    Text {
                                        text: Math.round(volumeSlider.value) + "%"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.textSecondary
                                        Layout.preferredWidth: 40
                                    }
                                }
                                
                                QDSlider {
                                    Layout.fillWidth: true
                                    enabled: false
                                    value: 0.3
                                }
                            }
                        }
                    }
                }
                
                // ============ Progress & ComboBox Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "进度条与下拉框 (Progress & ComboBox)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: progressContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: progressContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            Text {
                                text: "ProgressBar 进度条"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: Theme.spacingMedium
                                
                                RowLayout {
                                    spacing: Theme.spacingMedium
                                    
                                    Text {
                                        text: "下载进度:"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }
                                    
                                    QDProgressBar {
                                        Layout.fillWidth: true
                                        value: progressTimer.progress
                                        
                                        Timer {
                                            id: progressTimer
                                            property real progress: 0
                                            interval: 50
                                            repeat: true
                                            running: true
                                            onTriggered: {
                                                progress += 0.005
                                                if (progress > 1) progress = 0
                                            }
                                        }
                                    }
                                    
                                    Text {
                                        text: Math.round(progressTimer.progress * 100) + "%"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.textSecondary
                                        Layout.preferredWidth: 40
                                    }
                                }
                                
                                RowLayout {
                                    spacing: Theme.spacingMedium
                                    
                                    Text {
                                        text: "加载中:"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }
                                    
                                    QDProgressBar {
                                        Layout.fillWidth: true
                                        indeterminate: true
                                    }
                                }
                            }
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                            
                            Text {
                                text: "ComboBox 下拉框"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            RowLayout {
                                spacing: Theme.spacingXLarge
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "选择语言"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDComboBox {
                                        model: ["简体中文", "English", "日本語", "한국어"]
                                    }
                                }
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "选择分辨率"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDComboBox {
                                        model: ["1920x1080", "2560x1440", "3840x2160"]
                                        currentIndex: 1
                                    }
                                }
                            }
                        }
                    }
                }
                
                // ============ Badge Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "徽章 (Badge)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: badgeContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: badgeContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            RowLayout {
                                spacing: Theme.spacingXLarge
                                
                                // Badge with count
                                Item {
                                    width: 60
                                    height: 40
                                    
                                    Rectangle {
                                        anchors.centerIn: parent
                                        width: 40
                                        height: 40
                                        radius: Theme.radiusMedium
                                        color: Theme.surface
                                        border.width: 1
                                        border.color: Theme.border
                                        
                                        Text {
                                            anchors.centerIn: parent
                                            text: FluentIconGlyph.mailGlyph
                                            font.family: "Segoe Fluent Icons"
                                            font.pixelSize: 20
                                            color: Theme.text
                                        }
                                    }
                                    
                                    QDBadge {
                                        anchors.top: parent.top
                                        anchors.right: parent.right
                                        count: 5
                                        badgeType: QDBadge.Type.Error
                                    }
                                }
                                
                                // Badge with large count
                                Item {
                                    width: 60
                                    height: 40
                                    
                                    Rectangle {
                                        anchors.centerIn: parent
                                        width: 40
                                        height: 40
                                        radius: Theme.radiusMedium
                                        color: Theme.surface
                                        border.width: 1
                                        border.color: Theme.border
                                        
                                        Text {
                                            anchors.centerIn: parent
                                            text: FluentIconGlyph.chatBubblesGlyph
                                            font.family: "Segoe Fluent Icons"
                                            font.pixelSize: 20
                                            color: Theme.text
                                        }
                                    }
                                    
                                    QDBadge {
                                        anchors.top: parent.top
                                        anchors.right: parent.right
                                        count: 128
                                        badgeType: QDBadge.Type.Primary
                                    }
                                }
                                
                                // Dot badge
                                Item {
                                    width: 60
                                    height: 40
                                    
                                    Rectangle {
                                        anchors.centerIn: parent
                                        width: 40
                                        height: 40
                                        radius: Theme.radiusMedium
                                        color: Theme.surface
                                        border.width: 1
                                        border.color: Theme.border
                                        
                                        Text {
                                            anchors.centerIn: parent
                                            text: FluentIconGlyph.settingsGlyph
                                            font.family: "Segoe Fluent Icons"
                                            font.pixelSize: 20
                                            color: Theme.text
                                        }
                                    }
                                    
                                    QDBadge {
                                        anchors.top: parent.top
                                        anchors.right: parent.right
                                        dot: true
                                        badgeType: QDBadge.Type.Success
                                    }
                                }
                                
                                // Text badges
                                QDBadge {
                                    text: "New"
                                    badgeType: QDBadge.Type.Success
                                }
                                
                                QDBadge {
                                    text: "Beta"
                                    badgeType: QDBadge.Type.Warning
                                }
                                
                                QDBadge {
                                    text: "Error"
                                    badgeType: QDBadge.Type.Error
                                }
                            }
                        }
                    }
                }
                
                // ============ Radio & Chip & Avatar Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "单选、标签与头像 (Radio, Chip & Avatar)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: radioContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: radioContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            Text {
                                text: "RadioButton 单选按钮"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            ColumnLayout {
                                spacing: Theme.spacingMedium
                                
                                QQC.ButtonGroup {
                                    id: radioGroup
                                }
                                
                                QDRadioButton {
                                    text: "选项 1"
                                    checked: true
                                    QQC.ButtonGroup.group: radioGroup
                                }
                                
                                QDRadioButton {
                                    text: "选项 2"
                                    QQC.ButtonGroup.group: radioGroup
                                }
                                
                                QDRadioButton {
                                    text: "选项 3"
                                    QQC.ButtonGroup.group: radioGroup
                                }
                                
                                QDRadioButton {
                                    text: "禁用选项"
                                    enabled: false
                                }
                            }
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                            
                            Text {
                                text: "Chip 标签"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            Flow {
                                Layout.fillWidth: true
                                spacing: Theme.spacingMedium
                                
                                QDChip {
                                    text: "默认"
                                    chipType: QDChip.Type.Default
                                }
                                
                                QDChip {
                                    text: "主要"
                                    chipType: QDChip.Type.Primary
                                }
                                
                                QDChip {
                                    text: "成功"
                                    chipType: QDChip.Type.Success
                                    iconText: FluentIconGlyph.checkMarkGlyph
                                }
                                
                                QDChip {
                                    text: "警告"
                                    chipType: QDChip.Type.Warning
                                    iconText: FluentIconGlyph.warningGlyph
                                }
                                
                                QDChip {
                                    text: "错误"
                                    chipType: QDChip.Type.Error
                                    iconText: FluentIconGlyph.errorGlyph
                                }
                                
                                QDChip {
                                    text: "信息"
                                    chipType: QDChip.Type.Info
                                    iconText: FluentIconGlyph.infoGlyph
                                }
                                
                                QDChip {
                                    text: "可关闭"
                                    chipType: QDChip.Type.Primary
                                    closable: true
                                    onCloseClicked: globalToast.show("标签已关闭", QDToast.Type.Info)
                                }
                                
                                QDChip {
                                    text: "轮廓"
                                    chipType: QDChip.Type.Primary
                                    outlined: true
                                }
                            }
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                            
                            Text {
                                text: "Avatar 头像"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            Row {
                                spacing: Theme.spacingLarge
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDAvatar {
                                        name: "Zhang San"
                                        avatarSize: QDAvatar.Size.Small
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "Small"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDAvatar {
                                        name: "Li Si"
                                        avatarSize: QDAvatar.Size.Medium
                                        backgroundColor: Theme.success
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "Medium"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDAvatar {
                                        name: "Wang Wu"
                                        avatarSize: QDAvatar.Size.Large
                                        backgroundColor: Theme.warning
                                        showBadge: true
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "Large (在线)"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDAvatar {
                                        name: "Zhao Liu"
                                        avatarSize: QDAvatar.Size.XLarge
                                        backgroundColor: Theme.error
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "XLarge"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                            }
                        }
                    }
                }
                
                // ============ Spinner & Menu & MessageBox Section ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "加载、菜单与消息框 (Spinner, Menu & MessageBox)"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: spinnerContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: spinnerContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            Text {
                                text: "Spinner 加载动画"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            Row {
                                spacing: Theme.spacingXLarge
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDSpinner {
                                        size: 24
                                        spinnerColor: Theme.primary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "主色"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDSpinner {
                                        size: 32
                                        spinnerColor: Theme.success
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "成功色"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                                
                                ColumnLayout {
                                    spacing: Theme.spacingSmall
                                    
                                    QDSpinner {
                                        size: 40
                                        spinnerColor: Theme.warning
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                    
                                    Text {
                                        text: "警告色"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        Layout.alignment: Qt.AlignHCenter
                                    }
                                }
                            }
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                            
                            Text {
                                text: "Menu 菜单"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            QDButton {
                                text: "打开菜单"
                                iconText: FluentIconGlyph.moreGlyph
                                buttonType: QDButton.Type.Secondary
                                onClicked: contextMenu.open()
                                
                                QDMenu {
                                    id: contextMenu
                                    x: 0  // 左对齐
                                    y: parent.height
                                    
                                    QDMenuItem {
                                        text: "新建"
                                        onTriggered: globalToast.show("点击了新建", QDToast.Type.Info)
                                    }
                                    
                                    QDMenuItem {
                                        text: "打开"
                                        onTriggered: globalToast.show("点击了打开", QDToast.Type.Info)
                                    }
                                    
                                    QDMenuSeparator { }
                                    
                                    QDMenuItem {
                                        text: "保存"
                                        checkable: true
                                        checked: true
                                    }
                                    
                                    QDMenuItem {
                                        text: "另存为"
                                        checkable: true
                                    }
                                    
                                    QDMenuSeparator { }
                                    
                                    QDMenuItem {
                                        text: "退出"
                                        onTriggered: globalToast.show("点击了退出", QDToast.Type.Warning)
                                    }
                                }
                            }
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                            
                            Text {
                                text: "MessageBox 消息框"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                            }
                            
                            Row {
                                spacing: Theme.spacingMedium
                                
                                QDButton {
                                    text: "信息"
                                    buttonType: QDButton.Type.Primary
                                    onClicked: {
                                        globalMessageBox.title = "信息"
                                        globalMessageBox.message = "这是一条信息消息"
                                        globalMessageBox.messageType = QDMessageBox.Type.Information
                                        globalMessageBox.buttons = QDMessageBox.Buttons.Ok
                                        globalMessageBox.show()
                                    }
                                }
                                
                                QDButton {
                                    text: "警告"
                                    buttonType: QDButton.Type.Secondary
                                    onClicked: {
                                        globalMessageBox.title = "警告"
                                        globalMessageBox.message = "这是一条警告消息"
                                        globalMessageBox.detailMessage = "请注意检查相关设置"
                                        globalMessageBox.messageType = QDMessageBox.Type.Warning
                                        globalMessageBox.buttons = QDMessageBox.Buttons.OkCancel
                                        globalMessageBox.show()
                                    }
                                }
                                
                                QDButton {
                                    text: "错误"
                                    buttonType: QDButton.Type.Danger
                                    onClicked: {
                                        globalMessageBox.title = "错误"
                                        globalMessageBox.message = "操作失败"
                                        globalMessageBox.messageType = QDMessageBox.Type.Error
                                        globalMessageBox.buttons = QDMessageBox.Buttons.Ok
                                        globalMessageBox.show()
                                    }
                                }
                                
                                QDButton {
                                    text: "询问"
                                    buttonType: QDButton.Type.Secondary
                                    onClicked: {
                                        globalMessageBox.title = "确认操作"
                                        globalMessageBox.message = "确定要删除这些文件吗？"
                                        globalMessageBox.messageType = QDMessageBox.Type.Question
                                        globalMessageBox.buttons = QDMessageBox.Buttons.YesNo
                                        globalMessageBox.accepted.connect(function() {
                                            globalToast.show("已确认删除", QDToast.Type.Success)
                                        })
                                        globalMessageBox.show()
                                    }
                                }
                                
                                QDButton {
                                    text: "成功"
                                    buttonType: QDButton.Type.Success
                                    onClicked: {
                                        globalMessageBox.title = "操作成功"
                                        globalMessageBox.message = "所有文件已成功上传"
                                        globalMessageBox.messageType = QDMessageBox.Type.Success
                                        globalMessageBox.buttons = QDMessageBox.Buttons.Ok
                                        globalMessageBox.show()
                                    }
                                }
                            }
                        }
                    }
                }
                
                // ============ Section: TabBar ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "TabBar 标签栏"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: tabBarContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: tabBarContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        // 基础标签栏
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingSmall
                            
                            QDTabBar {
                                id: tabBar
                                width: parent.width
                                
                                QDTabButton {
                                    text: "主页"
                                    iconSource: FluentIconGlyph.homeGlyph
                                }
                                QDTabButton {
                                    text: "设置"
                                    iconSource: FluentIconGlyph.settingsGlyph
                                }
                                QDTabButton {
                                    text: "关于"
                                    iconSource: FluentIconGlyph.infoGlyph
                                }
                            }
                            
                            Rectangle {
                                width: parent.width
                                height: 100
                                color: Theme.surface
                                radius: Theme.radiusMedium
                                border.width: Theme.borderWidthThin
                                border.color: Theme.border
                                
                                Text {
                                    anchors.centerIn: parent
                                    text: {
                                        switch(tabBar.currentIndex) {
                                            case 0: return "主页内容"
                                            case 1: return "设置内容"
                                            case 2: return "关于内容"
                                            default: return ""
                                        }
                                    }
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeLarge
                                    color: Theme.text
                                }
                            }
                        }
                        
                        // 可关闭标签（动态管理）
                        ColumnLayout {
                            id: tabContainer
                            Layout.fillWidth: true
                            spacing: Theme.spacingSmall
                            
                            property var tabList: [
                                { id: 1, text: "文档 1" },
                                { id: 2, text: "文档 2" },
                                { id: 3, text: "文档 3" }
                            ]
                            property int nextTabId: 4
                            
                            signal closeTabRequested(int tabIndex, string tabText)
                            signal addTabRequested()
                            
                            onCloseTabRequested: function(tabIndex, tabText) {
                                var newList = []
                                for (var i = 0; i < tabList.length; i++) {
                                    if (i !== tabIndex) {
                                        newList.push(tabList[i])
                                    }
                                }
                                tabList = newList
                                root.children[0].show("已关闭: " + tabText, 1) // QDToast.Type.Info
                            }
                            
                            onAddTabRequested: {
                                var newList = tabList.slice()
                                newList.push({ 
                                    id: nextTabId, 
                                    text: "文档 " + nextTabId 
                                })
                                tabList = newList
                                nextTabId++
                                root.children[0].show("已创建新标签", 0) // QDToast.Type.Success
                            }
                            
                            RowLayout {
                                Layout.fillWidth: true
                                spacing: Theme.spacingSmall
                                
                                QDTabBar {
                                    id: dynamicTabBar
                                    Layout.fillWidth: true
                                    
                                    Repeater {
                                        model: tabContainer.tabList.length
                                        
                                        QDTabButton {
                                            required property int index
                                            text: tabContainer.tabList[index].text
                                            showCloseButton: true
                                            onCloseClicked: {
                                                tabContainer.closeTabRequested(index, text)
                                            }
                                        }
                                    }
                                }
                                
                                QDIconButton {
                                    iconSource: FluentIconGlyph.addGlyph
                                    buttonStyle: QDIconButton.Style.Subtle
                                    buttonSize: QDIconButton.Size.Small
                                    onClicked: {
                                        tabContainer.addTabRequested()
                                    }
                                    QDToolTip { text: "添加新标签" }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: IconButton ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "IconButton 图标按钮"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: iconButtonContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: iconButtonContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        // 不同样式
                        Row {
                            spacing: Theme.spacingLarge
                            
                            Column {
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "Standard"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                                
                                QDIconButton {
                                    iconSource: FluentIconGlyph.heartGlyph
                                    buttonStyle: QDIconButton.Style.Standard
                                    onClicked: globalToast.show("Standard 样式", QDToast.Type.Info)
                                }
                            }
                            
                            Column {
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "Subtle"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                                
                                QDIconButton {
                                    iconSource: FluentIconGlyph.favoriteStarGlyph
                                    buttonStyle: QDIconButton.Style.Subtle
                                    onClicked: globalToast.show("Subtle 样式", QDToast.Type.Info)
                                }
                            }
                            
                            Column {
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "Accent"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                                
                                QDIconButton {
                                    iconSource: FluentIconGlyph.likeGlyph
                                    buttonStyle: QDIconButton.Style.Accent
                                    iconColor: Theme.textOnPrimary
                                    onClicked: globalToast.show("Accent 样式", QDToast.Type.Success)
                                }
                            }
                            
                            Column {
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "Transparent"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                                
                                QDIconButton {
                                    iconSource: FluentIconGlyph.moreGlyph
                                    buttonStyle: QDIconButton.Style.Transparent
                                    onClicked: globalToast.show("Transparent 样式", QDToast.Type.Info)
                                }
                            }
                        }
                        
                        // 不同尺寸
                        Row {
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: "尺寸: "
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            
                            QDIconButton {
                                iconSource: FluentIconGlyph.addGlyph
                                buttonSize: QDIconButton.Size.Small
                                buttonStyle: QDIconButton.Style.Accent
                                iconColor: Theme.textOnPrimary
                            }
                            
                            QDIconButton {
                                iconSource: FluentIconGlyph.addGlyph
                                buttonSize: QDIconButton.Size.Medium
                                buttonStyle: QDIconButton.Style.Accent
                                iconColor: Theme.textOnPrimary
                            }
                            
                            QDIconButton {
                                iconSource: FluentIconGlyph.addGlyph
                                buttonSize: QDIconButton.Size.Large
                                buttonStyle: QDIconButton.Style.Accent
                                iconColor: Theme.textOnPrimary
                            }
                        }
                        
                        // 圆形按钮
                        Row {
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: "圆形: "
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            
                            QDIconButton {
                                iconSource: FluentIconGlyph.playGlyph
                                buttonStyle: QDIconButton.Style.Accent
                                iconColor: Theme.textOnPrimary
                                circular: true
                            }
                            
                            QDIconButton {
                                iconSource: FluentIconGlyph.pauseGlyph
                                buttonStyle: QDIconButton.Style.Standard
                                circular: true
                            }
                            
                            QDIconButton {
                                iconSource: FluentIconGlyph.stopGlyph
                                buttonStyle: QDIconButton.Style.Subtle
                                circular: true
                            }
                        }
                    }
                }
                }
                
                // ============ Section: Drawer ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Drawer 抽屉"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: drawerContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: drawerContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Row {
                            spacing: Theme.spacingMedium
                            
                            QDButton {
                                text: "从左侧打开"
                                iconText: FluentIconGlyph.globalNavButtonGlyph
                                onClicked: leftDrawer.open()
                            }
                            
                            QDButton {
                                text: "从右侧打开"
                                iconText: FluentIconGlyph.globalNavButtonGlyph
                                onClicked: rightDrawer.open()
                            }
                        }
                    }
                }
                }
                
                // ============ Section: ScrollBar ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "ScrollBar 滚动条"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: scrollBarContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: scrollBarContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Rectangle {
                            Layout.fillWidth: true
                            height: 200
                            color: Theme.surface
                            radius: Theme.radiusMedium
                            border.width: Theme.borderWidthThin
                            border.color: Theme.border
                            
                            ListView {
                                anchors.fill: parent
                                anchors.margins: Theme.spacingSmall
                                clip: true
                                model: 20
                                spacing: Theme.spacingSmall
                                
                                QQC.ScrollBar.vertical: QDScrollBar {
                                    policy: QQC.ScrollBar.AsNeeded
                                }
                                
                                delegate: Rectangle {
                                    width: ListView.view.width
                                    height: 40
                                    color: index % 2 === 0 ? Theme.surfaceVariant : Theme.surface
                                    radius: Theme.radiusSmall
                                    
                                    Text {
                                        anchors.centerIn: parent
                                        text: "列表项 " + (index + 1)
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: ListView ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "ListView 列表视图"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: listViewContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: listViewContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Rectangle {
                            Layout.fillWidth: true
                            height: 300
                            color: Theme.surface
                            radius: Theme.radiusMedium
                            border.width: Theme.borderWidthThin
                            border.color: Theme.border
                            
                            QDListView {
                                anchors.fill: parent
                                anchors.margins: Theme.spacingSmall
                                
                                model: ListModel {
                                    ListElement { 
                                        icon: "\uE80F"  // Home
                                        title: "主页"
                                        subtitle: "应用程序主页"
                                        trailing: ""
                                    }
                                    ListElement { 
                                        icon: "\uE713"  // Settings
                                        title: "设置"
                                        subtitle: "配置应用程序选项"
                                        trailing: ""
                                    }
                                    ListElement { 
                                        icon: "\uE946"  // Info
                                        title: "关于"
                                        subtitle: "查看版本信息和帮助"
                                        trailing: ""
                                    }
                                    ListElement { 
                                        icon: "\uE8F1"  // User
                                        title: "用户资料"
                                        subtitle: "管理个人信息"
                                        trailing: "编辑"
                                    }
                                    ListElement { 
                                        icon: "\uE7EE"  // Notification
                                        title: "通知"
                                        subtitle: "消息和提醒设置"
                                        trailing: "3"
                                    }
                                    ListElement { 
                                        icon: "\uE72E"  // Shield
                                        title: "安全与隐私"
                                        subtitle: "保护您的数据安全"
                                        trailing: ""
                                    }
                                    ListElement { 
                                        icon: "\uE897"  // Help
                                        title: "帮助与反馈"
                                        subtitle: "获取帮助或提交反馈"
                                        trailing: ""
                                    }
                                }
                                
                                delegate: QDListItem {
                                    iconSource: model.icon
                                    iconColor: Theme.primary
                                    text: model.title
                                    subtitle: model.subtitle
                                    trailing: model.trailing
                                    showChevron: true
                                    showSeparator: index < 6
                                    
                                    onClicked: {
                                        globalToast.show("点击: " + model.title, QDToast.Type.Info)
                                    }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: Breadcrumb ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Breadcrumb 面包屑导航"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: breadcrumbContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: breadcrumbContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        // 基础面包屑
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: "基础面包屑（带图标）:"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                            }
                            
                            QDBreadcrumb {
                                items: [
                                    {icon: FluentIconGlyph.homeGlyph, text: "首页"},
                                    {text: "文档"},
                                    {text: "组件"},
                                    {text: "面包屑"}
                                ]
                                onItemClicked: function(index, text) {
                                    globalToast.show("点击: " + text + " (索引: " + index + ")", QDToast.Type.Info)
                                }
                            }
                        }
                        
                        // 简单面包屑
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: "简单文本面包屑:"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                            }
                            
                            QDBreadcrumb {
                                items: ["项目", "源代码", "组件", "QDBreadcrumb.qml"]
                                onItemClicked: function(index, text) {
                                    globalToast.show("返回: " + text, QDToast.Type.Info)
                                }
                            }
                        }
                        
                        // 自定义分隔符
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: "自定义分隔符:"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                            }
                            
                            QDBreadcrumb {
                                items: ["C:", "Users", "Documents", "Projects"]
                                separator: "\\"
                                showSeparatorIcon: false
                                onItemClicked: function(index, text) {
                                    globalToast.show("打开: " + text, QDToast.Type.Info)
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: EmptyState ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "EmptyState 空状态"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: emptyStateContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: emptyStateContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Row {
                            Layout.fillWidth: true
                            spacing: Theme.spacingLarge
                            
                            // 无数据
                            Rectangle {
                                width: 300
                                height: 350
                                color: Theme.surface
                                radius: Theme.radiusMedium
                                border.width: Theme.borderWidthThin
                                border.color: Theme.border
                                
                                QDEmptyState {
                                    anchors.fill: parent
                                    iconSource: FluentIconGlyph.folderGlyph
                                    iconColor: Theme.textSecondary
                                    title: "暂无文件"
                                    description: "这个文件夹是空的，点击下方按钮添加文件"
                                    actionText: "添加文件"
                                    onActionClicked: globalToast.show("添加文件", QDToast.Type.Info)
                                }
                            }
                            
                            // 无搜索结果
                            Rectangle {
                                width: 300
                                height: 350
                                color: Theme.surface
                                radius: Theme.radiusMedium
                                border.width: Theme.borderWidthThin
                                border.color: Theme.border
                                
                                QDEmptyState {
                                    anchors.fill: parent
                                    iconSource: FluentIconGlyph.searchGlyph
                                    iconColor: Theme.textSecondary
                                    title: "无搜索结果"
                                    description: "找不到匹配的项目，请尝试其他搜索词"
                                    actionText: "清除搜索"
                                    onActionClicked: globalToast.show("清除搜索", QDToast.Type.Info)
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: Separator ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Separator 分隔线"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: separatorContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: separatorContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        // 垂直分隔线
                        Row {
                            spacing: 0
                            height: 100
                            
                            Rectangle {
                                width: 150
                                height: parent.height
                                color: Theme.surfaceVariant
                                radius: Theme.radiusSmall
                                
                                Text {
                                    anchors.centerIn: parent
                                    text: "左侧内容"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                            }
                            
                            QDSeparator {
                                orientation: QDSeparator.Orientation.Vertical
                                height: parent.height
                            }
                            
                            Rectangle {
                                width: 150
                                height: parent.height
                                color: Theme.surfaceVariant
                                radius: Theme.radiusSmall
                                
                                Text {
                                    anchors.centerIn: parent
                                    text: "中间内容"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                            }
                            
                            QDSeparator {
                                orientation: QDSeparator.Orientation.Vertical
                                height: parent.height
                            }
                            
                            Rectangle {
                                width: 150
                                height: parent.height
                                color: Theme.surfaceVariant
                                radius: Theme.radiusSmall
                                
                                Text {
                                    anchors.centerIn: parent
                                    text: "右侧内容"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                            }
                        }
                        
                        Text {
                            text: "水平分隔线（使用 QDDivider）:"
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                        }
                        
                        QDDivider {
                            Layout.fillWidth: true
                        }
                        
                        Text {
                            text: "QDSeparator 也支持水平方向"
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }
                        
                        QDSeparator {
                            Layout.fillWidth: true
                            orientation: QDSeparator.Orientation.Horizontal
                        }
                    }
                }
                }
                
                // ============ Section: Table ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Table 表格"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: tableContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: tableContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        QDTable {
                            Layout.fillWidth: true
                            height: 300
                            
                            columns: [
                                {title: "姓名", role: "name", width: 120},
                                {title: "职位", role: "position"},
                                {title: "部门", role: "department"},
                                {title: "状态", role: "status", width: 80}
                            ]
                            
                            model: ListModel {
                                ListElement { name: "张三"; position: "前端工程师"; department: "技术部"; status: "在线" }
                                ListElement { name: "李四"; position: "后端工程师"; department: "技术部"; status: "离线" }
                                ListElement { name: "王五"; position: "UI设计师"; department: "设计部"; status: "在线" }
                                ListElement { name: "赵六"; position: "产品经理"; department: "产品部"; status: "忙碌" }
                                ListElement { name: "孙七"; position: "测试工程师"; department: "技术部"; status: "在线" }
                                ListElement { name: "周八"; position: "运维工程师"; department: "技术部"; status: "离线" }
                                ListElement { name: "吴九"; position: "交互设计师"; department: "设计部"; status: "在线" }
                                ListElement { name: "郑十"; position: "架构师"; department: "技术部"; status: "忙碌" }
                            }
                            
                            showHeader: true
                            showBorder: true
                            alternatingRowColors: true
                            
                            onRowClicked: function(row, rowData) {
                                globalToast.show("选中: " + rowData.name + " - " + rowData.position, QDToast.Type.Info)
                            }
                        }
                    }
                }
                }
                
                // ============ Section: Accordion ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Accordion 折叠面板"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: accordionContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: accordionContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingSmall
                            
                            QDAccordion {
                                width: parent.width
                                title: "基本信息"
                                iconSource: FluentIconGlyph.infoGlyph
                                expanded: true
                                
                                Column {
                                    width: parent.width
                                    spacing: Theme.spacingMedium
                                    
                                    Row {
                                        width: parent.width
                                        spacing: Theme.spacingMedium
                                        
                                        Text {
                                            text: "姓名:"
                                            font.family: Theme.fontFamily
                                            font.pixelSize: Theme.fontSizeMedium
                                            color: Theme.textSecondary
                                            width: 80
                                        }
                                        
                                        QDTextField {
                                            placeholderText: "请输入姓名"
                                            width: parent.width - 80 - parent.spacing
                                        }
                                    }
                                    
                                    Row {
                                        width: parent.width
                                        spacing: Theme.spacingMedium
                                        
                                        Text {
                                            text: "邮箱:"
                                            font.family: Theme.fontFamily
                                            font.pixelSize: Theme.fontSizeMedium
                                            color: Theme.textSecondary
                                            width: 80
                                        }
                                        
                                        QDTextField {
                                            placeholderText: "请输入邮箱"
                                            width: parent.width - 80 - parent.spacing
                                        }
                                    }
                                }
                            }
                            
                            QDAccordion {
                                width: parent.width
                                title: "高级设置"
                                iconSource: FluentIconGlyph.settingsGlyph
                                
                                Column {
                                    width: parent.width
                                    spacing: Theme.spacingMedium
                                    
                                    Row {
                                        width: parent.width
                                        spacing: Theme.spacingMedium
                                        
                                        Text {
                                            text: "自动保存:"
                                            font.family: Theme.fontFamily
                                            font.pixelSize: Theme.fontSizeMedium
                                            color: Theme.text
                                            width: parent.width - autoSaveSwitch.width - parent.spacing
                                        }
                                        
                                        QDSwitch {
                                            id: autoSaveSwitch
                                            checked: true
                                        }
                                    }
                                    
                                    Row {
                                        width: parent.width
                                        spacing: Theme.spacingMedium
                                        
                                        Text {
                                            text: "通知提醒:"
                                            font.family: Theme.fontFamily
                                            font.pixelSize: Theme.fontSizeMedium
                                            color: Theme.text
                                            width: parent.width - notificationSwitch.width - parent.spacing
                                        }
                                        
                                        QDSwitch {
                                            id: notificationSwitch
                                            checked: false
                                        }
                                    }
                                    
                                    QDDivider {
                                        width: parent.width
                                    }
                                    
                                    Column {
                                        width: parent.width
                                        spacing: Theme.spacingSmall
                                        
                                        Text {
                                            text: "音量调节:"
                                            font.family: Theme.fontFamily
                                            font.pixelSize: Theme.fontSizeMedium
                                            color: Theme.text
                                        }
                                        
                                        QDSlider {
                                            width: parent.width
                                            value: 0.7
                                        }
                                    }
                                }
                            }
                            
                            QDAccordion {
                                width: parent.width
                                title: "关于"
                                iconSource: FluentIconGlyph.infoGlyph
                                
                                Column {
                                    width: parent.width
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "QuickDesk v1.0.0"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        font.weight: Font.DemiBold
                                        color: Theme.text
                                    }
                                    
                                    Text {
                                        text: "现代化的远程控制应用"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        wrapMode: Text.WordWrap
                                        width: parent.width
                                    }
                                    
                                    Text {
                                        text: "© 2026 QuickDesk Team"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: ContextMenu ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "ContextMenu 右键菜单"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: contextMenuContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: contextMenuContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Rectangle {
                            Layout.fillWidth: true
                            height: 200
                            color: Theme.surfaceVariant
                            radius: Theme.radiusMedium
                            
                            Column {
                                anchors.centerIn: parent
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: FluentIconGlyph.touchPointerGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 48
                                    color: Theme.primary
                                    anchors.horizontalCenter: parent.horizontalCenter
                                }
                                
                                Text {
                                    text: "在此区域右键点击"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Text {
                                    text: "尝试右键菜单功能"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                            }
                            
                            MouseArea {
                                anchors.fill: parent
                                acceptedButtons: Qt.RightButton
                                onClicked: function(mouse) {
                                    contextMenu2.x = mouse.x
                                    contextMenu2.y = mouse.y
                                    contextMenu2.open()
                                }
                            }
                            
                            QDContextMenu {
                                id: contextMenu2
                                
                                menuItems: [
                                    QDMenuItem {
                                        text: "复制"
                                        onTriggered: globalToast.show("复制", QDToast.Type.Info)
                                    },
                                    QDMenuItem {
                                        text: "粘贴"
                                        onTriggered: globalToast.show("粘贴", QDToast.Type.Info)
                                    },
                                    QDMenuSeparator {},
                                    QDMenuItem {
                                        text: "删除"
                                        onTriggered: globalToast.show("删除", QDToast.Type.Warning)
                                    },
                                    QDMenuSeparator {},
                                    QDMenuItem {
                                        text: "属性"
                                        onTriggered: globalToast.show("属性", QDToast.Type.Info)
                                    }
                                ]
                            }
                        }
                    }
                }
                }
                
                // ============ Section: StatusBar ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "StatusBar 状态栏"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: statusBarContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: statusBarContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium
                            
                            Text {
                                text: "基础状态栏:"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                            }
                            
                            QDStatusBar {
                                width: parent.width
                                message: "就绪"
                                rightText: "第 1 行，第 1 列"
                            }
                            
                            Text {
                                text: "带左侧文本:"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                            }
                            
                            QDStatusBar {
                                width: parent.width
                                leftText: "已连接"
                                message: "正在同步..."
                                rightText: "10:30 AM"
                            }
                            
                            Text {
                                text: "自定义内容:"
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                            }
                            
                            QDStatusBar {
                                width: parent.width
                                leftText: "QuickDesk"
                                
                                Row {
                                    spacing: Theme.spacingMedium
                                    
                                    QDIconButton {
                                        buttonSize: QDIconButton.Size.Small
                                        iconSource: FluentIconGlyph.wifiGlyph
                                        buttonStyle: QDIconButton.Style.Transparent
                                        QDToolTip { text: "网络连接" }
                                    }
                                    
                                    QDSeparator {
                                        orientation: QDSeparator.Orientation.Vertical
                                        height: Theme.buttonHeightSmall
                                    }
                                    
                                    QDIconButton {
                                        buttonSize: QDIconButton.Size.Small
                                        iconSource: FluentIconGlyph.volumeGlyph
                                        buttonStyle: QDIconButton.Style.Transparent
                                        QDToolTip { text: "音量" }
                                    }
                                    
                                    QDSeparator {
                                        orientation: QDSeparator.Orientation.Vertical
                                        height: Theme.buttonHeightSmall
                                    }
                                    
                                    QDIconButton {
                                        buttonSize: QDIconButton.Size.Small
                                        iconSource: FluentIconGlyph.batteryUnknownGlyph
                                        buttonStyle: QDIconButton.Style.Transparent
                                        QDToolTip { text: "电量 85%" }
                                    }
                                }
                                
                                rightText: "v1.0.0"
                            }
                        }
                    }
                }
                }
                
                // ============ Section: ToggleButtonGroup ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "ToggleButtonGroup 切换按钮组"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: toggleButtonGroupContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: toggleButtonGroupContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingLarge
                            
                            // 基础切换组
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "视图切换:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDToggleButtonGroup {
                                    options: [
                                        {icon: FluentIconGlyph.viewGlyph, text: "列表"},
                                        {icon: FluentIconGlyph.gridViewGlyph, text: "网格"},
                                        {icon: FluentIconGlyph.detailsGlyph, text: "详情"}
                                    ]
                                    onValueChanged: function(value) {
                                        globalToast.show("切换到: " + value, QDToast.Type.Info)
                                    }
                                }
                            }
                            
                            // 纯文本切换组
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "排序方式:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDToggleButtonGroup {
                                    options: ["名称", "日期", "大小", "类型"]
                                    currentIndex: 0
                                    onValueChanged: function(value) {
                                        globalToast.show("排序: " + value, QDToast.Type.Info)
                                    }
                                }
                            }
                            
                            // 不同尺寸
                            Row {
                                spacing: Theme.spacingLarge
                                
                                Column {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "小号"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDToggleButtonGroup {
                                        buttonSize: QDToggleButtonGroup.Size.Small
                                        options: ["选项1", "选项2", "选项3"]
                                    }
                                }
                                
                                Column {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "中号"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDToggleButtonGroup {
                                        buttonSize: QDToggleButtonGroup.Size.Medium
                                        options: ["选项1", "选项2", "选项3"]
                                    }
                                }
                                
                                Column {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "大号"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDToggleButtonGroup {
                                        buttonSize: QDToggleButtonGroup.Size.Large
                                        options: ["选项1", "选项2", "选项3"]
                                    }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: Rating ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Rating 评分组件"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: ratingContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: ratingContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingLarge
                            
                            // 基础评分
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Row {
                                    spacing: Theme.spacingMedium
                                    
                                    Text {
                                        text: "评分:"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                        anchors.verticalCenter: parent.verticalCenter
                                    }
                                    
                                    QDRating {
                                        id: rating1
                                        value: 3.5
                                        onRatingChanged: function(newValue) {
                                            ratingText.text = newValue.toFixed(1) + " 分"
                                        }
                                    }
                                    
                                    Text {
                                        id: ratingText
                                        text: "3.5 分"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.textSecondary
                                        anchors.verticalCenter: parent.verticalCenter
                                    }
                                }
                            }
                            
                            // 只读评分
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "商品评价 (只读):"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Row {
                                    spacing: Theme.spacingMedium
                                    
                                    QDRating {
                                        value: 4.5
                                        readOnly: true
                                    }
                                    
                                    Text {
                                        text: "4.5 分 (128 评价)"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        anchors.verticalCenter: parent.verticalCenter
                                    }
                                }
                            }
                            
                            // 不同尺寸
                            Row {
                                spacing: Theme.spacingLarge
                                
                                Column {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "小号"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDRating {
                                        ratingSize: QDRating.Size.Small
                                        value: 4
                                    }
                                }
                                
                                Column {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "中号"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDRating {
                                        ratingSize: QDRating.Size.Medium
                                        value: 4
                                    }
                                }
                                
                                Column {
                                    spacing: Theme.spacingSmall
                                    
                                    Text {
                                        text: "大号"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                    
                                    QDRating {
                                        ratingSize: QDRating.Size.Large
                                        value: 4
                                    }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: Pagination ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "Pagination 分页组件"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: paginationContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: paginationContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingLarge
                            
                            // 基础分页
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "基础分页:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDPagination {
                                    total: 100
                                    current: 1
                                    pageSize: 10
                                    showSizeChanger: false
                                    
                                    onPageChanged: function(page) {
                                        globalToast.show("切换到第 " + page + " 页", QDToast.Type.Info)
                                    }
                                }
                            }
                            
                            // 完整功能
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "完整功能 (可切换每页条数):"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDPagination {
                                    total: 500
                                    current: 5
                                    pageSize: 20
                                    showSizeChanger: true
                                    pageSizeOptions: [10, 20, 50, 100]
                                    
                                    onPageChanged: function(page) {
                                        globalToast.show("第 " + page + " 页", QDToast.Type.Info)
                                    }
                                    
                                    onPageSizeChanged: function(size) {
                                        globalToast.show("每页 " + size + " 条", QDToast.Type.Info)
                                    }
                                }
                            }
                        }
                    }
                }
                }
                
                // ============ Section: DateTimePicker ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "DateTimePicker 日期时间选择器"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: dateTimePickerContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: dateTimePickerContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            QDDivider {
                                Layout.fillWidth: true
                            }
                        
                        Column {
                            Layout.fillWidth: true
                            spacing: Theme.spacingLarge
                            
                            // 日期选择器
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "日期选择器:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDDateTimePicker {
                                    width: 280
                                    showTime: false
                                    
                                    onDateChanged: function(newDate) {
                                        globalToast.show("选择日期: " + Qt.formatDate(newDate, "yyyy-MM-dd"), QDToast.Type.Info)
                                    }
                                }
                            }
                            
                            // 日期时间选择器
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "日期时间选择器:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDDateTimePicker {
                                    width: 280
                                    showTime: true
                                    
                                    onDateChanged: function(newDate) {
                                        globalToast.show("选择: " + Qt.formatDateTime(newDate, "yyyy-MM-dd HH:mm"), QDToast.Type.Info)
                                    }
                                }
                            }
                        }
                    }
                }
                
                // ============ Section: TextArea ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "TextArea 多行文本输入框"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: textAreaContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: textAreaContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            // 基础 TextArea
                            Column {
                                Layout.fillWidth: true
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "基础多行输入:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDTextArea {
                                    Layout.fillWidth: true
                                    height: 120
                                    placeholderText: "请输入多行文本..."
                                }
                            }
                            
                            // 带错误状态
                            Column {
                                Layout.fillWidth: true
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "错误状态:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDTextArea {
                                    Layout.fillWidth: true
                                    height: 100
                                    hasError: true
                                    placeholderText: "这里有错误..."
                                    text: "输入内容不符合要求"
                                }
                            }
                            
                            // 禁用状态
                            Column {
                                width: parent.width
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "禁用状态:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDTextArea {
                                    Layout.fillWidth: true
                                    height: 80
                                    enabled: false
                                    text: "这是只读的文本内容"
                                }
                            }
                        }
                    }
                }
                
                // ============ Section: SearchBox ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "SearchBox 搜索框"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: searchBoxContent.implicitHeight + Theme.spacingXLarge * 2
                        elevation: 1
                        
                        ColumnLayout {
                            id: searchBoxContent
                            anchors.fill: parent
                            anchors.margins: Theme.spacingXLarge
                            spacing: Theme.spacingLarge
                            
                            // 基础搜索框
                            Column {
                                Layout.fillWidth: true
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "基础搜索框:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDSearchBox {
                                    width: 400
                                    placeholderText: "搜索..."
                                    
                                    onSearchRequested: function(query) {
                                        globalToast.show("搜索: " + query, QDToast.Type.Info)
                                    }
                                }
                            }
                            
                            // 不同尺寸
                            Column {
                                Layout.fillWidth: true
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "不同宽度:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                Row {
                                    spacing: Theme.spacingMedium
                                    
                                    QDSearchBox {
                                        width: 200
                                        placeholderText: "小号..."
                                        onSearchRequested: function(query) {
                                            globalToast.show("搜索: " + query, QDToast.Type.Info)
                                        }
                                    }
                                    
                                    QDSearchBox {
                                        width: 300
                                        placeholderText: "中号..."
                                        onSearchRequested: function(query) {
                                            globalToast.show("搜索: " + query, QDToast.Type.Info)
                                        }
                                    }
                                    
                                    QDSearchBox {
                                        width: 500
                                        placeholderText: "大号..."
                                        onSearchRequested: function(query) {
                                            globalToast.show("搜索: " + query, QDToast.Type.Info)
                                        }
                                    }
                                }
                            }
                            
                            // 带清除功能
                            Column {
                                Layout.fillWidth: true
                                spacing: Theme.spacingSmall
                                
                                Text {
                                    text: "清除按钮（输入内容后显示）:"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }
                                
                                QDSearchBox {
                                    width: 400
                                    placeholderText: "输入文本查看清除按钮..."
                                    text: "示例文本"
                                    
                                    onCleared: {
                                        globalToast.show("已清除搜索", QDToast.Type.Info)
                                    }
                                }
                            }
                        }
                    }
                }
                
                // ============ Section: NavigationView ============
                
                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacingLarge
                    
                    Text {
                        text: "NavigationView 导航视图"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeHeading
                        font.weight: Font.Bold
                        color: Theme.text
                    }
                    
                    QDCard {
                        Layout.fillWidth: true
                        Layout.preferredHeight: 500
                        elevation: 1
                        
                        QDNavigationView {
                            id: navView
                            anchors.fill: parent
                            
                            menuItems: [
                                { icon: FluentIconGlyph.homeGlyph, text: "首页" },
                                { icon: FluentIconGlyph.documentGlyph, text: "文档" },
                                { icon: FluentIconGlyph.favoriteStarGlyph, text: "收藏" },
                                { icon: FluentIconGlyph.mailGlyph, text: "邮件" },
                                { icon: FluentIconGlyph.calendarGlyph, text: "日历" },
                                { icon: FluentIconGlyph.contactGlyph, text: "联系人" },
                                { icon: FluentIconGlyph.settingsGlyph, text: "设置" }
                            ]
                            
                            header: RowLayout {
                                width: parent.width
                                height: 48
                                spacing: Theme.spacingMedium
                                
                                Rectangle {
                                    Layout.leftMargin: Theme.spacingMedium
                                    width: 32
                                    height: 32
                                    color: Theme.primary
                                    radius: 16
                                    
                                    Text {
                                        anchors.centerIn: parent
                                        text: "Q"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: 18
                                        font.bold: true
                                        color: "white"
                                    }
                                }
                                
                                Text {
                                    visible: navView.isExpanded
                                    text: "QuickDesk"
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.bold: true
                                    color: Theme.text
                                }
                            }
                            
                            footer: Rectangle {
                                width: parent.width
                                height: 48
                                color: "transparent"
                                
                                QDButton {
                                    anchors.centerIn: parent
                                    text: navView.isExpanded ? "账户设置" : ""
                                    iconText: FluentIconGlyph.contactGlyph
                                    buttonType: QDButton.Type.Ghost
                                    width: navView.isExpanded ? parent.width - Theme.spacingMedium * 2 : 40
                                    
                                    onClicked: {
                                        globalToast.show("打开账户设置", QDToast.Type.Info)
                                    }
                                }
                            }
                            
                            content: Rectangle {
                                color: Theme.background
                                
                                ColumnLayout {
                                    anchors.fill: parent
                                    anchors.margins: Theme.spacingLarge
                                    spacing: Theme.spacingMedium
                                    
                                    Text {
                                        text: "内容区域"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeXLarge
                                        font.bold: true
                                        color: Theme.text
                                    }
                                    
                                    Text {
                                        id: navContentText
                                        text: "选择左侧菜单项查看详情"
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.textSecondary
                                        wrapMode: Text.WordWrap
                                        Layout.fillWidth: true
                                    }
                                    
                                    Item { Layout.fillHeight: true }
                                }
                            }
                            
                            onItemClicked: function(index, item) {
                                navContentText.text = "当前选中: " + item.text + " (索引: " + index + ")"
                                globalToast.show("点击了 " + item.text, QDToast.Type.Info)
                            }
                            }
                        }
                    }
                }
                }
                
                // Footer
                Item {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 80
                    
                    Text {
                        anchors.centerIn: parent
                        text: "QuickDesk © 2026 - Modern Fluent Design"
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                    }
                }
            }
        }
    }
}