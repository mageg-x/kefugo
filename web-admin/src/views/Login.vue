<template>
    <div class="min-h-screen bg-white overflow-hidden relative">
        <!-- Left gradient background -->
        <div class="absolute left-0 top-0 w-full h-full bg-gradient-to-br from-blue-600 via-blue-700 to-indigo-900 z-0">
            <!-- Decorative shapes -->
            <div class="absolute top-[20%] left-[15%] w-40 h-40 bg-purple-700/30 rounded-2xl transform rotate-12"></div>
            <div class="absolute bottom-[25%] left-[30%] w-64 h-64 bg-indigo-800/20 rounded-2xl transform -rotate-12">
            </div>
            <div class="absolute top-[50%] left-[20%] w-32 h-32 bg-blue-600/25 rounded-2xl transform rotate-6"></div>

            <!-- Left content -->
            <div class="absolute left-0 top-0 w-3/5 h-full flex flex-col justify-center px-20 text-white z-10">
                <div class="mb-8 flex items-center animate-pulse">
                    <img src="@/assets/logo.png" :alt="t('pageLogin.brandFull')" class="w-24 h-24 mr-3 drop-shadow-lg animate-glow">
                    <span class="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-white to-blue-200 drop-shadow-md">{{ t('pageLogin.brandLeft') }}</span>
                    <span class="text-4xl font-bold ml-3 bg-clip-text text-transparent bg-gradient-to-r from-blue-300 to-cyan-200 drop-shadow-md">{{ t('pageLogin.brandRight') }}</span>
                </div>
                <h1 class="text-5xl font-bold mb-6 leading-tight bg-clip-text text-transparent bg-gradient-to-r from-white via-blue-100 to-cyan-200 drop-shadow-lg animate-pulse">{{ t('pageLogin.slogan') }}</h1>
                <p class="text-blue-100 opacity-95 max-w-md text-lg drop-shadow-md animate-pulse">
                    <span class="text-cyan-300 font-medium">{{ t('pageLogin.studio') }}</span>{{ t('pageLogin.studioDesc') }}<br>
                    {{ t('pageLogin.studioMission') }}
                </p>
            </div>
        </div>

        <!-- Right white curved panel -->
        <div
            class="absolute right-0 top-0 w-1/2 h-full bg-gradient-to-bl from-white to-blue-50 z-20 transform -skew-x-6 origin-top-right">
        </div>

        <!-- Login card -->
        <div
            class="login-card absolute right-[10%] top-1/2 transform -translate-y-1/2 bg-white/95 backdrop-blur-sm rounded-2xl shadow-2xl p-8 border border-blue-200 w-96 z-30 hover:shadow-blue-500/30 transition-all duration-300 animate-float animate-border-glow">
            <!-- Title and locale switch -->
            <div class="flex justify-between items-center mb-8">
                <h2 class="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-800 to-indigo-600">{{ t('pageLogin.staffLogin') }}</h2>
                <el-select
                    class="login-locale"
                    size="small"
                    :model-value="localeRef"
                    @change="handleLocaleChange"
                    :teleported="false"
                >
                    <el-option
                        v-for="opt in localeOptions"
                        :key="opt.value"
                        :label="opt.label"
                        :value="opt.value"
                    />
                </el-select>
            </div>

            <!-- Login form -->
            <el-form :model="loginForm" @submit.prevent="handleLogin">
                <!-- Account input -->
                <el-form-item class="mb-4">
                    <el-input v-model="loginForm.username" :placeholder="t('form.enterAccount')" :prefix-icon="UserFilled" size="large"
                        class="rounded-xl border-blue-200 hover:border-blue-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-300/50 transition-all duration-300" :disabled="loading" />
                </el-form-item>

                <!-- Password input -->
                <el-form-item class="mb-6">
                    <el-input v-model="loginForm.password" type="password" :placeholder="t('form.enterPassword')" :prefix-icon="Lock"
                        size="large" class="rounded-xl border-blue-200 hover:border-blue-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-300/50 transition-all duration-300" :disabled="loading" />
                </el-form-item>

                <el-form-item v-if="showCaptcha" class="mb-6">
                    <el-input
                        v-model="loginForm.captcha"
                        :placeholder="t('pageLogin.enterCaptcha')"
                        size="large"
                        class="rounded-xl border-blue-200 hover:border-blue-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-300/50 transition-all duration-300"
                        :disabled="loading"
                    />
                </el-form-item>

                <!-- Remember password -->
                <el-form-item class="mb-6">
                    <el-checkbox v-model="rememberPassword" size="large" class="text-blue-700 hover:text-blue-500 transition-colors duration-300">{{ t('pageLogin.rememberPassword') }}</el-checkbox>
                </el-form-item>

                <!-- Error message -->
                <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="mb-4 rounded-lg"
                    :closable="false" />

                <!-- Submit button -->
                <el-form-item>
                    <el-button type="primary" native-type="submit" size="large" :loading="loading"
                        class="w-full py-3 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 text-white font-medium shadow-lg hover:shadow-blue-500/50 transition-all duration-300 transform hover:scale-[1.02] active:scale-[0.98]">
                        {{ t('pageLogin.loginButton') }}
                    </el-button>
                </el-form-item>
            </el-form>
        </div>

        <!-- Footer -->
        <div class="absolute bottom-4 left-0 right-0 text-center text-xs z-40 drop-shadow-md animate-pulse">
            {{ t('pageLogin.studioFooter') }}
            <p class="mt-1.5 bg-clip-text text-transparent bg-gradient-to-r from-gray-100 to-gray-900 drop-shadow-sm">{{ t('pageLogin.copyright') }}</p>
        </div>
    </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { UserFilled, Lock } from '@element-plus/icons-vue'
import api from '@/script/api'
import { useStore } from '@/script/store'
import { getErrorMessageByCode } from '@/script/error-codes'
import { localeRef, localeOptions, setLocale } from '@/script/i18n'
import { t } from '@/script/i18n-text'

const store = useStore()
// Router
const router = useRouter()

// Form data
const loginForm = ref({
    username: '',
    password: '',
    captcha: ''
})
const showCaptcha = ref(false)
const captchaId = ref('')

// Remember password
const rememberPassword = ref(false)

// State
const loading = ref(false)
const errorMessage = ref('')

const handleLocaleChange = (value) => {
    setLocale(value)
}

// Error fallback mapping
const getErrorMessages = () => ({
    'invalid': t('auth.invalidCredentials'),
    'Invalid': t('auth.invalidCredentials'),
    'credentials': t('auth.invalidCredentials'),
    'password': t('auth.invalidCredentials'),
    'username': t('auth.invalidCredentials'),
    'parameter': t('auth.invalidCredentials'),
    'network': t('error.network'),
    'Network': t('error.network'),
    'connection': t('error.network'),
    'Connection': t('error.network'),
    'timeout': t('error.network'),
    'Timeout': t('error.network'),
    'server': t('error.serverRetry'),
    'Server': t('error.serverRetry'),
    '500': t('error.serverRetry'),
    '502': t('error.serverRetry'),
    '503': t('error.serverRetry'),
    '504': t('error.serverRetry')
})


// Load saved login info
const loadSavedLogin = () => {
    try {
        const savedLogin = localStorage.getItem('login_credentials')
        if (savedLogin) {
            const { username, remember } = JSON.parse(savedLogin)
            loginForm.value.username = username
            rememberPassword.value = remember
        }
    } catch (error) {
        console.error('failed to load saved login info:', error)
    }
}

// Save login info
const saveLogin = (username, remember) => {
    try {
        if (remember) {
            localStorage.setItem('login_credentials', JSON.stringify({
                username,
                remember
            }))
        } else {
            localStorage.removeItem('login_credentials')
        }
    } catch (error) {
        console.error('failed to save login info:', error)
    }
}

// Init
loadSavedLogin()

const loadCaptcha = async () => {
    try {
        const resp = await api.getCaptcha()
        const data = resp?.data?.data || {}
        if (data.enabled) {
            showCaptcha.value = true
            captchaId.value = String(data.captcha_id || '')
        } else {
            showCaptcha.value = false
            captchaId.value = ''
        }
    } catch (error) {
        console.error('failed to load captcha:', error)
    }
}


// Login action
const handleLogin = async () => {
    try {
        // Reset error message
        errorMessage.value = ''
        // Set loading state
        loading.value = true

        // Validate form
        if (!loginForm.value.username || !loginForm.value.password) {
            errorMessage.value = t('pageLogin.enterAccountAndPassword')
            return
        }

        // Call login API
        const response = await api.login(
            loginForm.value.username,
            loginForm.value.password,
            showCaptcha.value ? loginForm.value.captcha : '',
            showCaptcha.value ? captchaId.value : ''
        )

        // Save login info
        saveLogin(loginForm.value.username, rememberPassword.value)

        store.setUser(response?.data?.data?.token, {
            id: response?.data?.data?.user?.ID,
            username: response?.data?.data?.user?.username,
            name: response?.data?.data?.user?.username,
            role: response?.data?.data?.user?.role,
            avatar: response?.data?.data?.user?.avatar,
        })

        // Navigate after successful login
        router.push('/home')
    } catch (error) {
        console.error('login failed:', error)
        const mappedByCode = getErrorMessageByCode(error?.code, '')
        if (Number(error?.code) === 13005 || Number(error?.code) === 13006) {
            showCaptcha.value = true
            await loadCaptcha()
        }
        if (mappedByCode) {
            errorMessage.value = mappedByCode
            return
        }
        const errorMsg = error.message || ''
        const errorMessages = getErrorMessages()
        const matchedKey = Object.keys(errorMessages).find(key => errorMsg.includes(key))
        errorMessage.value = matchedKey ? errorMessages[matchedKey] : t('auth.loginFailedCheck')
    } finally {
        // Clear loading state
        loading.value = false
    }
}

loadCaptcha()
</script>

<style scoped>
/* Custom styles */
@media (max-width: 768px) {
    .absolute.left-0.top-0.w-3\/5 {
        width: 100%;
        height: 400px;
    }

    .absolute.right-0.top-0.w-1\/2 {
        width: 100%;
        top: 400px;
        height: calc(100% - 400px);
        transform: none;
    }

.login-card {
        right: 5%;
        left: 5%;
        top: 300px;
        transform: none;
        width: 90%;
        max-width: 400px;
    }
}

.login-locale {
    width: 108px;
}

/* Visual effects */
@keyframes pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.8;
    }
}

@keyframes glow {
    0%, 100% {
        filter: drop-shadow(0 0 8px rgba(59, 130, 246, 0.5));
    }
    50% {
        filter: drop-shadow(0 0 16px rgba(59, 130, 246, 0.8));
    }
}

@keyframes float {
    0%, 100% {
        transform: translateY(0) rotate(0deg);
    }
    50% {
        transform: translateY(-10px) rotate(2deg);
    }
}

@keyframes border-glow {
    0%, 100% {
        border-color: rgba(59, 130, 246, 0.5);
        box-shadow: 0 0 8px rgba(59, 130, 246, 0.3);
    }
    50% {
        border-color: rgba(59, 130, 246, 0.8);
        box-shadow: 0 0 16px rgba(59, 130, 246, 0.6);
    }
}

.animate-pulse {
    animation: pulse 3s ease-in-out infinite;
}

.animate-glow {
    animation: glow 2s ease-in-out infinite;
}

.animate-float {
    animation: float 6s ease-in-out infinite;
}

.animate-border-glow {
    animation: border-glow 2s ease-in-out infinite;
}

/* Element Plus overrides */
:deep(.el-input__wrapper) {
    border-radius: 0.5rem;
}

:deep(.el-button--primary) {
    border-radius: 0.5rem;
    font-weight: 500;
}

:deep(.el-form-item) {
    margin-bottom: 1rem;
}

:deep(.el-alert) {
    border-radius: 0.5rem;
}
</style>
