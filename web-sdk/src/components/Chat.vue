<template>
  <div class="kefu-chat-page">
    <header class="kefu-chat-header">
      <div class="kefu-chat-header-left">
        <img :src="chatLogo" alt="logo" class="kefu-chat-logo" />
        <div class="kefu-chat-meta">
          <h1 class="kefu-chat-title">{{ chatTitle }}</h1>
          <p class="kefu-chat-subtitle">独立聊天页</p>
        </div>
      </div>
    </header>

    <main class="kefu-chat-main">
      <TdChatCore
        :app-id="appId"
        :user-id="userId"
        :api-base-url="apiBaseUrl"
        :ws-url="wsUrl"
        :service-name="chatTitle"
        :service-avatar="chatLogo"
        @config-loaded="handleConfigLoaded"
      />
    </main>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import TdChatCore from "./TdChatCore.vue";
import { getChatInitOptionsFromQuery } from "../script/chat-runtime.js";

const initOptions = getChatInitOptionsFromQuery("default");
const appId = ref(initOptions.appId || "default");
const userId = ref(initOptions.userId || "");
const apiBaseUrl = ref(initOptions.apiBaseUrl || "");
const wsUrl = ref(initOptions.wsUrl || "");
const configName = ref("");
const configLogo = ref("");

const chatTitle = computed(() => configName.value || "零点客服");
const chatLogo = computed(() => configLogo.value || "./logo.png");

function handleConfigLoaded(config) {
  configName.value = config?.name || "";
  configLogo.value = config?.logo || "";
}
</script>

<style scoped>
.kefu-chat-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, #dbeafe 0%, #eff6ff 20%, #f8fafc 100%);
}

.kefu-chat-header {
  padding: 12px 16px;
  background: linear-gradient(90deg, #2563eb 0%, #1e40af 100%);
  color: #ffffff;
  box-shadow: 0 6px 14px rgba(30, 64, 175, 0.2);
}

.kefu-chat-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.kefu-chat-logo {
  width: 40px;
  height: 40px;
  object-fit: contain;
  border-radius: 9999px;
  background: #ffffff;
}

.kefu-chat-meta {
  display: flex;
  flex-direction: column;
}

.kefu-chat-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
}

.kefu-chat-subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  opacity: 0.92;
}

.kefu-chat-main {
  flex: 1;
  min-height: 0;
}
</style>
