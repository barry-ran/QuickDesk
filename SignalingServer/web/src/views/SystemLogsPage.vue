<template>
  <div class="system-logs-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('systemLogs.title') }}</h2>
      <el-button type="primary" :icon="Refresh" size="small" @click="loadLogs">
        {{ t('common.refresh') }}
      </el-button>
    </div>

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
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { downloadSystemLog, getSystemLogs } from '../api/system.js'

const { t, locale } = useI18n()
const loading = ref(false)
const downloading = ref('')
const logs = ref([])

function formatDate(value) {
  return value ? new Date(value).toLocaleString(locale.value) : '-'
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

onMounted(loadLogs)
</script>

<style scoped>
.system-logs-page { width: 100%; padding: 20px; box-sizing: border-box; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; font-size: 24px; font-weight: 600; color: #303133; }
</style>
