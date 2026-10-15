<template>
  <div>
    <h2 class="page-title">{{ $t('recordings.title') }}</h2>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('recordings.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm" style="margin-top:12px;" @click="showLogin()">
          {{ $t('user.login') }}
        </button>
      </div>
    </div>

    <template v-else>
      <div style="display:flex;justify-content:flex-end;margin-bottom:12px;">
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="refresh">
          🔄 {{ $t('recordings.refresh') }}
        </button>
      </div>

      <div class="card">
        <!-- Loading (initial load only) -->
        <div v-if="loading && recordings.length === 0" class="empty-state">
          <div class="empty-icon">⏳</div>
          <p>{{ $t('recordings.loading') }}</p>
        </div>

        <!-- Error -->
        <div v-else-if="error" class="empty-state">
          <div class="empty-icon">⚠️</div>
          <p class="hint error">{{ error }}</p>
          <button class="btn btn-secondary btn-sm" style="margin-top:12px;" @click="refresh">
            {{ $t('recordings.retry') }}
          </button>
        </div>

        <!-- Empty -->
        <div v-else-if="recordings.length === 0" class="empty-state">
          <div class="empty-icon">🎬</div>
          <p>{{ $t('recordings.empty') }}</p>
          <p class="hint">{{ $t('recordings.emptyHint') }}</p>
        </div>

        <!-- List -->
        <div v-else>
          <div v-for="r in recordings" :key="r.id" class="recording-item">
            <div :class="['status-dot', statusClass(r.status)]"></div>
            <div class="recording-info">
              <div class="recording-name">
                {{ r.device_name || r.device_id }}
                <span :class="['badge', statusClass(r.status)]">{{ statusLabel(r.status) }}</span>
              </div>
              <div class="recording-meta">
                <span>{{ formatTime(r.started_at) }}</span>
                <span v-if="r.duration_seconds">{{ formatDuration(r.duration_seconds) }}</span>
                <span v-if="r.format">{{ r.format.toUpperCase() }}</span>
                <span v-if="r.size_bytes">{{ formatSize(r.size_bytes) }}</span>
              </div>
              <div v-if="detailFor === r.id && detail" class="recording-detail">
                <div v-if="detail.device_id"><b>{{ $t('recordings.device') }}:</b> {{ detail.device_id }}</div>
                <div v-if="detail.session_id"><b>session_id:</b> {{ detail.session_id }}</div>
                <div v-if="detail.ended_at"><b>{{ $t('recordings.endedAt') }}:</b> {{ formatTime(detail.ended_at) }}</div>
              </div>
            </div>
            <div class="recording-actions">
              <button
                class="icon-btn"
                :title="$t('recordings.detail')"
                @click="toggleDetail(r)"
              >ℹ️</button>
              <button
                class="btn btn-primary btn-sm"
                :disabled="busy === r.id"
                :title="hasMedia(r) ? $t('recordings.play') : $t('recordings.noMedia')"
                @click="playRecording(r)"
              >{{ $t('recordings.play') }}</button>
              <button
                class="icon-btn"
                :disabled="busy === r.id"
                :title="hasMedia(r) ? $t('recordings.download') : $t('recordings.noMedia')"
                @click="downloadRecording(r)"
              >⬇️</button>
              <button
                class="icon-btn danger"
                :disabled="busy === r.id"
                :title="$t('recordings.delete')"
                @click="removeRecording(r)"
              >🗑</button>
            </div>
          </div>

          <div v-if="hasMore" style="text-align:center;margin-top:12px;">
            <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadMore">
              {{ loading ? '...' : $t('recordings.loadMore') }}
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, ref, inject, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'

const { t } = useI18n()
const showToast = inject('showToast')
const showLogin = inject('showLogin')
const authState = inject('authState')

const recordings = ref([])
const nextCursor = ref('')
const loading = ref(false)
const error = ref('')
const busy = ref(null)          // id currently being mutated
const detailFor = ref(null)     // id expanded
const detail = ref(null)        // fetched GET /v1/me/recordings/:id payload
const hasMore = computed(() => !!nextCursor.value)

function statusClass(status) {
  if (status === 'completed') return 'completed'
  if (status === 'recording') return 'recording'
  if (status === 'failed')    return 'failed'
  return 'unknown'
}

function statusLabel(status) {
  const key = `recordings.status.${status}`
  const str = t(key)
  return str === key ? t('recordings.status.unknown') : str
}

function formatTime(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleString()
}

function formatDuration(seconds) {
  const s = Math.max(0, Math.floor(seconds || 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  const pad = (n) => String(n).padStart(2, '0')
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${m}:${pad(sec)}`
}

function formatSize(bytes) {
  const b = Number(bytes) || 0
  if (b < 1024) return `${b} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let v = b / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(v >= 10 ? 0 : 1)} ${units[i]}`
}

// The recording engine owns the media bytes; the control plane only indexes
// metadata. Prefer artifact_url (a browser-reachable location); fall back to
// nothing when only a server-local artifact_path is present.
function resolveMediaUrl(row) {
  if (!row) return null
  const url = row.artifact_url
  if (!url) return null
  if (/^(https?:)?\/\//i.test(url)) return url
  // Relative URL → anchor against the HTTP form of the signaling server.
  let base = userApi.getServerUrl().replace(/\/+$/, '')
  if (base.startsWith('wss://')) base = base.replace(/^wss:\/\//, 'https://')
  else if (base.startsWith('ws://')) base = base.replace(/^ws:\/\//, 'http://')
  return `${base}${url.startsWith('/') ? '' : '/'}${url}`
}

function hasMedia(row) {
  return !!resolveMediaUrl(row)
}

// §2.2 / T2: list envelope is { items, next_cursor }.
async function loadPage(cursor) {
  loading.value = true
  error.value = ''
  try {
    const r = await userApi.fetchRecordings(cursor)
    if (!r.ok) {
      // A terminal 401 already triggered the session-ended flow; keep the
      // page quiet in that case and only surface non-auth failures.
      if (r.status !== 401) error.value = r.error || t('recordings.loadFailed')
      return
    }
    const items = Array.isArray(r.data?.items) ? r.data.items : []
    recordings.value = cursor ? recordings.value.concat(items) : items
    nextCursor.value = r.data?.next_cursor || ''
  } finally {
    loading.value = false
  }
}

function refresh() {
  detailFor.value = null
  detail.value = null
  return loadPage('')
}

function loadMore() {
  if (!nextCursor.value) return
  return loadPage(nextCursor.value)
}

// Fetch the freshest metadata before opening the artifact — the artifact_url
// may have been finalized after the session ended (GET /v1/me/recordings/:id).
async function playRecording(row) {
  const fresh = await loadDetail(row.id)
  const url = resolveMediaUrl(fresh || row)
  if (!url) { showToast(t('recordings.noMediaHint'), 'info'); return }
  window.open(url, `quickdesk_recording_${row.id}`)
}

async function downloadRecording(row) {
  const fresh = await loadDetail(row.id)
  const url = resolveMediaUrl(fresh || row)
  if (!url) { showToast(t('recordings.noMediaHint'), 'info'); return }
  const a = document.createElement('a')
  a.href = url
  a.download = suggestedFilename(fresh || row)
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

function suggestedFilename(row) {
  const ext = (row.format || 'mp4').toLowerCase()
  const base = `quickdesk-${row.device_id || 'session'}-${row.id}`
  return `${base}.${ext}`
}

async function loadDetail(id) {
  busy.value = id
  try {
    const r = await userApi.fetchRecording(id)
    if (!r.ok) {
      if (r.status !== 401 && r.status !== 404) showToast(r.error || t('recordings.detailFailed'), 'error')
      return null
    }
    // Keep the open detail panel in sync when the user is inspecting a row.
    if (detailFor.value === id) detail.value = r.data
    return r.data
  } finally {
    if (busy.value === id) busy.value = null
  }
}

function toggleDetail(row) {
  if (detailFor.value === row.id) {
    detailFor.value = null
    detail.value = null
    return
  }
  detailFor.value = row.id
  detail.value = null
  loadDetail(row.id)
}

async function removeRecording(row) {
  if (!confirm(t('recordings.confirmDelete'))) return
  busy.value = row.id
  try {
    const r = await userApi.deleteRecording(row.id)
    if (!r.ok) {
      showToast(r.error || t('recordings.deleteFailed'), 'error')
      return
    }
    recordings.value = recordings.value.filter(x => x.id !== row.id)
    if (detailFor.value === row.id) { detailFor.value = null; detail.value = null }
    showToast(t('recordings.deleted'), 'success')
  } finally {
    busy.value = null
  }
}

onMounted(() => {
  if (authState.isLoggedIn) refresh()
})
</script>

<style scoped>
.recording-item {
  display: flex; align-items: center; gap: 12px;
  padding: 12px; border-radius: var(--radius);
  transition: background 0.15s;
}
.recording-item:hover { background: rgba(255,255,255,0.05); }
.status-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.status-dot.completed { background: var(--success); }
.status-dot.recording { background: var(--accent); }
.status-dot.failed    { background: var(--error); }
.status-dot.unknown   { background: var(--text-disabled); }
.recording-info { flex: 1; min-width: 0; }
.recording-name { font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.recording-meta { font-size: 12px; color: var(--text-secondary); margin-top: 2px; display: flex; gap: 10px; flex-wrap: wrap; }
.recording-detail { font-size: 12px; color: var(--text-secondary); margin-top: 6px; display: flex; flex-direction: column; gap: 2px; font-family: 'Consolas', monospace; }
.badge { margin-left: 6px; padding: 1px 6px; border-radius: 4px; font-size: 11px; font-weight: 500; }
.badge.completed { background: rgba(76,175,80,0.15); color: var(--success); }
.badge.recording { background: rgba(233,69,96,0.15); color: var(--accent); }
.badge.failed    { background: rgba(244,67,54,0.15); color: var(--error); }
.badge.unknown   { background: rgba(255,255,255,0.1); color: var(--text-secondary); }
.recording-actions { display: flex; gap: 4px; flex-shrink: 0; align-items: center; }
.icon-btn { width: 28px; height: 28px; border: none; border-radius: 6px; background: transparent; color: var(--text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center; font-size: 14px; transition: all 0.15s; }
.icon-btn:hover:not(:disabled) { background: rgba(255,255,255,0.1); color: var(--text-primary); }
.icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.icon-btn.danger:hover:not(:disabled) { background: rgba(244,67,54,0.2); color: var(--error); }
</style>
