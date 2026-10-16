<template>
  <div class="update-releases-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('updateReleases.title') }}</h2>
      <div class="header-actions">
        <el-select v-model="channel" clearable :placeholder="t('updateReleases.allChannels')" style="width: 150px">
          <el-option label="stable" value="stable" />
          <el-option label="beta" value="beta" />
          <el-option label="nightly" value="nightly" />
        </el-select>
        <el-button :icon="Refresh" size="small" @click="loadReleases">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" :icon="Plus" size="small" @click="handlePublish">
          {{ t('updateReleases.publish') }}
        </el-button>
      </div>
    </div>

    <el-card shadow="never">
      <el-table :data="releases" stripe style="width: 100%" size="small">
        <el-table-column prop="channel" :label="t('updateReleases.channel')" width="80" />
        <el-table-column :label="t('updateReleases.platform')" width="100">
          <template #default="{ row }">{{ row.platform }} / {{ row.arch }}</template>
        </el-table-column>
        <el-table-column prop="version" :label="t('updateReleases.version')" width="110" />
        <el-table-column :label="t('updateReleases.mandatory')" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.mandatory" type="danger" size="small">{{ t('updateReleases.requiredTag') }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('updateReleases.downloadUrl')" min-width="230">
          <template #default="{ row }">
            <div class="url-cell">
              <el-tag v-if="row.hosted_here" size="small" type="success">
                {{ t('updateReleases.hostedHere') }}
              </el-tag>
              <el-tag v-else-if="!row.download_url" size="small" type="warning">
                {{ t('updateReleases.pendingArtifact') }}
              </el-tag>
              <el-link
                v-if="row.download_url"
                class="url-link"
                :href="row.download_url"
                :title="row.download_url"
                target="_blank"
                type="primary"
              >{{ row.download_url }}</el-link>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('updateReleases.sha256')" width="130">
          <template #default="{ row }">
            <span class="mono" :title="row.sha256">{{ shortDigest(row.sha256) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('updateReleases.size')" width="90">
          <template #default="{ row }">{{ formatSize(row.size_bytes) }}</template>
        </el-table-column>
        <el-table-column :label="t('updateReleases.publishedAt')" width="150">
          <template #default="{ row }">{{ formatTime(row.published_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              :loading="uploadingId === row.id"
              @click="handleUpload(row)"
            >{{ t('updateReleases.upload') }}</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="releases.length === 0 && !loading" class="empty-hint">{{ t('updateReleases.empty') }}</div>
    </el-card>

    <!-- Hidden picker: the per-row button chooses the release, then this input
         collects the file. Only one upload runs at a time. -->
    <input ref="fileInput" type="file" class="hidden-file-input" @change="handleFilePicked" />

    <el-dialog v-model="dialogVisible" :title="t('updateReleases.publish')" width="620px" destroy-on-close>
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('updateReleases.channel')">
          <el-select v-model="form.channel" style="width: 180px">
            <el-option label="stable" value="stable" />
            <el-option label="beta" value="beta" />
            <el-option label="nightly" value="nightly" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('updateReleases.platform')">
          <el-select v-model="form.platform" style="width: 180px">
            <el-option label="windows" value="windows" />
            <el-option label="macos" value="macos" />
            <el-option label="linux" value="linux" />
          </el-select>
          <el-select v-model="form.arch" style="width: 150px; margin-left: 12px">
            <el-option label="x64" value="x64" />
            <el-option label="arm64" value="arm64" />
            <el-option label="universal" value="universal" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('updateReleases.version')">
          <el-input v-model="form.version" placeholder="4.2.5.0" style="width: 220px" />
        </el-form-item>
        <el-form-item :label="t('updateReleases.downloadUrl')">
          <el-input v-model="form.download_url" placeholder="https://updates.example.com/QuickDesk-4.2.5.0.exe" />
          <span class="field-hint">{{ t('updateReleases.downloadUrlHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('updateReleases.sha256')">
          <el-input v-model="form.sha256" placeholder="64-character hex digest" />
          <span class="field-hint">{{ t('updateReleases.uploadHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('updateReleases.size')">
          <el-input-number v-model="form.size_bytes" :min="0" :step="1048576" controls-position="right" style="width: 220px" />
        </el-form-item>
        <el-form-item :label="t('updateReleases.notes')">
          <el-input v-model="form.notes" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item :label="t('updateReleases.mandatory')">
          <el-switch v-model="form.mandatory" />
          <span class="field-hint">{{ t('updateReleases.mandatoryHint') }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import {
  getUpdateReleases,
  publishUpdateRelease,
  deleteUpdateRelease,
  uploadUpdateReleaseArtifact,
} from '../api/updateReleases.js'

const { t } = useI18n()
const loading = ref(false)
const submitting = ref(false)
const uploadingId = ref(null)
const dialogVisible = ref(false)
const releases = ref([])
const channel = ref('')
const fileInput = ref(null)
// The row whose "upload" button opened the picker; the file arrives later.
const uploadTarget = ref(null)

const form = reactive({
  channel: 'stable',
  platform: 'windows',
  arch: 'x64',
  version: '',
  download_url: '',
  sha256: '',
  size_bytes: 0,
  notes: '',
  mandatory: false,
})

function shortDigest(value) {
  if (!value) return '-'
  return value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-6)}` : value
}

function formatSize(value) {
  if (!Number.isFinite(value) || value <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit ? 1 : 0)} ${units[unit]}`
}

function formatTime(value) {
  return value ? new Date(value).toLocaleString() : '-'
}

async function loadReleases() {
  loading.value = true
  try {
    const data = await getUpdateReleases({ channel: channel.value })
    releases.value = data.items || []
  } catch (e) {
    ElMessage.error(t('common.loadFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function handlePublish() {
  Object.assign(form, {
    channel: 'stable',
    platform: 'windows',
    arch: 'x64',
    version: '',
    download_url: '',
    sha256: '',
    size_bytes: 0,
    notes: '',
    mandatory: false,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!form.version || !form.sha256) {
    ElMessage.warning(t('updateReleases.fillRequired'))
    return
  }
  submitting.value = true
  try {
    await publishUpdateRelease({ ...form })
    ElMessage.success(t('common.success'))
    dialogVisible.value = false
    loadReleases()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    submitting.value = false
  }
}

// Re-hosting: the release already carries the SHA-256, so the server refuses an
// installer that does not hash to it. That is why the form asks for the digest
// up front even when the file is uploaded here instead of linked.
function handleUpload(row) {
  if (uploadingId.value) return
  uploadTarget.value = row
  if (fileInput.value) {
    fileInput.value.value = ''
    fileInput.value.click()
  }
}

async function handleFilePicked(event) {
  const file = event.target.files && event.target.files[0]
  const row = uploadTarget.value
  uploadTarget.value = null
  event.target.value = ''
  if (!file || !row) return
  uploadingId.value = row.id
  try {
    await uploadUpdateReleaseArtifact(row.id, file)
    ElMessage.success(t('common.success'))
    await loadReleases()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    uploadingId.value = null
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(t('updateReleases.confirmDelete'), t('common.tip'), { type: 'warning' })
    await deleteUpdateRelease(row.id)
    ElMessage.success(t('common.success'))
    loadReleases()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

onMounted(loadReleases)
</script>

<style scoped>
.update-releases-page { width: 100%; padding: 20px; box-sizing: border-box; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; gap: 12px; flex-wrap: wrap; }
.page-header h2 { margin: 0; font-size: 24px; font-weight: 600; color: #303133; }
.header-actions { display: flex; align-items: center; gap: 8px; }
.empty-hint { padding: 24px; text-align: center; color: #909399; font-size: 13px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
.field-hint { margin-left: 10px; color: #909399; font-size: 12px; }
/* The URL is long and the tag has to stay visible, so the cell is a flex row
   and only the link ellipsises; the full URL lives in its title attribute. */
.url-cell { display: flex; align-items: center; gap: 8px; min-width: 0; }
.url-link {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hidden-file-input { display: none; }
</style>
