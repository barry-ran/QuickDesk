include(FetchContent)

set(CRASHPAD_URL https://github.com/barry-ran/quickdesk-remoting/releases/download/v0.0.9)
if(WIN32)
    set(CRASHPAD_URL ${CRASHPAD_URL}/Windows-${QD_CPU_ARCH}.zip)
elseif(APPLE)
    set(CRASHPAD_URL ${CRASHPAD_URL}/Mac-${QD_CPU_ARCH}.zip)
endif()


# 如果FetchContent下载的顶级目录包含CMakeLists.txt文件，则调用add_subdirectory()将其添加到主构建中。
# 没有CMakeLists.txt文件也没问题，这允许仅使用FetchContent下载内容而不添加到构建流程中（例如导入外部构建好的二进制）
# FetchContent_Declare是配置阶段就去下载，所以URL不能使用生成器表达式，因为生成器表达式是在生成阶段才确定的
FetchContent_Declare(
    quickdesk-remoting
    URL             ${CRASHPAD_URL}
    SOURCE_DIR      ${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/quickdesk-remoting/${QD_CPU_ARCH}
)
FetchContent_MakeAvailable(quickdesk-remoting)
FetchContent_GetProperties(quickdesk-remoting)