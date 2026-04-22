<template>
  <div class="settings-page p-8" v-loading="loading">
    <div class="settings-head">
      <div>
        <h1 class="settings-title">{{ t('pageSettings.title') }}</h1>
        <p class="settings-subtitle">{{ t('pageSettings.subtitle') }}</p>
      </div>
      <el-button type="primary" :loading="saving" @click="save">{{ t('pageSettings.saveAll') }}</el-button>
    </div>

    <el-card class="settings-card" shadow="never">
      <el-tabs v-model="activeTab" class="settings-tabs">
        <el-tab-pane :label="t('pageSettings.tabBasic')" name="basic">
          <el-form label-width="120px" class="settings-form">
            <el-form-item :label="t('pageSettings.systemName')">
              <el-input v-model="settings.systemName" :placeholder="t('pageSettings.systemNameInput')" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.systemLogo')">
              <el-input v-model="settings.logo" :placeholder="t('pageSettings.systemLogoInput')" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.welcomeMsg')">
              <el-input v-model="settings.welcomeMsg" type="textarea" :rows="4" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('pageSettings.tabSession')" name="session">
          <el-form label-width="140px" class="settings-form">
            <el-form-item :label="t('pageSettings.maxSessions')">
              <el-input-number v-model="settings.maxSessions" :min="1" :max="200" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.timeoutSec')">
              <el-input-number v-model="settings.timeout" :min="30" :max="3600" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.autoAssign')">
              <el-switch v-model="settings.autoAssign" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('pageSettings.tabNotify')" name="notify">
          <el-form label-width="140px" class="settings-form">
            <el-form-item :label="t('pageSettings.emailNotify')">
              <el-switch v-model="settings.emailNotify" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.recipients')">
              <el-input v-model="settings.notifyEmail" placeholder="ops@example.com, alert@example.com" />
              <span class="form-tip">{{ t('pageSettings.recipientsTip') }}</span>
            </el-form-item>
            <el-form-item :label="t('pageSettings.smtpHost')">
              <el-input v-model="settings.smtpHost" placeholder="smtp.example.com" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.smtpPort')">
              <el-input-number v-model="settings.smtpPort" :min="1" :max="65535" />
              <span class="form-tip">{{ t('pageSettings.smtpPortTip') }}</span>
            </el-form-item>
            <el-form-item :label="t('pageSettings.smtpUser')">
              <el-input v-model="settings.smtpUser" placeholder="noreply@example.com" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.smtpPassword')">
              <el-input
                v-model="settings.smtpPassword"
                type="password"
                show-password
                :placeholder="t('pageSettings.smtpPasswordInput')"
              />
            </el-form-item>
            <el-form-item :label="t('pageSettings.smtpFrom')">
              <el-input v-model="settings.smtpFrom" placeholder="Support <noreply@example.com>" />
            </el-form-item>
            <el-alert type="info" show-icon :closable="false" :title="t('pageSettings.notifyRuleMail')" />
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('pageSettings.tabSecurity')" name="security">
          <el-form label-width="140px" class="settings-form">
            <el-form-item :label="t('pageSettings.sessionEncrypt')">
              <el-switch v-model="settings.sessionEncrypt" />
              <span class="form-tip">{{ t('pageSettings.sessionEncryptTip') }}</span>
            </el-form-item>
            <el-form-item :label="t('pageSettings.ipLimit')">
              <el-switch v-model="settings.ipLimit" />
              <span class="form-tip">{{ t('pageSettings.ipLimitTip') }}</span>
            </el-form-item>
            <el-form-item :label="t('pageSettings.captcha')">
              <el-switch v-model="settings.captcha" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.sensitiveWords')">
              <el-input v-model="settings.sensitiveWords" type="textarea" :rows="4" :placeholder="t('pageSettings.sensitiveWordsInput')" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.rateLimit')">
              <el-switch v-model="settings.rateLimitEnabled" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.rpm')">
              <el-input-number v-model="settings.rateLimitRpm" :min="30" :max="5000" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.burst')">
              <el-input-number v-model="settings.rateLimitBurst" :min="10" :max="1000" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.offlineWebhook')">
              <el-input v-model="settings.offlineNotifyUrl" placeholder="https://example.com/webhook" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('pageSettings.tabWecom')" name="wecom">
          <el-form label-width="140px" class="settings-form">
            <el-form-item :label="t('pageSettings.corpId')">
              <el-input v-model="wecomConfig.corpId" :placeholder="t('pageSettings.corpIdInput')" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.agentId')">
              <el-input-number v-model="wecomConfig.agentId" :min="1" :max="999999" style="width: 200px" />
              <span class="form-tip">{{ t('pageSettings.agentIdTip') }}</span>
            </el-form-item>
            <el-form-item :label="t('pageSettings.secret')">
              <el-input v-model="wecomConfig.secret" type="password" show-password :placeholder="t('pageSettings.secretInput')" />
            </el-form-item>
            <el-form-item :label="t('pageSettings.callbackDomain')">
              <el-input v-model="wecomConfig.callbackUrl" disabled />
            </el-form-item>
            <el-alert type="info" show-icon :closable="false" class="mb-4!">
              <template #title>{{ t('pageSettings.notifyRuleWecom') }}</template>
            </el-alert>
            <el-alert type="info" show-icon :closable="false" class="mb-4!">
              <template #title>{{ t('pageSettings.trustedDomainTip') }}</template>
            </el-alert>
            <el-form-item>
              <el-button type="primary" :loading="testingWecom" @click="testWecomConnection">{{ t('pageSettings.testConnection') }}</el-button>
              <el-button @click="resetWecomConfig">{{ t('pageSettings.reset') }}</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('pageSettings.tabLocale')" name="locale">
          <el-form label-width="140px" class="settings-form">
            <el-form-item :label="t('pageSettings.localeLabel')">
              <el-select v-model="localeRef" @change="handleLocaleChange" style="width: 240px">
                <el-option
                  v-for="opt in localeOptions"
                  :key="opt.value"
                  :label="opt.label"
                  :value="opt.value"
                />
              </el-select>
            </el-form-item>
            <el-alert type="info" show-icon :closable="false" :title="t('pageSettings.localeTip')" />
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";
import { localeRef, localeOptions, setLocale } from "@/script/i18n";

const route = useRoute();
const loading = ref(false);
const saving = ref(false);
const testingWecom = ref(false);
const activeTab = ref("basic");
const settings = ref({
  systemName: "Zero Support",
  logo: "",
  welcomeMsg: "Hello, how can I help you?",
  maxSessions: 5,
  timeout: 180,
  autoAssign: true,
  emailNotify: false,
  notifyEmail: "",
  smtpHost: "",
  smtpPort: 25,
  smtpUser: "",
  smtpPassword: "",
  smtpFrom: "kefu@localhost",
  sessionEncrypt: true,
  ipLimit: false,
  captcha: false,
  sensitiveWords: "",
  rateLimitEnabled: true,
  rateLimitRpm: 120,
  rateLimitBurst: 60,
  offlineNotifyUrl: "",
});

const wecomConfig = ref({
  corpId: "",
  agentId: 0,
  secret: "",
  callbackUrl: "",
});

const wecomConfigBackup = ref({});

function handleLocaleChange(value) {
  setLocale(value);
}

function fmt(key, vars = {}) {
  let out = t(key);
  for (const [k, v] of Object.entries(vars)) out = out.replace(`{${k}}`, String(v));
  return out;
}

async function loadSettings() {
  loading.value = true;
  try {
    const resp = await api.getSystemSettings();
    settings.value = { ...settings.value, ...(resp?.data?.data || {}) };

    const wecomResp = await api.getWecomConfig();
    const wecomData = wecomResp?.data?.data || {};
    wecomConfig.value = {
      corpId: wecomData.corpId || "",
      agentId: wecomData.agentId || 0,
      secret: "",
      callbackUrl: wecomData.callbackUrl || "",
    };
    wecomConfigBackup.value = { ...wecomConfig.value };
  } catch (error) {
    ElMessage.error(error.message || t("pageSettings.loadFailed"));
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    await api.updateSystemSettings(settings.value);

    if (wecomConfig.value.corpId || wecomConfig.value.secret) {
      await api.saveWecomConfig({
        corpId: wecomConfig.value.corpId,
        agentId: wecomConfig.value.agentId,
        secret: wecomConfig.value.secret,
      });
    }

    ElMessage.success(t("pageSettings.saveSuccess"));
    await loadSettings();
  } catch (error) {
    ElMessage.error(error.message || t("pageSettings.saveFailed"));
  } finally {
    saving.value = false;
  }
}

async function testWecomConnection() {
  if (!wecomConfig.value.corpId || !wecomConfig.value.secret) {
    ElMessage.warning(t("pageSettings.fillCorpAndSecret"));
    return;
  }

  testingWecom.value = true;
  try {
    const resp = await api.testWecomConnection({
      corpId: wecomConfig.value.corpId,
      agentId: wecomConfig.value.agentId,
      secret: wecomConfig.value.secret,
    });

    const data = resp?.data?.data || {};
    if (data.success) {
      ElMessage.success(fmt("pageSettings.connSuccess", { name: data.name }));
    } else {
      ElMessage.error(fmt("pageSettings.connFailed", { error: data.error }));
    }
  } catch (error) {
    ElMessage.error(error.message || t("pageSettings.testFailed"));
  } finally {
    testingWecom.value = false;
  }
}

function resetWecomConfig() {
  wecomConfig.value = { ...wecomConfigBackup.value };
}

onMounted(() => {
  const tab = String(route.query.tab || "").trim();
  if (["basic", "session", "notify", "security", "wecom", "locale"].includes(tab)) {
    activeTab.value = tab;
  }
  loadSettings();
});
</script>

<style scoped>
.settings-page {
  min-height: 100%;
  background: radial-gradient(circle at 0% 0%, #eef6ff 0%, #f8fbff 30%, #f5f7fa 100%);
}

.settings-head {
  margin-bottom: 16px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.settings-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
}

.settings-subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  color: #64748b;
}

.settings-card {
  border: 1px solid #dbe7ff;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 14px 40px rgba(37, 99, 235, 0.08);
}

.settings-tabs :deep(.el-tabs__header) {
  margin-bottom: 18px;
}

.settings-tabs :deep(.el-tabs__item) {
  height: 42px;
  line-height: 42px;
  font-size: 14px;
  font-weight: 600;
}

.settings-tabs :deep(.el-tabs__active-bar) {
  height: 3px;
  border-radius: 999px;
  background: linear-gradient(90deg, #2563eb 0%, #0ea5e9 100%);
}

.settings-form {
  max-width: 780px;
}

.form-tip {
  margin-left: 8px;
  color: #64748b;
  font-size: 12px;
}
</style>
