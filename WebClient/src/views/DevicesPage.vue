<template>
  <div>
    <h2 class="page-title">{{ $t('devices.title') }}</h2>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('devices.loginRequired') }}</p>
        <button class="btn btn-primary btn-sm" style="margin-top:12px;" @click="showLogin()">{{ $t('user.login') }}</button>
      </div>
    </div>

    <template v-else>
      <div style="display:flex;justify-content:flex-end;margin-bottom:12px;">
        <button class="btn btn-secondary btn-sm" @click="refresh">🔄 {{ $t('devices.refresh') }}</button>
      </div>

      <!-- My Devices -->
      <div class="card">
        <div class="card-title">{{ $t('devices.myDevices') }}</div>
        <div v-if="myDevices.length === 0" class="empty-state">
          <div class="empty-icon">📱</div>
          <p>{{ $t('devices.noDevices') }}</p>
        </div>
        <div v-for="d in myDevices" :key="d.device_id" class="device-item">
          <div :class="['device-status', statusClass(d)]"></div>
          <div class="device-info">
            <div class="device-name">{{ d.remark || d.device_name || d.device_id }}</div>
            <div class="device-id">
              {{ d.device_id }}
              <span v-if="!d.online" class="badge offline">{{ $t('devices.offline') }}</span>
              <span v-else-if="!d.logged_in" class="badge idle">{{ $t('devices.notLoggedIn') }}</span>
            </div>
          </div>
          <div class="device-actions">
            <button
              v-if="d.online && d.logged_in"
              class="btn btn-primary btn-sm"
              :disabled="connecting === d.device_id"
              @click="connectDevice(d.device_id, d.access_code)"
            >
              {{ connecting === d.device_id ? '...' : $t('devices.connect') }}
            </button>
            <button class="icon-btn" :title="$t('devices.setRemark')" @click="setRemark(d)">✏️</button>
            <button class="icon-btn danger" :title="$t('devices.unbind')" @click="unbindDevice(d.device_id)">🗑</button>
          </div>
        </div>
      </div>

      <!-- My Favorites -->
      <div class="card">
        <div class="card-title">{{ $t('devices.myFavorites') }}</div>
        <div v-if="myFavorites.length === 0" class="empty-state">
          <div class="empty-icon">⭐</div>
          <p>{{ $t('devices.noFavorites') }}</p>
        </div>
        <div v-for="f in myFavorites" :key="f.device_id" class="device-item">
          <span style="font-size:16px;flex-shrink:0;">⭐</span>
          <div class="device-info">
            <div class="device-name">{{ f.device_name || f.device_id }}</div>
            <div class="device-id">{{ f.device_id }}</div>
          </div>
          <div class="device-actions">
            <button
              class="btn btn-primary btn-sm"
              :disabled="connecting === f.device_id"
              @click="connectDevice(f.device_id, f.access_password)"
            >
              {{ connecting === f.device_id ? '...' : $t('devices.connect') }}
            </button>
            <button class="icon-btn danger" :title="$t('devices.removeFavorite')" @click="removeFav(f.device_id)">✕</button>
          </div>
        </div>
      </div>

      <!-- Connection Logs -->
      <div class="card">
        <div class="card-title">{{ $t('devices.connectionLogs') }}</div>
        <div v-if="logs.length === 0" class="empty-state">
          <div class="empty-icon">📋</div>
          <p>{{ $t('devices.noLogs') }}</p>
        </div>
        <div v-for="log in logs.slice(0, 20)" :key="log.id" class="device-item">
          <div :class="['device-status', log.status === 'success' ? 'online' : 'offline']"></div>
          <div class="device-info">
            <div class="device-name">{{ log.device_id }}</div>
            <div class="device-id">{{ formatTime(log.created_at) }}{{ log.duration ? ` · ${Math.floor(log.duration/60)}m${log.duration%60}s` : '' }}</div>
          </div>
        </div>
      </div>
    </template>

    <!-- Remark Dialog -->
    <div v-if="remarkDialog.show" class="dialog-overlay" @click.self="remarkDialog.show = false">
      <div class="dialog">
        <div class="card-title">{{ $t('devices.setRemark') }}</div>
        <div class="form-group">
          <input v-model="remarkDialog.value" class="form-input" type="text" @keyup.enter="saveRemark" />
        </div>
        <div style="display:flex;gap:8px;">
          <button class="btn btn-primary" style="flex:1;" @click="saveRemark">{{ $t('account.save') }}</button>
          <button class="btn btn-secondary" style="flex:1;" @click="remarkDialog.show = false">{{ $t('user.cancel') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { userSync } from '../api/userSync'
import { openRemoteSession } from '../utils/remoteLauncher'

const { t } = useI18n()
const showToast = inject('showToast')
const showLogin = inject('showLogin')
const authState = inject('authState')

const myDevices = ref([])
const myFavorites = ref([])
const logs = ref([])
const remarkDialog = ref({ show: false, device: null, value: '' })
const connecting = ref(null)

function statusClass(d) {
  if (d.online && d.logged_in) return 'online'
  if (d.online) return 'idle'
  return 'offline'
}

async function refresh() {
  await Promise.all([loadDevices(), loadFavorites(), loadLogs()])
}

// §2.2 / T2: list envelope is { items, next_cursor }.
async function loadDevices() {
  const r = await userApi.fetchMyDevices()
  if (r.ok && r.data) myDevices.value = Array.isArray(r.data.items) ? r.data.items : []
}

async function loadFavorites() {
  const r = await userApi.fetchFavorites()
  if (r.ok && r.data) myFavorites.value = Array.isArray(r.data.items) ? r.data.items : []
}

async function loadLogs() {
  const r = await userApi.fetchConnectionLogs()
  if (r.ok && r.data) logs.value = Array.isArray(r.data.items) ? r.data.items : []
}

// §2.18: verify → signal_token → open remote.html?st=<token>. The access
// code is passed to the remote window via sessionStorage (keyed by the
// short-lived signal_token) so it never enters the URL or browser history.
async function connectDevice(deviceId, accessCode) {
  if (!accessCode) { showToast(t('devices.noAccessCode'), 'error'); return }
  if (connecting.value) return
  connecting.value = deviceId
  try {
    const v = await userApi.verifyAccessCode(deviceId, accessCode)
    if (!v.ok) {
      // §2.10 / §2.15: on TOO_MANY_ATTEMPTS the server sets Retry-After
      // (seconds) so the UI can tell users how long to wait instead of
      // a vague "try again later".
      showToast(errorForCode(v.code, v.error, v.retryAfter), 'error')
      return
    }
    const signalToken = v.data?.signal_token
    if (!signalToken) { showToast(t('toast.networkError'), 'error'); return }
    openRemoteSession({ deviceId, signalToken, accessCode })
  } finally {
    connecting.value = null
  }
}

function errorForCode(code, fallback, retryAfter) {
  // §2.6 response code mapping to i18n keys (errors.* in locale files).
  // If the translation is missing fall back to the server-provided detail.
  if (code === 'TOO_MANY_ATTEMPTS' && retryAfter > 0) {
    return t('errors.TOO_MANY_ATTEMPTS_RETRY', { seconds: retryAfter })
  }
  if (code) {
    const key = `errors.${code}`
    const str = t(key)
    if (str !== key) return str
  }
  return fallback || t('toast.networkError')
}

function setRemark(device) {
  remarkDialog.value = { show: true, device, value: device.remark || device.device_name || '' }
}

// §2.2 / T5: PATCH for partial updates.
async function saveRemark() {
  const { device, value } = remarkDialog.value
  const r = await userApi.setDeviceRemark(device.device_id, value)
  if (!r.ok) {
    showToast(r.error || t('toast.networkError'), 'error')
    return
  }
  remarkDialog.value.show = false
  // Realtime `device.remark.changed` will refresh the list; fetch once as
  // belt-and-suspenders for snappy UI (§T8 refetch exception).
  await loadDevices()
}

async function unbindDevice(deviceId) {
  if (!confirm(t('devices.confirmUnbind'))) return
  const r = await userApi.unbindDevice(deviceId)
  if (!r.ok) { showToast(r.error || t('toast.networkError'), 'error'); return }
  showToast(t('devices.unbindSuccess'), 'success')
  // Realtime will emit device.unbound; patch locally for snappier feedback.
  myDevices.value = myDevices.value.filter(d => d.device_id !== deviceId)
}

async function removeFav(deviceId) {
  await userApi.removeFavorite(deviceId)
  // Realtime favorite.removed will patch the cached list; mirror locally.
  myFavorites.value = myFavorites.value.filter(f => f.device_id !== deviceId)
}

function formatTime(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleString()
}

// Realtime handlers — prefer userSync's in-memory cache which the sync
// layer already patched (§T8: do NOT re-fetch on ordinary updates).
function onSyncSnapshot(e) {
  myDevices.value   = e.detail.devices.slice()
  myFavorites.value = e.detail.favorites.slice()
}
function onDevicesChanged(e) {
  myDevices.value = e.detail.devices.slice()
}
function onFavoritesChanged(e) {
  myFavorites.value = e.detail.favorites.slice()
}

onMounted(() => {
  if (authState.isLoggedIn) {
    // Seed from userSync cache (non-empty if sync already ran).
    const cachedD = userSync.getDevices()
    const cachedF = userSync.getFavorites()
    if (cachedD.length) myDevices.value = cachedD
    if (cachedF.length) myFavorites.value = cachedF
    // Logs come from HTTP — sync stream doesn't cover connection history.
    loadLogs()
    // Also fetch once if cache empty; otherwise wait for snapshot.
    if (!cachedD.length) loadDevices()
    if (!cachedF.length) loadFavorites()
  }
  userSync.addEventListener('snapshot', onSyncSnapshot)
  userSync.addEventListener('devices-changed', onDevicesChanged)
  userSync.addEventListener('favorites-changed', onFavoritesChanged)
})

onBeforeUnmount(() => {
  userSync.removeEventListener('snapshot', onSyncSnapshot)
  userSync.removeEventListener('devices-changed', onDevicesChanged)
  userSync.removeEventListener('favorites-changed', onFavoritesChanged)
})
</script>

<style scoped>
.device-item {
  display: flex; align-items: center; gap: 12px;
  padding: 12px; border-radius: var(--radius);
  transition: background 0.15s;
}
.device-item:hover { background: rgba(255,255,255,0.05); }
.device-status { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.device-status.online { background: var(--success); }
.device-status.idle   { background: var(--warning); }
.device-status.offline { background: var(--text-disabled); }
.device-info { flex: 1; min-width: 0; }
.device-name { font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.device-id { font-size: 12px; color: var(--text-secondary); font-family: 'Consolas', monospace; margin-top: 2px; }
.badge { margin-left: 6px; padding: 1px 6px; border-radius: 4px; font-size: 11px; font-family: inherit; }
.badge.offline { background: rgba(255,255,255,0.1); color: var(--text-secondary); }
.badge.idle    { background: rgba(255,193,7,0.15); color: var(--warning); }
.device-actions { display: flex; gap: 4px; flex-shrink: 0; }
.icon-btn { width: 28px; height: 28px; border: none; border-radius: 6px; background: transparent; color: var(--text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center; font-size: 14px; transition: all 0.15s; }
.icon-btn:hover { background: rgba(255,255,255,0.1); color: var(--text-primary); }
.icon-btn.danger:hover { background: rgba(244,67,54,0.2); color: var(--error); }

.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.dialog { background: var(--bg-card); border: 1px solid var(--border); border-radius: 12px; padding: 24px; width: 320px; max-width: 90vw; }
</style>
