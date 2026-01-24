set(QD_TARGET_NAME "infra")

set(QD_TARGET_SOURCES
    db/dbbase.h
    db/dbbase.cpp    
    env/applicationcontext.h
    env/applicationcontext.cpp
    log/log.h
    log/log.cpp    
    http/httprequest.h
    http/httprequest.cpp
)
source_group(${QD_TARGET_NAME} FILES ${QD_TARGET_SOURCES})

# object库类似gn的source set，可以在多个target中使用，不会重复编译
add_library(${QD_TARGET_NAME} OBJECT
    ${QD_TARGET_SOURCES}
)

target_compile_definitions(${QD_TARGET_NAME} PUBLIC SPDLOG_ACTIVE_LEVEL=$<IF:$<OR:$<CONFIG:Release>,$<CONFIG:RelWithDebInfo>>,2,0>)

target_link_libraries(${QD_TARGET_NAME} PUBLIC
    spdlog::spdlog
    Qt${QT_VERSION_MAJOR}::Core
    Qt${QT_VERSION_MAJOR}::Sql
    Qt${QT_VERSION_MAJOR}::Network
)

target_link_libraries(${QD_TARGET_NAME} PRIVATE
    base
    spdlog::spdlog
)
