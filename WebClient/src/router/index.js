import { createRouter, createWebHashHistory } from 'vue-router'
import RemotePage from '../views/RemotePage.vue'
import DevicesPage from '../views/DevicesPage.vue'
import RecordingsPage from '../views/RecordingsPage.vue'
import AccountPage from '../views/AccountPage.vue'
import SettingsPage from '../views/SettingsPage.vue'
import AboutPage from '../views/AboutPage.vue'
import ResetPasswordPage from '../views/ResetPasswordPage.vue'

const routes = [
  { path: '/', redirect: '/remote' },
  { path: '/remote', component: RemotePage },
  { path: '/devices', component: DevicesPage },
  { path: '/recordings', component: RecordingsPage },
  { path: '/account', component: AccountPage },
  { path: '/settings', component: SettingsPage },
  { path: '/about', component: AboutPage },
  { path: '/reset-password', component: ResetPasswordPage },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
