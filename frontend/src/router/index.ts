import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import type { AppModule } from '@/types/app-module'

const modules: AppModule[] = ['sessions', 'workers', 'knowledge', 'skills', 'mcpServers', 'automations', 'settings']

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/sessions',
  },
  {
    path: '/sessions/:id?',
    name: 'sessions',
    component: () => import('@/views/SessionsView.vue'),
    props: (route) => ({ sessionId: route.params.id || null }),
  },
  ...modules.filter((m) => m !== 'sessions').map((m) => ({
    path: `/${m}`,
    name: m,
    component: () => import('@/views/ModuleView.vue'),
    props: { module: m },
  })),
]

declare const __ROUTER_BASE__: string

export const router = createRouter({
  history: createWebHistory(__ROUTER_BASE__),
  routes,
})
