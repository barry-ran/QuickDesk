include(spdlog)

include(quickdesk-remoting)

include(rapidocr)

add_subdirectory(base)
add_subdirectory(infra)

# Source files
set(MANAGER_SOURCES
    src/manager/NativeMessaging.cpp
    src/manager/ProcessManager.cpp
    src/manager/ServerManager.cpp
    src/manager/TurnServerManager.cpp
    src/manager/HostManager.cpp
    src/manager/ClientManager.cpp
    src/manager/SharedMemoryManager.cpp
    src/manager/RemoteDeviceManager.cpp
    src/manager/PresetManager.cpp
    src/manager/AgentManager.cpp
)

set(MANAGER_HEADERS
    src/manager/NativeMessaging.h
    src/manager/ProcessManager.h
    src/manager/ServerManager.h
    src/manager/TurnServerManager.h
    src/manager/HostManager.h
    src/manager/ClientManager.h
    src/manager/SharedMemoryManager.h
    src/manager/RemoteDeviceManager.h
    src/manager/PresetManager.h
    src/manager/AgentManager.h
)

set(CONTROLLER_SOURCES
    src/controller/MainController.cpp
)

set(CONTROLLER_HEADERS
    src/controller/MainController.h
)

set(API_SOURCES
    src/api/WebSocketServer.cpp
    src/api/ApiHandler.cpp
    src/api/SecurityManager.cpp
    src/api/OcrEngine.cpp
    src/api/OcrCache.cpp
    src/api/UiStateService.cpp
    src/api/VerificationService.cpp
    src/api/AgentHandler.cpp
)

set(API_HEADERS
    src/api/WebSocketServer.h
    src/api/ApiHandler.h
    src/api/SecurityManager.h
    src/api/OcrEngine.h
    src/api/OcrCache.h
    src/api/UiStateService.h
    src/api/VerificationService.h
    src/api/AgentHandler.h
)

set(COMMON_HEADERS
    src/common/ProcessStatus.h
)

set(COMPONENT_SOURCES
    src/component/VideoFrameProvider.cpp
    src/component/KeycodeMapper.cpp
    src/component/CursorImageProvider.cpp
    src/component/SystemTrayManager.cpp
)

set(COMPONENT_HEADERS
    src/component/VideoFrameProvider.h
    src/component/KeycodeMapper.h
    src/component/CursorImageProvider.h
    src/component/SystemTrayManager.h
)

set(CORE_SOURCES
    src/core/common/commontype.h
    src/core/db/appconfigdatabase.h
    src/core/db/appconfigdatabase.cpp
    src/core/db/userdatadatabase.h
    src/core/db/userdatadatabase.cpp
    src/core/localconfigcenter.h
    src/core/localconfigcenter.cpp
    src/core/userdatacenter.h
    src/core/userdatacenter.cpp
)
source_group(core FILES ${CORE_SOURCES})

set(VIEWMODEL_SOURCES
    src/viewmodel/configviewmodel.h
    src/viewmodel/configviewmodel.cpp
    src/viewmodel/connectionlistmodel.h
    src/viewmodel/connectionlistmodel.cpp
)
source_group(viewmodel FILES ${VIEWMODEL_SOURCES})

set(LANGUAGE_SOURCES
    src/language/languagemanage.h
    src/language/languagemanage.cpp
)
source_group(language FILES ${LANGUAGE_SOURCES})

set(ALL_SOURCES
    main.cpp
    ${MANAGER_SOURCES}
    ${CONTROLLER_SOURCES}
    ${API_SOURCES}
    ${COMPONENT_SOURCES}
)

set(ALL_HEADERS
    ${MANAGER_HEADERS}
    ${CONTROLLER_HEADERS}
    ${API_HEADERS}
    ${COMMON_HEADERS}
    ${COMPONENT_HEADERS}
)

# plantform file
if(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    # Define VERSION macros for .rc file
    add_compile_definitions(
        VERSION_MAJOR=${PROJECT_VERSION_MAJOR}
        VERSION_MINOR=${PROJECT_VERSION_MINOR}
        VERSION_PATCH=${PROJECT_VERSION_PATCH}
        VERSION_RC_STR="${VERSION_MAJOR}.${VERSION_MINOR}.${VERSION_PATCH}"
    )
    set(QC_PLANTFORM_SOURCES
        "${CMAKE_CURRENT_SOURCE_DIR}/res/QuickDesk.rc"
    )
endif()
if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    # 设置 macOS 最低部署版本，确保在新系统编译的应用可以在旧系统运行
    set(CMAKE_OSX_DEPLOYMENT_TARGET "12.0" CACHE STRING "Minimum OS X deployment version" FORCE)
    message(STATUS "[${PROJECT_NAME}] macOS deployment target: ${CMAKE_OSX_DEPLOYMENT_TARGET}")
    
    # Step 1. add icns to source file, for MACOSX_PACKAGE_LOCATION copy
    set(QC_PLANTFORM_SOURCES
        "${CMAKE_CURRENT_SOURCE_DIR}/res/QuickDesk.icns"
    )
endif()

qt_standard_project_setup(REQUIRES 6.5)

qt_add_executable(QuickDesk
    ${ALL_SOURCES}
    ${ALL_HEADERS}
    ${CORE_SOURCES}
    ${VIEWMODEL_SOURCES}
    ${LANGUAGE_SOURCES}
    ${QC_PLANTFORM_SOURCES}
    qml.qrc
    res/res.qrc
)

qt_add_qml_module(QuickDesk
    URI QuickDesk
    VERSION 1.0
    QML_FILES
        qml/views/MainEntry.qml
        qml/views/MainWindow.qml
        qml/views/RemoteWindow.qml
        qml/views/Example.qml
        qml/pages/RemoteControlPage.qml
        qml/pages/SettingsPage.qml
        qml/pages/AboutPage.qml
        qml/pages/McpConfigPopup.qml
        qml/component/Theme.qml
        qml/component/FluentIconGlyph.qml
        qml/component/QDButton.qml
        qml/component/QDTextField.qml
        qml/component/QDCheckBox.qml
        qml/component/QDRadioButton.qml
        qml/component/QDSwitch.qml
        qml/component/QDSlider.qml
        qml/component/QDProgressBar.qml
        qml/component/QDComboBox.qml
        qml/component/QDCard.qml
        qml/component/QDToast.qml
        qml/component/QDDialog.qml
        qml/component/QDMessageBox.qml
        qml/component/QDDivider.qml
        qml/component/QDBadge.qml
        qml/component/QDChip.qml
        qml/component/QDToolTip.qml
        qml/component/QDSpinner.qml
        qml/component/QDAvatar.qml
        qml/component/QDMenu.qml
        qml/component/QDMenuItem.qml
        qml/component/QDMenuSeparator.qml
        qml/component/QDShadow.qml
        qml/component/QDTabBar.qml
        qml/component/QDTabButton.qml
        qml/component/QDIconButton.qml
        qml/component/QDDrawer.qml
        qml/component/QDScrollBar.qml
        qml/component/QDListView.qml
        qml/component/QDListItem.qml
        qml/component/QDBreadcrumb.qml
        qml/component/QDEmptyState.qml
        qml/component/QDSeparator.qml
        qml/component/QDTable.qml
        qml/component/QDAccordion.qml
        qml/component/QDContextMenu.qml
        qml/component/QDStatusBar.qml
        qml/component/QDToggleButtonGroup.qml
        qml/component/QDRating.qml
        qml/component/QDPagination.qml
        qml/component/QDDateTimePicker.qml
        qml/component/QDTextArea.qml
        qml/component/QDSearchBox.qml
        qml/component/QDNavigationView.qml
        qml/component/QDAnnouncementBar.qml
        qml/component/QDText.qml
        qml/component/QDLabel.qml
        qml/quickdeskcomponent/RemoteDesktopView.qml
        qml/quickdeskcomponent/RemoteTab.qml
        qml/quickdeskcomponent/RemoteTabBar.qml
        qml/quickdeskcomponent/FloatingToolButton.qml
        qml/quickdeskcomponent/ConnectionListPanel.qml
        qml/quickdeskcomponent/RemoteDeviceSelector.qml
        qml/quickdeskcomponent/VideoStatsOverlay.qml
    RESOURCES
        res/font/SegoeFluentIcons.ttf
        qml/component/qmldir
)

# Include directories
target_include_directories(QuickDesk PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src
    ${RAPIDOCR_INCLUDE_DIRS}
)

set_target_properties(QuickDesk PROPERTIES
    MACOSX_BUNDLE_BUNDLE_VERSION ${PROJECT_VERSION}
    MACOSX_BUNDLE_SHORT_VERSION_STRING ${PROJECT_VERSION_MAJOR}.${PROJECT_VERSION_MINOR}
    MACOSX_BUNDLE TRUE
    WIN32_EXECUTABLE TRUE
)

# macOS bundle 配置
if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    # Step 2. use MACOSX_PACKAGE_LOCATION copy icns to Resources
    set_source_files_properties(
        ${CMAKE_CURRENT_SOURCE_DIR}/res/QuickDesk.icns
        PROPERTIES MACOSX_PACKAGE_LOCATION Resources
    )

    # use MACOSX_BUNDLE_INFO_PLIST custom plist, not use MACOSX_BUNDLE_BUNDLE_NAME etc..
    set(INFO_PLIST_TEMPLATE_FILE "${CMAKE_CURRENT_SOURCE_DIR}/res/Info_Mac.plist.in")
    set(INFO_PLIST_FILE "${CMAKE_CURRENT_SOURCE_DIR}/res/Info_Mac.plist")
    file(READ "${INFO_PLIST_TEMPLATE_FILE}" plist_contents)
    string(REPLACE "\${BUNDLE_VERSION}" "${PROJECT_VERSION}" plist_contents ${plist_contents})
    file(WRITE ${INFO_PLIST_FILE} ${plist_contents})
    set_target_properties(QuickDesk PROPERTIES
        MACOSX_BUNDLE_INFO_PLIST "${INFO_PLIST_FILE}"
        # "" disable code sign
        XCODE_ATTRIBUTE_CODE_SIGN_IDENTITY ""
    )
endif()

target_link_libraries(QuickDesk
    PRIVATE
    Qt6::Quick
    Qt6::QuickControls2
    Qt6::Core
    Qt6::Widgets
    Qt6::Multimedia
    Qt6::Network
    Qt6::WebSockets
    base
    infra
)

# -------------------------------------------------------------------------
# RapidOCR 链接（与 quick-recoil-assistant 方式一致）
# -------------------------------------------------------------------------
if(RAPIDOCR_FOUND)
    # 设置链接搜索目录
    target_link_directories(QuickDesk PRIVATE ${RAPIDOCR_LINK_DIRS}
    )
    target_link_libraries(QuickDesk PRIVATE ${RAPIDOCR_LIB_NAME})
    # 告知 OcrEngine.cpp 实际链接了 RapidOCR，编译完整实现而非 stub
    target_compile_definitions(QuickDesk PRIVATE QUICKDESK_OCR_ENABLED)
endif()

# -------------------------------------------------------------------------
# 部署 RapidOcrOnnx
# -------------------------------------------------------------------------
if(RAPIDOCR_FOUND)
    if(WIN32)
        add_custom_command(TARGET QuickDesk POST_BUILD
            COMMAND ${CMAKE_COMMAND} -E copy_if_different
                "${RAPIDOCR_DLLS}>"
                "$<TARGET_FILE_DIR:QuickDesk>/"
            COMMENT "Deploying RapidOcrOnnx.dll"
        )
    elseif(APPLE)
        add_custom_command(TARGET QuickDesk POST_BUILD
            COMMAND ${CMAKE_COMMAND} -E copy_if_different
                "${RAPIDOCR_DLLS}>"
                "$<TARGET_FILE_DIR:QuickDesk>/../Frameworks/"
            COMMENT "Deploying RapidOcrOnnx.dylib"
        )
    endif()
    
endif()

# -------------------------------------------------------------------------
# 部署 OCR 模型文件（Windows & macOS）
# -------------------------------------------------------------------------
if(RAPIDOCR_FOUND AND EXISTS "${RAPIDOCR_MODEL_DIR}")
    if(WIN32)
        add_custom_command(TARGET QuickDesk POST_BUILD
            COMMAND ${CMAKE_COMMAND} -E copy_directory
                "${RAPIDOCR_MODEL_DIR}"
                "$<TARGET_FILE_DIR:QuickDesk>/models"
            COMMENT "Deploying OCR models"
        )
    elseif(APPLE)
        add_custom_command(TARGET QuickDesk POST_BUILD
            COMMAND ${CMAKE_COMMAND} -E copy_directory
                "${RAPIDOCR_MODEL_DIR}"
                "$<TARGET_FILE_DIR:QuickDesk>/../Resources/models"
            COMMENT "Deploying OCR models to macOS bundle"
        )
    endif()
endif()

include(GNUInstallDirs)
install(TARGETS QuickDesk
    BUNDLE DESTINATION .
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
