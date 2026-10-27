cmake_minimum_required(VERSION 3.22)
project(QuickDeskFileTransferTests LANGUAGES CXX)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
find_package(Qt6 REQUIRED COMPONENTS Core Gui Test)
add_executable(file_transfer_test
    file_transfer_test.cpp
    ../../QuickDesk/src/manager/FileTransferQueue.cpp
    ../../QuickDesk/src/manager/FileTransferQueue.h
)
target_include_directories(file_transfer_test PRIVATE ../../QuickDesk/src/manager)
target_link_libraries(file_transfer_test PRIVATE Qt6::Core Qt6::Gui Qt6::Test)
if(MSVC)
    target_compile_options(file_transfer_test PRIVATE /utf-8)
endif()
enable_testing()
add_test(NAME file_transfer_state_machine COMMAND file_transfer_test)
set_tests_properties(file_transfer_state_machine PROPERTIES
    TIMEOUT 30 ENVIRONMENT "QT_QPA_PLATFORM=offscreen")
if(WIN32)
    set_tests_properties(file_transfer_state_machine PROPERTIES
        ENVIRONMENT_MODIFICATION "PATH=path_list_prepend:$<TARGET_FILE_DIR:Qt6::Core>")
endif()
