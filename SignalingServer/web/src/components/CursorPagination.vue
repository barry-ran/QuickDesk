<template>
  <!--
    Cursor-based pagination control for v1 admin list endpoints.

    Server contract (§2.2 / §3.1): every list GET accepts ?cursor=&limit=
    and returns { items, next_cursor, total? }. The cursor is opaque;
    clients must not introspect it.

    UI design:
      • "Previous" walks back through a local stack of cursors (seeded
        with '' for the first page). This lets the admin page backwards
        even though the server-side API is forward-only — perfectly
        acceptable for typical admin browsing behaviour on small-to-
        medium deployments.
      • "Next" is disabled when next_cursor is empty (=last page).
      • page-size selector resets the cursor stack (page size change
        invalidates any saved cursors since they encode per-limit state).
      • Total count is shown when the server provides it; otherwise
        we just show "Page N" with no grand total.
  -->
  <div class="cursor-pagination">
    <span class="summary">
      <template v-if="total !== null && total !== undefined">
        {{ t('pagination.total', { total }) }} ·
      </template>
      {{ t('pagination.page', { page: pageNumber }) }}
    </span>

    <el-select
      v-if="pageSizes && pageSizes.length > 1"
      :model-value="limit"
      size="small"
      class="limit-select"
      @update:model-value="onLimitChange"
    >
      <el-option v-for="sz in pageSizes" :key="sz" :value="sz" :label="t('pagination.perPage', { n: sz })" />
    </el-select>

    <el-button size="small" :disabled="!canPrev || loading" @click="goPrev">
      {{ t('pagination.prev') }}
    </el-button>
    <el-button size="small" :disabled="!canNext || loading" @click="goNext">
      {{ t('pagination.next') }}
    </el-button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  /** History of cursors visited. cursorStack[0] is always '' (first page).
   *  cursorStack.at(-1) is the cursor used to load the *current* page. */
  cursorStack: { type: Array, required: true },
  /** Cursor the server gave us for the NEXT page (empty string = last). */
  nextCursor:  { type: String, default: '' },
  /** Total row count from the server. null → hide the total label. */
  total:       { type: [Number, null], default: null },
  limit:       { type: Number, default: 20 },
  pageSizes:   { type: Array,  default: () => [10, 20, 50, 100] },
  loading:     { type: Boolean, default: false },
})

const emit = defineEmits(['prev', 'next', 'update:limit'])

const pageNumber = computed(() => Math.max(props.cursorStack.length, 1))
const canPrev = computed(() => props.cursorStack.length > 1)
const canNext = computed(() => !!props.nextCursor)

function goPrev() { if (canPrev.value) emit('prev') }
function goNext() { if (canNext.value) emit('next') }
function onLimitChange(v) { emit('update:limit', v) }
</script>

<style scoped>
.cursor-pagination {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}
.summary {
  color: #606266;
  font-size: 13px;
  margin-right: auto;
}
.limit-select {
  width: 110px;
}
</style>
