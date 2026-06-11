<template>
  <div>
    <h2 class="page-title">{{ $t('account.title') }}</h2>

    <div v-if="!authState.isLoggedIn" class="card">
      <div class="empty-state">
        <div class="empty-icon">🔒</div>
        <p>{{ $t('account.loginRequired') }}</p>
      </div>
    </div>

    <template v-else>
      <!-- Change Username -->
      <div class="card">
        <div class="card-title">{{ $t('account.changeUsername') }}</div>
        <div class="form-group">
          <label>{{ $t('account.newUsername') }}</label>
          <input v-model="newUsername" class="form-input" type="text" :placeholder="authState.userInfo?.username" />
        </div>
        <div v-if="msgs.username" :class="['hint', msgs.username.type]">{{ msgs.username.text }}</div>
        <button class="btn btn-primary" :disabled="loading.username || !newUsername" @click="changeUsername">{{ $t('account.save') }}</button>
      </div>

      <!-- Change Password -->
      <div class="card">
        <div class="card-title">{{ $t('account.changePassword') }}</div>
        <div class="form-group">
          <label>{{ $t('account.oldPassword') }}</label>
          <input v-model="oldPassword" class="form-input" type="password" />
        </div>
        <div class="form-group">
          <label>{{ $t('account.newPassword') }}</label>
          <input v-model="newPassword" class="form-input" type="password" />
          <div class="hint">{{ $t('user.passwordHint') }}</div>
        </div>
        <div class="form-group">
          <label>{{ $t('account.confirmPassword') }}</label>
          <input v-model="confirmPassword" class="form-input" type="password" />
        </div>
        <div v-if="msgs.password" :class="['hint', msgs.password.type]">{{ msgs.password.text }}</div>
        <button class="btn btn-primary" :disabled="loading.password" @click="changePassword">{{ $t('account.save') }}</button>
      </div>

      <!-- Change Phone (SMS required) -->
      <div v-if="authState.smsEnabled" class="card">
        <div class="card-title">{{ $t('account.changePhone') }}</div>
        <div class="form-group">
          <label>{{ $t('account.newPhone') }}</label>
          <div class="input-row">
            <input v-model="newPhone" class="form-input" type="text" />
            <button class="btn btn-secondary" :disabled="smsCountdown > 0 || !newPhone" @click="sendPhoneSms">
              {{ smsCountdown > 0 ? `${smsCountdown}s` : $t('user.sendCode') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>{{ $t('user.smsCode') }}</label>
          <input v-model="phoneSmsCode" class="form-input" type="text" />
        </div>
        <div v-if="msgs.phone" :class="['hint', msgs.phone.type]">{{ msgs.phone.text }}</div>
        <button class="btn btn-primary" :disabled="loading.phone || !newPhone || !phoneSmsCode" @click="changePhone">{{ $t('account.save') }}</button>
      </div>

      <!-- Change Email -->
      <div class="card">
        <div class="card-title">{{ $t('account.changeEmail') }}</div>
        <div class="form-group">
          <label>{{ $t('account.newEmail') }}</label>
          <input v-model="newEmail" class="form-input" type="email" :placeholder="authState.userInfo?.email || ''" />
        </div>
        <div v-if="msgs.email" :class="['hint', msgs.email.type]">{{ msgs.email.text }}</div>
        <button class="btn btn-primary" :disabled="loading.email || !newEmail" @click="changeEmail">{{ $t('account.save') }}</button>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, inject, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { updateAuthState } from '../store/auth'

const { t } = useI18n()
const showToast = inject('showToast')
const authState = inject('authState')

const newUsername = ref('')
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const newPhone = ref('')
const phoneSmsCode = ref('')
const newEmail = ref('')
const smsCountdown = ref(0)
let smsTimer = null

const loading = ref({ username: false, password: false, phone: false, email: false })
const msgs = ref({ username: null, password: null, phone: null, email: null })

function apiError(r) {
  if (r.code && t(`errors.${r.code}`) !== `errors.${r.code}`) return t(`errors.${r.code}`)
  return r.error || t('toast.networkError')
}

function setMsg(field, text, type = 'error') { msgs.value[field] = { text, type } }
function clearMsg(field) { msgs.value[field] = null }

async function changeUsername() {
  clearMsg('username')
  loading.value.username = true
  try {
    const r = await userApi.changeUsername(newUsername.value)
    if (r.ok) {
      setMsg('username', t('account.success'), 'success')
      newUsername.value = ''
      // Update stored user info
      const info = authState.userInfo || {}
      info.username = newUsername.value
      localStorage.setItem('quickdesk_user_info', JSON.stringify(info))
      updateAuthState()
      showToast(t('account.success'), 'success')
    } else {
      setMsg('username', apiError(r))
    }
  } finally { loading.value.username = false }
}

async function changePassword() {
  clearMsg('password')
  if (newPassword.value !== confirmPassword.value) {
    setMsg('password', t('account.passwordMismatch')); return
  }
  loading.value.password = true
  try {
    const r = await userApi.changePassword(oldPassword.value, newPassword.value)
    if (r.ok) {
      setMsg('password', t('account.success'), 'success')
      oldPassword.value = ''; newPassword.value = ''; confirmPassword.value = ''
      showToast(t('account.success'), 'success')
    } else {
      setMsg('password', apiError(r))
    }
  } finally { loading.value.password = false }
}

async function sendPhoneSms() {
  if (smsCountdown.value > 0) return
  // §2.2: scene=bind_phone for phone change verification.
  const r = await userApi.sendSmsCode(newPhone.value, 'bind_phone')
  if (r.ok) {
    smsCountdown.value = 60
    smsTimer = setInterval(() => {
      smsCountdown.value--
      if (smsCountdown.value <= 0) { clearInterval(smsTimer); smsTimer = null }
    }, 1000)
  } else {
    setMsg('phone', apiError(r))
  }
}

async function changePhone() {
  clearMsg('phone')
  loading.value.phone = true
  try {
    const r = await userApi.changePhone(newPhone.value, phoneSmsCode.value)
    if (r.ok) {
      setMsg('phone', t('account.success'), 'success')
      newPhone.value = ''; phoneSmsCode.value = ''
      showToast(t('account.success'), 'success')
    } else {
      setMsg('phone', apiError(r))
    }
  } finally { loading.value.phone = false }
}

async function changeEmail() {
  clearMsg('email')
  loading.value.email = true
  try {
    const r = await userApi.changeEmail(newEmail.value)
    if (r.ok) {
      setMsg('email', t('account.success'), 'success')
      newEmail.value = ''
      showToast(t('account.success'), 'success')
    } else {
      setMsg('email', apiError(r))
    }
  } finally { loading.value.email = false }
}

onUnmounted(() => { if (smsTimer) clearInterval(smsTimer) })
</script>

<style scoped>
.input-row { display: flex; gap: 8px; }
.input-row .form-input { flex: 1; }
</style>
