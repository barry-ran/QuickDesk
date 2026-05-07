<template>
  <div>
    <h2 class="page-title">{{ $t('connect.title') }}</h2>
    <div class="connect-grid">
      <!-- Connect Form -->
      <div class="card">
        <div class="card-title">{{ $t('connect.title') }}</div>
        <div class="form-group">
          <label>{{ $t('connect.server') }}</label>
          <input v-model="serverUrl" class="form-input" type="text" :placeholder="$t('settings.signalingServerPlaceholder')" @change="saveServerUrl" />
        </div>
        <div class="form-group">
          <label>{{ $t('connect.deviceId') }}</label>
          <input v-model="deviceId" class="form-input" type="text" :placeholder="$t('connect.deviceIdPlaceholder')" autocomplete="off" @keyup.enter="accessCodeInput?.focus()" />
        </div>
        <div class="form-group">
          <label>{{ $t('connect.accessCode') }}</label>
          <input ref="accessCodeInput" v-model="accessCode" class="form-input" type="password" :placeholder="$t('connect.accessCodePlaceholder')" autocomplete="off" @keyup.enter="connect" />
        </div>
        <button class="btn btn-primary btn-full" @click="connect">{{ $t('connect.button') }}</button>
      </div>

      <!-- Connection History -->
      <div class="card">
        <div class="card-title">{{ $t('history.title') }}</div>
        <div v-if="history.length === 0" class="empty-state">
          <div class="empty-icon">📋</div>
          <p>{{ $t('history.empty') }}</p>
        </div>
        <div v-else class="history-list">
          <div v-for="item in history" :key="item.deviceId" class="history-item" @click="fillFromHistory(item)">
            <div class="history-icon">🖥️</div>
            <div class="history-info">
              <div class="history-device">{{ item.deviceId }}</div>
              <div class="history-meta">{{ formatTime(item.lastConnected) }} · {{ $t('history.connectCount', { count: item.connectCount || 1 }) }}</div>
            </div>
            <div class="history-actions">
              <button v-if="authState.isLoggedIn" class="fav-star" :title="isFavorite(item.deviceId) ? $t('devices.removeFavorite') : $t('devices.addFavorite')" @click.stop="toggleFavorite(item.deviceId)">
                {{ isFavorite(item.deviceId) ? '⭐' : '☆' }}
              </button>
              <button class="icon-btn" :title="$t('history.fill')" @click.stop="fillFromHistory(item)">↗</button>
              <button class="icon-btn danger" :title="$t('history.delete')" @click.stop="deleteHistory(item.deviceId)">✕</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ConnectionHistory } from '../../js/storage/connection-history.js'
import { userApi, DEFAULT_SERVER } from '../api/userApi'

const { t } = useI18n()
const showToast = inject('showToast')
const authState = inject('authState')

const serverUrl = ref(localStorage.getItem('quickdesk_signaling_url') || DEFAULT_SERVER)
const deviceId = ref('')
const accessCode = ref('')
const accessCodeInput = ref(null)
const history = ref(ConnectionHistory.getAll())
const favorites = ref([])

function saveServerUrl() {
  localStorage.setItem('quickdesk_signaling_url', serverUrl.value)
  userApi.setBaseUrl(serverUrl.value)
}

function connect() {
  if (!deviceId.value || !accessCode.value) {
    showToast(t('connect.inputRequired'), 'error')
    return
  }
  saveServerUrl()
  const codec = localStorage.getItem('quickdesk_video_codec') || 'AV1'
  const params = new URLSearchParams({ server: serverUrl.value, device: deviceId.value, code: accessCode.value, codec })
  const url = `remote.html?${params}`
  const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
  if (isMobile) window.location.href = url
  else window.open(url, `quickdesk_${deviceId.value}`)
  showToast(t('connect.connecting', { deviceId: deviceId.value }), 'info')
}

function fillFromHistory(item) {
  deviceId.value = item.deviceId
  if (item.serverUrl) serverUrl.value = item.serverUrl
  accessCode.value = ''
  accessCodeInput.value?.focus()
}

function deleteHistory(id) {
  ConnectionHistory.remove(id)
  history.value = ConnectionHistory.getAll()
  showToast(t('history.deleted'), 'info')
}

function isFavorite(id) { return favorites.value.some(f => f.device_id === id) }

async function toggleFavorite(id) {
  if (isFavorite(id)) {
    await userApi.removeFavorite(id)
  } else {
    await userApi.addFavorite(id, '', id === deviceId.value ? accessCode.value : '')
  }
  await loadFavorites()
  history.value = ConnectionHistory.getAll()
}

async function loadFavorites() {
  if (!authState.isLoggedIn) return
  const r = await userApi.fetchFavorites()
  if (r.ok && r.data) favorites.value = r.data.favorites || []
}

function formatTime(ts) {
  if (!ts) return ''
  const diff = Date.now() - new Date(ts).getTime()
  const min = Math.floor(diff / 60000)
  if (min < 1) return t('time.justNow')
  if (min < 60) return t('time.minutesAgo', { n: min })
  const hr = Math.floor(min / 60)
  if (hr < 24) return t('time.hoursAgo', { n: hr })
  return t('time.daysAgo', { n: Math.floor(hr / 24) })
}

function onMessage(e) {
  if (e.data?.type !== 'quickdesk-connected') return
  const { deviceId: id, serverUrl: srv } = e.data
  if (id) { ConnectionHistory.save(id, srv); history.value = ConnectionHistory.getAll() }
}

// Apply URL params (for direct links)
const params = new URLSearchParams(window.location.search)
if (params.get('server')) serverUrl.value = params.get('server')
if (params.get('device')) deviceId.value = params.get('device')
if (params.get('code')) accessCode.value = params.get('code')

onMounted(() => {
  window.addEventListener('message', onMessage)
  loadFavorites()
})
onUnmounted(() => window.removeEventListener('message', onMessage))
</script>

<style scoped>
.connect-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

@media (max-width: 900px) { .connect-grid { grid-template-columns: 1fr; } }

.history-list { max-height: 400px; overflow-y: auto; }

.history-item {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px; border-radius: var(--radius);
  cursor: pointer; transition: background 0.15s;
}
.history-item:hover { background: rgba(255,255,255,0.05); }
.history-icon { width: 36px; height: 36px; border-radius: 8px; background: rgba(233,69,96,0.15); display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0; }
.history-info { flex: 1; min-width: 0; }
.history-device { font-size: 15px; font-weight: 600; font-family: 'Consolas', monospace; }
.history-meta { font-size: 12px; color: var(--text-secondary); margin-top: 2px; }
.history-actions { display: flex; gap: 4px; opacity: 0; transition: opacity 0.15s; }
.history-item:hover .history-actions { opacity: 1; }

.fav-star { background: none; border: none; font-size: 16px; cursor: pointer; padding: 4px; transition: transform 0.15s; }
.fav-star:hover { transform: scale(1.2); }

.icon-btn { width: 28px; height: 28px; border: none; border-radius: 6px; background: transparent; color: var(--text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center; font-size: 14px; transition: all 0.15s; }
.icon-btn:hover { background: rgba(255,255,255,0.1); color: var(--text-primary); }
.icon-btn.danger:hover { background: rgba(244,67,54,0.2); color: var(--error); }
</style>
