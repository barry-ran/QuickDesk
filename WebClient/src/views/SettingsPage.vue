<template>
  <div>
    <h2 class="page-title">{{ $t('settings.title') }}</h2>

    <div class="card">
      <div class="card-title">{{ $t('settings.network') }}</div>
      <div class="form-group">
        <label>{{ $t('settings.signalingServer') }}</label>
        <p class="card-subtitle">{{ $t('settings.signalingServerDesc') }}</p>
        <input v-model="serverUrl" class="form-input" type="text" :placeholder="$t('settings.signalingServerPlaceholder')" @change="saveServerUrl" />
      </div>
    </div>

    <div class="card">
      <div class="card-title">{{ $t('settings.video') }}</div>
      <div class="form-group">
        <label>{{ $t('settings.videoCodec') }}</label>
        <p class="card-subtitle">{{ $t('settings.videoCodecDesc') }}</p>
        <select v-model="videoCodec" class="form-input" @change="saveCodec">
          <option value="AV1">AV1</option>
          <option value="VP9">VP9</option>
          <option value="VP8">VP8</option>
          <option value="H264">H264</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { userApi, DEFAULT_SERVER } from '../api/userApi'

const { t } = useI18n()
const showToast = inject('showToast')

const serverUrl = ref(localStorage.getItem('quickdesk_signaling_url') || DEFAULT_SERVER)
const videoCodec = ref(localStorage.getItem('quickdesk_video_codec') || 'AV1')

function saveServerUrl() {
  const url = serverUrl.value.trim()
  if (url) {
    localStorage.setItem('quickdesk_signaling_url', url)
    userApi.setBaseUrl(url)
    showToast(t('settings.signalingServerSaved'), 'info')
  }
}

function saveCodec() {
  localStorage.setItem('quickdesk_video_codec', videoCodec.value)
  showToast(t('settings.codecChanged', { codec: videoCodec.value }), 'info')
}
</script>
