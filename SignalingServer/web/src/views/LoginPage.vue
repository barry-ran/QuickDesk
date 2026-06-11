<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h1 v-if="!loading">{{ siteName }}</h1>
        <h1 v-else>&nbsp;</h1>
        <p>{{ t('login.title') }}</p>
      </div>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        @submit.prevent="handleLogin"
        class="login-form"
      >
        <el-form-item prop="user">
          <el-input
            v-model="form.user"
            :placeholder="t('login.username')"
            size="large"
            :prefix-icon="User"
          />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            :placeholder="t('login.password')"
            size="large"
            :prefix-icon="Lock"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-form-item v-if="show2FA">
          <el-input
            v-model="form.totp_code"
            :placeholder="t('login.totpPlaceholder')"
            size="large"
            :prefix-icon="Key"
            maxlength="6"
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="handleLogin"
            class="login-btn"
          >
            {{ t('login.loginBtn') }}
          </el-button>
        </el-form-item>
      </el-form>
      <div class="cache-actions">
        <el-button
          text
          size="small"
          @click="clearCache"
          class="clear-cache-btn"
        >
          <el-icon><Delete /></el-icon>
          {{ t('login.clearCache') }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { User, Lock, Delete, Key } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { login, loginWithTotp } from '../api/auth.js'
import { getSettings } from '../api/settings.js'

const router = useRouter()
const { t } = useI18n()
const formRef = ref(null)
const loading = ref(true)
const siteName = ref('')

const show2FA = ref(false)
// §2.16 step-up: server hands us a short-lived pre_token on the first
// response so the TOTP call can authenticate the continuation without
// re-asking for the password.
const preToken = ref('')

const form = reactive({
  user: '',
  password: '',
  totp_code: ''
})

const rules = computed(() => ({
  user: [{ required: true, message: t('login.userRequired'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.passRequired'), trigger: 'blur' }]
}))

async function loadSiteName() {
  loading.value = true
  try {
    // §2.2: /v1/settings/public returns snake_case (site_name, site_enabled).
    const data = await getSettings()
    if (data.site_name) {
      siteName.value = data.site_name
      document.title = data.site_name + ' Admin'
    }
  } catch (e) {
    console.error('Failed to load site name:', e)
    siteName.value = 'QuickDesk'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadSiteName()
})

async function handleLogin() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  if (show2FA.value && !form.totp_code) {
    ElMessage.warning(t('login.totpRequired'))
    return
  }

  loading.value = true
  try {
    // Two flows:
    //   1) First attempt: POST /v1/admin/auth/sessions with username+password
    //      (+ totp_code if we're retrying in the same form). Server either
    //      grants tokens or throws { code:'TOTP_REQUIRED', preToken }.
    //   2) If we hold a pre_token from (1) and the user has now typed a
    //      code, go straight to /v1/admin/auth/sessions:totp — no
    //      password re-entry needed.
    if (show2FA.value && preToken.value && form.totp_code) {
      await loginWithTotp(preToken.value, form.totp_code)
    } else {
      await login(form.user, form.password, form.totp_code)
    }
    ElMessage.success(t('login.loginSuccess'))
    router.push('/home')
  } catch (e) {
    if (e.code === 'TOTP_REQUIRED') {
      show2FA.value = true
      if (e.preToken) preToken.value = e.preToken
      ElMessage.info(t('login.totpRequired'))
    } else {
      ElMessage.error(e.message || t('login.loginFailed'))
    }
  } finally {
    loading.value = false
  }
}

function clearCache() {
  try {
    localStorage.clear()
    sessionStorage.clear()
    document.cookie.split(';').forEach(cookie => {
      const [name] = cookie.split('=')
      document.cookie = `${name.trim()}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`
    })
    if ('caches' in window) {
      caches.keys().then(names => {
        names.forEach(name => caches.delete(name))
      })
    }
    ElMessage.success(t('login.cacheCleared'))
    setTimeout(() => {
      window.location.reload(true)
    }, 1000)
  } catch (e) {
    console.error('清除缓存失败:', e)
    ElMessage.error(t('login.cacheFailed'))
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1d1e1f 0%, #2c3e50 100%);
  padding: 0 16px;
  box-sizing: border-box;
}

.login-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  box-sizing: border-box;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-header h1 {
  margin: 0;
  font-size: 28px;
  color: #409eff;
}

.login-header p {
  margin: 8px 0 0;
  color: #909399;
  font-size: 14px;
}

.login-btn {
  width: 100%;
}

.cache-actions {
  margin-top: 16px;
  text-align: center;
}

.clear-cache-btn {
  color: #909399;
}

.clear-cache-btn:hover {
  color: #409eff;
}

@media (max-width: 768px) {
  .login-card {
    padding: 30px 24px;
  }

  .login-header {
    margin-bottom: 24px;
  }

  .login-header h1 {
    font-size: 24px;
  }

  .login-header p {
    font-size: 13px;
  }
}

@media (max-width: 480px) {
  .login-container {
    padding: 0 12px;
  }

  .login-card {
    padding: 24px 20px;
  }

  .login-header h1 {
    font-size: 22px;
  }
}
</style>
