<template>
  <div class="home-page" v-loading="loading">
    <div class="page-header">
      <h2>{{ t('dashboard.title') }}</h2>
      <el-button
        type="primary"
        size="small"
        @click="loadStats"
        :icon="Refresh"
      >
        {{ t('dashboard.refreshData') }}
      </el-button>
    </div>

    <!-- Overview Cards -->
    <div class="overview-cards">
      <div class="overview-card purple">
        <div class="overview-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-value">{{ metricValue(overview.totalDevices) }}</div>
          <div class="overview-label">{{ t('dashboard.totalDevices') }}</div>
          <div class="overview-desc">{{ t('dashboard.totalDevicesDesc') }}</div>
        </div>
      </div>
      <div class="overview-card blue">
        <div class="overview-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-value">{{ metricValue(overview.totalConnections) }}</div>
          <div class="overview-label">{{ t('dashboard.totalConnections') }}</div>
          <div class="overview-desc">{{ t('dashboard.totalConnectionsDesc') }}</div>
        </div>
      </div>
      <div class="overview-card green">
        <div class="overview-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-value">{{ metricValue(overview.webSocketConnections) }}</div>
          <div class="overview-label">{{ t('dashboard.wsConnections') }}</div>
          <div class="overview-desc">{{ t('dashboard.wsConnectionsDesc') }}</div>
        </div>
      </div>
      <div class="overview-card orange">
        <div class="overview-icon">
          <el-icon><DataLine /></el-icon>
        </div>
        <div class="overview-content">
          <div class="overview-value">{{ metricValue(overview.apiRequests) }}</div>
          <div class="overview-label">{{ t('dashboard.apiRequests') }}</div>
          <div class="overview-desc">{{ t('dashboard.apiRequestsDesc') }}</div>
        </div>
      </div>
    </div>

    <!-- Today Summary Cards -->
    <div class="today-cards">
      <div class="today-card">
        <div class="today-value">{{ todaySummary.todayNewDevices }}</div>
        <div class="today-label">{{ t('dashboard.todayNewDevices') }}</div>
      </div>
      <div class="today-card">
        <div class="today-value">{{ todaySummary.todayConnections }}</div>
        <div class="today-label">{{ t('dashboard.todayConnections') }}</div>
      </div>
      <div class="today-card">
        <div class="today-value">{{ todaySummary.todayActiveUsers }}</div>
        <div class="today-label">{{ t('dashboard.todayActiveUsers') }}</div>
      </div>
    </div>

    <!-- Trends Chart -->
    <el-card class="trends-card" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <el-icon><DataLine /></el-icon>
          <span>{{ t('dashboard.trends') }}</span>
          <div class="activity-actions">
            <el-radio-group v-model="trendRange" size="small" @change="loadTrends">
              <el-radio-button value="24h">24h</el-radio-button>
              <el-radio-button value="7d">7d</el-radio-button>
              <el-radio-button value="30d">30d</el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </template>
      <div ref="chartRef" style="width:100%;height:300px"></div>
    </el-card>

    <!-- Activity Section -->
    <el-card class="activity-card" style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <el-icon class="card-icon"><Timer /></el-icon>
          <span>{{ t('dashboard.recentActivity') }}</span>
          <div class="activity-actions">
            <el-button :icon="Download" size="small" @click="handleExportActivity">{{ t('common.export') }}</el-button>
            <el-button type="primary" size="small" @click="loadActivity" :icon="Refresh">{{ t('common.refresh') }}</el-button>
          </div>
        </div>
      </template>

      <!-- Activity Filters -->
      <div class="activity-filters">
        <el-select v-model="activityFilters.dateRange" style="width: 130px" size="small" @change="handleActivityFilter">
          <el-option :label="t('dashboard.today')" value="today" />
          <el-option :label="t('dashboard.last7Days')" value="7days" />
          <el-option :label="t('dashboard.last30Days')" value="30days" />
          <el-option :label="t('dashboard.allTime')" value="all" />
        </el-select>
        <el-input
          v-model="activityFilters.deviceId"
          :placeholder="t('dashboard.filterDevice')"
          clearable
          size="small"
          style="width: 160px"
          @clear="handleActivityFilter"
          @keyup.enter="handleActivityFilter"
        />
        <el-select v-model="activityFilters.status" :placeholder="t('dashboard.filterStatus')" clearable size="small" style="width: 120px" @change="handleActivityFilter">
          <el-option :label="t('common.success')" value="success" />
          <el-option :label="t('common.failed')" value="failed" />
        </el-select>
      </div>

      <el-table :data="activityList" stripe style="width: 100%" size="small" :row-class-name="rowClassName">
        <el-table-column prop="created_at" :label="t('dashboard.time')" width="180" />
        <el-table-column prop="device_id" :label="t('dashboard.deviceId')" width="120" />
        <el-table-column prop="device_name" :label="t('dashboard.activity')" width="150" />
        <el-table-column prop="error_msg" :label="t('dashboard.details')" show-overflow-tooltip />
        <el-table-column prop="status" :label="t('common.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'warning'" size="small">
              {{ row.status === 'success' ? t('common.success') : t('common.failed') }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="activityList.length === 0 && !loading" class="empty-state">
        <el-empty :description="t('dashboard.noActivity')" />
      </div>

      <div class="pagination-bar">
        <!-- §3.1 cursor-based pagination. See components/CursorPagination.vue.
             Note: GET /v1/admin/activity (admin_stats_handler.go:97) only
             consumes ?cursor=&limit= today — the activityFilters.deviceId/
             status/dateRange are forwarded but the server ignores them.
             They're kept in the UI so we don't lose the filter affordance
             when the server adds support later. -->
        <CursorPagination
          :cursor-stack="activityPagination.cursorStack"
          :next-cursor="activityPagination.nextCursor"
          :total="activityPagination.total"
          :limit="activityPagination.limit"
          :loading="loading"
          @prev="goPrevActivity"
          @next="goNextActivity"
          @update:limit="onActivityLimitChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Monitor, Connection, Timer, Refresh, DataLine, Download } from '@element-plus/icons-vue'
import { getStats, getSystemStatus, getConnectionStatus, getActivity, getTrends } from '../api/stats.js'
import { exportCSV } from '../utils/export.js'
import CursorPagination from '../components/CursorPagination.vue'
import * as echarts from 'echarts'

const { t } = useI18n()
const loading = ref(false)

const overview = ref({
  totalDevices: null,
  totalConnections: null,
  webSocketConnections: null,
  apiRequests: null
})

const todaySummary = ref({
  todayNewDevices: 0,
  todayConnections: 0,
  todayActiveUsers: 0
})

const stats = ref({
  totalDevices: 0,
  onlineDevices: 0,
  offlineDevices: 0,
  onlineRate: 0
})

const connectionStatus = ref({
  currentConnections: 0,
  todayConnections: 0,
  webSocketConnections: 0,
  apiRequests: 0
})

const chartRef = ref(null)
const trendRange = ref('24h')
let chartInstance = null

const activityList = ref([])

const activityFilters = reactive({
  dateRange: 'today',
  deviceId: '',
  status: ''
})

const activityPagination = reactive({
  cursorStack: [''],
  nextCursor: '',
  total: 0,
  limit: 20
})

function rowClassName({ row }) {
  return row.status === 'success' ? 'success-row' : 'failed-row'
}

function metricValue(value) {
  return typeof value === 'number' ? value : '—'
}

function applyDashboardStats(statsData, connectionData) {
  overview.value.totalDevices = statsData.devices_total || 0
  overview.value.totalConnections = connectionData.current_connections ?? 0
  overview.value.webSocketConnections = connectionData.websocket_connections || 0
  overview.value.apiRequests = connectionData.api_requests_today || 0

  connectionStatus.value = {
    currentConnections: overview.value.totalConnections || 0,
    todayConnections: connectionData.today_connections || 0,
    webSocketConnections: overview.value.webSocketConnections || 0,
    apiRequests: overview.value.apiRequests || 0
  }

  todaySummary.value.todayNewDevices = statsData.devices_new_today || 0
  todaySummary.value.todayConnections = connectionData.today_connections || 0
  todaySummary.value.todayActiveUsers = statsData.today_active_users || 0
}

async function loadActivity() {
  try {
    const data = await getActivity({
      cursor: activityPagination.cursorStack.at(-1),
      limit: activityPagination.limit
    })
    activityList.value = data.items || data.activity || []
    activityPagination.nextCursor = data.next_cursor || ''
    if (typeof data.total === 'number') activityPagination.total = data.total
  } catch (e) {
    ElMessage.error(t('dashboard.activityFailed') + ': ' + e.message)
  }
}

function resetActivityCursor() {
  activityPagination.cursorStack = ['']
  activityPagination.nextCursor = ''
  loadActivity()
}

function goPrevActivity() {
  if (activityPagination.cursorStack.length <= 1) return
  activityPagination.cursorStack.pop()
  loadActivity()
}

function goNextActivity() {
  if (!activityPagination.nextCursor) return
  activityPagination.cursorStack.push(activityPagination.nextCursor)
  loadActivity()
}

function onActivityLimitChange(n) {
  activityPagination.limit = n
  resetActivityCursor()
}

function handleActivityFilter() {
  // Server-side filtering for activity is not implemented yet; reload to
  // keep the UI deterministic when the user toggles filters.
  resetActivityCursor()
}

async function loadStats() {
  loading.value = true
  try {
    const [statsData, systemData, connectionData] = await Promise.all([
      getStats(),
      getSystemStatus(),
      getConnectionStatus()
    ])
    stats.value = statsData
    applyDashboardStats(statsData, connectionData)

    ElMessage.success(t('dashboard.statsUpdated'))
  } catch (e) {
    ElMessage.error(t('dashboard.statsFailed') + ': ' + e.message)
  } finally {
    loading.value = false
  }
}

function handleExportActivity() {
  const columns = [
    { key: 'created_at', label: 'Time' },
    { key: 'device_id', label: 'Device ID' },
    { key: 'device_name', label: 'Device Name' },
    { key: 'error_msg', label: 'Details' },
    { key: 'status', label: 'Status' }
  ]
  exportCSV(columns, activityList.value, 'activity.csv')
}

async function loadTrends() {
  try {
    const data = await getTrends(trendRange.value)
    if (!data) return
    await nextTick()
    if (!chartRef.value) return
    if (!chartInstance) {
      chartInstance = echarts.init(chartRef.value)
    }
    const rawConns = Array.isArray(data.connections) ? data.connections : []
    const rawDevices = Array.isArray(data.devices) ? data.devices : []
    // API returns [{day, count}, ...]; extract labels from the longer series.
    const longerSeries = rawConns.length >= rawDevices.length ? rawConns : rawDevices
    const labels = longerSeries.map(r => r.day ? r.day.substring(0, 10) : '')
    const connections = rawConns.map(r => r.count || 0)
    const newDevices = rawDevices.map(r => r.count || 0)
    chartInstance.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: [t('dashboard.totalConnections'), t('dashboard.todayNewDevices')] },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: { type: 'category', data: labels, axisLabel: { rotate: labels.length > 24 ? 45 : 0 } },
      yAxis: [
        { type: 'value', name: t('dashboard.totalConnections'), minInterval: 1 },
        { type: 'value', name: t('dashboard.todayNewDevices'), minInterval: 1 }
      ],
      series: [
        { name: t('dashboard.totalConnections'), type: 'line', smooth: true, data: connections },
        { name: t('dashboard.todayNewDevices'), type: 'line', smooth: true, yAxisIndex: 1, data: newDevices }
      ]
    })
  } catch (e) {
    console.error('Failed to load trends:', e.message)
  }
}

let systemStatusTimer = null

async function refreshSystemStatus() {
  try {
    const [statsData, connectionData] = await Promise.all([
      getStats(),
      getConnectionStatus()
    ])
    stats.value = statsData
    applyDashboardStats(statsData, connectionData)
  } catch (e) {
    console.error('Failed to refresh system status:', e.message)
  }
}

function startSystemStatusAutoRefresh() {
  if (systemStatusTimer) clearInterval(systemStatusTimer)
  systemStatusTimer = setInterval(refreshSystemStatus, 5000)
}

function stopSystemStatusAutoRefresh() {
  if (systemStatusTimer) {
    clearInterval(systemStatusTimer)
    systemStatusTimer = null
  }
}

onMounted(() => {
  loadStats()
  loadActivity()
  loadTrends()
  startSystemStatusAutoRefresh()
  window.addEventListener('resize', handleResize)
})

function handleResize() {
  if (chartInstance) chartInstance.resize()
}

onUnmounted(() => {
  stopSystemStatusAutoRefresh()
  window.removeEventListener('resize', handleResize)
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})
</script>

<style scoped>
.home-page {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
  overflow: hidden;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.overview-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.overview-card {
  display: flex;
  align-items: center;
  padding: 20px;
  border-radius: 12px;
  color: white;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.overview-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.overview-card.purple {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.overview-card.blue {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.overview-card.green {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.overview-card.orange {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.overview-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 28px;
}

.overview-content {
  flex: 1;
}

.overview-value {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: 4px;
}

.overview-label {
  font-size: 16px;
  font-weight: 500;
  margin-bottom: 4px;
  opacity: 0.95;
}

.overview-desc {
  font-size: 12px;
  opacity: 0.8;
}

.today-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.today-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px 20px;
  text-align: center;
}

.today-value {
  font-size: 28px;
  font-weight: 700;
  color: #409eff;
  margin-bottom: 4px;
}

.today-label {
  font-size: 13px;
  color: #909399;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.activity-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.activity-filters {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
  align-items: center;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}

@media (max-width: 1200px) {
  .overview-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .overview-cards {
    grid-template-columns: 1fr;
  }

  .today-cards {
    grid-template-columns: 1fr;
  }

  .home-page {
    padding: 12px;
  }
}
</style>
