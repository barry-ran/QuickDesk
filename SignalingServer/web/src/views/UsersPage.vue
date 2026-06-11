<template>
  <div class="users-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('userMgmt.title') }}</h2>
      <div class="header-actions">
        <el-button :icon="Download" size="small" @click="handleExport">{{ t('common.export') }}</el-button>
        <el-button type="primary" :icon="Plus" size="small" @click="handleAdd">{{ t('userMgmt.addUser') }}</el-button>
      </div>
    </div>

    <!-- Filters -->
    <el-card shadow="never" class="filter-card">
      <div class="filter-bar">
        <el-input
          v-model="filters.search"
          :placeholder="t('userMgmt.searchPlaceholder')"
          clearable
          style="width: 240px"
          @clear="handleSearch"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="filters.level" :placeholder="t('userMgmt.filterLevel')" clearable style="width: 120px" @change="handleFilter">
          <el-option label="V1" value="V1" />
          <el-option label="V2" value="V2" />
          <el-option label="V3" value="V3" />
          <el-option label="V4" value="V4" />
          <el-option label="V5" value="V5" />
        </el-select>
        <el-select v-model="filters.status" :placeholder="t('userMgmt.filterStatus')" clearable style="width: 120px" @change="handleFilter">
          <el-option :label="t('common.enabled')" value="true" />
          <el-option :label="t('common.disabled')" value="false" />
        </el-select>
        <el-select v-model="filters.channel_type" :placeholder="t('userMgmt.filterChannel')" clearable style="width: 140px" @change="handleFilter">
          <el-option :label="t('userMgmt.channelGlobal')" value="全球" />
          <el-option :label="t('userMgmt.channelChina')" value="中国大陆" />
        </el-select>
        <el-button :icon="Search" @click="handleSearch">{{ t('common.search') }}</el-button>
      </div>
    </el-card>

    <!-- Batch Toolbar -->
    <div v-if="selectedIds.length > 0" class="batch-bar">
      <span>{{ t('batch.selected', { count: selectedIds.length }) }}</span>
      <el-button size="small" @click="handleBatch('enable')">{{ t('batch.enable') }}</el-button>
      <el-button size="small" @click="handleBatch('disable')">{{ t('batch.disable') }}</el-button>
      <el-button size="small" type="danger" @click="handleBatch('delete')">{{ t('batch.delete') }}</el-button>
      <el-button size="small" @click="handleBatch('set_level')">{{ t('batch.setLevel') }}</el-button>
    </div>

    <!-- Table -->
    <el-card shadow="never" style="margin-top: 16px">
      <el-table
        :data="users"
        stripe
        style="width: 100%"
        size="small"
        @sort-change="handleSortChange"
        @row-click="handleRowClick"
        @selection-change="handleSelectionChange"
        row-class-name="clickable-row"
      >
        <el-table-column type="selection" width="40" />
        <el-table-column prop="id" label="ID" width="60" sortable="custom" />
        <el-table-column prop="username" :label="t('userMgmt.username')" min-width="120" sortable="custom" />
        <el-table-column prop="phone" :label="t('userMgmt.phone')" min-width="120" />
        <el-table-column prop="email" :label="t('userMgmt.email')" min-width="180" show-overflow-tooltip />
        <el-table-column prop="level" :label="t('userMgmt.level')" width="90" sortable="custom">
          <template #default="{ row }">
            <el-tag :type="getLevelType(row.level)" size="small">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="device_count" :label="t('userMgmt.deviceCount')" width="100" sortable="custom" />
        <el-table-column prop="channel_type" :label="t('userMgmt.channelType')" width="110">
          <template #default="{ row }">
            <el-tag :type="row.channel_type === '全球' ? 'success' : 'warning'" size="small">
              {{ row.channel_type === '全球' ? t('userMgmt.channelGlobal') : t('userMgmt.channelChina') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('common.status')" width="80">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              size="small"
              @click.stop
              @change="(val) => handleStatusChange(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('userMgmt.createdAt')" width="170" sortable="custom" prop="created_at">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click.stop="handleEdit(row)">
              <el-icon><Edit /></el-icon>
              {{ t('common.edit') }}
            </el-button>
            <el-button link type="danger" size="small" @click.stop="handleDelete(row)">
              <el-icon><Delete /></el-icon>
              {{ t('common.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-bar">
        <!-- §3.1 cursor-based pagination. The Element-Plus el-pagination
             jumper/pager model doesn't fit opaque cursors, so we use the
             custom CursorPagination component that emits prev / next /
             limit change events against the server's {items, next_cursor,
             total} envelope. -->
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

    <!-- Add/Edit Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? t('userMgmt.editUser') : t('userMgmt.addUser')"
      width="500px"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
      >
        <el-form-item :label="t('userMgmt.username')" prop="username">
          <el-input v-model="form.username" :placeholder="t('userMgmt.usernamePlaceholder')" :disabled="isEdit" />
        </el-form-item>
        <el-form-item :label="t('userMgmt.phone')" prop="phone">
          <el-input v-model="form.phone" :placeholder="t('userMgmt.phonePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('userMgmt.email')" prop="email">
          <el-input v-model="form.email" :placeholder="t('userMgmt.emailPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('userMgmt.password')" prop="password" v-if="!isEdit">
          <el-input v-model="form.password" type="password" :placeholder="t('userMgmt.passwordPlaceholder')" show-password />
        </el-form-item>
        <el-form-item :label="t('userMgmt.password')" prop="password" v-else>
          <el-input v-model="form.password" type="password" :placeholder="t('userMgmt.passwordHint')" show-password />
        </el-form-item>
        <el-form-item :label="t('userMgmt.level')" prop="level">
          <el-select v-model="form.level" :placeholder="t('userMgmt.levelPlaceholder')" style="width: 100%">
            <el-option label="V1" value="V1" />
            <el-option label="V2" value="V2" />
            <el-option label="V3" value="V3" />
            <el-option label="V4" value="V4" />
            <el-option label="V5" value="V5" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('userMgmt.deviceCount')" prop="device_count">
          <el-input-number v-model="form.device_count" :min="0" style="width: 100%" />
        </el-form-item>
        <el-form-item :label="t('userMgmt.channelType')" prop="channel_type">
          <el-select v-model="form.channel_type" :placeholder="t('userMgmt.channelPlaceholder')" style="width: 100%">
            <el-option :label="t('userMgmt.channelGlobal')" value="全球" />
            <el-option :label="t('userMgmt.channelChina')" value="中国大陆" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.status')" prop="status">
          <el-switch v-model="form.status" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Batch Set Level Dialog -->
    <el-dialog v-model="levelDialogVisible" :title="t('batch.setLevel')" width="400px" destroy-on-close>
      <el-form>
        <el-form-item :label="t('userMgmt.level')">
          <el-select v-model="selectedLevel" style="width:100%" :placeholder="t('userMgmt.levelPlaceholder')">
            <el-option label="V1" value="V1" />
            <el-option label="V2" value="V2" />
            <el-option label="V3" value="V3" />
            <el-option label="V4" value="V4" />
            <el-option label="V5" value="V5" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="levelDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmBatchLevel" :disabled="!selectedLevel">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Edit, Delete, Search, Download } from '@element-plus/icons-vue'
import { getUsers, createUser, updateUser, deleteUser, batchUsers } from '../api/users.js'
import { exportCSV } from '../utils/export.js'
import CursorPagination from '../components/CursorPagination.vue'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const users = ref([])
const selectedIds = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref(null)

const filters = reactive({
  search: '',
  level: '',
  status: '',
  channel_type: ''
})

// §3.1 cursor pagination state.
//   cursorStack[0] is '' (first page). The last entry is the cursor that
//   produced the *currently displayed* page. nextCursor comes from the
//   previous server response. See components/CursorPagination.vue.
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

const form = reactive({
  id: null,
  username: '',
  phone: '',
  email: '',
  password: '',
  level: 'V1',
  device_count: 0,
  channel_type: '全球',
  status: true
})

const rules = computed(() => ({
  username: [{ required: true, message: t('userMgmt.usernameRequired'), trigger: 'blur' }],
  password: [{ required: !isEdit.value, message: t('userMgmt.passRequired'), trigger: 'blur' }]
}))

function formatDate(dateStr) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

function getLevelType(level) {
  const types = { 'V1': '', 'V2': 'success', 'V3': 'warning', 'V4': 'danger', 'V5': 'info' }
  return types[level] || ''
}

async function loadUsers() {
  loading.value = true
  try {
    const data = await getUsers({
      cursor: pagination.cursorStack.at(-1),
      limit: pagination.limit,
      sort: sort.field,
      order: sort.order,
      search: filters.search,
      level: filters.level,
      status: filters.status,
      channel_type: filters.channel_type
    })
    users.value = data.items || data.users || []
    pagination.nextCursor = data.next_cursor || ''
    if (typeof data.total === 'number') pagination.total = data.total
  } catch (e) {
    ElMessage.error(t('userMgmt.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function resetCursorAndReload() {
  pagination.cursorStack = ['']
  pagination.nextCursor = ''
  loadUsers()
}

function goPrevPage() {
  if (pagination.cursorStack.length <= 1) return
  pagination.cursorStack.pop()
  loadUsers()
}

function goNextPage() {
  if (!pagination.nextCursor) return
  pagination.cursorStack.push(pagination.nextCursor)
  loadUsers()
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
  // Sort changes invalidate the cursor stack since the cursor encodes
  // a position under the previous order.
  resetCursorAndReload()
}

function handleRowClick(row) {
  router.push(`/users/${row.id}`)
}

function handleAdd() {
  isEdit.value = false
  Object.assign(form, {
    id: null,
    username: '',
    phone: '',
    email: '',
    password: '',
    level: 'V1',
    device_count: 0,
    channel_type: '全球',
    status: true
  })
  dialogVisible.value = true
}

function handleEdit(row) {
  isEdit.value = true
  Object.assign(form, {
    id: row.id,
    username: row.username,
    phone: row.phone,
    email: row.email,
    password: '',
    level: row.level,
    device_count: row.device_count,
    channel_type: row.channel_type,
    status: row.status
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateUser(form.id, form)
      ElMessage.success(t('userMgmt.updated'))
    } else {
      await createUser(form)
      ElMessage.success(t('userMgmt.created'))
    }
    dialogVisible.value = false
    loadUsers()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(t('userMgmt.confirmDelete'), t('common.tip'), { type: 'warning' })
    await deleteUser(row.id)
    ElMessage.success(t('userMgmt.deleted'))
    loadUsers()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error(t('userMgmt.deleteFailed') + ': ' + e.message)
    }
  }
}

async function handleStatusChange(row, status) {
  try {
    await updateUser(row.id, { status })
    ElMessage.success(t('userMgmt.statusUpdated'))
  } catch (e) {
    ElMessage.error(t('userMgmt.statusFailed') + ': ' + e.message)
    row.status = !status
  }
}

function handleExport() {
  const columns = [
    { key: 'id', label: 'ID' },
    { key: 'username', label: 'Username' },
    { key: 'phone', label: 'Phone' },
    { key: 'email', label: 'Email' },
    { key: 'level', label: 'Level' },
    { key: 'device_count', label: 'Device Count' },
    { key: 'channel_type', label: 'Channel' },
    { key: 'status', label: 'Status' },
    { key: 'created_at', label: 'Created At' }
  ]
  exportCSV(columns, users.value, 'users.csv')
}

const levelDialogVisible = ref(false)
const selectedLevel = ref('')

function handleSelectionChange(rows) {
  selectedIds.value = rows.map(r => r.id)
}

async function handleBatch(action) {
  if (selectedIds.value.length === 0) return
  try {
    // §2.2 batch op names match server enum: enable|disable|delete|set_level.
    if (action === 'set_level') {
      selectedLevel.value = ''
      levelDialogVisible.value = true
      return
    }
    await ElMessageBox.confirm(t('batch.confirmBatch', { count: selectedIds.value.length }), t('common.tip'), { type: 'warning' })
    await batchUsers(action, selectedIds.value)
    ElMessage.success(t('batch.success'))
    loadUsers()
  } catch (e) {
    if (e !== 'cancel' && e?.message) ElMessage.error(e.message)
  }
}

async function confirmBatchLevel() {
  if (!selectedLevel.value) return
  try {
    await batchUsers('set_level', selectedIds.value, selectedLevel.value)
    ElMessage.success(t('batch.success'))
    levelDialogVisible.value = false
    loadUsers()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(loadUsers)
</script>

<style scoped>
.users-page {
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
