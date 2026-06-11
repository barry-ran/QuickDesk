<template>
  <div class="webhooks-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('webhooks.title') }}</h2>
      <el-button type="primary" :icon="Plus" size="small" @click="handleAdd">{{ t('webhooks.addWebhook') }}</el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="webhooks" stripe style="width: 100%" size="small">
        <el-table-column prop="name" :label="t('webhooks.name')" min-width="120" />
        <el-table-column prop="url" label="URL" min-width="200" show-overflow-tooltip />
        <el-table-column :label="t('webhooks.events')" min-width="200">
          <template #default="{ row }">
            <el-tag v-for="ev in parseEvents(row.events)" :key="ev" size="small" style="margin-right: 4px">{{ ev }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.status')" width="80">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" @change="handleToggle(row)" />
          </template>
        </el-table-column>
        <el-table-column :label="t('webhooks.lastStatus')" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.last_status" :type="row.last_status < 300 ? 'success' : 'danger'" size="small">{{ row.last_status }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleTest(row)">{{ t('webhooks.test') }}</el-button>
            <el-button link type="primary" size="small" @click="handleEdit(row)">{{ t('common.edit') }}</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? t('webhooks.editWebhook') : t('webhooks.addWebhook')" width="550px" destroy-on-close>
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('webhooks.name')">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="URL">
          <el-input v-model="form.url" placeholder="https://example.com/webhook" />
        </el-form-item>
        <el-form-item label="Secret">
          <el-input v-model="form.secret" placeholder="HMAC signing secret (optional)" />
        </el-form-item>
        <el-form-item :label="t('webhooks.events')">
          <el-checkbox-group v-model="form.events">
            <el-checkbox label="device_online">device_online</el-checkbox>
            <el-checkbox label="device_offline">device_offline</el-checkbox>
            <el-checkbox label="new_device">new_device</el-checkbox>
            <el-checkbox label="new_user">new_user</el-checkbox>
            <el-checkbox label="connection_failed">connection_failed</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="t('common.enabled')">
          <el-switch v-model="form.enabled" />
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
import { getWebhooks, createWebhook, updateWebhook, deleteWebhook, testWebhook } from '../api/webhooks.js'

const { t } = useI18n()
const loading = ref(false)
const webhooks = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)

const form = reactive({ id: null, name: '', url: '', secret: '', events: [], enabled: true })

function parseEvents(events) {
  try { return JSON.parse(events) } catch { return [] }
}

async function loadWebhooks() {
  loading.value = true
  try {
    const data = await getWebhooks()
    webhooks.value = data.items || data.webhooks || []
  } catch (e) {
    ElMessage.error(t('common.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  isEdit.value = false
  Object.assign(form, { id: null, name: '', url: '', secret: '', events: [], enabled: true })
  dialogVisible.value = true
}

function handleEdit(row) {
  isEdit.value = true
  Object.assign(form, { id: row.id, name: row.name, url: row.url, secret: row.secret || '', events: parseEvents(row.events), enabled: row.enabled })
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!form.name || !form.url || form.events.length === 0) {
    ElMessage.warning(t('webhooks.fillRequired'))
    return
  }
  submitting.value = true
  try {
    if (isEdit.value) {
      await updateWebhook(form.id, { name: form.name, url: form.url, secret: form.secret, events: form.events, enabled: form.enabled })
    } else {
      await createWebhook({ name: form.name, url: form.url, secret: form.secret, events: form.events, enabled: form.enabled })
    }
    dialogVisible.value = false
    loadWebhooks()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function handleToggle(row) {
  try {
    await updateWebhook(row.id, { enabled: row.enabled })
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error(e.message)
  }
}

async function handleTest(row) {
  try {
    await testWebhook(row.id)
    ElMessage.success(t('webhooks.testSent'))
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(t('webhooks.confirmDelete'), t('common.tip'), { type: 'warning' })
    await deleteWebhook(row.id)
    ElMessage.success(t('common.success'))
    loadWebhooks()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

onMounted(loadWebhooks)
</script>

<style scoped>
.webhooks-page { width: 100%; padding: 20px; box-sizing: border-box; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { margin: 0; font-size: 24px; font-weight: 600; color: #303133; }
</style>
