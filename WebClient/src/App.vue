<template>
  <div class="app-layout">
    <!-- Sidebar -->
    <nav class="nav-sidebar">
      <div class="nav-header" @click="onUserAreaClick" style="cursor:pointer;">
        <template v-if="authState.isLoggedIn">
          <div class="user-avatar">{{ userInitial }}</div>
          <div class="user-info">
            <div class="user-name">{{ authState.userInfo?.username || 'User' }}</div>
            <div class="user-sub">QuickDesk</div>
          </div>
        </template>
        <template v-else>
          <div class="user-avatar-empty">👤</div>
          <div class="user-info">
            <div class="user-name">{{ $t('user.login') }}</div>
            <div class="user-sub">{{ $t('user.clickToLogin') }}</div>
          </div>
        </template>
      </div>

      <div class="nav-items">
        <router-link v-for="item in navItems" :key="item.path" :to="item.path" class="nav-item" active-class="active">
          <span class="nav-icon">{{ item.icon }}</span>
          <span>{{ $t(item.label) }}</span>
        </router-link>
      </div>

      <div class="nav-footer">
        <select class="lang-select" :value="locale" @change="onLangChange">
          <option value="zh-CN">中文</option>
          <option value="en-US">English</option>
        </select>
      </div>
    </nav>

    <!-- Content -->
    <main class="content-area">
      <router-view />
    </main>

    <!-- Login Dialog -->
    <LoginDialog v-if="showLogin" @close="showLogin = false" @success="onLoginSuccess" />

    <!-- User Menu -->
    <div v-if="showUserMenu" class="user-menu-overlay" @click="showUserMenu = false">
      <div class="user-menu" :style="userMenuStyle" @click.stop>
        <router-link to="/account" class="menu-item" @click="showUserMenu = false">
          <span>⚙️</span> {{ $t('account.title') }}
        </router-link>
        <div class="menu-item danger" @click="onLogout">
          <span>🚪</span> {{ $t('user.logout') }}
        </div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="toast.visible" :class="['toast', toast.type]">{{ toast.message }}</div>
  </div>
</template>

<script setup>
import { ref, computed, provide, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import LoginDialog from './components/LoginDialog.vue'
import { userApi } from './api/userApi'
import { authState, updateAuthState } from './store/auth'

const { locale, t } = useI18n()
const router = useRouter()

const showLogin = ref(false)
const showUserMenu = ref(false)
const userMenuStyle = ref({})
const toast = ref({ visible: false, message: '', type: 'info' })
let toastTimer = null

const navItems = computed(() => {
  const items = [
    { path: '/remote', icon: '🖥️', label: 'nav.remote' },
    { path: '/devices', icon: '📱', label: 'nav.devices' },
  ]
  if (authState.isLoggedIn) {
    items.push({ path: '/account', icon: '👤', label: 'nav.account' })
  }
  items.push(
    { path: '/settings', icon: '⚙️', label: 'nav.settings' },
    { path: '/about', icon: 'ℹ️', label: 'nav.about' },
  )
  return items
})

const userInitial = computed(() => {
  const name = authState.userInfo?.username || 'U'
  return name.charAt(0).toUpperCase()
})

function onUserAreaClick(e) {
  if (authState.isLoggedIn) {
    const rect = e.currentTarget.getBoundingClientRect()
    userMenuStyle.value = { left: rect.right + 4 + 'px', top: rect.top + 'px' }
    showUserMenu.value = true
  } else {
    showLogin.value = true
  }
}

function onLoginSuccess() {
  showLogin.value = false
  updateAuthState()
  showToast(t('user.loginSuccess'), 'success')
  // Fetch features for SMS toggle
  fetchFeatures()
}

async function onLogout() {
  showUserMenu.value = false
  await userApi.logout()
  updateAuthState()
  router.push('/remote')
  showToast(t('user.logoutSuccess'), 'info')
}

function onLangChange(e) {
  locale.value = e.target.value
  localStorage.setItem('quickdesk_lang', e.target.value)
}

async function fetchFeatures() {
  userApi.setBaseUrl(userApi.getServerUrl())
  const r = await userApi.fetchFeatures()
  if (r.ok && r.data) authState.smsEnabled = !!r.data.sms_enabled
}

async function restoreSession() {
  userApi.setBaseUrl(userApi.getServerUrl())
  fetchFeatures()

  // Handle ?token= auto-login from Qt client
  const params = new URLSearchParams(window.location.search)
  const token = params.get('token')
  if (token) {
    localStorage.setItem('quickdesk_user_token', token)
    updateAuthState()
  }

  if (!userApi.isLoggedIn()) return
  const r = await userApi.fetchMe()
  if (r.ok && r.data) {
    // Update stored user info with fresh data
    localStorage.setItem('quickdesk_user_info', JSON.stringify({
      id: r.data.id, username: r.data.username, phone: r.data.phone, email: r.data.email,
    }))
    updateAuthState()
  } else {
    userApi.clearSession()
    updateAuthState()
  }
}

function showToast(message, type = 'info') {
  toast.value = { visible: true, message, type }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value.visible = false }, 3000)
}

// Provide showToast and showLogin to child components
provide('showToast', showToast)
provide('showLogin', () => { showLogin.value = true })
provide('authState', authState)

onMounted(restoreSession)
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }

:root {
  --bg-primary: #1a1a2e;
  --bg-secondary: #16213e;
  --bg-card: #1e2a4a;
  --bg-input: #1a1a3e;
  --accent: #e94560;
  --accent-hover: #ff6b81;
  --text-primary: #eee;
  --text-secondary: #a0a0b0;
  --text-disabled: #555;
  --success: #4caf50;
  --warning: #ffc107;
  --error: #f44336;
  --border: rgba(255,255,255,0.1);
  --radius: 8px;
  --nav-width: 200px;
  --nav-bg: #0f1629;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  height: 100vh;
  overflow: hidden;
}

a { text-decoration: none; color: inherit; }

.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

/* Sidebar */
.nav-sidebar {
  width: var(--nav-width);
  background: var(--nav-bg);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.nav-header {
  padding: 20px 16px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-avatar {
  width: 36px; height: 36px;
  border-radius: 50%;
  background: var(--accent);
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; font-weight: 700; color: white;
  flex-shrink: 0;
}

.user-avatar-empty {
  width: 36px; height: 36px;
  border-radius: 50%;
  background: rgba(255,255,255,0.08);
  display: flex; align-items: center; justify-content: center;
  font-size: 18px; flex-shrink: 0;
}

.user-info { flex: 1; min-width: 0; }
.user-name { font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.user-sub { font-size: 12px; color: var(--text-secondary); }

.nav-items { flex: 1; padding: 8px 0; }

.nav-item {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 20px;
  font-size: 14px; color: var(--text-secondary);
  transition: all 0.15s;
  border-left: 3px solid transparent;
  cursor: pointer;
}

.nav-item:hover { background: rgba(255,255,255,0.04); color: var(--text-primary); }
.nav-item.active { background: rgba(233,69,96,0.1); color: var(--accent); border-left-color: var(--accent); }
.nav-icon { font-size: 18px; width: 24px; text-align: center; flex-shrink: 0; }

.nav-footer { padding: 12px 16px; border-top: 1px solid var(--border); }

.lang-select {
  width: 100%; padding: 4px 8px; font-size: 13px;
  background: var(--bg-card); color: var(--text-secondary);
  border: 1px solid var(--border); border-radius: 6px;
  outline: none; cursor: pointer;
}

/* Content */
.content-area { flex: 1; overflow-y: auto; padding: 32px 40px; }

/* User menu */
.user-menu-overlay { position: fixed; inset: 0; z-index: 200; }
.user-menu {
  position: fixed; z-index: 201;
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: 8px; padding: 4px; min-width: 160px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
}
.menu-item {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 16px; font-size: 13px; cursor: pointer;
  border-radius: 6px; transition: background 0.15s;
  color: var(--text-primary);
}
.menu-item:hover { background: rgba(255,255,255,0.06); }
.menu-item.danger { color: var(--error); }
.menu-item.danger:hover { background: rgba(244,67,54,0.1); }

/* Toast */
.toast {
  position: fixed; bottom: 24px; left: 50%;
  transform: translateX(-50%);
  padding: 10px 24px; border-radius: var(--radius);
  font-size: 14px; z-index: 10000; pointer-events: none;
  animation: slideUp 0.3s ease;
}
.toast.success { background: var(--success); color: white; }
.toast.error { background: var(--error); color: white; }
.toast.info { background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border); }

@keyframes slideUp {
  from { transform: translateX(-50%) translateY(20px); opacity: 0; }
  to { transform: translateX(-50%) translateY(0); opacity: 1; }
}

/* Shared card styles */
.card {
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: 12px; padding: 24px; margin-bottom: 20px;
}
.card-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.card-subtitle { font-size: 13px; color: var(--text-secondary); margin-bottom: 16px; }
.page-title { font-size: 24px; font-weight: 700; margin-bottom: 24px; }

/* Form */
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; color: var(--text-secondary); margin-bottom: 6px; }
.form-input {
  width: 100%; padding: 10px 14px;
  background: var(--bg-input); border: 1px solid var(--border);
  border-radius: var(--radius); color: var(--text-primary);
  font-size: 14px; outline: none; transition: border-color 0.2s;
}
.form-input:focus { border-color: var(--accent); }
.form-input::placeholder { color: var(--text-disabled); }

/* Buttons */
.btn {
  padding: 10px 20px; border: none; border-radius: var(--radius);
  font-size: 14px; font-weight: 600; cursor: pointer;
  transition: all 0.2s; display: inline-flex;
  align-items: center; justify-content: center; gap: 8px;
}
.btn:hover { opacity: 0.9; }
.btn:active { transform: scale(0.98); }
.btn:disabled { opacity: 0.4; cursor: not-allowed; transform: none; }
.btn-primary { background: var(--accent); color: white; }
.btn-secondary { background: rgba(255,255,255,0.08); color: var(--text-primary); }
.btn-sm { padding: 6px 14px; font-size: 13px; }
.btn-full { width: 100%; }

/* Empty state */
.empty-state { text-align: center; padding: 32px 16px; color: var(--text-secondary); }
.empty-icon { font-size: 32px; margin-bottom: 8px; opacity: 0.5; }
.empty-state p { font-size: 13px; }

/* Hint text */
.hint { font-size: 12px; color: var(--text-secondary); margin-top: 4px; }
.hint.error { color: var(--error); }
.hint.success { color: var(--success); }

@media (max-width: 600px) {
  .nav-sidebar { display: none; }
  .content-area { padding: 16px; }
}
</style>
