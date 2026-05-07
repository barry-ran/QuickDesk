<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <div class="dialog">
      <h3 class="dialog-title">{{ dialogTitle }}</h3>

      <!-- Username/password login -->
      <template v-if="mode === 'login'">
        <div class="form-group">
          <label>{{ $t('user.username') }}</label>
          <input v-model="username" class="form-input" type="text" autocomplete="username" @keyup.enter="submit" />
        </div>
        <div class="form-group">
          <label>{{ $t('user.password') }}</label>
          <input v-model="password" class="form-input" type="password" autocomplete="current-password" @keyup.enter="submit" />
        </div>
      </template>

      <!-- SMS login -->
      <template v-else-if="mode === 'sms-login'">
        <div class="form-group">
          <label>{{ $t('user.phone') }}</label>
          <input v-model="phone" class="form-input" type="text" />
        </div>
        <div class="form-group">
          <label>{{ $t('user.smsCode') }}</label>
          <div class="input-row">
            <input v-model="smsCode" class="form-input" type="text" @keyup.enter="submit" />
            <button class="btn btn-secondary" :disabled="smsCountdown > 0 || !phone" @click="sendSms('login')">
              {{ smsCountdown > 0 ? `${smsCountdown}s` : $t('user.sendCode') }}
            </button>
          </div>
        </div>
      </template>

      <!-- Register -->
      <template v-else>
        <div class="form-group">
          <label>{{ $t('user.username') }}</label>
          <input v-model="username" class="form-input" type="text" autocomplete="username" />
        </div>
        <div class="form-group">
          <label>{{ $t('user.password') }}</label>
          <input v-model="password" class="form-input" type="password" autocomplete="new-password" />
          <div class="hint">{{ $t('user.passwordHint') }}</div>
        </div>
        <div class="form-group">
          <label>{{ $t('user.phone') }} {{ authState.smsEnabled ? '' : `(${$t('user.optional')})` }}</label>
          <input v-model="phone" class="form-input" type="text" />
        </div>
        <div v-if="authState.smsEnabled" class="form-group">
          <label>{{ $t('user.smsCode') }}</label>
          <div class="input-row">
            <input v-model="smsCode" class="form-input" type="text" />
            <button class="btn btn-secondary" :disabled="smsCountdown > 0 || !phone" @click="sendSms('register')">
              {{ smsCountdown > 0 ? `${smsCountdown}s` : $t('user.sendCode') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>{{ $t('user.email') }} ({{ $t('user.optional') }})</label>
          <input v-model="email" class="form-input" type="email" />
        </div>
      </template>

      <div v-if="errorMsg" class="hint error" style="margin-bottom:12px;">{{ errorMsg }}</div>
      <div v-if="successMsg" class="hint success" style="margin-bottom:12px;">{{ successMsg }}</div>

      <button class="btn btn-primary btn-full" :disabled="loading" @click="submit" style="margin-bottom:12px;">
        {{ loading ? '...' : submitLabel }}
      </button>

      <div class="dialog-links">
        <span v-if="mode !== 'register'" class="link" @click="mode = 'register'; clearMessages()">{{ $t('user.noAccount') }}</span>
        <span v-else class="link" @click="mode = 'login'; clearMessages()">{{ $t('user.hasAccount') }}</span>

        <span v-if="authState.smsEnabled && mode === 'login'" class="link" @click="mode = 'sms-login'; clearMessages()">{{ $t('user.smsLogin') }}</span>
        <span v-if="mode === 'sms-login'" class="link" @click="mode = 'login'; clearMessages()">{{ $t('user.passwordLogin') }}</span>

        <span v-if="mode === 'login'" class="link" @click="mode = 'forgot'; clearMessages()">{{ $t('user.forgotPassword') }}</span>
      </div>

      <!-- Forgot password inline -->
      <template v-if="mode === 'forgot'">
        <div style="margin-top:16px;padding-top:16px;border-top:1px solid var(--border);">
          <div class="form-group">
            <label>{{ $t('user.phone') }}</label>
            <div class="input-row">
              <input v-model="phone" class="form-input" type="text" />
              <button class="btn btn-secondary" :disabled="smsCountdown > 0 || !phone" @click="sendSms('reset-password')">
                {{ smsCountdown > 0 ? `${smsCountdown}s` : $t('user.sendCode') }}
              </button>
            </div>
          </div>
          <div class="form-group">
            <label>{{ $t('user.smsCode') }}</label>
            <input v-model="smsCode" class="form-input" type="text" />
          </div>
          <div class="form-group">
            <label>{{ $t('account.newPassword') }}</label>
            <input v-model="newPassword" class="form-input" type="password" />
            <div class="hint">{{ $t('user.passwordHint') }}</div>
          </div>
          <button class="btn btn-primary btn-full" :disabled="loading" @click="submitResetPassword">
            {{ loading ? '...' : $t('account.resetPassword') }}
          </button>
          <div style="text-align:center;margin-top:8px;">
            <span class="link" @click="mode = 'login'; clearMessages()">{{ $t('user.hasAccount') }}</span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, inject, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'
import { updateAuthState } from '../store/auth'

const { t } = useI18n()
const authState = inject('authState')
const showToast = inject('showToast')
const emit = defineEmits(['close', 'success'])

const mode = ref('login') // login | sms-login | register | forgot
const username = ref('')
const password = ref('')
const newPassword = ref('')
const phone = ref('')
const email = ref('')
const smsCode = ref('')
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const smsCountdown = ref(0)
let smsTimer = null

const dialogTitle = computed(() => {
  if (mode.value === 'register') return t('user.register')
  if (mode.value === 'sms-login') return t('user.smsLogin')
  if (mode.value === 'forgot') return t('account.resetPassword')
  return t('user.login')
})

const submitLabel = computed(() => {
  if (mode.value === 'register') return t('user.register')
  return t('user.login')
})

function clearMessages() { errorMsg.value = ''; successMsg.value = '' }

function apiError(r) {
  if (r.code && t(`errors.${r.code}`) !== `errors.${r.code}`) {
    return t(`errors.${r.code}`)
  }
  return r.error || t('toast.networkError')
}

function ensureBaseUrl() {
  userApi.setBaseUrl(userApi.getServerUrl())
}

async function sendSms(scene) {
  if (smsCountdown.value > 0) return
  ensureBaseUrl()
  clearMessages()
  const r = await userApi.sendSmsCode(phone.value, scene)
  if (r.ok) {
    successMsg.value = t('user.codeSent')
    startCountdown()
  } else {
    errorMsg.value = apiError(r)
  }
}

function startCountdown() {
  smsCountdown.value = 60
  smsTimer = setInterval(() => {
    smsCountdown.value--
    if (smsCountdown.value <= 0) { clearInterval(smsTimer); smsTimer = null }
  }, 1000)
}

async function submit() {
  if (loading.value || mode.value === 'forgot') return
  clearMessages()
  ensureBaseUrl()
  loading.value = true

  try {
    if (mode.value === 'login') {
      if (!username.value || !password.value) { errorMsg.value = t('user.inputRequired'); return }
      const r = await userApi.login(username.value, password.value)
      if (r.ok) { updateAuthState(); emit('success') }
      else errorMsg.value = apiError(r)

    } else if (mode.value === 'sms-login') {
      if (!phone.value || !smsCode.value) { errorMsg.value = t('user.phoneCodeRequired'); return }
      const r = await userApi.loginWithSms(phone.value, smsCode.value)
      if (r.ok) { updateAuthState(); emit('success') }
      else errorMsg.value = apiError(r)

    } else if (mode.value === 'register') {
      if (!username.value || !password.value) { errorMsg.value = t('user.inputRequired'); return }
      if (authState.smsEnabled && (!phone.value || !smsCode.value)) {
        errorMsg.value = t('user.phoneCodeRequired'); return
      }
      const r = await userApi.register(username.value, password.value, phone.value, email.value, smsCode.value)
      if (r.ok) {
        successMsg.value = t('user.registerSuccess')
        mode.value = 'login'
      } else {
        errorMsg.value = apiError(r)
      }
    }
  } finally {
    loading.value = false
  }
}

async function submitResetPassword() {
  if (loading.value) return
  clearMessages()
  ensureBaseUrl()
  loading.value = true
  try {
    const r = await userApi.resetPassword(phone.value, smsCode.value, newPassword.value)
    if (r.ok) {
      showToast(t('account.success'), 'success')
      mode.value = 'login'
      clearMessages()
    } else {
      errorMsg.value = apiError(r)
    }
  } finally {
    loading.value = false
  }
}

onUnmounted(() => { if (smsTimer) clearInterval(smsTimer) })
</script>

<style scoped>
.dialog-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.5);
  z-index: 1000;
  display: flex; align-items: center; justify-content: center;
}

.dialog {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 32px;
  width: 360px;
  max-width: 90vw;
  max-height: 90vh;
  overflow-y: auto;
}

.dialog-title {
  text-align: center;
  margin-bottom: 20px;
  font-size: 20px;
}

.input-row {
  display: flex; gap: 8px;
}

.input-row .form-input { flex: 1; }

.dialog-links {
  display: flex; flex-direction: column;
  align-items: center; gap: 6px;
  margin-top: 4px;
}

.link {
  color: var(--accent);
  font-size: 13px;
  cursor: pointer;
}

.link:hover { text-decoration: underline; }
</style>
