import { createRouter, createWebHashHistory } from 'vue-router'
import LoginPage from '../views/LoginPage.vue'
import HomePage from '../views/HomePage.vue'
import PresetPage from '../views/PresetPage.vue'
import DeviceListPage from '../views/DeviceListPage.vue'
import DeviceDetailPage from '../views/DeviceDetailPage.vue'
import UsersPage from '../views/UsersPage.vue'
import UserDetailPage from '../views/UserDetailPage.vue'
import AdminUserPage from '../views/AdminUserPage.vue'
import DeviceGroupPage from '../views/DeviceGroupPage.vue'
import AuditLogPage from '../views/AuditLogPage.vue'
import SystemLogsPage from '../views/SystemLogsPage.vue'
import WebhooksPage from '../views/WebhooksPage.vue'
import SettingsPage from '../views/SettingsPage.vue'

const routes = [
  { path: '/login', name: 'Login', component: LoginPage, meta: { public: true } },
  { path: '/', redirect: '/home' },
  { path: '/home', name: 'Home', component: HomePage, meta: { title: 'nav.dashboard' } },
  { path: '/preset', name: 'Preset', component: PresetPage, meta: { title: 'nav.preset' } },
  { path: '/devices', name: 'Devices', component: DeviceListPage, meta: { title: 'nav.devices' } },
  { path: '/devices/:deviceId', name: 'DeviceDetail', component: DeviceDetailPage, meta: { title: 'nav.devices' } },
  { path: '/users', name: 'Users', component: UsersPage, meta: { title: 'nav.users' } },
  { path: '/users/:id', name: 'UserDetail', component: UserDetailPage, meta: { title: 'nav.users' } },
  { path: '/admin-users', name: 'AdminUsers', component: AdminUserPage, meta: { title: 'nav.adminUsers' } },
  { path: '/device-groups', name: 'DeviceGroups', component: DeviceGroupPage, meta: { title: 'nav.deviceGroups' } },
  { path: '/audit-logs', name: 'AuditLogs', component: AuditLogPage, meta: { title: 'nav.auditLogs' } },
  { path: '/system-logs', name: 'SystemLogs', component: SystemLogsPage, meta: { title: 'nav.systemLogs' } },
  { path: '/webhooks', name: 'Webhooks', component: WebhooksPage, meta: { title: 'nav.webhooks' } },
  { path: '/settings', name: 'Settings', component: SettingsPage, meta: { title: 'nav.settings' } }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to) => {
  if (to.meta.public) return
  const token = localStorage.getItem('quickdesk_admin_access_token')
  if (!token) {
    return '/login'
  }
})

export default router
