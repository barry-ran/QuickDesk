<template>
  <div class="profile-page">
    <div class="console-content">
      <el-row :gutter="20">
        <el-col :span="8">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>基本信息</span>
                <el-button type="primary" size="small" @click="editMode = !editMode">
                  {{ editMode ? '取消' : '编辑' }}
                </el-button>
              </div>
            </template>
            <div class="profile-info">
              <div class="avatar-section">
                <el-avatar :size="80" :src="userInfo.avatar || defaultAvatar" />
                <h3>{{ userInfo.username }}</h3>
                <el-tag size="small" type="success">{{ userInfo.level }}</el-tag>
              </div>
              
              <el-form v-if="editMode" :model="editForm" label-width="80px">
                <el-form-item label="用户名">
                  <el-input v-model="editForm.username" disabled />
                </el-form-item>
                <el-form-item label="手机号">
                  <el-input v-model="editForm.phone" placeholder="请输入手机号" />
                </el-form-item>
                <el-form-item label="邮箱">
                  <el-input v-model="editForm.email" placeholder="请输入邮箱" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="saveProfile" style="width: 100%">
                    保存修改
                  </el-button>
                </el-form-item>
              </el-form>
              
              <div v-else class="info-list">
                <div class="info-item">
                  <span class="label">用户名:</span>
                  <span class="value">{{ userInfo.username }}</span>
                </div>
                <div class="info-item">
                  <span class="label">手机号:</span>
                  <span class="value">{{ userInfo.phone || '未设置' }}</span>
                </div>
                <div class="info-item">
                  <span class="label">邮箱:</span>
                  <span class="value">{{ userInfo.email || '未设置' }}</span>
                </div>
                <div class="info-item">
                  <span class="label">等级:</span>
                  <span class="value">{{ userInfo.level }}</span>
                </div>
                <div class="info-item">
                  <span class="label">通道类型:</span>
                  <span class="value">{{ userInfo.channelType }}</span>
                </div>
                <div class="info-item">
                  <span class="label">注册时间:</span>
                  <span class="value">{{ formatDate(userInfo.createdAt) }}</span>
                </div>
              </div>
            </div>
          </el-card>
          
          <el-card shadow="hover" style="margin-top: 20px;">
            <template #header>
              <div class="card-header">
                <span>修改密码</span>
              </div>
            </template>
            <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="100px">
              <el-form-item label="原密码" prop="oldPassword">
                <el-input v-model="passwordForm.oldPassword" type="password" placeholder="请输入原密码" show-password />
              </el-form-item>
              <el-form-item label="新密码" prop="newPassword">
                <el-input v-model="passwordForm.newPassword" type="password" placeholder="请输入新密码" show-password />
              </el-form-item>
              <el-form-item label="确认密码" prop="confirmPassword">
                <el-input v-model="passwordForm.confirmPassword" type="password" placeholder="请再次输入新密码" show-password />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="changePassword" style="width: 100%">
                  修改密码
                </el-button>
              </el-form-item>
            </el-form>
          </el-card>
        </el-col>
        
        <el-col :span="16">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>账户统计</span>
              </div>
            </template>
            <div class="stats-grid">
              <div class="stat-item">
                <el-icon class="stat-icon"><Monitor /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">{{ userInfo.deviceCount || 0 }}</div>
                  <div class="stat-label">设备数量</div>
                </div>
              </div>
              <div class="stat-item">
                <el-icon class="stat-icon"><Connection /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">0</div>
                  <div class="stat-label">今日连接</div>
                </div>
              </div>
              <div class="stat-item">
                <el-icon class="stat-icon"><Timer /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">0</div>
                  <div class="stat-label">本月时长</div>
                </div>
              </div>
              <div class="stat-item">
                <el-icon class="stat-icon"><DataAnalysis /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">0</div>
                  <div class="stat-label">本月流量</div>
                </div>
              </div>
            </div>
          </el-card>
          
          <el-card shadow="hover" style="margin-top: 20px;">
            <template #header>
              <div class="card-header">
                <span>登录日志</span>
              </div>
            </template>
            <el-table :data="loginLogs" style="width: 100%">
              <el-table-column prop="time" label="登录时间" width="180" />
              <el-table-column prop="ip" label="IP地址" width="150" />
              <el-table-column prop="device" label="设备" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="scope">
                  <el-tag :type="scope.row.status === '成功' ? 'success' : 'danger'" size="small">
                    {{ scope.row.status }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Monitor, Connection, Timer, DataAnalysis } from '@element-plus/icons-vue'

const router = useRouter()
const userInfo = ref(JSON.parse(localStorage.getItem('userInfo') || '{}'))
const defaultAvatar = 'https://cube.elemecdn.com/3/7c/3ea6beec64369c2642b92c6726f1epng.png'

const editMode = ref(false)
const editForm = ref({
  username: userInfo.value.username,
  phone: userInfo.value.phone,
  email: userInfo.value.email
})

const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const passwordFormRef = ref(null)

const passwordRules = {
  oldPassword: [
    { required: true, message: '请输入原密码', trigger: 'blur' }
  ],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== passwordForm.value.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

const loginLogs = ref([
  { time: '2026-03-08 12:30:00', ip: '192.168.1.100', device: 'Chrome / Windows', status: '成功' },
  { time: '2026-03-07 15:20:00', ip: '192.168.1.101', device: 'Safari / macOS', status: '成功' },
  { time: '2026-03-06 09:15:00', ip: '192.168.1.102', device: 'Firefox / Windows', status: '失败' }
])

function formatDate(dateString) {
  if (!dateString) return '未知'
  const date = new Date(dateString)
  return date.toLocaleString()
}

function saveProfile() {
  ElMessage.success('个人信息保存成功')
  editMode.value = false
  // 这里可以添加实际的保存逻辑
}

async function changePassword() {
  if (!passwordFormRef.value) return
  
  await passwordFormRef.value.validate((valid) => {
    if (valid) {
      ElMessage.success('密码修改成功')
      passwordForm.value = {
        oldPassword: '',
        newPassword: '',
        confirmPassword: ''
      }
    }
  })
}

onMounted(() => {
  if (!localStorage.getItem('quickdesk_admin_token')) {
    ElMessage.error('请先登录')
    router.push('/user-login')
  }
})
</script>

<style scoped>
.profile-page {
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

.profile-info {
  padding: 20px 0;
}

.avatar-section {
  text-align: center;
  margin-bottom: 30px;
}

.avatar-section h3 {
  margin: 10px 0 5px 0;
  color: #303133;
}

.info-list {
  padding: 10px 0;
}

.info-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 15px;
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
}

.info-item:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.label {
  color: #909399;
  font-size: 14px;
}

.value {
  color: #303133;
  font-size: 14px;
  font-weight: 500;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.stat-item {
  display: flex;
  align-items: center;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  transition: all 0.3s;
}

.stat-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.stat-icon {
  font-size: 28px;
  color: #409eff;
  margin-right: 15px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

@media (max-width: 768px) {
  .profile-page {
    padding: 10px;
  }
  
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
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