include(spdlog)

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
)

set(MANAGER_HEADERS
    src/manager/NativeMessaging.h
    src/manager/ProcessManager.h
    src/manager/ServerManager.h
    src/manager/TurnServerManager.h
    src/manager/HostManager.h
    src/manager/ClientManager.h
    src/manager/SharedMemoryManager.h
)

set(CONTROLLER_SOURCES
    src/controller/MainController.cpp
)

set(CONTROLLER_HEADERS
    src/controller/MainController.h
)

set(COMMON_HEADERS
    src/common/ProcessStatus.h
)

set(COMPONENT_SOURCES
    src/component/VideoFrameProvider.cpp
    src/component/KeycodeMapper.cpp
    src/component/CursorImageProvider.cpp
)

set(COMPONENT_HEADERS
    src/component/VideoFrameProvider.h
    src/component/KeycodeMapper.h
    src/component/CursorImageProvider.h
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
    ${COMPONENT_SOURCES}
)

set(ALL_HEADERS
    ${MANAGER_HEADERS}
    ${CONTROLLER_HEADERS}
    ${COMMON_HEADERS}
    ${COMPONENT_HEADERS}
)

qt_standard_project_setup(REQUIRES 6.5)

qt_add_executable(QuickDesk
    ${ALL_SOURCES}
    ${ALL_HEADERS}
    ${CORE_SOURCES}
    ${VIEWMODEL_SOURCES}
    ${LANGUAGE_SOURCES}
    qml.qrc
)

qt_add_qml_module(QuickDesk
    URI QuickDesk
    VERSION 1.0
    QML_FILES 
        qml/views/Main.qml
        qml/views/MainEntry.qml
        qml/views/MainWindow.qml
        qml/views/RemoteWindow.qml
        qml/views/Example.qml
        qml/pages/RemoteControlPage.qml
        qml/pages/SettingsPage.qml
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
        qml/component/ConnectionListPanel.qml
        qml/quickdeskcomponent/RemoteDesktopView.qml
        qml/quickdeskcomponent/RemoteTab.qml
        qml/quickdeskcomponent/RemoteTabBar.qml
        qml/quickdeskcomponent/FloatingToolButton.qml
    RESOURCES
        res/font/SegoeFluentIcons.ttf
        qml/component/qmldir
)

# Include directories
target_include_directories(QuickDesk PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src    
)

set_target_properties(QuickDesk PROPERTIES
    MACOSX_BUNDLE_BUNDLE_VERSION ${PROJECT_VERSION}
    MACOSX_BUNDLE_SHORT_VERSION_STRING ${PROJECT_VERSION_MAJOR}.${PROJECT_VERSION_MINOR}
    MACOSX_BUNDLE TRUE
    WIN32_EXECUTABLE TRUE
)

target_link_libraries(QuickDesk
    PRIVATE 
    Qt6::Quick 
    Qt6::QuickControls2
    Qt6::Core
    Qt6::Multimedia
    base
    infra
)

include(GNUInstallDirs)
install(TARGETS QuickDesk
    BUNDLE DESTINATION .
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
