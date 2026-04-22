<!-- src/views/Home.vue -->
<template>
    <div class="flex h-screen bg-gradient-to-br from-slate-50 to-blue-50">
        <!-- Left sidebar -->
        <aside :class="['sidebar-wrapper', isCollapsed ? 'w-20' : 'w-64']">
            <!-- Logo area -->
            <div class="sidebar-logo  " :class="[isCollapsed ? 'justify-center' : 'justify-between']">
                <div v-if="!isCollapsed" class="flex items-center">
                    <div class="logo-icon">
                        <img src="@/assets/logo.png" alt="Logo" class="w-full h-full object-contain" />
                    </div>
                    <div class="ml-3">
                        <h1 class="text-lg font-bold text-gray-800">{{ t('app.supportSystem') }}</h1>
                        <p class="text-xs text-gray-400">{{ t('pageHomeLayout.systemDesc') }}</p>
                    </div>
                </div>

                <!-- Collapse/expand button -->
                <el-tooltip :content="isCollapsed ? t('pageHomeLayout.expandSidebar') : t('pageHomeLayout.collapseSidebar')" placement="right">
                    <button 
                        @click="toggleCollapse" 
                        class="collapse-btn"
                    >
                        <component :is="isCollapsed ? Expand : Fold" class="w-5 h-5" />
                    </button>
                </el-tooltip>
            </div>

            <!-- Navigation menu -->
            <nav class="flex-1 py-4 overflow-y-auto sidebar-nav">
                <ul class="space-y-1 px-3">
                    <li v-for="item in menuItems" :key="item.name">
                        <el-tooltip :content="item.name" placement="right" :disabled="!isCollapsed">
                            <router-link :to="item.path"
                                class="nav-item"
                                active-class="nav-item-active"
                                :class="[isCollapsed?'justify-center':'px-4']">
                                <component :is="item.icon" class="w-5 h-5" />
                                <span v-if="!isCollapsed" class="font-medium ml-3">{{ item.name }}</span>
                            </router-link>
                        </el-tooltip>
                    </li>
                </ul>
            </nav>

            <!-- User area -->
            <div class="sidebar-user">
                <el-tooltip :content="store.user?.username || store.user?.name || t('pageHomeLayout.defaultUser')" placement="right" :disabled="!isCollapsed">
                    <div class="user-info" :class="[isCollapsed ? 'justify-center' : 'px-3']">
                        <div class="user-avatar">
                            <span class="text-sm font-semibold">{{ (store.user?.username || store.user?.name || t('pageHomeLayout.defaultUser')).charAt(0).toUpperCase() }}</span>
                        </div>
                        <div v-if="!isCollapsed" class="flex-1 min-w-0 ml-3">
                            <p class="text-sm font-medium text-gray-800 truncate">{{ store.user?.username || store.user?.name || t('pageHomeLayout.defaultUser') }}</p>
                            <p class="text-xs text-gray-400">{{ store.isAdmin ? t('role.admin') : t('role.agent') }}</p>
                        </div>
                    </div>
                </el-tooltip>
                <el-tooltip :content="t('auth.signOut')" placement="right" :disabled="!isCollapsed">
                    <button @click="logout" class="logout-btn" :class="[isCollapsed ? 'justify-center' : '']">
                        <svg class="w-5 h-5" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
                            <path d="M136.2 513.8v366.7c0 26.4 23.4 47.8 52.4 47.8h261.8c28.9 0 52.4-21.4 52.4-47.8s-23.4-47.8-52.4-47.8H241V194.8h209.5c28.9 0 52.4-21.4 52.4-47.8s-23.4-47.8-52.4-47.8H188.6c-28.9 0-52.4 21.4-52.4 47.8v366.8z m757.5-34.6c10.1 8.8 16.4 21.5 16.4 35.6 0 14.2-6.3 26.8-16.4 35.6L728.3 693.8c-8.8 7.6-20.5 12.3-33.2 12.3-27.4 0-49.6-21.5-49.6-47.9 0-14.1 6.4-26.8 16.5-35.5l69.3-60.1h-301c-27.4 0-49.6-21.4-49.6-47.8s22.2-47.8 49.6-47.8h300.8l-69.3-60.1c-10.1-8.7-16.5-21.5-16.5-35.6 0-26.4 22.3-47.8 49.6-47.8 12.8 0 24.4 4.6 33.2 12.3l165.6 143.4z" fill="currentColor"/>
                        </svg>
                        <span v-if="!isCollapsed" class="ml-2">{{ t('auth.signOut') }}</span>
                    </button>
                </el-tooltip>
            </div>
        </aside>

        <!-- Right content -->
        <main class="flex-1 overflow-auto bg-transparent">
            <router-view />
        </main>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useStore } from '@/script/store'
import api from '@/script/api'
import { House, Avatar, Service, Setting, ChatDotRound, User, Fold, Expand, Monitor, Menu, DocumentChecked } from '@element-plus/icons-vue'
import { t } from '@/script/i18n-text'

const router = useRouter()
const route = useRoute()
const store = useStore()

// Sidebar collapsed state
const isCollapsed = ref(false)

// Load initial collapsed state
onMounted(() => {
    const savedCollapsed = localStorage.getItem('sidebar_collapsed')
    if (savedCollapsed !== null) {
        isCollapsed.value = savedCollapsed === 'true'
    }
})

// Toggle collapsed state
const toggleCollapse = () => {
    isCollapsed.value = !isCollapsed.value
    localStorage.setItem('sidebar_collapsed', isCollapsed.value.toString())
}

const menuItems = computed(() => {
    if (store.isAdmin) {
        return [
            { name: t('nav.dashboard'), path: '/home/dashboard', icon: House },
            { name: t('nav.inbox'), path: '/home/inbox', icon: ChatDotRound },
            { name: t('nav.visitors'), path: '/home/visitors', icon: Avatar },
            { name: t('nav.agents'), path: '/home/staff', icon: Service },
            { name: t('nav.apps'), path: '/home/apps', icon: Monitor },
            { name: t('nav.settings'), path: '/home/settings', icon: Setting },
            { name: t('nav.auditLogs'), path: '/home/audit', icon: DocumentChecked }
        ]
    } else {
        return [
            { name: t('nav.mySessions'), path: '/home/inbox', icon: ChatDotRound },
            { name: t('nav.moreSettings'), path: '/home/more', icon: Menu },
            { name: t('nav.profile'), path: '/home/profile', icon: User }
        ]
    }
})

const logout = async () => {
    try {
        await api.logout()
    } catch (error) {
        console.error('logout failed:', error)
    } finally {
        router.push('/login')
    }
}
</script>

<style scoped>
.sidebar-wrapper {
    background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
    border-right: 1px solid rgba(229, 231, 235, 0.6);
    display: flex;
    flex-direction: column;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.02);
    transition: width 0.3s ease;
}

.sidebar-logo {
    padding: 1rem;
    border-bottom: 1px solid rgba(229, 231, 235, 0.4);
    display: flex;
    align-items: center;
    
    min-height: 4.5rem;
}

.logo-icon {
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 0.75rem;
    background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
    padding: 0.375rem;
    box-shadow: 0 2px 4px rgba(59, 130, 246, 0.1);
}

.collapse-btn {
    width: 2rem;
    height: 2rem;
    border-radius: 0.5rem;
    border: 1px solid #e5e7eb;
    background: white;
    color: #6b7280;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s ease;
}

.collapse-btn:hover {
    background: #f3f4f6;
    border-color: #d1d5db;
    color: #374151;
}

.sidebar-nav {
    scrollbar-width: thin;
    scrollbar-color: #e2e8f0 transparent;
}

.sidebar-nav::-webkit-scrollbar {
    width: 4px;
}

.sidebar-nav::-webkit-scrollbar-track {
    background: transparent;
}

.sidebar-nav::-webkit-scrollbar-thumb {
    background: #e2e8f0;
    border-radius: 2px;
}

.nav-item {
    display: flex;
    align-items: center;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    border-radius: 0.75rem;
    color: #64748b;
    transition: all 0.2s ease;
    margin: 0.125rem 0;
}

.nav-item:hover {
    background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
    color: #334155;
}

.nav-item-active {
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
    color: white;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.25);
}

.nav-item-active:hover {
    background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
    color: white;
}

.sidebar-user {
    padding: 1rem;
    border-top: 1px solid rgba(229, 231, 235, 0.4);
    background: linear-gradient(180deg, #f8fafc 0%, #ffffff 100%);
}

.user-info {
    display: flex;
    align-items: center;
    margin-bottom: 0.75rem;
}

.user-avatar {
    width: 2.5rem;
    height: 2.5rem;
    border-radius: 50%;
    background: linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.logout-btn {
    width: 100%;
    padding: 0.5rem 0.75rem;
    border-radius: 0.5rem;
    border: 1px solid #fecaca;
    background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
    color: #dc2626;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
}

.logout-btn:hover {
    background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
    border-color: #fca5a5;
    box-shadow: 0 2px 4px rgba(220, 38, 38, 0.1);
}
</style>
