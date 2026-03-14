<template>
  <div class="remote-console">
    <div class="console-content">
      <el-row :gutter="20">
        <el-col :span="24">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>设备连接</span>
                <el-button type="primary" size="small" @click="loadDevices" :loading="loading">
                  <el-icon><Refresh /></el-icon>
                  刷新
                </el-button>
              </div>
            </template>
            <div class="device-list">
              <el-empty v-if="!devices.length && !loading" description="暂无设备连接" />
              <el-skeleton v-else-if="loading" :rows="3" animated />
              <div v-else class="device-grid">
                <el-card v-for="device in devices" :key="device.id" class="device-card" shadow="hover">
                  <div class="device-info">
                    <div class="device-icon">
                      <el-icon size="40"><Monitor /></el-icon>
                    </div>
                    <div class="device-details">
                      <h4>{{ device.device_name || '未命名设备' }}</h4>
                      <p>ID: {{ device.device_id }}</p>
                      <p v-if="device.remark" class="device-remark">备注: {{ device.remark }}</p>
                      <el-tag :type="device.online ? 'success' : 'info'" size="small">
                        {{ device.online ? '在线' : '离线' }}
                      </el-tag>
                    </div>
                  </div>
                  <div class="device-actions">
                    <el-button 
                      type="primary" 
                      size="small" 
                      :disabled="!device.online"
                      @click="connectDevice(device)"
                    >
                      连接
                    </el-button>
                    <el-button size="small" @click="viewDeviceInfo(device)">
                      详情
                    </el-button>
                    <el-button 
                      size="small" 
                      type="danger" 
                      plain
                      @click="unbindDevice(device)"
                    >
                      解绑
                    </el-button>
                  </div>
                </el-card>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
      
      <el-row :gutter="20" style="margin-top: 20px;">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>连接历史</span>
              </div>
            </template>
            <el-table :data="connectionHistory" style="width: 100%">
              <el-table-column prop="deviceName" label="设备名称" />
              <el-table-column prop="connectTime" label="连接时间" width="180" />
              <el-table-column prop="duration" label="时长" width="100" />
              <el-table-column prop="status" label="状态" width="80">
                <template #default="scope">
                  <el-tag :type="scope.row.status === '成功' ? 'success' : 'danger'" size="small">
                    {{ scope.row.status }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
        
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>快速连接</span>
              </div>
            </template>
            <div class="quick-connect">
              <el-form :model="quickConnectForm" label-width="80px">
                <el-form-item label="设备ID">
                  <el-input v-model="quickConnectForm.deviceId" placeholder="请输入设备ID（9位数字）" maxlength="9" />
                </el-form-item>
                <el-form-item label="设备名称">
                  <el-input v-model="quickConnectForm.deviceName" placeholder="请输入设备名称（可选）" maxlength="100" />
                </el-form-item>
                <el-form-item label="访问码">
                  <el-input v-model="quickConnectForm.accessCode" type="password" placeholder="请输入访问码（6位数字）" show-password maxlength="6" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="quickConnect" style="width: 100%">
                    快速连接
                  </el-button>
                </el-form-item>
              </el-form>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Monitor, Refresh } from '@element-plus/icons-vue'
import { getUserDevices, unbindDevice as unbindDeviceAPI } from '../api/device.js'

const router = useRouter()

const devices = ref([])
const connectionHistory = ref([])
const loading = ref(false)

const quickConnectForm = ref({
  deviceId: '',
  deviceName: '',
  accessCode: ''
})

// 获取当前用户ID
function getCurrentUserId() {
  const userInfo = localStorage.getItem('userInfo')
  if (!userInfo) return null
  try {
    const user = JSON.parse(userInfo)
    return user.id
  } catch (e) {
    return null
  }
}

// 加载用户设备列表
async function loadDevices() {
  const userId = getCurrentUserId()
  if (!userId) {
    ElMessage.error('无法获取用户信息')
    return
  }

  loading.value = true
  try {
    const result = await getUserDevices()

    if (result.devices) {
      // 获取设备在线状态
      const devicesWithStatus = await Promise.all(
        result.devices.map(async (device) => {
          const status = await getDeviceOnlineStatus(device.device_id)
          return {
            ...device,
            online: status.online
          }
        })
      )
      devices.value = devicesWithStatus
    } else if (result.error) {
      ElMessage.error(result.error)
    }

    // 加载连接历史
    await loadConnectionHistory()
  } catch (error) {
    ElMessage.error('加载设备列表失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

// 加载连接历史（近3天）
async function loadConnectionHistory() {
  const userId = getCurrentUserId()
  if (!userId) {
    return
  }

  try {
    const response = await fetch(`/api/v1/user/devices/logs?user_id=${userId}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('quickdesk_token')}`,
        'Content-Type': 'application/json'
      }
    })

    const result = await response.json()

    if (result.logs) {
      connectionHistory.value = result.logs
    }
  } catch (error) {
    console.error('加载连接历史失败:', error)
  }
}

// 获取设备在线状态
async function getDeviceOnlineStatus(deviceId) {
  try {
    const response = await fetch(`/api/v1/devices/${deviceId}/status`)
    const result = await response.json()
    return result || { online: false }
  } catch (error) {
    return { online: false }
  }
}

// 刷新设备列表
function refreshDevices() {
  loadDevices()
  ElMessage.success('设备列表已刷新')
}

// 连接设备
function connectDevice(device) {
  if (!device.online) {
    ElMessage.warning('设备离线，无法连接')
    return
  }

  // 从数据库读取访问码
  const accessCode = device.access_code || ''
  
  if (!accessCode) {
    ElMessage.error('设备未设置访问码')
    return
  }

  ElMessage.success(`正在连接设备: ${device.device_name || device.device_id}`)

  // 打开远程连接页面 - 指向 remote.html
  const apiServer = localStorage.getItem('quickdesk_api_server') || 'http://localhost:8000'
  const remoteUrl = `/remote.html?server=${encodeURIComponent(apiServer.replace('http', 'ws'))}&device=${device.device_id}&code=${accessCode}`

  // 在新窗口打开远程连接页面
  window.open(remoteUrl, '_blank')

  // 记录连接历史
  connectionHistory.value.unshift({
    deviceName: device.device_name || `设备-${device.device_id}`,
    connectTime: new Date().toLocaleString(),
    duration: '-',
    status: '连接中'
  })
}

// 查看设备详情
function viewDeviceInfo(device) {
  ElMessageBox.alert(`
    <div style="text-align: left;">
      <p><strong>设备ID:</strong> ${device.device_id}</p>
      <p><strong>设备名称:</strong> ${device.device_name || '未命名'}</p>
      <p><strong>访问码:</strong> ${device.access_code || '未设置'}</p>
      <p><strong>绑定类型:</strong> ${device.bind_type === 'auto' ? '自动绑定' : '手动绑定'}</p>
      <p><strong>最后连接:</strong> ${device.last_connect || '从未连接'}</p>
      <p><strong>连接次数:</strong> ${device.connect_count || 0}</p>
      <p><strong>绑定时间:</strong> ${new Date(device.created_at).toLocaleString()}</p>
    </div>
  `, '设备详情', {
    dangerouslyUseHTMLString: true,
    confirmButtonText: '关闭'
  })
}

// 解绑设备
async function unbindDevice(device) {
  try {
    await ElMessageBox.confirm(
      `确定要解绑设备 "${device.device_name || device.device_id}" 吗？`,
      '确认解绑',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const userId = getCurrentUserId()
    if (!userId) {
      ElMessage.error('无法获取用户信息')
      return
    }

    const result = await unbindDeviceAPI({
      user_id: userId,
      device_id: device.device_id
    })

    if (result.message === '设备解绑成功') {
      ElMessage.success('设备解绑成功')
      loadDevices()
    } else if (result.error) {
      ElMessage.error(result.error)
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('解绑设备失败: ' + error.message)
    }
  }
}

// 快速连接
async function quickConnect() {
  if (!quickConnectForm.value.deviceId || !quickConnectForm.value.accessCode) {
    ElMessage.error('请输入设备ID和访问码')
    return
  }

  const deviceId = quickConnectForm.value.deviceId.trim()
  const accessCode = quickConnectForm.value.accessCode.trim()
  const deviceName = quickConnectForm.value.deviceName.trim()

  // 验证设备ID格式（9位数字）
  if (!/^\d{9}$/.test(deviceId)) {
    ElMessage.error('设备ID必须是9位数字')
    return
  }

  // 验证访问码格式（6位数字）
  if (!/^\d{6}$/.test(accessCode)) {
    ElMessage.error('访问码必须是6位数字')
    return
  }

  const userId = getCurrentUserId()
  if (!userId) {
    ElMessage.error('无法获取用户信息')
    return
  }

  // 绑定设备名称、访问码和用户ID到devices表
  try {
    const bindResponse = await fetch('/api/v1/user/devices/quick-connect', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('quickdesk_token')}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        user_id: userId,
        device_id: deviceId,
        device_name: deviceName,
        access_code: accessCode
      })
    })

    const bindResult = await bindResponse.json()
    if (bindResult.error) {
      ElMessage.error('绑定设备失败: ' + bindResult.error)
      return
    }
  } catch (error) {
    console.error('绑定设备失败:', error)
    // 绑定失败不影响连接，继续执行
  }

  ElMessage.success('正在连接设备...')

  // 打开远程连接页面 - 指向 remote.html
  const apiServer = localStorage.getItem('quickdesk_api_server') || 'http://localhost:8000'
  const remoteUrl = `/remote.html?server=${encodeURIComponent(apiServer.replace('http', 'ws'))}&device=${deviceId}&code=${accessCode}`

  // 在新窗口打开远程连接页面
  window.open(remoteUrl, '_blank')

  // 记录连接历史（可选）
  connectionHistory.value.unshift({
    deviceName: deviceName || `设备-${deviceId}`,
    connectTime: new Date().toLocaleString(),
    duration: '-',
    status: '连接中'
  })

  // 清空表单
  quickConnectForm.value.deviceId = ''
  quickConnectForm.value.deviceName = ''
  quickConnectForm.value.accessCode = ''
}

onMounted(() => {
  if (!localStorage.getItem('quickdesk_token')) {
    ElMessage.error('请先登录')
    router.push('/user-login')
    return
  }

  // 加载用户设备列表
  loadDevices()
})
</script>

<style scoped>
.remote-console {
  min-height: 100vh;
  background: #f5f7fa;
  padding: 20px;
}

.console-content {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.device-list {
  padding: 20px 0;
}

.device-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.device-card {
  transition: all 0.3s;
}

.device-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.device-info {
  display: flex;
  align-items: center;
  margin-bottom: 15px;
}

.device-icon {
  margin-right: 15px;
  color: #409eff;
}

.device-details h4 {
  margin: 0 0 5px 0;
  color: #303133;
}

.device-details p {
  margin: 0 0 5px 0;
  color: #909399;
  font-size: 12px;
}

.device-remark {
  color: #606266;
  font-size: 12px;
  margin-top: 5px;
  padding: 4px 8px;
  background: #f4f4f5;
  border-radius: 4px;
}

.device-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.quick-connect {
  padding: 20px;
}

@media (max-width: 768px) {
  .remote-console {
    padding: 10px;
  }
  
  .device-grid {
    grid-template-columns: 1fr;
  }
  
  .el-row {
    flex-direction: column;
  }
  
  .el-col {
    width: 100% !important;
    margin-bottom: 20px;
  }
}
</style>
