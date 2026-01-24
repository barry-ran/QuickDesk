#pragma once

#include "xcheck.h"

#define CHECK_RETURN_VOID(x) \
    if (x) {                 \
        return;              \
    }

#define CHECK_RETURN_VOID_ASSERT(x) \
    if (x) {                        \
        Q_ASSERT(0);                \
        return;                     \
    }

#define CHECK_RETURN_VALUE(x, ret) \
    if (x) {                       \
        return (ret);              \
    }

#define CHECK_RETURN_VALUE_ASSERT(x, ret) \
    if (x) {                              \
        Q_ASSERT(0);                      \
        return (ret);                     \
    }