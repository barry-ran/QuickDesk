# rapidocr.cmake
#
# 集成 RapidOCR（PP-OCRv4 + ONNX Runtime）预编译库
#
# 使用方式（与 quick-recoil-assistant 完全一致）：
#   将预编译包解压到 QuickDesk/3rdparty/rapidocr/ 即可，无需任何环境变量。
#
# 目录结构（Windows x64 shared MD）：
#   3rdparty/rapidocr/
#   ├── include/                       # OcrLiteCApi.h 等头文件
#   ├── lib-shared-x64-md/             # RapidOcrOnnx.lib（release 导入库）
#   ├── bin-shared-x64-md/             # RapidOcrOnnx.dll（release，需部署到 exe 旁）
#   ├── lib-shared-x64-md-debug/       # RapidOcrOnnx.lib（debug 导入库）
#   ├── bin-shared-x64-md-debug/       # RapidOcrOnnx.dll（debug）
#   └── models/                        # PP-OCRv4 .onnx 模型文件
#       ├── ch_PP-OCRv4_det_infer.onnx
#       ├── ch_PP-OCRv4_rec_infer.onnx
#       ├── ch_ppocr_mobile_v2.0_cls_infer.onnx
#       └── ppocr_keys_v1.txt
#
# 目录结构（macOS arm64 static）：
#   3rdparty/rapidocr/
#   ├── include/
#   ├── lib-arm64/                     # libRapidOcrOnnx.a + libonnxruntime.a + libopencv_*.a
#   └── models/
#
# 说明：
#   - Windows DLL 已将 OpenCV + ONNX Runtime 静态链接进去，应用只需链接导入库
#   - macOS 静态库需要额外链接 ONNX Runtime 和 OpenCV（它们未合并进 .a）
#   - 预编译包来源：quick-recoil-assistant/QuickRecoilAssistant/3rdparty/rapidocr（Windows）
#                   RapidAI/RapidOcrOnnx GitHub Releases（macOS）

# CMAKE_CURRENT_SOURCE_DIR 在 include() 调用时指向调用方的目录
# 即 QuickDesk/QuickDesk/，与 quick-recoil-assistant 的用法完全一致
set(rapidocr_path "${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/rapidocr")

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
if(WIN32)
    # 与 quick-recoil-assistant shared MD 配置完全一致
    set(rapidocr_lib_path_rel "${rapidocr_path}/lib-shared-x64-md")
    set(rapidocr_bin_path_rel "${rapidocr_path}/bin-shared-x64-md")
    set(rapidocr_lib_path_dbg "${rapidocr_path}/lib-shared-x64-md-debug")
    set(rapidocr_bin_path_dbg "${rapidocr_path}/bin-shared-x64-md-debug")

    # 链接目录（供 target_link_directories 使用）
    set(RAPIDOCR_LINK_DIRS_REL "${rapidocr_lib_path_rel}")
    set(RAPIDOCR_LINK_DIRS_DBG "${rapidocr_lib_path_dbg}")

    # 链接库名（不含路径，由 LINK_DIRS 解析）
    set(RAPIDOCR_LIB_NAME "RapidOcrOnnx")

    # 需要 POST_BUILD 部署的 DLL
    set(RAPIDOCR_DLLS     "${rapidocr_bin_path_rel}/RapidOcrOnnx.dll")
    set(RAPIDOCR_DLLS_DBG "${rapidocr_bin_path_dbg}/RapidOcrOnnx.dll")

elseif(APPLE)
    set(rapidocr_lib_path "${rapidocr_path}/lib-${QD_CPU_ARCH}")

    if(NOT EXISTS "${rapidocr_lib_path}")
        message(WARNING "[rapidocr] macOS lib dir not found: ${rapidocr_lib_path}")
        set(RAPIDOCR_FOUND FALSE)
        return()
    endif()

    set(RAPIDOCR_LINK_DIRS_REL "${rapidocr_lib_path}")
    set(RAPIDOCR_LINK_DIRS_DBG "${rapidocr_lib_path}")

    # macOS 静态库：RapidOcrOnnx 本身 + ONNX Runtime + 必要的 OpenCV 模块
    set(RAPIDOCR_LIB_NAME
        RapidOcrOnnx
        onnxruntime
        opencv_imgproc
        opencv_imgcodecs
        opencv_core
    )
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
