include(FetchContent)

set(CRASHPAD_URL https://github.com/barry-ran/quickdesk-remoting/releases/download/v1.0.4)
set(QD_REMOTING_DIR ${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/quickdesk-remoting/${QD_CPU_ARCH})
if(WIN32)
    set(CRASHPAD_URL ${CRASHPAD_URL}/Windows-${QD_CPU_ARCH}.zip)
elseif(APPLE)
    set(CRASHPAD_URL ${CRASHPAD_URL}/Mac-${QD_CPU_ARCH}.zip)
endif()


# 如果FetchContent下载的顶级目录包含CMakeLists.txt文件，则调用add_subdirectory()将其添加到主构建中。
# 没有CMakeLists.txt文件也没问题，这允许仅使用FetchContent下载内容而不添加到构建流程中（例如导入外部构建好的二进制）
# FetchContent_Declare是配置阶段就去下载，所以URL不能使用生成器表达式，因为生成器表达式是在生成阶段才确定的
#
# A core that is already on disk wins: it may be built from source or copied in
# by hand, and re-downloading over it would be pointless. Linux release archives
# have no prebuilt core, so without one the Qt shell still has to configure —
# see docs/dev/linux-build.md for how to obtain or build the Linux core.
if(EXISTS "${QD_REMOTING_DIR}/icudtl.dat" OR EXISTS "${QD_REMOTING_DIR}/quickdesk_host"
   OR EXISTS "${QD_REMOTING_DIR}/quickdesk_host.exe")
    message(STATUS "quickdesk-remoting: using core already present in ${QD_REMOTING_DIR}")
elseif(UNIX AND NOT APPLE)
    message(WARNING
        "quickdesk-remoting: no Linux core in ${QD_REMOTING_DIR}; building the Qt shell only. "
        "Place quickdesk_host, quickdesk_client and icudtl.dat there (or build them from "
        "quickdesk-remoting) to get a usable Linux build.")
else()
    FetchContent_Declare(
        quickdesk-remoting
        URL             ${CRASHPAD_URL}
        SOURCE_DIR      ${QD_REMOTING_DIR}
    )
    FetchContent_MakeAvailable(quickdesk-remoting)
    FetchContent_GetProperties(quickdesk-remoting)
endif()
