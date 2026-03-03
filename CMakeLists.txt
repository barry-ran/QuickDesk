# For VS2019 and Xcode 12+ support.
cmake_minimum_required(VERSION 3.19 FATAL_ERROR)

if(DEFINED ENV{Qt6_DIR})
    set(CMAKE_PREFIX_PATH $ENV{Qt6_DIR})
elseif(EXISTS "D:/Qt/Qt5.15.2/6.8.3/msvc2022_64")
    set(CMAKE_PREFIX_PATH "D:/Qt/Qt5.15.2/6.8.3/msvc2022_64")
endif()

# Read version numbers from file
file(STRINGS ${CMAKE_CURRENT_SOURCE_DIR}/version QD_FILE_VERSION)

# project通常指的是逻辑上相对独立、完整，能够独立编译的集合
# 一个project可以单独设置名称、版本和语言
# 一般你的项目中某一部分需要有独立的名称/版本/不同的语言，那么就可以放到单独的project中
project(QuickDesk VERSION ${QD_FILE_VERSION} LANGUAGES CXX)

message(STATUS "QD_FILE_VERSION:${QD_FILE_VERSION}, CMAKE_PROJECT_VERSION:${CMAKE_PROJECT_VERSION}")

add_compile_definitions(
    APP_VERSION_MAJOR=${PROJECT_VERSION_MAJOR}
    APP_VERSION_MINOR=${PROJECT_VERSION_MINOR}
    APP_VERSION_PATCH=${PROJECT_VERSION_PATCH}
    APP_VERSION_TWEAK=${PROJECT_VERSION_TWEAK}
    APP_VERSION_STR="${PROJECT_VERSION_MAJOR}.${PROJECT_VERSION_MINOR}.${PROJECT_VERSION_PATCH}.${PROJECT_VERSION_TWEAK}"
)

# API Key for official signaling server authentication.
# Official CI builds set this via: cmake -DQUICKDESK_API_KEY="your-secret-key"
# Open-source / self-hosted builds leave this empty (default).
if(DEFINED QUICKDESK_API_KEY AND NOT QUICKDESK_API_KEY STREQUAL "")
    add_compile_definitions(QUICKDESK_API_KEY="${QUICKDESK_API_KEY}")
    message(STATUS "QUICKDESK_API_KEY: configured")
else()
    message(STATUS "QUICKDESK_API_KEY: not set (open-source build)")
endif()
message(STATUS "APP_VERSION_MAJOR:${PROJECT_VERSION_MAJOR}, APP_VERSION_MINOR:${PROJECT_VERSION_MINOR}, APP_VERSION_PATCH:${PROJECT_VERSION_PATCH}, APP_VERSION_TWEAK:${PROJECT_VERSION_TWEAK}")
message(STATUS "APP_VERSION_STR:${APP_VERSION_STR}")
# CMAKE_PROJECT_NAME和PROJECT_NAME都表示最近的project的名称
# 区别是只有最顶层的CMakeLists.txt中的project会影响CMAKE_PROJECT_NAME
# add_subdirectory中的project只会影响PROJECT_NAME

#
# 整个项目的全局配置（target无关）
#

# CMake

list(APPEND CMAKE_MODULE_PATH ${CMAKE_CURRENT_SOURCE_DIR}/cmake)

# 对每个subdirectories（递归）：自动添加subdirectories的CMAKE_CURRENT_BINARY_DIR和CMAKE_CURRENT_SOURCE_DIR到subdirectories的include_directories
# 注意：上述include_directories只作用于对应的subdirectories，不会向下传播到subdirectories的subdirectories
set(CMAKE_INCLUDE_CURRENT_DIR ON)
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/QuickDesk)

set(CMAKE_AUTOUIC ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# https://cmake.org/cmake/help/latest/prop_gbl/GENERATOR_IS_MULTI_CONFIG.html
# 介绍CMake构建类型较好的文章 https://blog.csdn.net/jjjstephen/article/details/122496126
get_property(GENERATOR_IS_MULTI_CONFIG GLOBAL PROPERTY GENERATOR_IS_MULTI_CONFIG)
message(STATUS "is multi config:" ${GENERATOR_IS_MULTI_CONFIG})

if(GENERATOR_IS_MULTI_CONFIG)
    # 限制多配置类型
    set(CMAKE_CONFIGURATION_TYPES "Debug;Release;RelWithDebInfo" CACHE STRING "limit build type" FORCE)
    # 多配置类型在配置时有多个构建类型，不知道具体编译的构建类型，在编译时才知道
else()
    set(allowableBuildTypes Debug Release RelWithDebInfo)
    # default RelWithDebInfo
    if(NOT CMAKE_BUILD_TYPE)
        set(CMAKE_BUILD_TYPE RelWithDebInfo)
    elseif(NOT CMAKE_BUILD_TYPE IN_LIST allowableBuildTypes)
        message(FATAL_ERROR "[${PROJECT_NAME}] Invalid build type: ${CMAKE_BUILD_TYPE}")
    endif()
    message(STATUS "[${PROJECT_NAME}] single config:BUILD_TYPE:${CMAKE_BUILD_TYPE}")
endif()

# Compiler set
message(STATUS "[${PROJECT_NAME}] C++ compiler ID is: ${CMAKE_CXX_COMPILER_ID}")
if (MSVC)
    # FFmpeg cannot be compiled natively by MSVC version < 12.0 (2013)
    if(MSVC_VERSION LESS 1800)
        message(FATAL_ERROR "[${PROJECT_NAME}] ERROR: MSVC version is older than 12.0 (2013).")
    endif()

    message(STATUS "[${PROJECT_NAME}] Set Warnings as error")
    # warning level 3 and all warnings as errors
    add_compile_options($<$<COMPILE_LANGUAGE:CXX>:/W3> $<$<COMPILE_LANGUAGE:CXX>:/WX> /wd4566 /wd4996)

    # avoid warning C4819    
    add_compile_options(/utf-8)

    # ensure we use minimal "windows.h" lib without the crazy min max macros
    add_compile_definitions(NOMINMAX WIN32_LEAN_AND_MEAN)
    
    # disable SAFESEH - avoid "LNK2026: module unsafe"(Qt5.15&&vs2019)     
    add_link_options(/SAFESEH:NO)
endif()

if (NOT MSVC)
    message(STATUS "[${PROJECT_NAME}] Set warnings as error")
    # lots of warnings and all warnings as errors
    add_compile_options(-Wall -Wextra -pedantic -Werror)

    # disable some warning
    add_compile_options(-Wno-unknown-warning-option -Wno-nested-anon-types -Wno-c++17-extensions -Wno-overloaded-virtual -Wno-variadic-macro-arguments-omitted -Wno-gnu-zero-variadic-macro-arguments -Wno-deprecated-declarations)
endif()

# 开启qml import trace(这里设置无效，需要在项目配置设置该环境变量)
# SET(ENV{QML_IMPORT_TRACE} 1)

# QD define

# check arch
if(CMAKE_SIZEOF_VOID_P EQUAL 8)
    set(QD_CPU_ARCH x64)
else()
    set(QD_CPU_ARCH x86)
endif()

# MacOS
if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    # mac default arch arm64
    if(NOT CMAKE_OSX_ARCHITECTURES)
        set(CMAKE_OSX_ARCHITECTURES arm64)
    endif()

    if (CMAKE_OSX_ARCHITECTURES MATCHES "arm64")
        set(QD_CPU_ARCH arm64)
    endif()
endif()

message(STATUS "[${PROJECT_NAME}] CPU_ARCH:${QD_CPU_ARCH}")

set(QD_OUTPUT_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}/output/${QD_CPU_ARCH}/${CMAKE_BUILD_TYPE}")
message(STATUS "[${PROJECT_NAME}] QD_OUTPUT_DIRECTORY:${QD_OUTPUT_DIRECTORY}")
# Qt

find_package(QT NAMES Qt6 Qt5 REQUIRED COMPONENTS Core Quick QuickControls2 Widgets Multimedia Sql WebSockets)
find_package(Qt${QT_VERSION_MAJOR} REQUIRED COMPONENTS Core Quick QuickControls2 Widgets Multimedia Sql WebSockets)
message(STATUS "[${PROJECT_NAME}] Qt version is: ${QT_VERSION_MAJOR}.${QT_VERSION_MINOR}")

# 在配置阶段生成qm
if(CMAKE_SYSTEM_NAME STREQUAL "Windows")
    execute_process(
        COMMAND ${CMAKE_CURRENT_SOURCE_DIR}/scripts/lrelease.bat ${CMAKE_PREFIX_PATH}/bin
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        RESULT_VARIABLE result
        OUTPUT_VARIABLE output
        ERROR_VARIABLE error
    )
else()
    execute_process(
        COMMAND bash ${CMAKE_CURRENT_SOURCE_DIR}/scripts/lrelease.sh ${CMAKE_PREFIX_PATH}/bin
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        RESULT_VARIABLE result
        OUTPUT_VARIABLE output
        ERROR_VARIABLE error
    )
endif()

if(NOT result EQUAL 0)
    message(FATAL_ERROR "gen qm failed: ${result} ${output} ${error}")
endif()
message("gen qm: ${output}")

# Subdirectory

add_subdirectory(QuickDesk)

# Output dir

function (_get_all_cmake_targets out_var current_dir)
    get_property(targets DIRECTORY ${current_dir} PROPERTY BUILDSYSTEM_TARGETS)
    get_property(subdirs DIRECTORY ${current_dir} PROPERTY SUBDIRECTORIES)

    foreach(subdir ${subdirs})
        _get_all_cmake_targets(subdir_targets ${subdir})
        list(APPEND targets ${subdir_targets})
    endforeach()

    set(${out_var} ${targets} PARENT_SCOPE)
endfunction()

_get_all_cmake_targets(all_targets ${CMAKE_CURRENT_LIST_DIR})
message(STATUS "[${PROJECT_NAME}] all targets: ${all_targets}")

# $<0:> 使用生成器表达式为每个config设置RUNTIME_OUTPUT_DIRECTORY，这样multi config就不会自动追加CMAKE_BUILD_TYPE子目录了
# 1. multi config介绍 https://cmake.org/cmake/help/latest/prop_gbl/GENERATOR_IS_MULTI_CONFIG.html
# 2. multi config在不用表达式生成器时自动追加子目录说明 https://cmake.org/cmake/help/latest/prop_tgt/RUNTIME_OUTPUT_DIRECTORY.html
# 3. 使用表达式生成器禁止multi config自动追加子目录解决方案 https://stackoverflow.com/questions/7747857/in-cmake-how-do-i-work-around-the-debug-and-release-directories-visual-studio-2
# 一般多配置时CMAKE_BUILD_TYPE默认为空，不应该使用此变量，当前项目打包脚本执行CMake时会指定CMAKE_BUILD_TYPE变量
foreach(target ${all_targets})
    set_target_properties(${target} PROPERTIES
        RUNTIME_OUTPUT_DIRECTORY "${QD_OUTPUT_DIRECTORY}/$<0:>"
        LIBRARY_OUTPUT_DIRECTORY "${QD_OUTPUT_DIRECTORY}/$<0:>"
        ARCHIVE_OUTPUT_DIRECTORY "${QD_OUTPUT_DIRECTORY}/$<0:>")
endforeach()
