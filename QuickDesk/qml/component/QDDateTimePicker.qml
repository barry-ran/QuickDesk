// Fluent Design DateTimePicker Component
import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import QtQuick.Effects

Item {
    id: control
    
    // ============ Custom Properties ============
    
    property date selectedDate: new Date()
    property bool showTime: false
    property string format: showTime ? "yyyy-MM-dd HH:mm" : "yyyy-MM-dd"
    
    signal dateChanged(date newDate)
    
    // ============ Size & Style ============
    
    implicitWidth: 280
    implicitHeight: Theme.buttonHeightMedium
    
    // ============ Content ============
    
    Rectangle {
        anchors.fill: parent
        color: dateButton.hovered ? Theme.surfaceHover : Theme.surface
        border.width: Theme.borderWidthThin
        border.color: datePopup.opened ? Theme.primary : Theme.border
        radius: Theme.radiusMedium
        
        RowLayout {
            anchors.fill: parent
            anchors.leftMargin: Theme.spacingMedium
            anchors.rightMargin: Theme.spacingMedium
            spacing: Theme.spacingSmall
            
            Text {
                text: FluentIconGlyph.calendarGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.primary
            }
            
            Text {
                text: Qt.formatDateTime(control.selectedDate, control.format)
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.text
                Layout.fillWidth: true
            }
            
            Text {
                text: FluentIconGlyph.chevronDownGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 12
                color: Theme.textSecondary
                rotation: datePopup.opened ? 180 : 0
                
                Behavior on rotation {
                    NumberAnimation {
                        duration: Theme.animationDurationFast
                        easing.type: Easing.OutCubic
                    }
                }
            }
        }
        
        HoverHandler {
            id: dateButton
        }
        
        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: datePopup.open()
        }
        
        Behavior on color {
            ColorAnimation { duration: Theme.animationDurationFast }
        }
    }
    
    // ============ Popup ============
    
    // ============ Popup ============
    
    Controls.Popup {
        id: datePopup
        x: 0  // 左对齐
        y: control.height + Theme.spacingSmall
        width: 320
        height: control.showTime ? 400 : 350
        padding: 0  // 移除默认 padding
        modal: true
        dim: false
        closePolicy: Controls.Popup.CloseOnEscape | Controls.Popup.CloseOnPressOutside
        
        background: Rectangle {
            color: Theme.surface
            border.width: Theme.borderWidthThin
            border.color: Theme.border
            radius: Theme.radiusMedium
            
            layer.enabled: true
            layer.effect: MultiEffect {
                shadowEnabled: true
                shadowHorizontalOffset: 0
                shadowVerticalOffset: 2
                shadowBlur: 0.75
                shadowColor: Qt.rgba(0, 0, 0, 0.3)
            }
        }
        
        contentItem: ColumnLayout {
            id: calendarLayout
            anchors.fill: parent
            anchors.margins: Theme.spacingMedium
            spacing: Theme.spacingMedium
            
            property int currentMonth: control.selectedDate.getMonth()
            property int currentYear: control.selectedDate.getFullYear()
            property var pickerControl: control  // 保存 control 引用
            
            function handleDayClick(year, month, day, isTimeMode) {
                var newDate = new Date(year, month, day)
                control.selectedDate = newDate
                if (!isTimeMode) {
                    control.dateChanged(newDate)
                    datePopup.close()
                }
            }
            
            property var calendarModel: {
                var model = []
                var firstDay = new Date(currentYear, currentMonth, 1)
                var lastDay = new Date(currentYear, currentMonth + 1, 0)
                var startWeekDay = firstDay.getDay()
                
                // Previous month days
                var prevMonthLastDay = new Date(currentYear, currentMonth, 0).getDate()
                for (var i = startWeekDay - 1; i >= 0; i--) {
                    model.push({
                        day: prevMonthLastDay - i,
                        month: currentMonth - 1,
                        year: currentYear,
                        isCurrentMonth: false,
                        isToday: false,
                        isSelected: false
                    })
                }
                
                // Current month days
                var today = new Date()
                for (var j = 1; j <= lastDay.getDate(); j++) {
                    var isToday = j === today.getDate() && 
                                 currentMonth === today.getMonth() && 
                                 currentYear === today.getFullYear()
                    var isSelected = j === control.selectedDate.getDate() && 
                                    currentMonth === control.selectedDate.getMonth() && 
                                    currentYear === control.selectedDate.getFullYear()
                    model.push({
                        day: j,
                        month: currentMonth,
                        year: currentYear,
                        isCurrentMonth: true,
                        isToday: isToday,
                        isSelected: isSelected
                    })
                }
                
                // Next month days
                var remainingDays = 42 - model.length
                for (var k = 1; k <= remainingDays; k++) {
                    model.push({
                        day: k,
                        month: currentMonth + 1,
                        year: currentYear,
                        isCurrentMonth: false,
                        isToday: false,
                        isSelected: false
                    })
                }
                
                return model
            }
            
            // Month/Year selector
            RowLayout {
                Layout.fillWidth: true
                
                QDIconButton {
                    buttonSize: QDIconButton.Size.Small
                    iconSource: FluentIconGlyph.chevronLeftGlyph
                    buttonStyle: QDIconButton.Style.Subtle
                    onClicked: {
                        var newDate = new Date(calendarLayout.currentYear, calendarLayout.currentMonth - 1, 1)
                        calendarLayout.currentMonth = newDate.getMonth()
                        calendarLayout.currentYear = newDate.getFullYear()
                    }
                }
                
                Text {
                    text: calendarLayout.currentYear + " 年 " + (calendarLayout.currentMonth + 1) + " 月"
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeMedium
                    font.weight: Font.DemiBold
                    color: Theme.text
                    Layout.fillWidth: true
                    horizontalAlignment: Text.AlignHCenter
                }
                
                QDIconButton {
                    buttonSize: QDIconButton.Size.Small
                    iconSource: FluentIconGlyph.chevronRightGlyph
                    buttonStyle: QDIconButton.Style.Subtle
                    onClicked: {
                        var newDate = new Date(calendarLayout.currentYear, calendarLayout.currentMonth + 1, 1)
                        calendarLayout.currentMonth = newDate.getMonth()
                        calendarLayout.currentYear = newDate.getFullYear()
                    }
                }
            }
            
            QDDivider {
                Layout.fillWidth: true
            }
            
            // Calendar grid
            GridLayout {
                Layout.fillWidth: true
                columns: 7
                rowSpacing: Theme.spacingSmall
                columnSpacing: Theme.spacingSmall
                
                // Week day headers
                Repeater {
                    model: ["日", "一", "二", "三", "四", "五", "六"]
                    
                    Text {
                        text: modelData
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.textSecondary
                        horizontalAlignment: Text.AlignHCenter
                        Layout.preferredWidth: 36
                        Layout.preferredHeight: 24
                    }
                }
                
                // Calendar days
                Repeater {
                    model: calendarLayout.calendarModel
                    
                    Rectangle {
                        required property var modelData
                        
                        Layout.preferredWidth: 36
                        Layout.preferredHeight: 36
                        color: {
                            if (modelData.isToday) return Qt.rgba(Theme.primary.r, Theme.primary.g, Theme.primary.b, 0.1)
                            if (modelData.isSelected) return Theme.primary
                            if (dayArea.containsMouse) return Theme.surfaceHover
                            return "transparent"
                        }
                        radius: Theme.radiusSmall
                        
                        property var dayData: modelData
                        property bool showTimeMode: control.showTime
                        
                        Text {
                            anchors.centerIn: parent
                            text: parent.dayData.day
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeSmall
                            color: {
                                if (!parent.dayData.isCurrentMonth) return Theme.textDisabled
                                if (parent.dayData.isSelected) return Theme.textOnPrimary
                                if (parent.dayData.isToday) return Theme.primary
                                return Theme.text
                            }
                        }
                        
                        MouseArea {
                            id: dayArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            enabled: parent.dayData.isCurrentMonth
                            
                            onClicked: {
                                calendarLayout.handleDayClick(
                                    parent.dayData.year, 
                                    parent.dayData.month, 
                                    parent.dayData.day,
                                    parent.showTimeMode
                                )
                            }
                        }
                        
                        Behavior on color {
                            ColorAnimation { duration: Theme.animationDurationFast }
                        }
                    }
                }
            }
            
            // Time picker (if showTime is true)
            Row {
                visible: control.showTime
                Layout.alignment: Qt.AlignHCenter
                spacing: Theme.spacingSmall
                
                Text {
                    text: "时间:"
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    anchors.verticalCenter: parent.verticalCenter
                }
                
                QDComboBox {
                    width: 60
                    model: Array.from({length: 24}, (_, i) => String(i).padStart(2, '0'))
                    currentIndex: control.selectedDate.getHours()
                    onCurrentIndexChanged: {
                        var newDate = new Date(control.selectedDate)
                        newDate.setHours(currentIndex)
                        control.selectedDate = newDate
                    }
                }
                
                Text {
                    text: ":"
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.text
                    anchors.verticalCenter: parent.verticalCenter
                }
                
                QDComboBox {
                    width: 60
                    model: Array.from({length: 60}, (_, i) => String(i).padStart(2, '0'))
                    currentIndex: control.selectedDate.getMinutes()
                    onCurrentIndexChanged: {
                        var newDate = new Date(control.selectedDate)
                        newDate.setMinutes(currentIndex)
                        control.selectedDate = newDate
                    }
                }
            }
            
            // Action buttons
            Row {
                visible: control.showTime
                Layout.alignment: Qt.AlignRight
                spacing: Theme.spacingSmall
                
                QDButton {
                    text: "取消"
                    buttonType: QDButton.Type.Secondary
                    onClicked: datePopup.close()
                }
                
                QDButton {
                    text: "确定"
                    buttonType: QDButton.Type.Primary
                    onClicked: {
                        control.dateChanged(control.selectedDate)
                        datePopup.close()
                    }
                }
            }
        }
    }
}
