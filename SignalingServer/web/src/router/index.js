import { createRouter, createWebHashHistory } from 'vue-router'
import LoginPage from '../views/LoginPage.vue'
import UserLoginPage from '../user/UserLoginPage.vue'
import RegisterPage from '../user/RegisterPage.vue'
import UserLayout from '../user/UserLayout.vue'
import UserConsolePage from '../user/UserConsolePage.vue'
import RemoteConsolePage from '../user/RemoteConsolePage.vue'
import ProfilePage from '../user/ProfilePage.vue'
import HomePage from '../views/HomePage.vue'
import PresetPage from '../views/PresetPage.vue'
import DeviceListPage from '../views/DeviceListPage.vue'
import UsersPage from '../views/UsersPage.vue'
import AdminUserPage from '../views/AdminUserPage.vue'
import SettingsPage from '../views/SettingsPage.vue'

const routes = [
  { path: '/login', name: 'Login', component: LoginPage, meta: { public: true } },
  { path: '/user-login', name: 'UserLogin', component: UserLoginPage, meta: { public: true } },
  { path: '/register', name: 'Register', component: RegisterPage, meta: { public: true } },
  { path: '/', redirect: '/user-login' },
  {
    path: '/user-console',
    component: UserLayout,
    redirect: '/user-console/index',
    children: [
      { path: 'index', name: 'UserConsole', component: UserConsolePage, meta: { title: '用户控制台' } }
    ]
  },
  {
    path: '/remote-console',
    component: UserLayout,
    children: [
      { path: '', name: 'RemoteConsole', component: RemoteConsolePage, meta: { title: '远程控制台' } }
    ]
  },
  {
    path: '/profile',
    component: UserLayout,
    children: [
      { path: '', name: 'Profile', component: ProfilePage, meta: { title: '个人中心' } }
    ]
  },
  { path: '/home', name: 'Home', component: HomePage, meta: { title: '监控面板' } },
  { path: '/preset', name: 'Preset', component: PresetPage, meta: { title: '预设管理' } },
  { path: '/devices', name: 'Devices', component: DeviceListPage, meta: { title: '设备列表' } },
  { path: '/users', name: 'Users', component: UsersPage, meta: { title: '用户管理' } },
  { path: '/admin-users', name: 'AdminUsers', component: AdminUserPage, meta: { title: '管理员账户' } },
  { path: '/settings', name: 'Settings', component: SettingsPage, meta: { title: '系统设置' } }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to) => {
  const token = localStorage.getItem('quickdesk_token') || localStorage.getItem('quickdesk_admin_token')
  if (!to.meta.public && !token) {
    return '/user-login'
  }
})

export default router
