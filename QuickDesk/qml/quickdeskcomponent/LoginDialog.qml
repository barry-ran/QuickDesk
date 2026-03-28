import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

import "../component"

Popup {
    id: loginDialog

    required property var mainController

    modal: true
    anchors.centerIn: parent
    width: 380
    padding: Theme.spacingXLarge

    background: Rectangle {
        color: Theme.surface
        radius: Theme.radiusLarge
        border.width: Theme.borderWidthThin
        border.color: Theme.border
    }

    property string mode: "login"  // "login", "register", or "sms-login"
    property string errorMessage: ""
    property bool isLoading: false
    property int smsCountdown: 0
    readonly property bool smsEnabled: loginDialog.mainController
                                       ? loginDialog.mainController.authManager.smsEnabled
                                       : false

    Timer {
        id: smsTimer
        interval: 1000
        repeat: true
        onTriggered: {
            loginDialog.smsCountdown--
            if (loginDialog.smsCountdown <= 0) {
                smsTimer.stop()
            }
        }
    }

    onOpened: {
        errorMessage = ""
        usernameField.text = ""
        passwordField.text = ""
        phoneField.text = ""
        emailField.text = ""
        smsPhoneField.text = ""
        smsCodeField.text = ""
        regSmsCodeField.text = ""
        usernameField.forceActiveFocus()
    }

    // Connect to AuthManager signals
    Connections {
        target: loginDialog.mainController ? loginDialog.mainController.authManager : null

        function onLoginSuccess() {
            loginDialog.isLoading = false
            loginDialog.close()
        }

        function onLoginFailed(errorCode, errorMsg) {
            loginDialog.isLoading = false
            loginDialog.errorMessage = loginDialog._translateError(errorCode, errorMsg)
        }

        function onRegisterSuccess() {
            loginDialog.isLoading = false
            loginDialog.mode = "login"
            loginDialog.errorMessage = qsTr("Registration successful! Please login.")
        }

        function onRegisterFailed(errorCode, errorMsg) {
            loginDialog.isLoading = false
            loginDialog.errorMessage = loginDialog._translateError(errorCode, errorMsg)
        }

        function onSmsCodeSent() {
            loginDialog.smsCountdown = 60
            smsTimer.start()
        }

        function onSmsCodeFailed(errorCode, errorMsg) {
            loginDialog.errorMessage = loginDialog._translateError(errorCode, errorMsg)
        }
    }

    ColumnLayout {
        id: contentColumn
        anchors.fill: parent
        spacing: Theme.spacingMedium

        // Title
        Text {
            text: {
                switch (loginDialog.mode) {
                case "login": return qsTr("Login")
                case "register": return qsTr("Register")
                case "sms-login": return qsTr("SMS Login")
                }
            }
            font.pixelSize: Theme.fontSizeXLarge
            font.weight: Font.Bold
            color: Theme.text
            Layout.alignment: Qt.AlignHCenter
        }

        // ---- Username/Password Login ----
        ColumnLayout {
            visible: loginDialog.mode === "login"
            Layout.fillWidth: true
            spacing: Theme.spacingMedium

            QDTextField {
                id: usernameField
                Layout.fillWidth: true
                placeholderText: qsTr("Username")
                enabled: !loginDialog.isLoading
            }

            QDTextField {
                id: passwordField
                Layout.fillWidth: true
                placeholderText: qsTr("Password")
                echoMode: TextInput.Password
                enabled: !loginDialog.isLoading

                Keys.onReturnPressed: confirmAction()
            }
        }

        // ---- SMS Login ----
        ColumnLayout {
            visible: loginDialog.mode === "sms-login"
            Layout.fillWidth: true
            spacing: Theme.spacingMedium

            QDTextField {
                id: smsPhoneField
                Layout.fillWidth: true
                placeholderText: qsTr("Phone number")
                enabled: !loginDialog.isLoading
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingSmall

                QDTextField {
                    id: smsCodeField
                    Layout.fillWidth: true
                    placeholderText: qsTr("Verification code")
                    enabled: !loginDialog.isLoading

                    Keys.onReturnPressed: confirmAction()
                }

                QDButton {
                    text: loginDialog.smsCountdown > 0
                          ? qsTr("%1s").arg(loginDialog.smsCountdown)
                          : qsTr("Send")
                    enabled: !loginDialog.isLoading
                             && smsPhoneField.text.length > 0
                             && loginDialog.smsCountdown <= 0

                    onClicked: {
                        loginDialog.errorMessage = ""
                        loginDialog.mainController.authManager.sendSmsCode(
                            smsPhoneField.text, "login")
                    }
                }
            }
        }

        // ---- Register ----
        ColumnLayout {
            visible: loginDialog.mode === "register"
            Layout.fillWidth: true
            spacing: Theme.spacingMedium

            QDTextField {
                id: regUsernameField
                Layout.fillWidth: true
                placeholderText: qsTr("Username")
                enabled: !loginDialog.isLoading
            }

            QDTextField {
                id: regPasswordField
                Layout.fillWidth: true
                placeholderText: qsTr("Password")
                echoMode: TextInput.Password
                enabled: !loginDialog.isLoading
            }

            Text {
                text: qsTr("At least 8 characters with letters and digits")
                color: Theme.textSecondary
                font.pixelSize: Theme.fontSizeSmall
                Layout.fillWidth: true
            }

            // Phone + SMS code (required when SMS enabled, optional otherwise)
            QDTextField {
                id: phoneField
                Layout.fillWidth: true
                placeholderText: loginDialog.smsEnabled
                                 ? qsTr("Phone number")
                                 : qsTr("Phone (optional)")
                enabled: !loginDialog.isLoading
            }

            RowLayout {
                visible: loginDialog.smsEnabled
                Layout.fillWidth: true
                spacing: Theme.spacingSmall

                QDTextField {
                    id: regSmsCodeField
                    Layout.fillWidth: true
                    placeholderText: qsTr("Verification code")
                    enabled: !loginDialog.isLoading
                }

                QDButton {
                    text: loginDialog.smsCountdown > 0
                          ? qsTr("%1s").arg(loginDialog.smsCountdown)
                          : qsTr("Send")
                    enabled: !loginDialog.isLoading
                             && phoneField.text.length > 0
                             && loginDialog.smsCountdown <= 0

                    onClicked: {
                        loginDialog.errorMessage = ""
                        loginDialog.mainController.authManager.sendSmsCode(
                            phoneField.text, "register")
                    }
                }
            }

            QDTextField {
                id: emailField
                Layout.fillWidth: true
                placeholderText: qsTr("Email (optional)")
                enabled: !loginDialog.isLoading
            }
        }

        // Error message
        Text {
            visible: loginDialog.errorMessage !== ""
            text: loginDialog.errorMessage
            color: loginDialog.errorMessage.indexOf(qsTr("successful")) >= 0
                   ? Theme.success : Theme.error
            font.pixelSize: Theme.fontSizeSmall
            wrapMode: Text.Wrap
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignHCenter
            horizontalAlignment: Text.AlignHCenter
            Layout.minimumHeight: implicitHeight
        }

        Item {
            visible: loginDialog.errorMessage === ""
            Layout.fillWidth: true
            Layout.preferredHeight: Theme.fontSizeSmall + 4
        }

        // Confirm button
        QDButton {
            Layout.fillWidth: true
            text: {
                if (loginDialog.isLoading) return qsTr("Please wait...")
                switch (loginDialog.mode) {
                case "login": return qsTr("Login")
                case "register": return qsTr("Register")
                case "sms-login": return qsTr("Login")
                }
            }
            highlighted: true
            enabled: !loginDialog.isLoading && isFormValid()

            onClicked: confirmAction()
        }

        // Mode switch links
        ColumnLayout {
            Layout.alignment: Qt.AlignHCenter
            spacing: Theme.spacingSmall

            Text {
                visible: loginDialog.mode !== "register"
                text: loginDialog.mode === "login"
                      ? qsTr("Don't have an account? Register")
                      : qsTr("Don't have an account? Register")
                color: Theme.primary
                font.pixelSize: Theme.fontSizeSmall
                Layout.alignment: Qt.AlignHCenter

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        loginDialog.errorMessage = ""
                        loginDialog.mode = "register"
                    }
                }
            }

            Text {
                visible: loginDialog.mode !== "login"
                text: qsTr("Already have an account? Login")
                color: Theme.primary
                font.pixelSize: Theme.fontSizeSmall
                Layout.alignment: Qt.AlignHCenter

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        loginDialog.errorMessage = ""
                        loginDialog.mode = "login"
                    }
                }
            }

            Text {
                visible: loginDialog.smsEnabled && loginDialog.mode === "login"
                text: qsTr("Login with SMS verification code")
                color: Theme.primary
                font.pixelSize: Theme.fontSizeSmall
                Layout.alignment: Qt.AlignHCenter

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        loginDialog.errorMessage = ""
                        loginDialog.mode = "sms-login"
                    }
                }
            }

            Text {
                visible: loginDialog.smsEnabled && loginDialog.mode === "sms-login"
                text: qsTr("Login with username and password")
                color: Theme.primary
                font.pixelSize: Theme.fontSizeSmall
                Layout.alignment: Qt.AlignHCenter

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        loginDialog.errorMessage = ""
                        loginDialog.mode = "login"
                    }
                }
            }

            Text {
                visible: loginDialog.mode === "login" || loginDialog.mode === "sms-login"
                text: qsTr("Forgot password?")
                color: Theme.primary
                font.pixelSize: Theme.fontSizeSmall
                Layout.alignment: Qt.AlignHCenter

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        var webclientUrl = loginDialog.mainController.presetManager.webclientUrl
                        if (!webclientUrl) {
                            loginDialog.errorMessage = qsTr("WebClient URL not configured on the server")
                            return
                        }
                        Qt.openUrlExternally(webclientUrl + "/#/account")
                    }
                }
            }
        }
    }

    function _translateError(code, fallback) {
        var map = {
            "INVALID_CREDENTIALS": qsTr("Invalid username or password"),
            "ACCOUNT_DISABLED": qsTr("Account is disabled"),
            "USERNAME_EXISTS": qsTr("Username already exists"),
            "PHONE_EXISTS": qsTr("Phone number already registered"),
            "PHONE_NOT_FOUND": qsTr("Phone number not registered"),
            "PHONE_INVALID": qsTr("Invalid phone number format"),
            "PASSWORD_WEAK": qsTr("Password too weak: at least 8 characters with letters and digits"),
            "SMS_DISABLED": qsTr("SMS service not enabled"),
            "SMS_RATE_LIMIT": qsTr("Too many requests, please try again later"),
            "SMS_DAILY_LIMIT": qsTr("Daily SMS limit reached"),
            "SMS_CODE_INVALID": qsTr("Invalid verification code"),
            "SMS_CODE_EXPIRED": qsTr("Verification code expired"),
            "SMS_MAX_ATTEMPTS": qsTr("Too many verification attempts"),
        }
        if (code && map[code]) return map[code]
        return fallback || qsTr("Unknown error")
    }

    function isFormValid() {
        switch (loginDialog.mode) {
        case "login":
            return usernameField.text.length > 0 && passwordField.text.length > 0
        case "sms-login":
            return smsPhoneField.text.length > 0 && smsCodeField.text.length > 0
        case "register":
            var baseValid = regUsernameField.text.length > 0 && regPasswordField.text.length > 0
            if (loginDialog.smsEnabled) {
                return baseValid && phoneField.text.length > 0 && regSmsCodeField.text.length > 0
            }
            return baseValid
        }
        return false
    }

    function confirmAction() {
        if (loginDialog.isLoading) return
        loginDialog.errorMessage = ""
        loginDialog.isLoading = true

        switch (loginDialog.mode) {
        case "login":
            loginDialog.mainController.authManager.login(
                usernameField.text, passwordField.text)
            break
        case "sms-login":
            loginDialog.mainController.authManager.loginWithSms(
                smsPhoneField.text, smsCodeField.text)
            break
        case "register":
            loginDialog.mainController.authManager.registerUser(
                regUsernameField.text, regPasswordField.text,
                phoneField.text, emailField.text, regSmsCodeField.text)
            break
        }
    }
}
