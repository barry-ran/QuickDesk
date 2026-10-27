cmake_minimum_required(VERSION 3.22)
project(QuickDeskTrustTests LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Sql)
find_package(spdlog CONFIG QUIET)
if(NOT TARGET spdlog::spdlog)
    include(FetchContent)
    FetchContent_Declare(spdlog
        GIT_REPOSITORY https://github.com/gabime/spdlog.git
        GIT_TAG ad0e89c
    )
    FetchContent_MakeAvailable(spdlog)
endif()

set(QUICKDESK_SOURCE_DIR "${CMAKE_CURRENT_LIST_DIR}/../../QuickDesk")
add_executable(trust_handler_test
    trust_handler_test.cpp
    "${QUICKDESK_SOURCE_DIR}/src/api/TrustHandler.cpp"
    "${QUICKDESK_SOURCE_DIR}/src/api/TrustHandler.h"
    "${QUICKDESK_SOURCE_DIR}/src/core/localconfigcenter.cpp"
    "${QUICKDESK_SOURCE_DIR}/src/core/localconfigcenter.h"
    "${QUICKDESK_SOURCE_DIR}/src/core/db/appconfigdatabase.cpp"
    "${QUICKDESK_SOURCE_DIR}/infra/db/dbbase.cpp"
    "${QUICKDESK_SOURCE_DIR}/infra/env/applicationcontext.cpp"
    "${QUICKDESK_SOURCE_DIR}/infra/log/log.cpp"
    "${QUICKDESK_SOURCE_DIR}/base/util.cpp"
)
target_include_directories(trust_handler_test PRIVATE
    "${QUICKDESK_SOURCE_DIR}"
    "${QUICKDESK_SOURCE_DIR}/src"
)
target_link_libraries(trust_handler_test PRIVATE Qt6::Core Qt6::Sql spdlog::spdlog)
if(MSVC)
    target_compile_options(trust_handler_test PRIVATE /utf-8)
    target_compile_definitions(trust_handler_test PRIVATE NOMINMAX WIN32_LEAN_AND_MEAN)
endif()

enable_testing()
add_test(NAME trust_handler_thread_affinity COMMAND trust_handler_test)
set_tests_properties(trust_handler_thread_affinity PROPERTIES TIMEOUT 30)
if(WIN32)
    set_tests_properties(trust_handler_thread_affinity PROPERTIES
        ENVIRONMENT_MODIFICATION "PATH=path_list_prepend:$<TARGET_FILE_DIR:Qt6::Core>"
    )
endif()
