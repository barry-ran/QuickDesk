<template>
  <div class="reset-page">
    <div class="reset-card">
      <h2 class="reset-title">{{ $t('account.resetPassword') }}</h2>
      <p class="reset-desc">{{ $t('account.resetPasswordDesc') }}</p>

      <div class="form-group">
        <label>{{ $t('user.phone') }}</label>
        <input v-model="phone" class="form-input" type="text" />
      </div>
      <div class="form-group">
        <label>{{ $t('user.smsCode') }}</label>
        <div class="input-row">
          <input v-model="smsCode" class="form-input" type="text" />
          <button class="btn btn-secondary" :disabled="smsCountdown > 0 || !phone" @click="sendSms">
            {{ smsCountdown > 0 ? `${smsCountdown}s` : $t('user.sendCode') }}
          </button>
        </div>
      </div>
      <div class="form-group">
        <label>{{ $t('account.newPassword') }}</label>
        <input v-model="newPassword" class="form-input" type="password" />
        <div class="hint">{{ $t('user.passwordHint') }}</div>
      </div>

      <div v-if="errorMsg" class="hint error" style="margin-bottom:12px;">{{ errorMsg }}</div>
      <div v-if="successMsg" class="hint success" style="margin-bottom:12px;">{{ successMsg }}</div>

      <button class="btn btn-primary btn-full" :disabled="loading" @click="submitReset">
        {{ loading ? '...' : $t('account.resetPassword') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi } from '../api/userApi'

const { t } = useI18n()
const showToast = inject('showToast')

const phone = ref('')
const smsCode = ref('')
const newPassword = ref('')
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const smsCountdown = ref(0)
let smsTimer = null

function ensureBaseUrl() {
  userApi.setBaseUrl(userApi.getServerUrl())
}

function apiError(r) {
  if (r.code && t(`errors.${r.code}`) !== `errors.${r.code}`) return t(`errors.${r.code}`)
  return r.error || t('toast.networkError')
}

async function sendSms() {
  if (smsCountdown.value > 0) return
  ensureBaseUrl()
  errorMsg.value = ''
  successMsg.value = ''
  const r = await userApi.sendResetPasswordCode(phone.value)
  if (r.ok) {
    successMsg.value = t('user.codeSent')
    smsCountdown.value = 60
    smsTimer = setInterval(() => {
      smsCountdown.value--
      if (smsCountdown.value <= 0) { clearInterval(smsTimer); smsTimer = null }
    }, 1000)
  } else {
    errorMsg.value = apiError(r)
  }
}

async function submitReset() {
  if (loading.value) return
  errorMsg.value = ''
  successMsg.value = ''
  ensureBaseUrl()

  if (!phone.value || !smsCode.value || !newPassword.value) {
    errorMsg.value = t('user.inputRequired')
    return
  }

  loading.value = true
  try {
    const r = await userApi.resetPassword(phone.value, smsCode.value, newPassword.value)
    if (r.ok) {
      successMsg.value = t('account.resetPasswordSuccess')
      showToast(t('account.success'), 'success')
      phone.value = ''
      smsCode.value = ''
      newPassword.value = ''
    } else {
      errorMsg.value = apiError(r)
    }
  } finally {
    loading.value = false
  }
}

onMounted(ensureBaseUrl)
onUnmounted(() => { if (smsTimer) clearInterval(smsTimer) })
</script>

<style scoped>
.reset-page {
  display: flex;
  justify-content: center;
  padding-top: 40px;
}
.reset-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 32px;
  width: 400px;
  max-width: 90vw;
}
.reset-title {
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 8px;
}
.reset-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}
.input-row { display: flex; gap: 8px; }
.input-row .form-input { flex: 1; }
</style>
