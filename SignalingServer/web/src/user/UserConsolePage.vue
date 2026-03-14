<template>
  <div class="user-console">
    <div class="console-content">
      <el-row :gutter="20">
        <el-col :span="8">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>账户信息</span>
              </div>
            </template>
            <div class="user-details">
              <div class="detail-item">
                <span class="label">用户名:</span>
                <span class="value">{{ userInfo.username }}</span>
              </div>
              <div class="detail-item">
                <span class="label">手机号:</span>
                <span class="value">{{ userInfo.phone || '未设置' }}</span>
              </div>
              <div class="detail-item">
                <span class="label">邮箱:</span>
                <span class="value">{{ userInfo.email || '未设置' }}</span>
              </div>
              <div class="detail-item">
                <span class="label">等级:</span>
                <span class="value">{{ userInfo.level }}</span>
              </div>
              <div class="detail-item">
                <span class="label">通道类型:</span>
                <span class="value">{{ userInfo.channelType }}</span>
              </div>
              <div class="detail-item">
                <span class="label">设备数量:</span>
                <span class="value">{{ userInfo.deviceCount || 0 }}</span>
              </div>
              <div class="detail-item">
                <span class="label">注册时间:</span>
                <span class="value">{{ formatDate(userInfo.createdAt) }}</span>
              </div>
            </div>
          </el-card>
        </el-col>
        
        <el-col :span="16">
          <el-card shadow="hover">
            <template #header>
              <div class="card-header">
                <span>使用统计</span>
              </div>
            </template>
            <div class="stats-grid">
              <div class="stat-item">
                <el-icon class="stat-icon"><Monitor /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">0</div>
                  <div class="stat-label">活跃设备</div>
                </div>
              </div>
              <div class="stat-item">
                <el-icon class="stat-icon"><Connection /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">0</div>
                  <div class="stat-label">当前连接</div>
                </div>
              </div>
              <div class="stat-item">
                <el-icon class="stat-icon"><Timer /></el-icon>
                <div class="stat-info">
                  <div class="stat-value">0</div>
                  <div class="stat-label">今日时长</div>
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
                <span>最近活动</span>
              </div>
            </template>
            <el-table :data="recentActivities" style="width: 100%">
              <el-table-column prop="time" label="时间" width="180" />
              <el-table-column prop="action" label="操作" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="scope">
                  <el-tag :type="scope.row.status === '成功' ? 'success' : 'danger'">
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
const recentActivities = ref([
  { time: '2026-03-08 12:00', action: '登录系统', status: '成功' },
  { time: '2026-03-07 15:30', action: '添加设备', status: '成功' },
  { time: '2026-03-06 09:15', action: '修改密码', status: '成功' }
])

function formatDate(dateString) {
  if (!dateString) return '未知'
  const date = new Date(dateString)
  return date.toLocaleString()
}

onMounted(() => {
  if (!localStorage.getItem('quickdesk_admin_token')) {
    ElMessage.error('请先登录')
    router.push('/user-login')
  }
})
</script>

<style scoped>
.user-console {
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

.user-details {
  padding: 10px 0;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  padding: 5px 0;
  border-bottom: 1px solid #f0f0f0;
}

.detail-item:last-child {
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
  .user-console {
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