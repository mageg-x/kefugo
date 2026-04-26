<template>
  <div class="more-page p-8">
    <div class="more-head">
      <div>
        <h1 class="page-title">{{ t("pageMore.title") }}</h1>
        <p class="page-subtitle">{{ t("pageMore.subtitle") }}</p>
      </div>
      <div class="seat-card" @click.stop>
        <span class="seat-text" :class="isOnline ? 'seat-online' : 'seat-offline'">{{ isOnline ? t("pageMore.status.onSeat") : t("pageMore.status.offSeat") }}</span>
        <el-switch v-model="isOnline" @change="toggleStatus" size="small" />
      </div>
    </div>

    <el-card class="panel-card mb-6" shadow="never">
      <template #header>
        <div class="panel-header">{{ t("pageMore.section.business") }}</div>
      </template>
      <div class="feature-grid">
        <button type="button" class="feature-item" @click="goGreeting">
          <div class="feature-icon"><el-icon><ChatRound /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.greeting") }}</p>
        </button>

        <button type="button" class="feature-item" @click="openAIConfig">
          <div class="feature-icon"><el-icon><Cpu /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.aiBot") }}</p>
        </button>

        <button type="button" class="feature-item" @click="goKnowledge">
          <div class="feature-icon"><el-icon><Collection /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.knowledge") }}</p>
        </button>

        <button type="button" class="feature-item" @click="goSnippets()">
          <div class="feature-icon"><el-icon><Postcard /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.snippets") }}</p>
        </button>

        <button type="button" class="feature-item" @click="openFaqDialog">
          <div class="feature-icon"><el-icon><ChatLineRound /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.faq") }}</p>
        </button>
      </div>
    </el-card>

    <el-card class="panel-card mb-6" shadow="never">
      <template #header>
        <div class="panel-header">{{ t("pageMore.section.system") }}</div>
      </template>
      <div class="feature-grid feature-grid-system">
        <button type="button" class="feature-item" @click="preferencesVisible = true">
          <div class="feature-icon"><el-icon><Setting /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.preferences") }}</p>
        </button>

        <button type="button" class="feature-item" @click="openApiKeyDialog">
          <div class="feature-icon"><el-icon><Key /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.apiKey") }}</p>
        </button>

        <button type="button" class="feature-item" @click="openSensitiveWordsDialog">
          <div class="feature-icon"><el-icon><Warning /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.sensitiveWords") }}</p>
        </button>

        <button type="button" class="feature-item" @click="goStatistics">
          <div class="feature-icon"><el-icon><DataAnalysis /></el-icon></div>
          <p class="feature-title">{{ t("pageMore.feature.statistics") }}</p>
        </button>

        <button type="button" class="feature-item" @click="openWecomBindDialog">
          <div class="feature-icon wechat-icon"><img src="@/assets/wechat.png" :alt="t('pageMore.feature.wechatNotify')" /></div>
          <p class="feature-title">{{ t("pageMore.feature.wechatNotify") }}</p>
        </button>
      </div>
    </el-card>

    <el-dialog v-model="preferencesVisible" :title="t('pageMore.dialog.preferences')" width="520px" class="more-dialog">
      <el-form label-width="140px">
        <el-form-item :label="t('pageMore.preferences.sound')"><el-switch v-model="preferences.soundEnabled" /></el-form-item>
        <el-form-item :label="t('pageMore.preferences.desktopNotify')"><el-switch v-model="preferences.desktopNotifyEnabled" /></el-form-item>
        <el-form-item :label="t('pageMore.preferences.typing')"><el-switch v-model="preferences.typingIndicatorEnabled" /></el-form-item>
        <el-form-item :label="t('pageMore.preferences.enterToSend')"><el-switch v-model="preferences.enterToSend" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPreferences">{{ t("pageMore.action.restoreDefault") }}</el-button>
        <el-button type="primary" :loading="savingPreferences" @click="savePreferences">{{ t("action.save") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="aiDialogVisible" :title="t('pageMore.dialog.aiBot')" width="760px" class="more-dialog">
      <el-form label-width="120px">
        <el-form-item :label="t('pageMore.ai.enabled')"><el-switch v-model="aiConfig.enabled" /></el-form-item>
        <el-form-item :label="t('pageMore.ai.botName')"><el-input v-model="aiConfig.botName" :placeholder="t('pageMore.ai.botNamePlaceholder')" /></el-form-item>
        <el-form-item :label="t('pageMore.ai.model')">
          <el-select v-model="aiConfig.model" filterable clearable :placeholder="t('pageMore.ai.modelPlaceholder')">
            <el-option
              v-for="item in aiModelOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageMore.ai.triggerScope')">
          <el-select v-model="aiConfig.whenAssigned">
            <el-option :value="false" :label="t('pageMore.ai.unassignedOnly')" />
            <el-option :value="true" :label="t('pageMore.ai.allSessions')" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageMore.ai.topK')">
          <el-input-number v-model="aiConfig.topK" :min="1" :max="20" />
        </el-form-item>
        <el-form-item :label="t('pageMore.ai.style')">
          <el-select v-model="aiConfig.style">
            <el-option :label="t('pageMore.ai.styleProfessional')" value="professional" />
            <el-option :label="t('pageMore.ai.styleFriendly')" value="friendly" />
            <el-option :label="t('pageMore.ai.styleFormal')" value="formal" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageMore.ai.prompt')">
          <el-input v-model="aiConfig.prompt" type="textarea" :rows="4" :placeholder="t('pageMore.ai.promptPlaceholder')" />
        </el-form-item>
      </el-form>

      <div class="test-wrap">
        <div class="test-head">{{ t("pageMore.ai.testTitle") }}</div>
        <div class="test-row">
          <el-select v-model="aiTestAppID" filterable :placeholder="t('pageMore.placeholder.selectApp')" style="width: 220px">
            <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
          </el-select>
          <el-input v-model="aiTestQuestion" :placeholder="t('pageMore.ai.testQuestionPlaceholder')" clearable />
          <el-button type="primary" :loading="aiTesting" @click="runAITest">{{ t("action.test") }}</el-button>
        </div>
        <el-input v-model="aiTestAnswer" type="textarea" :rows="8" readonly :placeholder="t('pageMore.ai.testAnswerPlaceholder')" />
      </div>

      <template #footer>
        <el-button @click="resetAiConfig">{{ t("pageMore.action.restoreDefault") }}</el-button>
        <el-button type="primary" :loading="savingAIConfig" @click="saveAiConfig">{{ t("action.save") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="knowledgeDialogVisible" :title="t('pageMore.dialog.knowledge')" width="980px" top="6vh" class="more-dialog more-dialog-wide">
      <div class="workspace-grid">
        <aside class="workspace-side">
          <div class="workspace-side-title">{{ t("pageMore.word.filterAndAction") }}</div>
          <div class="workspace-stack">
            <el-select v-model="knowledgeAppID" filterable :placeholder="t('pageMore.placeholder.selectApp')" @change="loadKnowledge">
              <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
            </el-select>
            <el-input v-model="knowledgeKeyword" :placeholder="t('pageMore.placeholder.searchKnowledge')" @keyup.enter="loadKnowledge" />
            <div class="workspace-actions">
              <el-button @click="loadKnowledge">{{ t("action.search") }}</el-button>
              <el-button type="primary" @click="openKnowledgeEditor()">{{ t("action.create") }}</el-button>
            </div>
            <el-upload
              :auto-upload="false"
              :show-file-list="false"
              :on-change="onKnowledgeFileSelected"
              accept=".txt,.md,.csv,.json,.log,.html,.htm"
            >
              <template #trigger>
                <el-button type="primary" plain>{{ t("pageMore.btn.uploadDocToKnowledge") }}</el-button>
              </template>
            </el-upload>
            <p class="tip-text">{{ t("pageMore.tip.uploadTypes") }}</p>
          </div>
          <div class="workspace-stats">
            <div class="workspace-stat-item">
              <span>{{ t("pageMore.word.knowledgeItems") }}</span>
              <strong>{{ knowledgeRows.length }}</strong>
            </div>
            <div class="workspace-stat-item">
              <span>{{ t("status.enabled") }}</span>
              <strong>{{ knowledgeEnabledCount }}</strong>
            </div>
          </div>
        </aside>
        <section class="workspace-main">
          <el-table :data="knowledgeRows" v-loading="knowledgeLoading" border class="more-table">
            <el-table-column prop="title" :label="t('pageMore.table.title')" min-width="220" />
            <el-table-column prop="tags" :label="t('pageMore.table.tags')" width="120" show-overflow-tooltip />
            <el-table-column prop="source_type" :label="t('pageMore.table.source')" width="110" />
            <el-table-column prop="source_name" :label="t('pageMore.table.sourceFile')" min-width="160" show-overflow-tooltip />
            <el-table-column prop="enabled" :label="t('status.label')" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? t("status.enabled") : t("status.disabled") }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('action.actions')" width="180" align="center">
              <template #default="{ row }">
                <el-button type="primary" link @click="openKnowledgeEditor(row)">{{ t("action.edit") }}</el-button>
                <el-button type="danger" link @click="removeKnowledge(row)">{{ t("action.delete") }}</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="test-wrap">
            <div class="test-head">{{ t("pageMore.word.ragTest") }}</div>
            <div class="dialog-toolbar">
              <el-input v-model="ragQuery" :placeholder="t('pageMore.placeholder.ragQuestion')" @keyup.enter="runRAGTest" />
              <el-button type="primary" :loading="ragTesting" @click="runRAGTest">{{ t("pageMore.btn.runRetrieve") }}</el-button>
            </div>
            <el-table :data="ragRows" size="small" border class="more-table">
              <el-table-column prop="score" :label="t('pageMore.table.score')" width="80" />
              <el-table-column prop="title" :label="t('pageMore.table.hitDoc')" min-width="160" show-overflow-tooltip />
              <el-table-column prop="source_type" :label="t('pageMore.table.source')" width="90" />
              <el-table-column prop="content" :label="t('pageMore.table.hitSegment')" min-width="320" show-overflow-tooltip />
            </el-table>
          </div>
        </section>
      </div>
    </el-dialog>

    <el-dialog v-model="knowledgeEditVisible" :title="knowledgeForm.id ? t('pageMore.dialog.editKnowledge') : t('pageMore.dialog.newKnowledge')" width="620px" class="more-dialog">
      <el-form label-width="92px">
        <el-form-item :label="t('app.label')">
          <el-select v-model="knowledgeForm.app_id" filterable style="width: 100%">
            <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageMore.table.title')"><el-input v-model="knowledgeForm.title" maxlength="255" show-word-limit /></el-form-item>
        <el-form-item :label="t('pageMore.table.tags')"><el-input v-model="knowledgeForm.tags" :placeholder="t('pageMore.placeholder.commaSeparated')" /></el-form-item>
        <el-form-item :label="t('pageMore.table.content')"><el-input v-model="knowledgeForm.content" type="textarea" :rows="6" maxlength="5000" show-word-limit /></el-form-item>
        <el-form-item :label="t('status.enabled')"><el-switch v-model="knowledgeForm.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="knowledgeEditVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :loading="savingKnowledge" @click="saveKnowledge">{{ t("action.save") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="faqDialogVisible" :title="t('pageMore.dialog.faq')" width="920px" top="6vh" class="more-dialog more-dialog-wide">
      <div class="workspace-grid workspace-grid-compact">
        <aside class="workspace-side">
          <div class="workspace-side-title">{{ t("pageMore.word.filterAndAction") }}</div>
          <div class="workspace-stack">
            <el-select v-model="faqAppID" filterable :placeholder="t('pageMore.placeholder.selectApp')" @change="loadFaq">
              <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
            </el-select>
            <el-input v-model="faqCategory" :placeholder="t('pageMore.placeholder.filterCategory')" @keyup.enter="loadFaq" />
            <div class="workspace-actions">
              <el-button @click="loadFaq">{{ t("action.search") }}</el-button>
              <el-button type="primary" @click="openFaqEditor()">{{ t("action.create") }}</el-button>
            </div>
          </div>
          <div class="workspace-stats">
            <div class="workspace-stat-item">
              <span>{{ t("pageMore.word.faqItems") }}</span>
              <strong>{{ faqRows.length }}</strong>
            </div>
            <div class="workspace-stat-item">
              <span>{{ t("status.enabled") }}</span>
              <strong>{{ faqEnabledCount }}</strong>
            </div>
          </div>
        </aside>
        <section class="workspace-main">
          <el-table :data="faqRows" v-loading="faqLoading" border class="more-table">
            <el-table-column prop="question" :label="t('pageMore.table.question')" min-width="160" show-overflow-tooltip />
            <el-table-column prop="category" :label="t('pageMore.table.category')" width="140" />
            <el-table-column prop="enabled" :label="t('status.label')" width="60" align="center">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? t("status.enabled") : t("status.disabled") }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('action.actions')" width="180" align="center">
              <template #default="{ row }">
                <el-button type="primary" link @click="openFaqEditor(row)">{{ t("action.edit") }}</el-button>
                <el-button type="danger" link @click="removeFaq(row)">{{ t("action.delete") }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </section>
      </div>
    </el-dialog>

    <el-dialog v-model="faqEditVisible" :title="faqForm.id ? t('pageMore.dialog.editFaq') : t('pageMore.dialog.newFaq')" width="620px" class="more-dialog">
      <el-form label-width="92px">
        <el-form-item :label="t('app.label')">
          <el-select v-model="faqForm.app_id" filterable style="width: 100%">
            <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageMore.table.category')"><el-input v-model="faqForm.category" maxlength="64" /></el-form-item>
        <el-form-item :label="t('pageMore.table.question')"><el-input v-model="faqForm.question" maxlength="255" show-word-limit /></el-form-item>
        <el-form-item :label="t('pageMore.table.answer')"><el-input v-model="faqForm.answer" type="textarea" :rows="6" maxlength="5000" show-word-limit /></el-form-item>
        <el-form-item :label="t('status.enabled')"><el-switch v-model="faqForm.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="faqEditVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :loading="savingFaq" @click="saveFaq">{{ t("action.save") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="apiKeyDialogVisible" :title="t('pageMore.dialog.apiKey')" width="920px" top="6vh" class="more-dialog more-dialog-wide">
      <div class="workspace-grid workspace-grid-compact">
        <aside class="workspace-side">
          <div class="workspace-side-title">{{ t("pageMore.word.filterAndAction") }}</div>
          <div class="workspace-stack">
            <el-select v-model="apiKeyAppID" filterable :placeholder="t('pageMore.placeholder.selectApp')" @change="loadApiKeys">
              <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
            </el-select>
            <div class="workspace-actions workspace-actions-single">
              <el-button type="primary" @click="openApiKeyCreate">{{ t("pageMore.btn.newApiKey") }}</el-button>
            </div>
          </div>
          <div class="workspace-stats">
            <div class="workspace-stat-item">
              <span>{{ t("pageMore.word.totalKeys") }}</span>
              <strong>{{ apiKeyRows.length }}</strong>
            </div>
            <div class="workspace-stat-item">
              <span>{{ t("status.enabled") }}</span>
              <strong>{{ apiKeyEnabledCount }}</strong>
            </div>
          </div>
        </aside>
        <section class="workspace-main">
          <el-table :data="apiKeyRows" v-loading="apiKeyLoading" border class="more-table">
            <el-table-column prop="name" :label="t('name')" min-width="180" />
            <el-table-column prop="key_id" label="Key ID" min-width="220" />
            <el-table-column prop="secret_masked" label="Secret" min-width="180" />
            <el-table-column prop="enabled" :label="t('status.label')" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? t("status.enabled") : t("status.disabled") }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('action.actions')" width="220" align="center">
              <template #default="{ row }">
                <el-button type="primary" link @click="rotateApiKey(row)">{{ t("pageMore.action.rotate") }}</el-button>
                <el-button type="warning" link @click="toggleApiKey(row)">{{ row.enabled ? t("status.disabled") : t("status.enabled") }}</el-button>
                <el-button type="danger" link @click="removeApiKey(row)">{{ t("action.delete") }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </section>
      </div>
    </el-dialog>

    <el-dialog v-model="apiKeyCreateVisible" :title="t('pageMore.dialog.newApiKey')" width="520px" class="more-dialog">
      <el-form label-width="92px">
        <el-form-item :label="t('app.label')">
          <el-select v-model="apiKeyCreateForm.app_id" filterable style="width: 100%">
            <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('name')"><el-input v-model="apiKeyCreateForm.name" maxlength="128" show-word-limit /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="apiKeyCreateVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :loading="savingApiKey" @click="saveApiKey">{{ t("action.create") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="apiKeySecretVisible" :title="t('pageMore.dialog.newSecret')" width="600px" class="more-dialog">
      <p class="text-gray-600 mb-2">{{ t("pageMore.tip.copySecret") }}</p>
      <el-input v-model="latestSecretValue" readonly />
      <template #footer>
        <el-button @click="copySecret">{{ t("pageMore.action.copySecret") }}</el-button>
        <el-button type="primary" @click="apiKeySecretVisible = false">{{ t("action.close") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="statsDialogVisible" :title="t('pageMore.dialog.myStats')" width="520px" class="more-dialog">
      <div v-loading="statsLoading" class="stats-grid">
        <div class="stats-item">
          <div class="stats-label">{{ t("pageMore.stats.todaySessions") }}</div>
          <div class="stats-value">{{ statsData.sessions_today }}</div>
        </div>
        <div class="stats-item">
          <div class="stats-label">{{ t("pageMore.stats.totalAssigned") }}</div>
          <div class="stats-value">{{ statsData.total_assigned }}</div>
        </div>
        <div class="stats-item">
          <div class="stats-label">{{ t("pageMore.stats.avgRating") }}</div>
          <div class="stats-value">{{ statsData.rating.toFixed(2) }}</div>
        </div>
        <div class="stats-item">
          <div class="stats-label">{{ t("pageMore.stats.ratingCount") }}</div>
          <div class="stats-value">{{ statsData.rated_count }}</div>
        </div>
      </div>
    </el-dialog>

    <el-dialog v-model="sensitiveDialogVisible" :title="t('pageMore.dialog.sensitiveWords')" width="620px" class="more-dialog">
      <el-form label-width="110px">
        <el-form-item :label="t('pageMore.label.sensitiveWordList')">
          <el-input
            v-model="sensitiveWordsText"
            type="textarea"
            :rows="8"
            :placeholder="t('pageMore.placeholder.sensitiveWords')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sensitiveDialogVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :loading="savingSensitiveWords" @click="saveSensitiveWords">{{ t("action.save") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="wecomBindVisible" :title="t('pageMore.dialog.wechatNotify')" width="480px" class="more-dialog">
      <div v-if="!wecomConfigured" class="wecom-not-configured">
        <el-empty :description="t('pageMore.wecom.notConfigured')" />
      </div>
      <div v-else-if="wecomBindInfo.isBound" class="wecom-bound">
        <el-result icon="success" :title="t('pageMore.wecom.bound')" :sub-title="`${t('pageMore.wecom.boundAccount')}: ${wecomBindInfo.userId}`">
          <template #extra>
            <el-button type="danger" :loading="unbindingWecom" @click="handleUnbindWecom">{{ t("pageMore.action.unbind") }}</el-button>
          </template>
        </el-result>
      </div>
      <div v-else class="wecom-qrcode">
        <div id="wx_qrcode_container" class="qrcode-container" ref="qrcodeContainer"></div>
        <p class="qrcode-tip">{{ t("pageMore.wecom.scanTip") }}</p>
        <p class="qrcode-expire">{{ t("pageMore.wecom.expire") }}</p>
        <p class="qrcode-status">{{ t("status.label") }}:{{ wecomBindStatus }}</p>
      </div>
      <template #footer>
        <el-button @click="wecomBindVisible = false">{{ t("action.close") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import {
  ChatRound,
  ChatLineRound,
  Collection,
  Cpu,
  DataAnalysis,
  Key,
  Postcard,
  Service,
  Setting,
  Warning,
} from "@element-plus/icons-vue";
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";

const router = useRouter();
const isOnline = ref(true);
const preferencesVisible = ref(false);
const aiDialogVisible = ref(false);
const savingPreferences = ref(false);
const savingAIConfig = ref(false);

const PREFERENCES_STORAGE_KEY = "kefu_agent_preferences";
const defaultPreferences = () => ({
  soundEnabled: true,
  desktopNotifyEnabled: false,
  typingIndicatorEnabled: true,
  enterToSend: true,
});

const defaultAiConfig = () => ({
  enabled: false,
  botName: "AI Bot",
  model: "",
  whenAssigned: false,
  topK: 5,
  style: "professional",
  prompt: "You are a customer support assistant. Keep answers accurate, concise, and polite.",
});

const preferences = ref(defaultPreferences());
const aiConfig = ref(defaultAiConfig());
const aiModelOptions = ref([]);
const appOptions = ref([]);

const aiTestAppID = ref("");
const aiTestQuestion = ref("");
const aiTestAnswer = ref("");
const aiTesting = ref(false);
const statsDialogVisible = ref(false);
const statsLoading = ref(false);
const statsData = ref({
  sessions_today: 0,
  total_assigned: 0,
  rating: 0,
  rated_count: 0,
});
const sensitiveDialogVisible = ref(false);
const sensitiveWordsText = ref("");
const savingSensitiveWords = ref(false);

const wecomBindVisible = ref(false);
const wecomConfigured = ref(false);
const wecomBindInfo = ref({ isBound: false, userId: "" });
const wecomBindStatus = ref("");
const wecomBindState = ref("");
const unbindingWecom = ref(false);
const qrcodeContainer = ref(null);
let wecomPollingTimer = null;
let wecomPollingExpireTimer = null;

const knowledgeDialogVisible = ref(false);
const knowledgeLoading = ref(false);
const knowledgeRows = ref([]);
const knowledgeKeyword = ref("");
const knowledgeAppID = ref("");
const knowledgeEditVisible = ref(false);
const savingKnowledge = ref(false);
const knowledgeForm = ref({
  id: 0,
  app_id: "",
  title: "",
  tags: "",
  content: "",
  enabled: true,
});

const ragQuery = ref("");
const ragRows = ref([]);
const ragTesting = ref(false);

const faqDialogVisible = ref(false);
const faqLoading = ref(false);
const faqRows = ref([]);
const faqCategory = ref("");
const faqAppID = ref("");
const faqEditVisible = ref(false);
const savingFaq = ref(false);
const faqForm = ref({
  id: 0,
  app_id: "",
  question: "",
  answer: "",
  category: "",
  enabled: true,
});

const apiKeyDialogVisible = ref(false);
const apiKeyLoading = ref(false);
const apiKeyRows = ref([]);
const apiKeyAppID = ref("");
const apiKeyCreateVisible = ref(false);
const savingApiKey = ref(false);
const apiKeyCreateForm = ref({
  app_id: "",
  name: "",
});
const apiKeySecretVisible = ref(false);
const latestSecretValue = ref("");

const knowledgeEnabledCount = computed(() => knowledgeRows.value.filter((item) => item?.enabled !== false).length);
const faqEnabledCount = computed(() => faqRows.value.filter((item) => item?.enabled !== false).length);
const apiKeyEnabledCount = computed(() => apiKeyRows.value.filter((item) => item?.enabled).length);

function loadLocalPreferences() {
  try {
    const raw = localStorage.getItem(PREFERENCES_STORAGE_KEY);
    if (!raw) {
      preferences.value = defaultPreferences();
      return;
    }
    preferences.value = {
      ...defaultPreferences(),
      ...JSON.parse(raw),
    };
  } catch {
    preferences.value = defaultPreferences();
  }
}

async function savePreferences() {
  savingPreferences.value = true;
  try {
    const payload = {
      ...preferences.value,
    };
    await api.updateAgentSettings(payload);
    localStorage.setItem(PREFERENCES_STORAGE_KEY, JSON.stringify(preferences.value));
    preferencesVisible.value = false;
    ElMessage.success(t("pageMore.toast.preferencesSaved"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.saveFailed"));
  } finally {
    savingPreferences.value = false;
  }
}

function resetPreferences() {
  preferences.value = defaultPreferences();
}

async function saveAiConfig() {
  savingAIConfig.value = true;
  try {
    const resp = await api.getSystemSettings();
    const settings = { ...(resp?.data?.data || {}) };
    settings.aiBotEnabled = Boolean(aiConfig.value.enabled);
    settings.aiBotName = String(aiConfig.value.botName || "").trim();
    settings.aiBotModel = String(aiConfig.value.model || "").trim();
    settings.aiBotWhenAssigned = Boolean(aiConfig.value.whenAssigned);
    settings.aiBotTopK = Number(aiConfig.value.topK || 5);
    settings.aiBotStyle = String(aiConfig.value.style || "professional");
    settings.aiBotPrompt = String(aiConfig.value.prompt || "").trim();
    await api.updateSystemSettings(settings);
    aiDialogVisible.value = false;
    ElMessage.success(t("pageMore.toast.aiSaved"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.saveFailed"));
  } finally {
    savingAIConfig.value = false;
  }
}

function resetAiConfig() {
  aiConfig.value = defaultAiConfig();
}

async function runAITest() {
  if (!aiTestAppID.value || !aiTestQuestion.value.trim()) {
    ElMessage.warning(t("pageMore.toast.selectAppAndQuestion"));
    return;
  }
  aiTesting.value = true;
  try {
    const resp = await api.testAIBot(aiTestAppID.value, aiTestQuestion.value.trim());
    aiTestAnswer.value = String(resp?.data?.data?.suggestion || "");
    if (!aiTestAnswer.value) ElMessage.warning(t("pageMore.toast.noSuggestion"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.aiTestFailed"));
  } finally {
    aiTesting.value = false;
  }
}

const fetchStatus = async () => {
  try {
    const response = await api.getUserStatus();
    if (response?.data?.data) {
      const status = response.data.data.status;
      isOnline.value = status === 1;
    }
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadSeatStatusFailed"));
  }
};

const toggleStatus = async (value) => {
  try {
    const status = value ? 1 : 0;
    await api.setUserStatus(status);
    isOnline.value = value;
    ElMessage.success(value ? t("pageMore.toast.onSeat") : t("pageMore.toast.offSeat"));
  } catch (error) {
    isOnline.value = !value;
    ElMessage.error(error.message || t("pageMore.toast.toggleSeatFailed"));
  }
};

function goSnippets(category = "") {
  router.push({
    path: "/home/snippets",
    query: category ? { category } : undefined,
  });
}

function goGreeting() {
  goSnippets("greeting");
}

function goKnowledge() {
  router.push("/home/knowledge");
}

async function openAIConfig() {
  await loadAiModelOptions();
  await loadAiConfigFromSystemSettings();
  aiDialogVisible.value = true;
}

async function loadAiModelOptions() {
  try {
    const resp = await api.listAPIModels();
    const rows = resp?.data?.data?.data || [];
    aiModelOptions.value = rows.map((row) => ({
      value: String(row?.model_name || ""),
      label: `${row?.model_name || ""}${row?.status === 1 ? ` (${t("status.enabled")})` : ""}`,
    })).filter((item) => item.value);
  } catch {
    aiModelOptions.value = [];
  }
}

async function loadAiConfigFromSystemSettings() {
  try {
    const resp = await api.getSystemSettings();
    const data = resp?.data?.data || {};
    aiConfig.value = {
      enabled: Boolean(data.aiBotEnabled),
      botName: String(data.aiBotName || defaultAiConfig().botName),
      model: String(data.aiBotModel || defaultAiConfig().model),
      whenAssigned: Boolean(data.aiBotWhenAssigned),
      topK: Number(data.aiBotTopK || defaultAiConfig().topK),
      style: String(data.aiBotStyle || defaultAiConfig().style),
      prompt: String(data.aiBotPrompt || defaultAiConfig().prompt),
    };
  } catch {
    aiConfig.value = defaultAiConfig();
  }
}

async function loadAppOptions() {
  try {
    const appSet = new Set();
    const byAppID = new Map();
    const resp = await api.listApps({ page: 1, page_size: 500 });
    const rows = resp?.data?.data?.data || [];
    for (const app of rows) {
      const appID = String(app?.app_id || "").trim();
      if (!appID) continue;
      appSet.add(appID);
      byAppID.set(appID, `${app.name} (${appID})`);
    }
    appOptions.value = Array.from(appSet).map((appID) => ({
      label: byAppID.get(appID) || appID,
      value: appID,
    }));
  } catch {
    try {
      const sessionResp = await api.listSessions({ page: 1, page_size: 200 });
      const rows = sessionResp?.data?.data?.data || [];
      const appIDs = Array.from(new Set(rows.map((s) => String(s.app_id || "").trim()).filter(Boolean)));
      appOptions.value = appIDs.map((appID) => ({ label: appID, value: appID }));
    } catch {
      appOptions.value = [];
    }
  }
  if (!knowledgeAppID.value && appOptions.value[0]) knowledgeAppID.value = appOptions.value[0].value;
  if (!faqAppID.value && appOptions.value[0]) faqAppID.value = appOptions.value[0].value;
  if (!apiKeyAppID.value && appOptions.value[0]) apiKeyAppID.value = appOptions.value[0].value;
  if (!aiTestAppID.value && appOptions.value[0]) aiTestAppID.value = appOptions.value[0].value;
}

function toKnowledgeForm(row) {
  return {
    id: row?.id || 0,
    app_id: row?.app_id || knowledgeAppID.value || appOptions.value[0]?.value || "",
    title: row?.title || "",
    tags: row?.tags || "",
    content: row?.content || "",
    enabled: row?.enabled !== false,
  };
}

function toFaqForm(row) {
  return {
    id: row?.id || 0,
    app_id: row?.app_id || faqAppID.value || appOptions.value[0]?.value || "",
    question: row?.question || "",
    answer: row?.answer || "",
    category: row?.category || "",
    enabled: row?.enabled !== false,
  };
}

async function loadKnowledge() {
  if (!knowledgeAppID.value) return;
  knowledgeLoading.value = true;
  try {
    const resp = await api.listKnowledge({
      app_id: knowledgeAppID.value,
      keyword: knowledgeKeyword.value || undefined,
    });
    knowledgeRows.value = resp?.data?.data?.data || [];
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadKnowledgeFailed"));
  } finally {
    knowledgeLoading.value = false;
  }
}

async function runRAGTest() {
  if (!knowledgeAppID.value || !ragQuery.value.trim()) {
    ElMessage.warning(t("pageMore.toast.selectAppAndQuestion"));
    return;
  }
  ragTesting.value = true;
  try {
    const resp = await api.ragTestKnowledge({
      app_id: knowledgeAppID.value,
      query: ragQuery.value.trim(),
      top_k: 8,
    });
    ragRows.value = resp?.data?.data?.results || [];
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.ragTestFailed"));
  } finally {
    ragTesting.value = false;
  }
}

async function onKnowledgeFileSelected(file) {
  if (!knowledgeAppID.value) {
    ElMessage.warning(t("pageMore.toast.selectAppFirst"));
    return;
  }
  const raw = file?.raw;
  if (!raw) return;
  try {
    const uploadResp = await api.uploadFile(knowledgeAppID.value, raw, "file");
    const data = uploadResp?.data?.data || {};
    if (!data.url) {
      ElMessage.error(t("pageMore.toast.uploadFailed"));
      return;
    }
    await api.uploadKnowledge({
      app_id: knowledgeAppID.value,
      name: raw.name || "knowledge",
      url: data.url,
      tags: "",
    });
    await loadKnowledge();
    ElMessage.success(t("pageMore.toast.docAdded"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.docAddFailed"));
  }
}

function openKnowledgeEditor(row = null) {
  knowledgeForm.value = toKnowledgeForm(row);
  knowledgeEditVisible.value = true;
}

async function saveKnowledge() {
  if (!knowledgeForm.value.app_id || !knowledgeForm.value.title.trim() || !knowledgeForm.value.content.trim()) {
    ElMessage.warning(t("pageMore.toast.fillKnowledgeRequired"));
    return;
  }
  savingKnowledge.value = true;
  try {
    const payload = {
      app_id: knowledgeForm.value.app_id,
      title: knowledgeForm.value.title.trim(),
      tags: knowledgeForm.value.tags.trim(),
      content: knowledgeForm.value.content.trim(),
      enabled: knowledgeForm.value.enabled,
    };
    if (knowledgeForm.value.id) {
      await api.updateKnowledge({ id: knowledgeForm.value.id, ...payload });
    } else {
      await api.createKnowledge(payload);
    }
    knowledgeEditVisible.value = false;
    await loadKnowledge();
    ElMessage.success(t("pageMore.toast.saveSuccess"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.saveFailed"));
  } finally {
    savingKnowledge.value = false;
  }
}

async function removeKnowledge(row) {
  try {
    await ElMessageBox.confirm(`${t("pageMore.confirm.deleteKnowledge")}「${row.title}」？`, t("pageMore.confirm.title"), { type: "warning" });
    await api.deleteKnowledge(row.id);
    await loadKnowledge();
    ElMessage.success(t("pageMore.toast.deleteSuccess"));
  } catch (error) {
    if (error !== "cancel") ElMessage.error(error.message || t("pageMore.toast.deleteFailed"));
  }
}

async function loadFaq() {
  if (!faqAppID.value) return;
  faqLoading.value = true;
  try {
    const resp = await api.listFAQ({
      app_id: faqAppID.value,
      category: faqCategory.value || undefined,
    });
    faqRows.value = resp?.data?.data?.data || [];
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadFaqFailed"));
  } finally {
    faqLoading.value = false;
  }
}

async function loadApiKeys() {
  if (!apiKeyAppID.value) return;
  apiKeyLoading.value = true;
  try {
    const resp = await api.listAppAPIKeys(apiKeyAppID.value);
    apiKeyRows.value = resp?.data?.data?.data || [];
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadApiKeysFailed"));
  } finally {
    apiKeyLoading.value = false;
  }
}

async function openFaqDialog() {
  faqDialogVisible.value = true;
  await loadFaq();
}

function openFaqEditor(row = null) {
  faqForm.value = toFaqForm(row);
  faqEditVisible.value = true;
}

async function saveFaq() {
  if (!faqForm.value.app_id || !faqForm.value.question.trim() || !faqForm.value.answer.trim()) {
    ElMessage.warning(t("pageMore.toast.fillFaqRequired"));
    return;
  }
  savingFaq.value = true;
  try {
    const payload = {
      app_id: faqForm.value.app_id,
      question: faqForm.value.question.trim(),
      answer: faqForm.value.answer.trim(),
      category: faqForm.value.category.trim(),
      enabled: faqForm.value.enabled,
    };
    if (faqForm.value.id) {
      await api.updateFAQ({ id: faqForm.value.id, ...payload });
    } else {
      await api.createFAQ(payload);
    }
    faqEditVisible.value = false;
    await loadFaq();
    ElMessage.success(t("pageMore.toast.saveSuccess"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.saveFailed"));
  } finally {
    savingFaq.value = false;
  }
}

async function removeFaq(row) {
  try {
    await ElMessageBox.confirm(`${t("pageMore.confirm.deleteFaq")}「${row.question}」？`, t("pageMore.confirm.title"), { type: "warning" });
    await api.deleteFAQ(row.id);
    await loadFaq();
    ElMessage.success(t("pageMore.toast.deleteSuccess"));
  } catch (error) {
    if (error !== "cancel") ElMessage.error(error.message || t("pageMore.toast.deleteFailed"));
  }
}

async function openApiKeyDialog() {
  apiKeyDialogVisible.value = true;
  await loadApiKeys();
}

function openApiKeyCreate() {
  apiKeyCreateForm.value = {
    app_id: apiKeyAppID.value || appOptions.value[0]?.value || "",
    name: "",
  };
  apiKeyCreateVisible.value = true;
}

async function saveApiKey() {
  if (!apiKeyCreateForm.value.app_id || !apiKeyCreateForm.value.name.trim()) {
    ElMessage.warning(t("pageMore.toast.fillApiKeyRequired"));
    return;
  }
  savingApiKey.value = true;
  try {
    const resp = await api.createAppAPIKey({
      app_id: apiKeyCreateForm.value.app_id,
      name: apiKeyCreateForm.value.name.trim(),
    });
    latestSecretValue.value = resp?.data?.data?.secret || "";
    apiKeyCreateVisible.value = false;
    apiKeySecretVisible.value = true;
    await loadApiKeys();
    ElMessage.success(t("pageMore.toast.createSuccess"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.createFailed"));
  } finally {
    savingApiKey.value = false;
  }
}

async function rotateApiKey(row) {
  try {
    const resp = await api.rotateAppAPIKey(row.id);
    latestSecretValue.value = resp?.data?.data?.secret || "";
    apiKeySecretVisible.value = true;
    await loadApiKeys();
    ElMessage.success(t("pageMore.toast.rotateSuccess"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.rotateFailed"));
  }
}

async function toggleApiKey(row) {
  try {
    await api.setAppAPIKeyEnabled(row.id, !row.enabled);
    await loadApiKeys();
    ElMessage.success(t("pageMore.toast.statusUpdated"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.updateFailed"));
  }
}

async function removeApiKey(row) {
  try {
    await ElMessageBox.confirm(`${t("pageMore.confirm.deleteApiKey")}「${row.name}」？`, t("pageMore.confirm.title"), { type: "warning" });
    await api.deleteAppAPIKey(row.id);
    await loadApiKeys();
    ElMessage.success(t("pageMore.toast.deleteSuccess"));
  } catch (error) {
    if (error !== "cancel") ElMessage.error(error.message || t("pageMore.toast.deleteFailed"));
  }
}

async function copySecret() {
  try {
    await navigator.clipboard.writeText(latestSecretValue.value || "");
    ElMessage.success(t("copy.copied"));
  } catch {
    ElMessage.error(t("copy.failed"));
  }
}

async function openSensitiveWordsDialog() {
  try {
    const resp = await api.getSystemSettings();
    const settings = resp?.data?.data || {};
    sensitiveWordsText.value = String(settings.sensitiveWords || "");
    sensitiveDialogVisible.value = true;
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadSensitiveWordsFailed"));
  }
}

async function saveSensitiveWords() {
  savingSensitiveWords.value = true;
  try {
    const resp = await api.getSystemSettings();
    const settings = { ...(resp?.data?.data || {}) };
    settings.sensitiveWords = String(sensitiveWordsText.value || "").trim();
    await api.updateSystemSettings(settings);
    sensitiveDialogVisible.value = false;
    ElMessage.success(t("pageMore.toast.sensitiveWordsSaved"));
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.saveFailed"));
  } finally {
    savingSensitiveWords.value = false;
  }
}

async function openWecomBindDialog() {
  wecomBindVisible.value = true;
  wecomBindStatus.value = t("pageMore.wecom.waiting");
  // Default to configured mode, switch to not-configured only when backend returns a dedicated not-configured code.
  wecomConfigured.value = true;
  stopWecomPolling();
  
  try {
    const bindInfoResp = await api.getWecomBindInfo();
    wecomBindInfo.value = bindInfoResp?.data?.data || { isBound: false, userId: "" };
    
    if (wecomBindInfo.value.isBound) {
      stopWecomPolling();
      return;
    }

    if (!wecomBindInfo.value.isBound) {
      try {
        const qrcodeResp = await api.getWecomQrcode();
        const qrcodeData = qrcodeResp?.data?.data || {};
        
        if (qrcodeData.corpId && qrcodeData.agentId) {
          wecomConfigured.value = true;
          wecomBindState.value = qrcodeData.state;
          let qrReady = false;
          
          await new Promise(resolve => setTimeout(resolve, 100));
          
          if (qrcodeContainer.value) {
            qrcodeContainer.value.innerHTML = '';
            
            if (window.WwLogin) {
              new window.WwLogin({
                id: "wx_qrcode_container",
                appid: qrcodeData.corpId,
                agentid: String(qrcodeData.agentId),
                redirect_uri: encodeURIComponent(qrcodeData.redirectUri),
                state: qrcodeData.state,
                href: "",
              });
              qrReady = true;
            } else {
              ElMessage.error(t("pageMore.toast.wecomSdkLoadFailed"));
            }
          }
          
          if (qrReady) {
            startWecomPolling();
          }
        }
      } catch (error) {
        if (error?.code === 32105 || error?.code === 32001) {
          wecomConfigured.value = false;
        } else {
          ElMessage.error(error.message || t("pageMore.toast.loadQrFailed"));
        }
      }
    }
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadBindInfoFailed"));
  }
}

function startWecomPolling() {
  stopWecomPolling();
  if (!wecomBindState.value) {
    return;
  }
  
  wecomPollingTimer = setInterval(async () => {
    try {
      const resp = await api.getWecomBindStatus(wecomBindState.value);
      const data = resp?.data?.data || {};
      
      if (data.status === 'success') {
        stopWecomPolling();
        wecomBindStatus.value = t("pageMore.wecom.bindSuccess");
        wecomBindInfo.value = { isBound: true, userId: data.userId };
        wecomBindVisible.value = false;
        ElMessage.success(t("pageMore.toast.bindSuccess"));
      } else if (data.status === 'expired') {
        stopWecomPolling();
        wecomBindStatus.value = t("pageMore.wecom.qrExpired");
        ElMessage.warning(t("pageMore.toast.qrExpired"));
      } else if (data.status === 'failed') {
        stopWecomPolling();
        wecomBindStatus.value = t("pageMore.wecom.bindFailed");
        ElMessage.error(t("pageMore.toast.bindFailed"));
      } else {
        wecomBindStatus.value = t("pageMore.wecom.waiting");
      }
    } catch (error) {
      console.error('polling error:', error);
    }
  }, 2000);
  
  wecomPollingExpireTimer = setTimeout(() => {
    stopWecomPolling();
    if (wecomBindStatus.value === t("pageMore.wecom.waiting")) {
      wecomBindStatus.value = t("pageMore.wecom.qrExpired");
    }
  }, 5 * 60 * 1000);
}

function stopWecomPolling() {
  if (wecomPollingTimer) {
    clearInterval(wecomPollingTimer);
    wecomPollingTimer = null;
  }
  if (wecomPollingExpireTimer) {
    clearTimeout(wecomPollingExpireTimer);
    wecomPollingExpireTimer = null;
  }
}

async function handleUnbindWecom() {
  try {
    await ElMessageBox.confirm(t("pageMore.confirm.unbindWecom"), t("pageMore.confirm.unbindTitle"), {
      confirmButtonText: t("action.confirm"),
      cancelButtonText: t("action.cancel"),
      type: "warning",
    });
    
    unbindingWecom.value = true;
    await api.unbindWecom();
    wecomBindInfo.value = { isBound: false, userId: "" };
    ElMessage.success(t("pageMore.toast.unbindSuccess"));
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || t("pageMore.toast.unbindFailed"));
    }
  } finally {
    unbindingWecom.value = false;
  }
}

function goStatistics() {
  openAgentStats();
}

async function openAgentStats() {
  statsDialogVisible.value = true;
  statsLoading.value = true;
  try {
    const resp = await api.getProfileSummary();
    const d = resp?.data?.data || {};
    statsData.value = {
      sessions_today: Number(d.sessions_today || 0),
      total_assigned: Number(d.total_assigned || 0),
      rating: Number(d.rating || 0),
      rated_count: Number(d.rated_count || 0),
    };
  } catch (error) {
    ElMessage.error(error.message || t("pageMore.toast.loadStatsFailed"));
  } finally {
    statsLoading.value = false;
  }
}

onMounted(() => {
  fetchStatus();
  loadLocalPreferences();
  loadAppOptions();
  api
    .getAgentSettings()
    .then((resp) => {
      const data = resp?.data?.data || {};
      preferences.value = {
        soundEnabled: data.soundEnabled !== false,
        desktopNotifyEnabled: Boolean(data.desktopNotifyEnabled),
        typingIndicatorEnabled: data.typingIndicatorEnabled !== false,
        enterToSend: data.enterToSend !== false,
      };
    })
    .catch(() => {});
  loadAiModelOptions();
  loadAiConfigFromSystemSettings();
});

onBeforeUnmount(() => {
  stopWecomPolling();
});
</script>

<style scoped>
.more-page {
  min-height: 100%;
  background:
    radial-gradient(circle at 10% 0%, rgba(37, 99, 235, 0.11) 0%, rgba(37, 99, 235, 0) 35%),
    radial-gradient(circle at 100% 100%, rgba(14, 165, 233, 0.12) 0%, rgba(14, 165, 233, 0) 40%),
    #f7fafc;
}

.more-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.page-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  color: #0f172a;
}

.page-subtitle {
  margin: 6px 0 0;
  font-size: 13px;
  color: #64748b;
}

.seat-card {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid #dbeafe;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
}

.seat-text {
  font-size: 12px;
  font-weight: 600;
}

.seat-online {
  color: #16a34a;
}

.seat-offline {
  color: #6b7280;
}

.panel-card {
  border: 1px solid #dbe7ff;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 14px 36px rgba(15, 23, 42, 0.08);
}

.panel-header {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.feature-grid-system {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.feature-item {
  border: 1px solid #dbeafe;
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  padding: 14px 10px;
  text-align: center;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.feature-item:hover {
  transform: translateY(-2px);
  border-color: #93c5fd;
  box-shadow: 0 10px 26px rgba(37, 99, 235, 0.14);
}

.feature-icon {
  width: 52px;
  height: 52px;
  margin: 0 auto;
  border-radius: 999px;
  background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
  color: #1d4ed8;
  display: flex;
  align-items: center;
  justify-content: center;
}

.feature-icon :deep(.el-icon) {
  font-size: 24px;
}

.feature-icon.wechat-icon {
  background: linear-gradient(135deg, #dcfce7 0%, #bbf7d0 100%);
  padding: 0;
}

.feature-icon.wechat-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.feature-title {
  margin: 10px 0 0;
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
}

.dialog-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px;
  border-radius: 10px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  border: 1px solid #dbeafe;
  flex-wrap: wrap;
}

.tip-text {
  color: #6b7280;
  font-size: 12px;
}

.test-wrap {
  margin-top: 14px;
  border: 1px solid #dbeafe;
  border-radius: 12px;
  padding: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
}

.test-head {
  margin-bottom: 10px;
  font-weight: 600;
  color: #1f2937;
}

.workspace-grid {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 14px;
  min-height: 480px;
}

.workspace-grid-compact {
  grid-template-columns: 250px minmax(0, 1fr);
}

.workspace-side {
  border: 1px solid #dbeafe;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.workspace-side-title {
  font-size: 13px;
  font-weight: 700;
  color: #334155;
}

.workspace-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.workspace-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.workspace-actions-single {
  grid-template-columns: minmax(0, 1fr);
}

.workspace-stats {
  margin-top: auto;
  border-top: 1px dashed #cbd5e1;
  padding-top: 10px;
  display: grid;
  gap: 8px;
}

.workspace-stat-item {
  border: 1px solid #dbeafe;
  border-radius: 10px;
  background: #ffffff;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
}

.workspace-stat-item span {
  font-size: 12px;
  color: #64748b;
}

.workspace-stat-item strong {
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
}

.workspace-main {
  min-width: 0;
}

.test-row {
  display: grid;
  grid-template-columns: 220px 1fr 120px;
  gap: 8px;
  margin-bottom: 10px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.stats-item {
  border: 1px solid #dbeafe;
  border-radius: 10px;
  padding: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
}

.more-page :deep(.more-dialog .el-dialog) {
  border: 1px solid #dbe3ef;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 22px 48px rgba(15, 23, 42, 0.2);
}

.more-page :deep(.more-dialog .el-dialog__header) {
  margin-right: 0;
  padding: 14px 18px 12px;
  border-bottom: 1px solid rgba(229, 231, 235, 0.7);
  background: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
}

.more-page :deep(.more-dialog .el-dialog__title) {
  font-size: 15px;
  font-weight: 700;
  color: #1e293b;
}

.more-page :deep(.more-dialog .el-dialog__body) {
  padding: 14px 16px 16px;
  background: #ffffff;
}

.more-page :deep(.more-dialog .el-dialog__footer) {
  padding: 10px 16px 14px;
  border-top: 1px solid rgba(229, 231, 235, 0.7);
  background: #f8fbff;
}

.more-page :deep(.more-table) {
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #e2e8f0;
}

.more-page :deep(.more-table .el-table__header-wrapper th) {
  background: #f8fbff !important;
  color: #334155;
  font-weight: 600;
  border-bottom-color: #dbeafe !important;
}

.more-page :deep(.more-table .el-table__row:hover > td) {
  background: #eff6ff !important;
}

.more-page :deep(.el-input__wrapper),
.more-page :deep(.el-textarea__inner),
.more-page :deep(.el-select .el-select__wrapper) {
  border-radius: 10px;
}

.stats-label {
  font-size: 12px;
  color: #6b7280;
}

.stats-value {
  margin-top: 4px;
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
}

@media (max-width: 1100px) {
  .feature-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .feature-grid-system {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .test-row {
    grid-template-columns: 1fr;
  }

  .workspace-grid,
  .workspace-grid-compact {
    grid-template-columns: 1fr;
  }

  .workspace-side {
    order: 2;
  }

  .workspace-main {
    order: 1;
  }
}

.wecom-not-configured {
  padding: 40px 20px;
  text-align: center;
}

.wecom-bound {
  padding: 20px;
}

.wecom-qrcode {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
}

.qrcode-container {
  width: 200px;
  height: 200px;
  margin-bottom: 16px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
}

.qrcode-tip {
  font-size: 14px;
  color: #1f2937;
  font-weight: 500;
  margin: 0 0 8px;
}

.qrcode-expire {
  font-size: 12px;
  color: #6b7280;
  margin: 0 0 8px;
}

.qrcode-status {
  font-size: 12px;
  color: #059669;
  margin: 0;
}
</style>
