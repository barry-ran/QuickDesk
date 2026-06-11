import { ref } from 'vue'
import { defineStore } from 'pinia'
import { getSettings } from '../api/settings.js'

export const useSettingsStore = defineStore('settings', () => {
  const siteName = ref('')
  const siteEnabled = ref(true)
  const loading = ref(true)

  async function loadSettings() {
    loading.value = true
    try {
      // §2.2: /v1/settings/public returns snake_case (site_name etc.).
      // The store keeps camelCase variable names because JS convention
      // is camelCase, only the wire field names changed.
      const data = await getSettings()
      siteName.value = data.site_name || 'QuickDesk'
      siteEnabled.value = data.site_enabled !== false
    } catch (e) {
      console.error('加载设置失败:', e)
    } finally {
      loading.value = false
    }
  }

  function updateSettings(data) {
    if (data.site_name !== undefined) {
      siteName.value = data.site_name
    }
    if (data.site_enabled !== undefined) {
      siteEnabled.value = data.site_enabled
    }
  }

  return {
    siteName,
    siteEnabled,
    loading,
    loadSettings,
    updateSettings
  }
})
