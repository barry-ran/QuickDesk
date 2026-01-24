set(QD_TARGET_NAME "base")

enable_language(ASM_MASM)

set(QD_TARGET_SOURCES
    singleton.h
    def.h
    xcheck.h
    system.h
    system.cpp
    util.h
    util.cpp
    qaesencryption.h
    qaesencryption.cpp    
)
source_group(${QD_TARGET_NAME} FILES ${QD_TARGET_SOURCES})

# object库类似gn的source set，可以在多个target中使用，不会重复编译
add_library(${QD_TARGET_NAME} OBJECT
    ${QD_TARGET_SOURCES}
)

target_link_libraries(${QD_TARGET_NAME} PUBLIC
    Qt${QT_VERSION_MAJOR}::Core
)
