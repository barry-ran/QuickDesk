include(FetchContent)
FetchContent_Declare(
    spdlog
    GIT_REPOSITORY https://github.com/gabime/spdlog.git
    GIT_TAG        ad0e89c
    SOURCE_DIR     ${CMAKE_CURRENT_SOURCE_DIR}/3rdparty/spdlog
)
FetchContent_MakeAvailable(spdlog)