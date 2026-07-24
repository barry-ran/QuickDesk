<template>
  <div class="device-list-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('nav.devices') }}</h2>
      <div class="header-actions">
        <el-button :icon="Download" size="small" @click="handleExport">{{ t('common.export') }}</el-button>
        <el-button type="primary" :icon="Refresh" size="small" @click="loadDevices">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <!-- Filters -->
    <el-card shadow="never" class="filter-card">
      <div class="filter-bar">
        <el-input
          v-model="filters.search"
          :placeholder="t('devices.searchPlaceholder')"
          clearable
          style="width: 240px"
          @clear="handleSearch"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="filters.os" :placeholder="t('devices.filterOS')" clearable style="width: 140px" @change="handleFilter">
          <el-option label="Windows" value="Windows" />
          <el-option label="macOS" value="macOS" />
          <el-option label="Linux" value="Linux" />
          <el-option label="Unknown" value="Unknown" />
        </el-select>
        <el-select v-model="filters.online" :placeholder="t('devices.filterOnline')" clearable style="width: 140px" @change="handleFilter">
          <el-option :label="t('devices.online')" value="true" />
          <el-option :label="t('devices.offline')" value="false" />
        </el-select>
        <el-button :icon="Search" @click="handleSearch">{{ t('common.search') }}</el-button>
      </div>
    </el-card>

    <!-- Batch Toolbar -->
    <div v-if="selectedIds.length > 0" class="batch-bar">
      <span>{{ t('batch.selected', { count: selectedIds.length }) }}</span>
      <el-button size="small" type="danger" @click="handleBatch('delete')">{{ t('batch.delete') }}</el-button>
      <el-button size="small" @click="handleBatch('assign_group')">{{ t('batch.assignGroup') }}</el-button>
      <el-button size="small" @click="handleBatch('remove_group')">{{ t('batch.removeGroup') }}</el-button>
    </div>

    <!-- Table -->
    <el-card shadow="never" style="margin-top: 16px">
      <el-table
        ref="tableRef"
        :data="devices"
        stripe
        style="width: 100%"
        size="small"
        @sort-change="handleSortChange"
        @row-click="handleRowClick"
        @selection-change="handleSelectionChange"
        row-class-name="clickable-row"
      >
        <el-table-column type="selection" width="40" />
        <el-table-column prop="device_id" :label="t('devices.deviceId')" width="110" sortable="custom" />
        <el-table-column prop="device_uuid" label="UUID" min-width="180" show-overflow-tooltip />
        <el-table-column :label="t('devices.os')" min-width="120" sortable="custom" prop="os">
          <template #default="{ row }">
            {{ row.os }}{{ row.os_version ? ' ' + row.os_version : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="app_version" :label="t('devices.appVersion')" width="110" sortable="custom" />
        <el-table-column :label="t('devices.status')" width="100" sortable="custom" prop="online">
          <template #default="{ row }">
            <el-tag :type="isOnline(row) ? 'success' : 'info'" size="small">
              {{ isOnline(row) ? t('devices.online') : t('devices.offline') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('devices.heartbeat')" width="105">
          <template #default="{ row }">
            <el-tag :type="row.presence?.heartbeat ? 'success' : 'info'" size="small">
              {{ row.presence?.heartbeat ? t('common.on') : t('common.off') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('devices.wsCount')" width="95">
          <template #default="{ row }">{{ row.presence?.ws_count ?? 0 }}</template>
        </el-table-column>
        <el-table-column :label="t('devices.offlineReason')" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            {{ isOnline(row) ? '-' : row.offline_reason || '-' }}
          </template>
        </el-table-column>
        <el-table-column :label="t('devices.lastSeen')" width="170" sortable="custom" prop="last_seen_at">
          <template #default="{ row }">
            {{ formatDate(row.last_seen_at) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('devices.createdAt')" width="170" sortable="custom" prop="created_at">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-bar">
        <!-- §3.1 cursor-based pagination. See components/CursorPagination.vue. -->
        <CursorPagination
          :cursor-stack="pagination.cursorStack"
          :next-cursor="pagination.nextCursor"
          :total="pagination.total"
          :limit="pagination.limit"
          :loading="loading"
          @prev="goPrevPage"
          @next="goNextPage"
          @update:limit="onLimitChange"
        />
      </div>
    </el-card>

    <!-- Batch Group Dialog -->
    <el-dialog v-model="groupDialogVisible" :title="t('batch.assignGroup')" width="400px" destroy-on-close>
      <el-form>
        <el-form-item :label="t('deviceGroups.name')">
          <el-select v-model="selectedGroupId" style="width:100%" :placeholder="t('deviceGroups.filterGroup')">
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmBatchGroup" :disabled="!selectedGroupId">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Search, Download } from '@element-plus/icons-vue'
import { getDevices, batchDevices } from '../api/admin_device.js'
import { getGroups } from '../api/device_groups.js'
import { exportCSV } from '../utils/export.js'
import CursorPagination from '../components/CursorPagination.vue'

const { t, locale } = useI18n()
const router = useRouter()
const loading = ref(false)
const devices = ref([])
const tableRef = ref(null)
const selectedIds = ref([])
const groups = ref([])

const filters = reactive({
  search: '',
  os: '',
  online: ''
})

// §3.1 cursor pagination state. See UsersPage.vue for the full rationale.
const pagination = reactive({
  cursorStack: [''],
  nextCursor: '',
  total: 0,
  limit: 20
})

const sort = reactive({
  field: 'created_at',
  order: 'desc'
})

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString(locale.value)
}

function isOnline(device) {
  return Boolean(device.online)
}

async function loadDevices() {
  loading.value = true
  try {
    const data = await getDevices({
      cursor: pagination.cursorStack.at(-1),
      limit: pagination.limit,
      sort: sort.field,
      order: sort.order,
      search: filters.search,
      os: filters.os,
      online: filters.online
    })
    devices.value = data.items || []
    pagination.nextCursor = data.next_cursor || ''
    if (typeof data.total === 'number') pagination.total = data.total
  } catch (e) {
    ElMessage.error(t('common.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function resetCursorAndReload() {
  pagination.cursorStack = ['']
  pagination.nextCursor = ''
  loadDevices()
}

function goPrevPage() {
  if (pagination.cursorStack.length <= 1) return
  pagination.cursorStack.pop()
  loadDevices()
}

function goNextPage() {
  if (!pagination.nextCursor) return
  pagination.cursorStack.push(pagination.nextCursor)
  loadDevices()
}

function onLimitChange(n) {
  pagination.limit = n
  resetCursorAndReload()
}

function handleSearch() {
  resetCursorAndReload()
}

function handleFilter() {
  resetCursorAndReload()
}

function handleSortChange({ prop, order }) {
  sort.field = prop || 'created_at'
  sort.order = order === 'ascending' ? 'asc' : 'desc'
  resetCursorAndReload()
}

function handleRowClick(row) {
  router.push(`/devices/${row.device_id}`)
}

function handleExport() {
  const columns = [
    { key: 'device_id', label: 'Device ID' },
    { key: 'device_uuid', label: 'UUID' },
    { key: 'os', label: 'OS' },
    { key: 'app_version', label: 'Version' },
    { key: 'online', label: 'Online' },
    { key: 'last_seen_at', label: 'Last Seen' },
    { key: 'created_at', label: 'Created At' }
  ]
  exportCSV(columns, devices.value, 'devices.csv')
}

function handleSelectionChange(rows) {
  selectedIds.value = rows.map(r => r.device_id)
}

const groupDialogVisible = ref(false)
const selectedGroupId = ref(null)

async function loadGroups() {
  try {
    const data = await getGroups()
    groups.value = data.items || data.groups || []
  } catch (e) {
    groups.value = []
  }
}

async function handleBatch(action) {
  if (selectedIds.value.length === 0) return
  try {
    // §2.2 batch op names match server enum: delete|assign_group|remove_group.
    if (action === 'assign_group') {
      await loadGroups()
      if (groups.value.length === 0) {
        ElMessage.warning(t('deviceGroups.noGroups'))
        return
      }
      selectedGroupId.value = null
      groupDialogVisible.value = true
      return
    }
    await ElMessageBox.confirm(t('batch.confirmBatch', { count: selectedIds.value.length }), t('common.tip'), { type: 'warning' })
    await batchDevices(action, selectedIds.value)
    ElMessage.success(t('batch.success'))
    loadDevices()
  } catch (e) {
    if (e !== 'cancel' && e?.message) ElMessage.error(e.message)
  }
}

async function confirmBatchGroup() {
  if (!selectedGroupId.value) return
  try {
    await batchDevices('assign_group', selectedIds.value, selectedGroupId.value)
    ElMessage.success(t('batch.success'))
    groupDialogVisible.value = false
    loadDevices()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(loadDevices)
</script>

<style scoped>
.device-list-page {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.filter-card {
  border-radius: 8px;
}

.filter-bar {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

:deep(.clickable-row) {
  cursor: pointer;
}

:deep(.clickable-row:hover td) {
  color: var(--el-color-primary);
}

.batch-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
  padding: 10px 16px;
  background: #ecf5ff;
  border-radius: 4px;
  font-size: 13px;
}
</style>
