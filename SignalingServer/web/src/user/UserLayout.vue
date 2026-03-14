<template>
  <div class="user-layout">
    <el-container>
      <el-aside width="220px" class="sidebar">
        <div class="logo">
          <h2>用户中心</h2>
        </div>
        <el-menu
          :default-active="activeMenu"
          class="sidebar-menu"
          router
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409eff"
        >
          <el-menu-item index="/user-console">
            <el-icon><HomeFilled /></el-icon>
            <span>用户控制台</span>
          </el-menu-item>
          <el-menu-item index="/remote-console">
            <el-icon><Monitor /></el-icon>
            <span>远程控制台</span>
          </el-menu-item>
          <el-menu-item index="/profile">
            <el-icon><User /></el-icon>
            <span>个人中心</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
      
      <el-container>
        <el-header class="header">
          <div class="header-left">
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/user-console' }">首页</el-breadcrumb-item>
              <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          <div class="header-right">
            <el-dropdown @command="handleCommand">
              <span class="user-dropdown">
                {{ userInfo.username }}
                <el-icon class="el-icon--right"><arrow-down /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                  <el-dropdown-item command="settings">系统设置</el-dropdown-item>
                  <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </el-header>
        
        <el-main class="main-content">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { HomeFilled, Monitor, User, ArrowDown } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userInfo = ref(JSON.parse(localStorage.getItem('userInfo') || '{}'))

const activeMenu = computed(() => route.path)

const pageTitle = computed(() => {
  const titles = {
    '/user-console': '用户控制台',
    '/remote-console': '远程控制台',
    '/profile': '个人中心'
  }
  return titles[route.path] || '用户控制台'
})

function handleCommand(command) {
  switch (command) {
    case 'profile':
      router.push('/profile')
      break
    case 'settings':
      ElMessage.info('系统设置功能开发中')
      break
    case 'logout':
      handleLogout()
      break
  }
}

function handleLogout() {
  localStorage.removeItem('quickdesk_token')
  localStorage.removeItem('userInfo')
  ElMessage.success('已退出登录')
  router.push('/user-login')
}

onMounted(() => {
  console.log('UserLayout onMounted - localStorage检查:', {
    quickdesk_token: localStorage.getItem('quickdesk_token'),
    quickdesk_admin_token: localStorage.getItem('quickdesk_admin_token')
  })
  
  const token = localStorage.getItem('quickdesk_token') || localStorage.getItem('quickdesk_admin_token')
  if (!token) {
    console.log('UserLayout: 未登录，跳转到登录页')
    ElMessage.error('请先登录')
    router.push('/user-login')
  }
})
</script>

<style scoped>
.user-layout {
  min-height: 100vh;
}

.sidebar {
  background-color: #304156;
  min-height: 100vh;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #2b3649;
  border-bottom: 1px solid #1f2d3d;
}

.logo h2 {
  margin: 0;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}

.sidebar-menu {
  border-right: none;
}

.sidebar-menu :deep(.el-menu-item) {
  height: 50px;
  line-height: 50px;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background-color: #263445 !important;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background-color: #263445 !important;
}

.header {
  background-color: #fff;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-dropdown {
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  display: flex;
  align-items: center;
}

.main-content {
  background-color: #f0f2f5;
  padding: 20px;
  overflow-y: auto;
}

:deep(.el-container) {
  min-height: 100vh;
}

@media (max-width: 768px) {
  .sidebar {
    width: 64px !important;
  }
  
  .logo h2 {
    display: none;
  }
  
  .sidebar-menu :deep(.el-menu-item span) {
    display: none;
  }
}
</style>