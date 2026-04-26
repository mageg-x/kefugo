<template>
  <div class="kefu-widget-root">
    <button
      v-if="configError && !isOpen"
      type="button"
      class="kefu-widget-error-float"
      @click="isOpen = true"
    >
      当前域名未授权，点击查看详情
    </button>

    <button
      v-if="!isOpen"
      type="button"
      class="kefu-widget-trigger"
      :class="{ 'has-error': configError }"
      :title="configError ? configErrorMessage : '在线咨询'"
      @click="isOpen = true"
    >
      <img :src="widgetLogo" alt="logo" class="kefu-widget-trigger-logo" />
      <span class="kefu-widget-trigger-text">在线咨询</span>
      <span v-if="unreadBadge > 0" class="kefu-widget-badge">{{ unreadBadgeText }}</span>
    </button>

    <section v-else class="kefu-widget-panel" :style="panelStyle">
      <header class="kefu-widget-header">
        <div class="kefu-widget-header-left">
          <img :src="widgetLogo" alt="logo" class="kefu-widget-header-logo" />
          <div class="kefu-widget-header-meta">
            <h3 class="kefu-widget-title">{{ widgetName }}</h3>
            <span class="kefu-widget-status" :class="{ offline: !configOnline }">
              {{ configOnline ? "在线" : "离线" }}
            </span>
          </div>
        </div>
        <button type="button" class="kefu-widget-close" @click="isOpen = false">×</button>
      </header>

      <div class="kefu-widget-body">
        <TdChatCore
          :key="chatInstanceKey"
          :app-id="appId"
          :api-base-url="apiBaseUrl"
          :ws-url="wsUrl"
          :user-id="userId"
          :service-name="widgetName"
          :service-avatar="widgetLogo"
          @config-loaded="handleConfigLoaded"
          @config-error="handleConfigError"
          @unread-change="handleUnreadChange"
        />
      </div>

      <div v-if="configError" class="kefu-widget-error">
        <p class="kefu-widget-error-title">当前域名未授权</p>
        <p class="kefu-widget-error-desc">{{ configErrorMessage }}</p>
        <div class="kefu-widget-error-actions">
          <button type="button" class="kefu-widget-error-btn" @click="retryLoad">重试连接</button>
          <a class="kefu-widget-error-link" :href="adminAppsUrl" target="_blank" rel="noreferrer">打开应用设置</a>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import TdChatCore from "./TdChatCore.vue";

const props = defineProps({
  appId: { type: String, default: "default" },
  apiBaseUrl: {
    type: String,
    default: () =>
      typeof window !== "undefined" && window.location?.origin
        ? window.location.origin
        : "http://localhost:5300",
  },
  wsUrl: { type: String, default: "" },
  userId: { type: String, default: "" },
});

const isOpen = ref(false);
const configError = ref(false);
const configErrorMessage = ref("配置加载失败，请检查 appId 或服务地址。");
const configName = ref("");
const configLogo = ref("");
const configOnline = ref(true);
const chatInstanceKey = ref(0);
const panelHeight = ref(640);
const panelWidth = ref(390);
const unreadBadge = ref(0);

const widgetName = computed(() => configName.value || "零点客服");
const unreadBadgeText = computed(() => (unreadBadge.value > 99 ? "99+" : String(unreadBadge.value)));
const adminAppsUrl = computed(() => {
  if (typeof window !== "undefined" && window.location?.origin) {
    return `${window.location.origin}/home/apps`;
  }
  return "/home/apps";
});

function resolveDefaultLogo() {
  const script = document.currentScript || document.querySelector("script[data-kefu-appid]");
  if (script && script.src) {
    const base = script.src.slice(0, script.src.lastIndexOf("/"));
    return `${base}/logo.png`;
  }
  return "./logo.png";
}

const widgetLogo = computed(() => configLogo.value || resolveDefaultLogo());
const panelStyle = computed(() => ({
  width: `${panelWidth.value}px`,
  height: `${panelHeight.value}px`,
}));

function syncPanelSize() {
  if (typeof window === "undefined") {
    return;
  }
  const viewportHeight = window.innerHeight || 800;
  const viewportWidth = window.innerWidth || 1200;

  if (viewportWidth <= 768) {
    panelWidth.value = Math.min(Math.max(Math.floor(viewportWidth - 12), 260), 420);
    panelHeight.value = Math.min(Math.max(Math.floor(viewportHeight - 12), 240), 860);
    return;
  }

  panelWidth.value = Math.min(Math.max(Math.floor(viewportWidth * 0.28), 360), 430);
  panelHeight.value = Math.min(Math.max(Math.floor(viewportHeight - 28), 300), 900);
}

function handleConfigLoaded(config) {
  configError.value = false;
  configErrorMessage.value = "";
  configName.value = config?.name || "";
  configLogo.value = config?.logo || "";
  configOnline.value = Boolean(config?.online);
}

function handleUnreadChange(count) {
  const normalized = Math.max(0, Number(count || 0));
  unreadBadge.value = isOpen.value ? 0 : normalized;
}

function handleConfigError(error) {
  configError.value = true;
  configErrorMessage.value = error?.message || "配置加载失败，请检查 appId 或服务地址。";
  isOpen.value = true;
}

function retryLoad() {
  configError.value = false;
  configErrorMessage.value = "";
  chatInstanceKey.value += 1;
}

onMounted(() => {
  syncPanelSize();
  window.addEventListener("resize", syncPanelSize);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", syncPanelSize);
});
</script>

<style scoped>
.kefu-widget-root {
  position: fixed;
  right: 12px;
  bottom: 12px;
  z-index: 50;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

.kefu-widget-trigger {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  border: none;
  border-radius: 20px;
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  box-shadow: 0 8px 32px rgba(59, 130, 246, 0.2), 0 2px 8px rgba(0, 0, 0, 0.04);
  padding: 10px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
}

.kefu-widget-trigger:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 40px rgba(59, 130, 246, 0.28), 0 4px 12px rgba(0, 0, 0, 0.06);
}

.kefu-widget-trigger:active {
  transform: translateY(0);
}

.kefu-widget-trigger.has-error {
  background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
  box-shadow: 0 8px 32px rgba(220, 38, 38, 0.2);
}

.kefu-widget-trigger-logo {
  width: 48px;
  height: 48px;
  object-fit: contain;
  border-radius: 12px;
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
  padding: 6px;
}

.kefu-widget-trigger-text {
  color: #2563eb;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.kefu-widget-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: #ef4444;
  color: #ffffff;
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
  box-shadow: 0 4px 10px rgba(239, 68, 68, 0.35);
}

.kefu-widget-panel {
  border-radius: 18px;
  border: 1px solid rgba(229, 231, 235, 0.6);
  background: #ffffff;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.18), 0 8px 24px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 24px);
  animation: widget-slide-up 0.3s ease-out;
}

@keyframes widget-slide-up {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.kefu-widget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  color: #ffffff;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 50%, #1d4ed8 100%);
  position: relative;
  overflow: hidden;
}

.kefu-widget-header::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, transparent 50%);
  pointer-events: none;
}

.kefu-widget-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  position: relative;
  z-index: 1;
}

.kefu-widget-header-logo {
  width: 36px;
  height: 36px;
  object-fit: contain;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.95);
  padding: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.kefu-widget-header-meta {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
}

.kefu-widget-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.01em;
}

.kefu-widget-status {
  font-size: 11px;
  opacity: 0.9;
  display: flex;
  align-items: center;
  gap: 4px;
}

.kefu-widget-status::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #4ade80;
  display: inline-block;
}

.kefu-widget-status.offline {
  color: #fecaca;
}

.kefu-widget-status.offline::before {
  background: #fca5a5;
}

.kefu-widget-close {
  border: none;
  background: rgba(255, 255, 255, 0.15);
  color: #ffffff;
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  padding: 4px 10px;
  border-radius: 8px;
  transition: all 0.15s ease;
  position: relative;
  z-index: 1;
}

.kefu-widget-close:hover {
  background: rgba(255, 255, 255, 0.25);
}

.kefu-widget-body {
  flex: 1;
  min-height: 0;
  background: linear-gradient(180deg, #fafbfc 0%, #ffffff 100%);
}

.kefu-widget-error {
  padding: 12px 16px;
  font-size: 12px;
  color: #dc2626;
  border-top: 1px solid #fecaca;
  background: linear-gradient(180deg, #fef2f2 0%, #fee2e2 100%);
}

.kefu-widget-error-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.kefu-widget-error-desc {
  margin: 6px 0 0;
  color: #991b1b;
}

.kefu-widget-error-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
}

.kefu-widget-error-btn {
  border: 1px solid #dc2626;
  background: #ffffff;
  color: #dc2626;
  border-radius: 8px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.kefu-widget-error-btn:hover {
  background: #fef2f2;
  border-color: #b91c1c;
}

.kefu-widget-error-link {
  color: #b91c1c;
  text-decoration: underline;
  text-underline-offset: 2px;
  font-size: 12px;
}

.kefu-widget-error-float {
  margin-bottom: 10px;
  border: none;
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: #ffffff;
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 8px 24px rgba(220, 38, 38, 0.3);
  transition: all 0.2s ease;
}

.kefu-widget-error-float:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 28px rgba(220, 38, 38, 0.35);
}

@media (max-width: 640px) {
  .kefu-widget-root {
    right: 6px;
    bottom: 6px;
  }

  .kefu-widget-panel {
    max-width: calc(100vw - 12px);
    max-height: calc(100vh - 12px);
    border-radius: 14px;
  }

  .kefu-widget-trigger {
    border-radius: 16px;
    padding: 8px;
  }

  .kefu-widget-trigger-logo {
    width: 40px;
    height: 40px;
  }
}
</style>
