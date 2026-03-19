include(FetchContent)

set(RAPIDOCR_URL https://github.com/barry-ran/3rdparty-prebuilt/releases/download/1.0.0)
if(WIN32)
    set(RAPIDOCR_URL ${RAPIDOCR_URL}/rapidocr-windows-${QD_CPU_ARCH}-shared-release-mt.zip)
elseif(APPLE)
    set(RAPIDOCR_URL ${RAPIDOCR_URL}/rapidocr-mac-${QD_CPU_ARCH}-shared-release.zip)
endif()


# 如果FetchContent下载的顶级目录包含CMakeLists.txt文件，则调用add_subdirectory()将其添加到主构建中。
# 没有CMakeLists.txt文件也没问题，这允许仅使用FetchContent下载内容而不添加到构建流程中（例如导入外部构建好的二进制）
# FetchContent_Declare是配置阶段就去下载，所以URL不能使用生成器表达式，因为生成器表达式是在生成阶段才确定的
FetchContent_Declare(
    rapidocr
    URL             ${RAPIDOCR_URL}
    SOURCE_DIR      ${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/rapidocr/${QD_CPU_ARCH}
)
FetchContent_MakeAvailable(rapidocr)
FetchContent_GetProperties(rapidocr)

set(rapidocr_path "${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/rapidocr/${QD_CPU_ARCH}")

# -------------------------------------------------------------------------
# 检查预编译库是否存在
# -------------------------------------------------------------------------
if(NOT EXISTS "${rapidocr_path}/include")
    message(WARNING
        "[rapidocr] Prebuilt libraries not found at: ${rapidocr_path}\n"
        "OCR features (get_screen_text / find_element / click_text) will be disabled.\n"
        "To enable: copy the rapidocr prebuilt directory to ${rapidocr_path}\n"
        "Source (Windows): quick-recoil-assistant/QuickRecoilAssistant/3rdparty/rapidocr\n"
        "Source (macOS):   https://github.com/RapidAI/RapidOcrOnnx/releases"
    )
    set(RAPIDOCR_FOUND FALSE)
    return()
endif()

set(RAPIDOCR_FOUND TRUE)

# -------------------------------------------------------------------------
# Include 目录
# -------------------------------------------------------------------------
set(RAPIDOCR_INCLUDE_DIRS "${rapidocr_path}/include")
message(STATUS "[rapidocr] Found at: ${rapidocr_path}")

# -------------------------------------------------------------------------
# 平台相关：库路径、链接库、需部署的二进制
# -------------------------------------------------------------------------
set(rapidocr_lib_path "${rapidocr_path}/lib")
set(rapidocr_bin_path "${rapidocr_path}/bin")

# 链接目录（供 target_link_directories 使用）
set(RAPIDOCR_LINK_DIRS "${rapidocr_lib_path}")
    
# 链接库名（不含路径，由 LINK_DIRS 解析）
set(RAPIDOCR_LIB_NAME "RapidOcrOnnx")

if(WIN32)    
    # 需要 POST_BUILD 部署的 DLL
    set(RAPIDOCR_DLLS "${rapidocr_bin_path}/RapidOcrOnnx.dll")

elseif(APPLE)
    set(RAPIDOCR_DLLS "${rapidocr_bin_path}/RapidOcrOnnx.dylib")
endif()

# -------------------------------------------------------------------------
# 模型文件目录
# -------------------------------------------------------------------------
set(RAPIDOCR_MODEL_DIR "${rapidocr_path}/models")
if(NOT EXISTS "${RAPIDOCR_MODEL_DIR}")
    message(WARNING "[rapidocr] Model dir not found: ${RAPIDOCR_MODEL_DIR}\n"
        "OCR will fail at runtime. Place PP-OCRv4 model files in that directory.")
endif()

message(STATUS "[rapidocr] Include : ${RAPIDOCR_INCLUDE_DIRS}")
message(STATUS "[rapidocr] Models  : ${RAPIDOCR_MODEL_DIR}")
