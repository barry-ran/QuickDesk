<template>
  <div class="group-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('deviceGroups.title') }}</h2>
      <el-button type="primary" :icon="Plus" size="small" @click="handleAdd">{{ t('deviceGroups.addGroup') }}</el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="groups" stripe style="width: 100%" size="small">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" :label="t('deviceGroups.name')" min-width="150">
          <template #default="{ row }">
            <span :style="{ borderLeft: '3px solid ' + (row.color || '#409eff'), paddingLeft: '8px' }">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" :label="t('deviceGroups.description')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="device_count" :label="t('deviceGroups.deviceCount')" width="100" />
        <el-table-column :label="t('deviceGroups.createdAt')" width="170">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEdit(row)">{{ t('common.edit') }}</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? t('deviceGroups.editGroup') : t('deviceGroups.addGroup')" width="450px" destroy-on-close>
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('deviceGroups.name')">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('deviceGroups.description')">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item :label="t('deviceGroups.color')">
          <el-color-picker v-model="form.color" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { getGroups, createGroup, updateGroup, deleteGroup } from '../api/device_groups.js'

const { t } = useI18n()
const loading = ref(false)
const groups = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)

const form = reactive({ id: null, name: '', description: '', color: '#409eff' })

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN')
}

async function loadGroups() {
  loading.value = true
  try {
    const data = await getGroups()
    groups.value = data.items || data.groups || []
  } catch (e) {
    ElMessage.error(t('common.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isEdit.value = false
  Object.assign(form, { id: null, name: '', description: '', color: '#409eff' })
  dialogVisible.value = true
}

function handleEdit(row) {
  isEdit.value = true
  Object.assign(form, { id: row.id, name: row.name, description: row.description, color: row.color || '#409eff' })
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!form.name) return
  submitting.value = true
  try {
    if (isEdit.value) {
      await updateGroup(form.id, { name: form.name, description: form.description, color: form.color })
    } else {
      await createGroup({ name: form.name, description: form.description, color: form.color })
    }
    dialogVisible.value = false
    loadGroups()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(t('deviceGroups.confirmDelete'), t('common.tip'), { type: 'warning' })
    await deleteGroup(row.id)
    ElMessage.success(t('common.success'))
    loadGroups()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

onMounted(loadGroups)
</script>

<style scoped>
.group-page { width: 100%; padding: 20px; box-sizing: border-box; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; font-size: 24px; font-weight: 600; color: #303133; }
</style>
