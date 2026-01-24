#pragma once

#include "base/check.h"
#include "base/check_op.h"

#define XCHECK(condition) CHECK(condition)
#define XPCHECK(condition) PCHECK(condition)
#define XDCHECK(condition) DCHECK(condition)
#define XDPCHECK(condition) DPCHECK(condition)

#define XCHECK_EQ(val1, val2) CHECK_EQ(val1, val2)
#define XCHECK_NE(val1, val2) CHECK_NE(val1, val2)
#define XCHECK_LE(val1, val2) CHECK_LE(val1, val2)
#define XCHECK_LT(val1, val2) CHECK_LT(val1, val2)
#define XCHECK_GE(val1, val2) CHECK_GE(val1, val2)
#define XCHECK_GT(val1, val2) CHECK_GT(val1, val2)

#define XDCHECK_EQ(val1, val2) DCHECK_EQ(val1, val2)
#define XDCHECK_NE(val1, val2) DCHECK_NE(val1, val2)
#define XDCHECK_LE(val1, val2) DCHECK_LE(val1, val2)
#define XDCHECK_LT(val1, val2) DCHECK_LT(val1, val2)
#define XDCHECK_GE(val1, val2) DCHECK_GE(val1, val2)
#define XDCHECK_GT(val1, val2) DCHECK_GT(val1, val2)