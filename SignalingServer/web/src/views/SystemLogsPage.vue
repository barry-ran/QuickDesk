<template>
  <div class="system-logs-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('systemLogs.title') }}</h2>
      <div class="header-actions">
        <el-button :icon="Search" size="small" :loading="searching" @click="runSearch">
          {{ t('common.search') }}
        </el-button>
        <el-button type="primary" :icon="Refresh" size="small" @click="loadLogs">
          {{ t('common.refresh') }}
        </el-button>
      </div>
    </div>

    <el-card shadow="never" class="filter-card">
      <div class="filter-bar">
        <el-select v-model="filters.scope" style="width: 170px" @change="onScopeChange">
          <el-option :label="t('systemLogs.scopeLocal')" value="local" />
          <el-option :label="t('systemLogs.scopeAll')" value="all" />
        </el-select>
        <el-select
          v-model="filters.file"
          :placeholder="t('systemLogs.source')"
          clearable
          style="width: 170px"
        >
          <el-option :label="t('systemLogs.sourceAll')" value="all" />
          <el-option v-for="file in logs" :key="file.name" :label="file.name" :value="file.name" />
        </el-select>
        <el-input
          v-model="filters.q"
          :placeholder="t('systemLogs.queryPlaceholder')"
          clearable
          style="width: 240px"
          @keyup.enter="runSearch"
        />
        <el-input
          v-model="filters.match"
          :placeholder="t('systemLogs.matchPlaceholder')"
          clearable
          style="width: 240px"
          @keyup.enter="runSearch"
        />
        <el-input
          v-model="filters.component"
          :placeholder="t('systemLogs.componentPlaceholder')"
          clearable
          style="width: 180px"
          @keyup.enter="runSearch"
        />
        <el-date-picker
          v-model="timeRange"
          type="datetimerange"
          :start-placeholder="t('systemLogs.startTime')"
          :end-placeholder="t('systemLogs.endTime')"
          value-format="YYYY-MM-DDTHH:mm:ss"
          style="width: 340px"
        />
        <el-select v-model="filters.limit" style="width: 120px">
          <el-option v-for="size in [100, 200, 500, 1000, 2000]" :key="size" :label="`${size}`" :value="size" />
        </el-select>
        <el-input
          v-if="filters.scope === 'all'"
          v-model="filters.instance"
          :placeholder="t('systemLogs.instancePlaceholder')"
          clearable
          style="width: 240px"
          @keyup.enter="runSearch"
        />
        <el-button type="primary" :icon="Search" :loading="searching" @click="runSearch">
          {{ t('common.search') }}
        </el-button>
      </div>
      <div v-if="filters.scope === 'all'" class="scope-hint">{{ t('systemLogs.aggregatedHint') }}</div>
    </el-card>

    <el-card shadow="never" class="result-card">
      <template #header>
        <div class="card-header">
          <span>{{ t('systemLogs.results') }}</span>
          <span v-if="summary" class="summary">{{ summary }}</span>
        </div>
      </template>
      <el-table :data="entries" stripe style="width: 100%" size="small">
        <el-table-column :label="t('systemLogs.time')" width="190">
          <template #default="{ row }">{{ formatEntryTime(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('systemLogs.sourceColumn')" width="240">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ filters.scope === 'all' ? row.instance : row.file }}</el-tag>
            <span v-if="filters.scope === 'all'" class="source-detail">{{ row.source }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('systemLogs.line')" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="log-line">{{ row.line }}</span>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!searching && entries.length === 0" :description="t('systemLogs.resultEmpty')" />
    </el-card>

    <div class="section-title">{{ t('systemLogs.filesTitle') }}</div>
    <el-card shadow="never">
      <el-table :data="logs" stripe style="width: 100%" size="small">
        <el-table-column prop="name" :label="t('systemLogs.name')" min-width="260" />
        <el-table-column :label="t('systemLogs.size')" width="160">
          <template #default="{ row }">{{ formatSize(row.size_bytes) }}</template>
        </el-table-column>
        <el-table-column :label="t('systemLogs.modifiedAt')" width="190">
          <template #default="{ row }">{{ formatDate(row.modified_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="120">
          <template #default="{ row }">
            <el-button link type="primary" :loading="downloading === row.name" @click="downloadLog(row)">
              {{ t('systemLogs.download') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && logs.length === 0" :description="t('systemLogs.empty')" />
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import { downloadSystemLog, getSystemLogs, searchSystemLogs } from '../api/system.js'

const { t, locale } = useI18n()
const loading = ref(false)
const searching = ref(false)
const downloading = ref('')
const logs = ref([])
const entries = ref([])
const resultMeta = ref(null)
const timeRange = ref([])

const filters = ref({
  scope: 'local',
  file: 'signaling.log',
  q: '',
  match: '',
  component: '',
  instance: '',
  limit: 200,
})

// One line under the results header that says what the query actually cost, so
// "nothing matched" can be told from "the scan gave up" and, for the aggregate,
// from "this instance dropped lines while shipping".
const summary = computed(() => {
  const meta = resultMeta.value
  if (!meta) return ''
  const parts = [`${t('systemLogs.returned')}: ${meta.count ?? entries.value.length}`]
  if (meta.scope === 'all') {
    if (Array.isArray(meta.instances)) parts.push(`${t('systemLogs.instances')}: ${meta.instances.length}`)
    if (Number.isFinite(meta.stored_lines)) parts.push(`${t('systemLogs.stored')}: ${meta.stored_lines}`)
    if (Number.isFinite(meta.dropped_lines)) parts.push(`${t('systemLogs.dropped')}: ${meta.dropped_lines}`)
  } else {
    if (Number.isFinite(meta.scanned_lines)) parts.push(`${t('systemLogs.scannedLines')}: ${meta.scanned_lines}`)
    if (Number.isFinite(meta.scanned_bytes)) parts.push(`${t('systemLogs.scannedBytes')}: ${formatSize(meta.scanned_bytes)}`)
  }
  parts.push(meta.truncated ? t('systemLogs.truncated') : t('systemLogs.complete'))
  return parts.join(' · ')
})

function onScopeChange() {
  // `file=all` means "every source on this instance"; the aggregate reads the
  // shared store instead, where an empty source filter already means every
  // source, so carrying the value over would look like a filter it is not.
  if (filters.value.scope === 'all' && filters.value.file === 'all') {
    filters.value.file = ''
  }
  if (filters.value.scope === 'local' && !filters.value.file) {
    filters.value.file = 'signaling.log'
  }
  if (filters.value.scope === 'local') {
    filters.value.instance = ''
  }
}

function formatDate(value) {
  return value ? new Date(value).toLocaleString(locale.value) : '-'
}

function formatEntryTime(row) {
  return row.timestamp ? new Date(row.timestamp).toLocaleString(locale.value) : '-'
}

function formatSize(value) {
  if (!Number.isFinite(value)) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit ? 1 : 0)} ${units[unit]}`
}

async function loadLogs() {
  loading.value = true
  try {
    const data = await getSystemLogs()
    logs.value = data.items || []
  } catch (error) {
    ElMessage.error(`${t('systemLogs.loadFailed')}: ${error.message}`)
  } finally {
    loading.value = false
  }
}

// The date picker hands back local wall-clock strings; the API compares against
// the line's own UTC timestamp, so the conversion happens here rather than
// leaving the operator to work out which clock they typed into.
function toRFC3339(value) {
  if (!value) return undefined
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? undefined : parsed.toISOString()
}

async function runSearch() {
  searching.value = true
  try {
    const [since, until] = timeRange.value || []
    const data = await searchSystemLogs({
      scope: filters.value.scope,
      file: filters.value.file,
      q: filters.value.q?.trim(),
      match: filters.value.match?.trim(),
      component: filters.value.component?.trim(),
      instance: filters.value.scope === 'all' ? filters.value.instance?.trim() : undefined,
      since: toRFC3339(since),
      until: toRFC3339(until),
      limit: filters.value.limit,
    })
    entries.value = data.items || []
    resultMeta.value = data
  } catch (error) {
    entries.value = []
    resultMeta.value = null
    ElMessage.error(`${t('systemLogs.searchFailed')}: ${error.message}`)
  } finally {
    searching.value = false
  }
}

async function downloadLog(log) {
  downloading.value = log.name
  try {
    const { blob, filename } = await downloadSystemLog(log.name)
    saveBlob(blob, filename || log.name)
  } catch (error) {
    ElMessage.error(`${t('systemLogs.downloadFailed')}: ${error.message}`)
  } finally {
    downloading.value = ''
  }
}

function saveBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(async () => {
  await loadLogs()
  await runSearch()
})
</script>

<style scoped>
.system-logs-page { width: 100%; padding: 20px; box-sizing: border-box; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; font-size: 24px; font-weight: 600; color: #303133; }
.header-actions { display: flex; gap: 8px; }
.filter-card { margin-bottom: 16px; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.scope-hint { margin-top: 8px; font-size: 12px; color: #909399; }
.result-card { margin-bottom: 16px; }
.card-header { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.card-header .summary { font-size: 12px; color: #909399; }
.source-detail { margin-left: 6px; font-size: 12px; color: #909399; }
.log-line { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
.section-title { margin: 0 0 8px; font-size: 16px; font-weight: 600; color: #303133; }
</style>
