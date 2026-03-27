<template>
  <div class="settings-page" v-loading="loading">
    <div class="page-header">
      <h2>系统设置</h2>
      <el-button type="primary" size="small" @click="handleSave" :loading="saving" :icon="Check">
        保存设置
      </el-button>
    </div>

    <div class="settings-container">
      <!-- 基础配置 -->
      <el-card class="settings-card" shadow="never">
        <template #header>
          <div class="card-header">
            <el-icon><Setting /></el-icon>
            <span>Basic</span>
          </div>
        </template>
        <el-form label-width="120px" label-position="top">
          <el-form-item label="Site Enabled">
            <el-switch v-model="form.siteEnabled" active-text="On" inactive-text="Off" />
          </el-form-item>
          <el-form-item label="Site Name">
            <el-input v-model="form.siteName" placeholder="QuickDesk" style="max-width:400px" />
          </el-form-item>
        </el-form>
      </el-card>

      <!-- ICE / TURN 配置 -->
      <el-card class="settings-card" shadow="never">
        <template #header>
          <div class="card-header">
            <el-icon><Connection /></el-icon>
            <span>ICE / TURN / STUN</span>
          </div>
        </template>
        <el-form label-width="180px" label-position="top">
          <el-form-item label="TURN Server URLs">
            <div class="form-tip">WebRTC relay servers. Clients will use these when direct connection fails.</div>
            <ListEditor
              v-model="turnUrlList"
              placeholder="turn:your-server.com:3478"
              add-text="Add TURN Server"
            />
          </el-form-item>

          <el-form-item label="TURN Auth Secret">
            <el-input
              v-model="form.turnAuthSecret"
              placeholder="Shared secret matching coturn static-auth-secret"
              style="max-width:500px"
              show-password
              type="password"
            />
            <div class="form-tip">Must match the static-auth-secret in your coturn configuration.</div>
          </el-form-item>

          <el-form-item label="TURN Credential TTL (seconds)">
            <el-input-number v-model="form.turnCredentialTtl" :min="300" :max="604800" :step="3600" />
            <div class="form-tip">How long TURN credentials remain valid. Default: 86400 (24 hours).</div>
          </el-form-item>

          <el-form-item label="STUN Server URLs">
            <div class="form-tip">NAT traversal servers for discovering public IP addresses.</div>
            <ListEditor
              v-model="stunUrlList"
              placeholder="stun:stun.example.com:3478"
              add-text="Add STUN Server"
            />
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 安全配置 -->
      <el-card class="settings-card" shadow="never">
        <template #header>
          <div class="card-header">
            <el-icon><Lock /></el-icon>
            <span>Security</span>
          </div>
        </template>
        <el-form label-width="180px" label-position="top">
          <el-form-item label="API Key">
            <el-input
              v-model="form.apiKey"
              placeholder="Leave empty to allow all clients"
              style="max-width:500px"
              show-password
              type="password"
            />
            <div class="form-tip">
              When set, clients must send this key via X-API-Key header to access signaling APIs.
              Leave empty to disable API key authentication.
            </div>
          </el-form-item>

          <el-form-item label="Allowed Origins">
            <div class="form-tip">
              Restrict WebClient access to specific domains. Leave empty to disable origin checking.
            </div>
            <ListEditor
              v-model="allowedOriginList"
              placeholder="https://web.quickdesk.example.com"
              add-text="Add Origin"
            />
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 阿里云短信配置 -->
      <el-card class="settings-card" shadow="never">
        <template #header>
          <div class="card-header">
            <el-icon><ChatLineRound /></el-icon>
            <span>Aliyun SMS</span>
            <el-tag v-if="isSmsEnabled" type="success" size="small" style="margin-left:auto">Enabled</el-tag>
            <el-tag v-else type="info" size="small" style="margin-left:auto">Disabled</el-tag>
          </div>
        </template>
        <el-form label-width="180px" label-position="top">
          <div class="form-tip" style="margin-bottom:12px">
            Configure Aliyun SMS for phone number verification (login/register).
            All four fields must be filled to enable SMS features. Leave empty to disable.
          </div>

          <el-form-item label="Access Key ID">
            <el-input
              v-model="form.smsAccessKeyId"
              placeholder="e.g. LTAI5txxxxxxxxxx"
              style="max-width:500px"
            />
          </el-form-item>

          <el-form-item label="Access Key Secret">
            <el-input
              v-model="form.smsAccessKeySecret"
              placeholder="e.g. eHjxxxxxxxxxxxxxxxxxx"
              style="max-width:500px"
              show-password
              type="password"
            />
          </el-form-item>

          <el-form-item label="Sign Name">
            <el-input
              v-model="form.smsSignName"
              placeholder="e.g. QuickDesk"
              style="max-width:400px"
            />
            <div class="form-tip">The SMS signature approved in your Aliyun SMS console.</div>
          </el-form-item>

          <el-form-item label="Template Code">
            <el-input
              v-model="form.smsTemplateCode"
              placeholder="e.g. SMS_123456789"
              style="max-width:400px"
            />
            <div class="form-tip">The template code for verification code SMS, must contain ${code} variable.</div>
          </el-form-item>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Setting, Check, Connection, Lock, ChatLineRound } from '@element-plus/icons-vue'
import { useSettingsStore } from '../stores/settings.js'
import { authFetch } from '../api/auth.js'
import ListEditor from './ListEditor.vue'

const settingsStore = useSettingsStore()
const loading = ref(false)
const saving = ref(false)

const form = reactive({
  siteEnabled: true,
  siteName: 'QuickDesk',
  loginLogo: '',
  smallLogo: '',
  favicon: '',
  turnUrls: '',
  turnAuthSecret: '',
  turnCredentialTtl: 86400,
  stunUrls: '',
  apiKey: '',
  allowedOrigins: '',
  smsAccessKeyId: '',
  smsAccessKeySecret: '',
  smsSignName: '',
  smsTemplateCode: ''
})

const isSmsEnabled = computed(() =>
  form.smsAccessKeyId && form.smsAccessKeySecret && form.smsSignName && form.smsTemplateCode
)

const turnUrlList = computed({
  get: () => textToList(form.turnUrls),
  set: (v) => { form.turnUrls = listToText(v) }
})

const stunUrlList = computed({
  get: () => textToList(form.stunUrls),
  set: (v) => { form.stunUrls = listToText(v) }
})

const allowedOriginList = computed({
  get: () => textToList(form.allowedOrigins),
  set: (v) => { form.allowedOrigins = listToText(v) }
})

function textToList(text) {
  if (!text) return []
  return text.split('\n').map(s => s.trim()).filter(Boolean)
}

function listToText(list) {
  return list.filter(Boolean).join('\n')
}

async function loadSettings() {
  loading.value = true
  try {
    const res = await authFetch('/api/v1/admin/settings')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    Object.assign(form, data)
    if (!form.turnCredentialTtl) form.turnCredentialTtl = 86400
  } catch (e) {
    ElMessage.error('Failed to load settings: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    const res = await authFetch('/api/v1/admin/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form)
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    settingsStore.updateSettings(form)
    ElMessage.success('Settings saved')
  } catch (e) {
    ElMessage.error('Failed to save settings: ' + e.message)
  } finally {
    saving.value = false
  }
}

onMounted(loadSettings)
</script>

<style scoped>
.settings-page {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.settings-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.settings-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.form-tip {
  margin-top: 4px;
  margin-bottom: 8px;
  color: #909399;
  font-size: 13px;
  line-height: 1.5;
}

@media (max-width: 768px) {
  .settings-page { padding: 10px; }
  .page-header { flex-direction: column; align-items: flex-start; gap: 10px; }
}
</style>
