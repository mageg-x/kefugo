<template>
  <div class="knowledge-page">
    <div class="kb-ambient"></div>

    <section class="kb-header-card">
      <div class="kb-head-left">
        <h1 class="kb-title">{{ t("pageKnowledge.title") }}</h1>
        <p class="kb-subtitle">{{ t("pageKnowledge.subtitle") }}</p>
      </div>
      <div class="header-actions">
        <el-button plain class="kb-create-btn" @click="openModelDialog">{{ t("pageKnowledge.btn.modelConfig") }}</el-button>
        <el-button plain class="kb-create-btn" @click="newBaseVisible = true">+ {{ t("pageKnowledge.btn.newBase") }}</el-button>
      </div>
    </section>

    <section class="kb-selector-card">
      <div class="selector-main">
        <div class="selector-label">{{ t("pageKnowledge.label.currentBase") }}</div>
        <el-select
          :key="`kb-${baseOptions.length}-${currentBaseID}`"
          v-model="currentBaseID"
          :placeholder="t('pageKnowledge.placeholder.base')"
          class="kb-selector"
          filterable
          @change="onBaseChanged"
        >
          <el-option v-for="base in baseOptions" :key="base.id" :label="base.name" :value="base.id" />
        </el-select>
        <el-button plain class="kb-delete-btn" :disabled="!currentBaseID" @click="removeCurrentBase">
          {{ t("pageKnowledge.btn.deleteBase") }}
        </el-button>
      </div>
      <div class="selector-stats">
        <span class="stat-pill">{{ t("pageKnowledge.stats.base") }} {{ baseOptions.length }}</span>
        <span class="stat-pill">{{ t("pageKnowledge.stats.document") }} {{ documents.length }}</span>
        <span class="stat-pill">{{ t("pageKnowledge.stats.chunk") }} {{ chunks.length }}</span>
      </div>
    </section>

    <section class="kb-tabs-wrap">
      <button
        v-for="tab in tabItems"
        :key="tab.key"
        type="button"
        class="kb-tab"
        :class="{ active: activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        <span class="tab-icon">{{ tab.icon }}</span>
        {{ t(tab.labelKey) }}
      </button>
    </section>

    <section class="kb-panel" v-loading="contentLoading">
      <template v-if="activeTab === 'documents'">
        <div class="kb-toolbar">
          <el-button plain class="toolbar-btn" :disabled="!canUploadDocs" :loading="kbBackendChecking || uploadingDocuments" @click="pickFiles">{{ t("pageKnowledge.btn.uploadDocs") }}</el-button>
          <el-tag size="small" :type="kbBackendReady ? 'success' : (kbBackendDegraded ? 'warning' : 'danger')">
            {{ kbBackendReady ? t("pageKnowledge.status.vectorReady") : t("pageKnowledge.status.vectorNotReady") }}
          </el-tag>
          <span v-if="uploadingDocuments" class="model-hint">{{ uploadProgressText }}</span>
          <input
            ref="fileInputRef"
            type="file"
            multiple
            class="hidden-input"
            accept=".txt,.md,.pdf,.docx,.csv,.tsv,.xlsx"
            @change="onFilesSelected"
          />
          <el-input
            v-model="documentKeyword"
            :placeholder="t('pageKnowledge.placeholder.documentKeyword')"
            clearable
            class="kb-search"
            @keyup.enter="loadDocuments"
          />
        </div>

        <div class="doc-list">
          <div v-for="doc in documents" :key="doc.id" class="doc-item">
            <div class="doc-main">
              <div class="doc-name">{{ doc.name }}</div>
              <div class="doc-meta">
                <span>{{ (doc.file_type || "").toUpperCase() }}</span>
                <span>·</span>
                <span>{{ formatFileSize(doc.file_size) }}</span>
                <span>·</span>
                <span>{{ formatDateTime(doc.updated_at) }}</span>
              </div>
            </div>

            <div class="doc-status">
              <span class="status-chip" :class="statusClass(doc.status)">
                {{ statusLabel(doc.status) }}
              </span>
              <el-tooltip v-if="doc.error_message" :content="doc.error_message" placement="top">
                <span class="doc-error">{{ t("pageKnowledge.word.error") }}</span>
              </el-tooltip>
            </div>

            <el-dropdown trigger="click" class="doc-menu">
              <span class="doc-more">⋯</span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="reindexDocument(doc)">{{ t("pageKnowledge.action.reindex") }}</el-dropdown-item>
                  <el-dropdown-item divided class="danger-item" @click="removeDocument(doc)">{{ t("action.delete") }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <el-empty v-if="!documents.length" :description="t('pageKnowledge.empty.documents')" />
        </div>
      </template>

      <template v-else-if="activeTab === 'chunks'">
        <div class="kb-toolbar">
          <el-input
            v-model="chunkKeyword"
            :placeholder="t('pageKnowledge.placeholder.chunkKeyword')"
            clearable
            class="kb-search"
            @keyup.enter="loadChunks"
          />
          <el-select
            v-model="chunkDocumentFilter"
            clearable
            :placeholder="t('pageKnowledge.placeholder.chunkDocument')"
            class="doc-filter"
            @change="loadChunks"
          >
            <el-option v-for="doc in documents" :key="doc.id" :label="doc.name" :value="doc.id" />
          </el-select>
          <el-button plain class="toolbar-btn" @click="loadChunks">{{ t("action.refresh") }}</el-button>
        </div>

        <div class="chunk-list">
          <article v-for="chunk in chunks" :key="chunk.id" class="chunk-card">
            <div class="chunk-head">
              <span class="chunk-doc">{{ chunk.document_name || t("pageKnowledge.word.unknownDocument") }}</span>
              <span class="chunk-seq">{{ t("pageKnowledge.word.chunk") }} {{ chunk.chunk_seq }}</span>
              <div class="chunk-actions">
                <button type="button" class="text-btn" @click="startEditChunk(chunk)">{{ t("action.edit") }}</button>
                <button type="button" class="text-btn danger" @click="removeChunk(chunk)">{{ t("action.delete") }}</button>
              </div>
            </div>

            <div v-if="editingChunkID === chunk.id" class="chunk-edit">
              <el-input v-model="editingChunkContent" type="textarea" :rows="4" />
              <div class="edit-actions">
                <el-button size="small" @click="cancelEditChunk">{{ t("action.cancel") }}</el-button>
                <el-button size="small" type="primary" @click="saveChunkEdit(chunk)">{{ t("action.save") }}</el-button>
              </div>
            </div>
            <p v-else class="chunk-body">{{ chunk.content }}</p>

            <div class="chunk-meta">
              {{ chunk.content_chars }}{{ t("pageKnowledge.word.chars") }} · {{ t("pageKnowledge.word.similarity") }} {{ Number(chunk.avg_score || 0).toFixed(2) }}
            </div>
          </article>

          <el-empty v-if="!chunks.length" :description="t('pageKnowledge.empty.chunks')" />
        </div>
      </template>

      <template v-else-if="activeTab === 'retrieve'">
        <div class="kb-toolbar">
          <el-input
            v-model="retrieveQuery"
            :placeholder="t('pageKnowledge.placeholder.retrieveQuery')"
            clearable
            class="kb-search"
            @keyup.enter="runRetrieveTest"
          />
          <el-select v-model="retrieveTopK" class="topk-select">
            <el-option v-for="k in topKOptions" :key="k" :label="`Top ${k}`" :value="k" />
          </el-select>
          <el-button type="primary" class="toolbar-btn primary" @click="runRetrieveTest">{{ t("pageKnowledge.btn.retrieve") }}</el-button>
        </div>

        <div class="retrieve-list">
          <div v-for="(item, idx) in retrieveResults" :key="`${item.id}-${idx}`" class="retrieve-item">
            <div class="retrieve-line">
              <span class="retrieve-score">{{ Math.round((item.score || 0) * 100) }}%</span>
              <span class="retrieve-doc">{{ item.document_name || t("pageKnowledge.word.unknownDocument") }}</span>
            </div>
            <div class="retrieve-body">{{ item.content }}</div>
          </div>
          <el-empty v-if="!retrieveResults.length" :description="t('pageKnowledge.empty.retrieve')" />
        </div>
      </template>

      <template v-else>
        <div class="kb-toolbar">
          <el-tag :type="activeAPIModel ? 'success' : 'warning'">
            {{ activeAPIModel ? t("pageKnowledge.status.modelEnabled") : t("pageKnowledge.status.modelDisabled") }}
          </el-tag>
          <span class="model-hint">{{ activeAPIModelHint }}</span>
        </div>

        <div class="qa-list">
          <article v-for="(item, idx) in qaHistory" :key="`${idx}-${item.question}`" class="qa-card">
            <div class="qa-question">👤 {{ item.question }}</div>
            <div class="qa-answer">🤖 {{ item.answer }}</div>
            <div class="qa-sources" v-if="item.sources?.length">
              <span>{{ t("pageKnowledge.word.references") }}:</span>
              <el-tooltip
                v-for="src in item.sources"
                :key="`${src.doc_name}-${src.vector_id}`"
                :content="src.vector_id"
                placement="top"
              >
                <span class="src-chip">{{ src.doc_name || t("pageKnowledge.word.unnamedSource") }}</span>
              </el-tooltip>
            </div>
            <div class="qa-meta" v-if="item.model_provider || item.model_name">
              {{ t("pageKnowledge.word.model") }}:{{ item.model_provider || "-" }} · {{ item.model_name || "-" }}
            </div>
            <div class="qa-actions">
              <button type="button" class="icon-btn" :class="{ active: item.helpful === true }" @click="feedback(item, true)">👍</button>
              <button type="button" class="icon-btn" :class="{ active: item.helpful === false }" @click="feedback(item, false)">👎</button>
              <button type="button" class="icon-btn" @click="regenerate(item)">↻ {{ t("pageKnowledge.action.regenerate") }}</button>
            </div>
          </article>

          <el-empty v-if="!qaHistory.length" :description="t('pageKnowledge.empty.qa')" />
        </div>

        <div class="qa-input-row">
          <el-input v-model="qaInput" :disabled="qaStreaming" :placeholder="t('pageKnowledge.placeholder.qaInput')" @keyup.enter="runQATest" />
          <el-button type="primary" :loading="qaStreaming" @click="runQATest">{{ t("action.send") }}</el-button>
          <el-button text @click="clearQAHistory">{{ t("action.clear") }}</el-button>
        </div>
      </template>
    </section>

    <footer class="kb-footer">{{ chunkSummary }}</footer>

    <el-dialog v-model="newBaseVisible" :title="t('pageKnowledge.dialog.newBase')" width="520px">
      <el-form label-width="80px">
        <el-form-item :label="t('app.label')">
          <el-select v-model="newBaseForm.app_id" filterable style="width: 100%">
            <el-option v-for="app in appOptions" :key="app.value" :label="app.label" :value="app.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('name')">
          <el-input v-model="newBaseForm.name" maxlength="128" show-word-limit :placeholder="t('pageKnowledge.placeholder.baseName')" />
        </el-form-item>
        <el-form-item :label="t('form.description')">
          <el-input v-model="newBaseForm.description" type="textarea" :rows="3" :placeholder="t('pageKnowledge.placeholder.baseDescription')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="newBaseVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :loading="creatingBase" @click="createBase">{{ t("action.create") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="llmDialogVisible" :title="t('pageKnowledge.dialog.modelConfig')" width="1080px" class="llm-config-dialog">
      <div class="model-type-tabs">
        <button
          v-for="mt in modelTypeTabs"
          :key="mt.key"
          type="button"
          class="model-type-tab"
          :class="{ active: activeModelType === mt.key }"
          @click="activeModelType = mt.key"
        >
          <span class="model-type-icon">{{ mt.icon }}</span>
          {{ t(mt.labelKey) }}
        </button>
      </div>
      <div class="model-status-cards">
        <div v-for="mt in modelTypeTabs" :key="mt.key + '-status'" class="model-status-card" :class="{ 'has-default': getActiveModelLabel(mt.key) }">
          <span class="model-status-icon">{{ mt.icon }}</span>
          <div class="model-status-info">
            <div class="model-status-label">{{ t(mt.labelKey) }}</div>
            <div class="model-status-value">{{ getActiveModelLabel(mt.key) || t("pageKnowledge.status.modelDisabled") }}</div>
          </div>
          <el-tag size="small" :type="getActiveModelLabel(mt.key) ? 'success' : 'info'">
            {{ getActiveModelLabel(mt.key) ? t("status.enabled") : t("status.disabled") }}
          </el-tag>
        </div>
      </div>
      <div class="kb-toolbar">
        <el-button plain class="toolbar-btn" @click="openAPIModelForm()">+ {{ t("pageKnowledge.btn.newConfig") }}</el-button>
        <el-button plain class="toolbar-btn" @click="loadAPIModels">{{ t("action.refresh") }}</el-button>
      </div>
      <el-table :data="filteredAPIModels" border style="width: 100%">
        <el-table-column prop="name" :label="t('pageKnowledge.table.configName')" min-width="100" />
        <el-table-column prop="provider" :label="t('pageKnowledge.table.provider')" min-width="90" />
        <el-table-column prop="model_name" :label="t('pageKnowledge.table.modelName')" min-width="180" />
        <el-table-column v-if="activeModelType === 'embedding'" prop="dims" label="Dims" width="70" />
        <el-table-column v-if="activeModelType === 'embedding'" prop="top_k" label="TopK" width="70" />
        <el-table-column v-if="activeModelType === 'rerank'" prop="top_n" label="TopN" width="70" />
        <el-table-column prop="api_key_mask" :label="t('pageKnowledge.table.apiKey')" min-width="120" />
        <el-table-column prop="base_url" label="Base URL" min-width="180" show-overflow-tooltip />
        <el-table-column :label="t('pageKnowledge.table.default')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.is_default ? 'success' : 'info'" size="small">
              {{ row.is_default ? t("status.enabled") : "-" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('status.label')" width="80">
          <template #default="{ row }">
            <el-tag :type="Number(row.status) === 1 ? 'success' : 'info'" size="small">
              {{ Number(row.status) === 1 ? t("status.enabled") : t("status.disabled") }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('action.actions')" width="320" fixed="right">
          <template #default="{ row }">
            <el-button v-if="!row.is_default" text type="success" @click="setDefaultAPIModel(row)">{{ t("pageKnowledge.btn.setDefault") }}</el-button>
            <el-button text type="primary" @click="testAPIModel(row)">{{ t("action.test") }}</el-button>
            <el-button v-if="activeModelType === 'embedding' && row.status === 1" text type="warning" @click="triggerRebuild(row)">{{ t("pageKnowledge.btn.rebuild") }}</el-button>
            <el-button text @click="openAPIModelForm(row)">{{ t("action.edit") }}</el-button>
            <el-button text type="danger" @click="removeAPIModel(row)">{{ t("action.delete") }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="rebuildTask" class="rebuild-progress">
        <el-divider />
        <div style="display: flex; align-items: center; gap: 12px;">
          <span style="font-weight: 600;">{{ t("pageKnowledge.rebuild.title") }}</span>
          <el-tag :type="rebuildTaskStatusType" size="small">{{ rebuildTask.status }}</el-tag>
        </div>
        <el-progress :percentage="rebuildTask.progress" :status="rebuildTaskProgressStatus" style="margin-top: 8px;" />
        <div v-if="rebuildTask.status === 'running'" style="margin-top: 4px; font-size: 12px; color: #909399;">
          {{ rebuildTask.done_docs }} / {{ rebuildTask.total_docs }}
        </div>
        <div v-if="rebuildTask.status === 'failed'" style="margin-top: 4px; font-size: 12px; color: #F56C6C;">
          {{ rebuildTask.error_message }}
        </div>
      </div>
      <template #footer>
        <el-button @click="llmDialogVisible = false">{{ t("action.close") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="apiModelFormVisible" :title="apiModelForm.id ? t('pageKnowledge.dialog.editModelConfig') : t('pageKnowledge.dialog.newModelConfig')" width="620px">
      <el-form label-width="120px">
        <el-form-item :label="t('pageKnowledge.table.modelType')">
          <el-select v-model="apiModelForm.model_type" style="width: 100%" :disabled="!!apiModelForm.id">
            <el-option v-for="mt in modelTypeTabs" :key="mt.key" :label="t(mt.labelKey)" :value="mt.key" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageKnowledge.table.configName')">
          <el-input v-model="apiModelForm.name" :placeholder="t('pageKnowledge.placeholder.configName')" />
        </el-form-item>
        <el-form-item :label="t('pageKnowledge.table.provider')">
          <el-select v-model="apiModelForm.provider" style="width: 100%" @change="onProviderChange">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Qwen" value="qwen" />
            <el-option label="DeepSeek" value="deepseek" />
            <el-option label="Zhipu AI" value="zhipu" />
            <el-option label="Ollama" value="ollama" />
            <el-option v-if="apiModelForm.model_type === 'rerank'" label="Cohere" value="cohere" />
            <el-option v-if="apiModelForm.model_type === 'rerank' || apiModelForm.model_type === 'embedding'" label="Jina" value="jina" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="apiModelForm.provider !== 'ollama'" :label="t('pageKnowledge.table.apiKey')">
          <el-input
            v-model="apiModelForm.api_key"
            type="password"
            show-password
            :placeholder="apiModelForm.id ? (apiModelForm.api_key_mask ? `${t('pageKnowledge.placeholder.keepApiKey')} (${t('pageKnowledge.word.current')}: ${apiModelForm.api_key_mask})` : t('pageKnowledge.placeholder.keepApiKey')) : 'sk-xxxx'"
          />
          <div v-if="apiModelForm.id" class="model-hint" style="margin-top: 6px;">
            {{ t("pageKnowledge.hint.keepApiKeyInEdit") }}
          </div>
        </el-form-item>
        <el-form-item label="Base URL">
          <el-input v-model="apiModelForm.base_url" :placeholder="apiModelForm.provider === 'ollama' ? 'http://localhost:11434' : t('pageKnowledge.placeholder.defaultBaseURL')" />
        </el-form-item>
        <el-form-item :label="t('pageKnowledge.table.modelName')">
          <el-input v-model="apiModelForm.model_name" :placeholder="t('pageKnowledge.placeholder.modelName')" />
        </el-form-item>
        <el-form-item v-if="apiModelForm.model_type === 'embedding'" label="Dims">
          <el-input-number v-model="apiModelForm.dims" :min="64" :max="4096" :step="64" />
        </el-form-item>
        <el-form-item v-if="apiModelForm.model_type === 'embedding'" label="TopK">
          <el-input-number v-model="apiModelForm.top_k" :min="1" :max="100" />
        </el-form-item>
        <el-form-item v-if="apiModelForm.model_type === 'rerank'" label="TopN">
          <el-input-number v-model="apiModelForm.top_n" :min="1" :max="50" />
        </el-form-item>
        <el-form-item :label="t('pageKnowledge.table.timeoutSec')">
          <el-input-number v-model="apiModelForm.timeout_sec" :min="10" :max="600" />
        </el-form-item>
        <el-form-item v-if="apiModelForm.model_type === 'chat'" :label="t('pageKnowledge.table.temperature')">
          <el-input-number v-model="apiModelForm.temperature" :min="0" :max="2" :step="0.1" />
        </el-form-item>
        <el-form-item v-if="apiModelForm.model_type === 'chat'" label="Top P">
          <el-input-number v-model="apiModelForm.top_p" :min="0.1" :max="1" :step="0.1" />
        </el-form-item>
        <el-form-item v-if="apiModelForm.model_type === 'chat'" :label="t('pageKnowledge.table.maxTokens')">
          <el-input-number v-model="apiModelForm.max_tokens" :min="64" :max="8192" :step="64" />
        </el-form-item>
        <el-form-item :label="t('status.enabled')">
          <el-switch v-model="apiModelForm.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="apiModelFormVisible = false">{{ t("action.cancel") }}</el-button>
        <el-button type="primary" :loading="savingAPIModel" @click="saveAPIModel">{{ t("action.save") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";
import { useStore } from "@/script/store";

const activeTab = ref("documents");
const contentLoading = ref(false);
const tabItems = [
  { key: "documents", labelKey: "pageKnowledge.tab.documents", icon: "📄" },
  { key: "chunks", labelKey: "pageKnowledge.tab.chunks", icon: "📝" },
  { key: "retrieve", labelKey: "pageKnowledge.tab.retrieve", icon: "🔍" },
  { key: "qa", labelKey: "pageKnowledge.tab.qa", icon: "💬" },
];

const baseOptions = ref([]);
const currentBaseID = ref(0);
const appOptions = ref([]);

const documents = ref([]);
const documentKeyword = ref("");
const fileInputRef = ref(null);
const uploadingDocuments = ref(false);
const uploadDoneCount = ref(0);
const uploadTotalCount = ref(0);

const chunks = ref([]);
const chunkKeyword = ref("");
const chunkDocumentFilter = ref(undefined);
const editingChunkID = ref(0);
const editingChunkContent = ref("");

const retrieveQuery = ref("");
const retrieveTopK = ref(5);
const topKOptions = [1, 3, 5, 8, 10, 15, 20];
const retrieveResults = ref([]);
const kbBackendReady = ref(false);
const kbBackendDegraded = ref(false);
const kbBackendChecking = ref(false);

const qaInput = ref("");
const qaHistory = ref([]);
const qaStreaming = ref(false);
const store = useStore();
let qaAbortController = null;

const newBaseVisible = ref(false);
const creatingBase = ref(false);
const newBaseForm = ref({
  app_id: "",
  name: "",
  description: "",
});

const llmDialogVisible = ref(false);
const apiModels = ref([]);
const apiModelFormVisible = ref(false);
const savingAPIModel = ref(false);
const activeModelType = ref("chat");
const rebuildTask = ref(null);
let rebuildPollTimer = null;
const modelTypeTabs = [
  { key: "chat", labelKey: "pageKnowledge.modelType.chat", icon: "💬" },
  { key: "embedding", labelKey: "pageKnowledge.modelType.embedding", icon: "🔢" },
  { key: "rerank", labelKey: "pageKnowledge.modelType.rerank", icon: "🔄" },
];
const apiModelForm = ref({
  id: 0,
  model_type: "chat",
  name: "",
  provider: "openai",
  api_key: "",
  api_key_mask: "",
  base_url: "",
  model_name: "",
  dims: 384,
  top_k: 20,
  top_n: 5,
  timeout_sec: 60,
  temperature: 0.7,
  top_p: 0.9,
  max_tokens: 512,
  is_default: false,
  status: 1,
});

const filteredAPIModels = computed(() => {
  return (apiModels.value || []).filter((m) => (m.model_type || "chat") === activeModelType.value);
});

function getEnabledModel(modelType) {
  const items = (apiModels.value || []).filter(
    (m) => (m.model_type || "chat") === modelType && Number(m.status) === 1,
  );
  if (!items.length) return null;
  return items.find((m) => Boolean(m.is_default)) || items[0];
}

function getActiveModelLabel(modelType) {
  const m = getEnabledModel(modelType);
  return m ? `${m.provider}/${m.model_name}` : "";
}

const chunkSummary = computed(() => {
  const count = chunks.value.length;
  let latest = "";
  const sorted = [...documents.value].sort(
    (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
  );
  if (sorted[0]?.updated_at) latest = ` · ${t("pageKnowledge.word.lastUpdated")} ${formatDateTime(sorted[0].updated_at)}`;
  return `${count}${t("pageKnowledge.word.chunk")}${latest}`;
});

const activeAPIModel = computed(() => {
  return getEnabledModel("chat");
});

const activeAPIModelHint = computed(() => {
  if (!activeAPIModel.value) return t("pageKnowledge.hint.noEnabledModel");
  const item = activeAPIModel.value;
  return `${item.provider} · ${item.model_name}`;
});

function getModelTimeoutSec(modelType, fallbackSec = 60) {
  const item = getEnabledModel(modelType);
  const timeout = Number(item?.timeout_sec || 0);
  if (Number.isFinite(timeout) && timeout > 0) {
    return timeout;
  }
  return fallbackSec;
}

function getRetrieveTimeoutSec() {
  const embeddingTimeout = getModelTimeoutSec("embedding", 60);
  const rerankModel = getEnabledModel("rerank");
  const rerankTimeout = rerankModel ? getModelTimeoutSec("rerank", 30) + 10 : 0;
  return Math.min(600, Math.max(45, embeddingTimeout + rerankTimeout + 20));
}

function getQATimeoutSec() {
  const chatTimeout = getModelTimeoutSec("chat", 120);
  return Math.min(900, Math.max(90, getRetrieveTimeoutSec() + chatTimeout + 20));
}

const canUploadDocs = computed(() => Boolean(currentBaseID.value) && (kbBackendReady.value || kbBackendDegraded.value) && !kbBackendChecking.value);
const uploadProgressText = computed(() => {
  if (!uploadingDocuments.value || uploadTotalCount.value <= 0) return "";
  return `${uploadDoneCount.value} / ${uploadTotalCount.value}`;
});

function formatDateTime(value) {
  if (!value) return "-";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return String(value);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate()
  ).padStart(2, "0")} ${String(d.getHours()).padStart(2, "0")}:${String(
    d.getMinutes()
  ).padStart(2, "0")}`;
}

function formatFileSize(size) {
  const n = Number(size || 0);
  if (n < 1024) return `${n}B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)}KB`;
  return `${(n / 1024 / 1024).toFixed(1)}MB`;
}

function statusLabel(status) {
  switch ((status || "").toLowerCase()) {
    case "indexed":
      return t("pageKnowledge.docStatus.indexed");
    case "indexing":
      return t("pageKnowledge.docStatus.indexing");
    case "failed":
      return t("pageKnowledge.docStatus.failed");
    default:
      return t("pageKnowledge.docStatus.pending");
  }
}

function statusClass(status) {
  switch ((status || "").toLowerCase()) {
    case "indexed":
      return "ok";
    case "indexing":
      return "running";
    case "failed":
      return "bad";
    default:
      return "pending";
  }
}

async function loadAppOptions() {
  try {
    const resp = await api.listApps({ page: 1, page_size: 200 });
    const rows = resp?.data?.data?.data || [];
    appOptions.value = rows.map((item) => ({ value: item.app_id, label: `${item.name} (${item.app_id})` }));
    if (!newBaseForm.value.app_id && appOptions.value[0]) {
      newBaseForm.value.app_id = appOptions.value[0].value;
    }
  } catch {
    try {
      const resp = await api.listSessions({ page: 1, page_size: 200 });
      const rows = resp?.data?.data?.data || [];
      const appIDs = Array.from(
        new Set(rows.map((item) => String(item?.app_id || "").trim()).filter(Boolean))
      );
      appOptions.value = appIDs.map((appID) => ({ value: appID, label: appID }));
      if (!newBaseForm.value.app_id && appOptions.value[0]) {
        newBaseForm.value.app_id = appOptions.value[0].value;
      }
    } catch (err) {
      console.error(err);
      appOptions.value = [];
    }
  }
}

function normalizeBaseID(value) {
  const id = Number(value || 0);
  return Number.isFinite(id) ? id : 0;
}

function upsertBaseOption(item) {
  const id = normalizeBaseID(item?.id);
  if (!id) return;
  const next = {
    ...item,
    id,
  };
  const idx = baseOptions.value.findIndex((row) => normalizeBaseID(row?.id) === id);
  if (idx >= 0) {
    baseOptions.value[idx] = next;
    return;
  }
  baseOptions.value = [next, ...baseOptions.value];
}

async function loadBases(options = {}) {
  const silent = options?.silent === true;
  const preferID = normalizeBaseID(options?.preferID);
  const keepLocalCreated = options?.keepLocalCreated === true;
  try {
    const resp = await api.listKnowledgeBases();
    const rows = (resp?.data?.data?.data || []).map((item) => ({
      ...item,
      id: normalizeBaseID(item?.id),
    }));
    const remote = rows.filter((item) => item.id > 0);
    if (keepLocalCreated && preferID > 0) {
      const hasRemotePrefer = remote.some((item) => item.id === preferID);
      if (!hasRemotePrefer) {
        const localPrefer = baseOptions.value.find((item) => normalizeBaseID(item?.id) === preferID);
        if (localPrefer) {
          baseOptions.value = [localPrefer, ...remote];
        } else {
          baseOptions.value = remote;
        }
      } else {
        baseOptions.value = remote;
      }
    } else {
      baseOptions.value = remote;
    }

    const hasCurrent = baseOptions.value.some((item) => item.id === normalizeBaseID(currentBaseID.value));
    if (preferID && baseOptions.value.some((item) => item.id === preferID)) {
      currentBaseID.value = preferID;
    } else if (!hasCurrent && baseOptions.value[0]) {
      currentBaseID.value = baseOptions.value[0].id;
    } else if (!baseOptions.value.length) {
      currentBaseID.value = 0;
    }
  } catch (err) {
    if (!silent) {
      ElMessage.error(err.message || t("pageKnowledge.toast.loadBaseFailed"));
    }
  }
}

async function createBase() {
  if (!newBaseForm.value.name.trim()) {
    ElMessage.warning(t("pageKnowledge.toast.inputBaseName"));
    return;
  }
  if (!newBaseForm.value.app_id) {
    ElMessage.warning(t("pageKnowledge.toast.selectApp"));
    return;
  }
  creatingBase.value = true;
  try {
    const resp = await api.createKnowledgeBase({
      app_id: newBaseForm.value.app_id,
      name: newBaseForm.value.name.trim(),
      description: newBaseForm.value.description.trim(),
    });
    if (Number(resp?.data?.code ?? -1) !== 0) {
      throw new Error(String(resp?.data?.msg || t("pageKnowledge.toast.createBaseFailed")));
    }
    const created = resp?.data?.data?.item || {};
    const createdID = normalizeBaseID(created?.id);
    if (createdID > 0) {
      upsertBaseOption(created);
      currentBaseID.value = createdID;
    }

    ElMessage.success(t("pageKnowledge.toast.createBaseSuccess"));
    newBaseVisible.value = false;
    newBaseForm.value.name = "";
    newBaseForm.value.description = "";
    void loadBases({ silent: true, preferID: createdID, keepLocalCreated: true });
    await refreshKnowledgeBackendHealth(false);
    if (currentBaseID.value) {
      await Promise.all([loadDocuments(true), loadChunks(true)]);
    }
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.createBaseFailed"));
  } finally {
    creatingBase.value = false;
  }
}

async function removeCurrentBase() {
  const id = normalizeBaseID(currentBaseID.value);
  if (!id) {
    ElMessage.warning(t("pageKnowledge.toast.selectBaseFirst"));
    return;
  }
  const base = baseOptions.value.find((item) => normalizeBaseID(item?.id) === id);
  const baseName = String(base?.name || `#${id}`);
  try {
    await ElMessageBox.confirm(
      `${t("pageKnowledge.confirm.deleteBase")}「${baseName}」？${t("pageKnowledge.confirm.deleteBaseSuffix")}`,
      t("pageKnowledge.confirm.title"),
      { type: "warning" }
    );
    await api.deleteKnowledgeBase(id);
    baseOptions.value = baseOptions.value.filter((item) => normalizeBaseID(item?.id) !== id);
    currentBaseID.value = normalizeBaseID(baseOptions.value[0]?.id);
    documents.value = [];
    chunks.value = [];
    retrieveResults.value = [];
    qaHistory.value = [];
    ElMessage.success(t("pageKnowledge.toast.baseDeleted"));
    await loadBases({ silent: true, preferID: currentBaseID.value });
    if (currentBaseID.value) {
      await Promise.all([loadDocuments(), loadChunks()]);
    }
  } catch (err) {
    if (err !== "cancel" && err !== "close") {
      ElMessage.error(err.message || t("pageKnowledge.toast.deleteBaseFailed"));
    }
  }
}

function ensureBaseID() {
  if (!currentBaseID.value) {
    ElMessage.warning(t("pageKnowledge.toast.selectBaseFirst"));
    return false;
  }
  return true;
}

async function refreshKnowledgeBackendHealth(showToast = false) {
  kbBackendChecking.value = true;
  try {
    const resp = await api.checkKnowledgeBackend();
    const payload = resp?.data?.data || {};
    const status = String(payload?.status || "").toLowerCase();
    const ok = status === "ok" && payload?.ready !== false;
    const degraded = status === "degraded";
    const usable = ok || degraded;
    kbBackendReady.value = ok;
    kbBackendDegraded.value = degraded;
    if (showToast) {
      if (ok) {
        ElMessage.success(t("pageKnowledge.toast.vectorReady"));
      } else if (degraded) {
        ElMessage.warning(payload?.message || t("pageKnowledge.toast.vectorNotReady"));
      } else {
        ElMessage.warning(payload?.message || t("pageKnowledge.toast.vectorNotReady"));
      }
    }
    return usable;
  } catch (err) {
    kbBackendReady.value = false;
    kbBackendDegraded.value = false;
    if (showToast) {
      ElMessage.error(err.message || t("pageKnowledge.toast.vectorUnavailable"));
    }
    return false;
  } finally {
    kbBackendChecking.value = false;
  }
}

async function loadDocuments(silent = false) {
  if (!ensureBaseID()) return;
  if (!silent) contentLoading.value = true;
  try {
    const resp = await api.listKnowledgeDocuments(currentBaseID.value, {
      keyword: documentKeyword.value || undefined,
    });
    documents.value = resp?.data?.data?.data || [];
  } catch (err) {
    if (!silent) {
      ElMessage.error(err.message || t("pageKnowledge.toast.loadDocumentFailed"));
    }
  } finally {
    if (!silent) contentLoading.value = false;
  }
}

function pickFiles() {
  if (!ensureBaseID()) return;
  if (!kbBackendReady.value && !kbBackendDegraded.value) {
    refreshKnowledgeBackendHealth(false).then((ok) => {
      if (!ok) {
        ElMessage.error(t("pageKnowledge.toast.vectorNotReadyRetry"));
        return;
      }
      fileInputRef.value?.click();
    });
    return;
  }
  fileInputRef.value?.click();
}

async function onFilesSelected(event) {
  const files = Array.from(event?.target?.files || []);
  if (!files.length) return;
  if (!ensureBaseID()) return;
  if (!kbBackendReady.value && !kbBackendDegraded.value) {
    const ok = await refreshKnowledgeBackendHealth(false);
    if (!ok) {
      ElMessage.error(t("pageKnowledge.toast.vectorNotReadyUpload"));
      if (event?.target) event.target.value = "";
      return;
    }
  }
  uploadingDocuments.value = true;
  uploadDoneCount.value = 0;
  uploadTotalCount.value = files.length;
  try {
    const failed = [];
    for (const file of files) {
      try {
        await api.uploadKnowledgeDocument(currentBaseID.value, file);
        uploadDoneCount.value += 1;
        await loadDocuments(true);
      } catch (err) {
        failed.push({ file, err });
      }
    }
    if (!failed.length) {
      ElMessage.success(t("pageKnowledge.toast.uploadSuccess"));
    } else if (uploadDoneCount.value > 0) {
      ElMessage.warning(`${uploadDoneCount.value}/${files.length} uploaded, ${failed.length} failed`);
    } else {
      throw failed[0].err;
    }
    await loadChunks(true);
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.uploadFailed"));
  } finally {
    uploadingDocuments.value = false;
    uploadDoneCount.value = 0;
    uploadTotalCount.value = 0;
    if (event?.target) event.target.value = "";
  }
}

async function reindexDocument(doc) {
  try {
    await api.reindexKnowledgeDocument(currentBaseID.value, doc.id);
    ElMessage.success(t("pageKnowledge.toast.reindexSuccess"));
    await loadDocuments();
    await loadChunks();
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.reindexFailed"));
  }
}

async function removeDocument(doc) {
  try {
    await ElMessageBox.confirm(`${t("pageKnowledge.confirm.deleteDocument")}「${doc.name}」？`, t("pageKnowledge.confirm.title"), { type: "warning" });
    await api.deleteKnowledgeDocument(currentBaseID.value, doc.id);
    ElMessage.success(t("pageKnowledge.toast.documentDeleted"));
    await loadDocuments();
    await loadChunks();
  } catch (err) {
    if (err !== "cancel" && err !== "close") {
      ElMessage.error(err.message || t("pageKnowledge.toast.deleteFailed"));
    }
  }
}

async function loadChunks(silent = false) {
  if (!ensureBaseID()) return;
  if (!silent) contentLoading.value = true;
  try {
    const resp = await api.listKnowledgeChunks(currentBaseID.value, {
      keyword: chunkKeyword.value || undefined,
      document_id: chunkDocumentFilter.value || undefined,
    });
    chunks.value = resp?.data?.data?.data || [];
  } catch (err) {
    if (!silent) {
      ElMessage.error(err.message || t("pageKnowledge.toast.loadChunkFailed"));
    }
  } finally {
    if (!silent) contentLoading.value = false;
  }
}

function startEditChunk(chunk) {
  editingChunkID.value = chunk.id;
  editingChunkContent.value = chunk.content || "";
}

function cancelEditChunk() {
  editingChunkID.value = 0;
  editingChunkContent.value = "";
}

async function saveChunkEdit(chunk) {
  const content = editingChunkContent.value.trim();
  if (!content) {
    ElMessage.warning(t("pageKnowledge.toast.chunkContentRequired"));
    return;
  }
  try {
    await api.updateKnowledgeChunk(currentBaseID.value, chunk.id, content);
    ElMessage.success(t("pageKnowledge.toast.chunkUpdated"));
    cancelEditChunk();
    await loadChunks();
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.updateChunkFailed"));
  }
}

async function removeChunk(chunk) {
  try {
    await ElMessageBox.confirm(t("pageKnowledge.confirm.deleteChunk"), t("pageKnowledge.confirm.title"), { type: "warning" });
    await api.deleteKnowledgeChunk(currentBaseID.value, chunk.id);
    ElMessage.success(t("pageKnowledge.toast.chunkDeleted"));
    await loadChunks();
  } catch (err) {
    if (err !== "cancel" && err !== "close") {
      ElMessage.error(err.message || t("pageKnowledge.toast.deleteChunkFailed"));
    }
  }
}

async function runRetrieveTest() {
  if (!ensureBaseID()) return;
  if (!retrieveQuery.value.trim()) {
    ElMessage.warning(t("pageKnowledge.toast.inputRetrieveQuery"));
    return;
  }
  contentLoading.value = true;
  try {
    const resp = await api.knowledgeRetrieveTest(
      currentBaseID.value,
      retrieveQuery.value.trim(),
      retrieveTopK.value,
      getRetrieveTimeoutSec(),
    );
    retrieveResults.value = resp?.data?.data?.results || [];
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.retrieveFailed"));
  } finally {
    contentLoading.value = false;
  }
}

function cancelQATestStream() {
  if (qaAbortController) {
    qaAbortController.abort();
    qaAbortController = null;
  }
  qaStreaming.value = false;
}

function parseSSEChunk(chunk) {
  const blocks = chunk.split("\n\n");
  const events = [];
  for (const block of blocks) {
    const text = String(block || "").trim();
    if (!text) continue;
    let event = "message";
    const dataLines = [];
    for (const line of text.split("\n")) {
      if (line.startsWith("event:")) {
        event = line.slice(6).trim() || "message";
        continue;
      }
      if (line.startsWith("data:")) {
        dataLines.push(line.slice(5).trim());
      }
    }
    if (!dataLines.length) continue;
    let payload = {};
    const dataText = dataLines.join("\n");
    try {
      payload = JSON.parse(dataText);
    } catch {
      payload = { message: dataText };
    }
    events.push({ event, payload });
  }
  return events;
}

async function runQATest() {
  if (!ensureBaseID()) return;
  if (qaStreaming.value) return;
  const query = qaInput.value.trim();
  if (!query) {
    ElMessage.warning(t("pageKnowledge.toast.inputQuestion"));
    return;
  }
  if (!activeAPIModel.value) {
    ElMessage.warning(t("pageKnowledge.toast.enableModelFirst"));
    return;
  }
  qaStreaming.value = true;
  const historyItem = reactive({
    question: query,
    answer: "正在生成...",
    sources: [],
    chunks: [],
    model_provider: "",
    model_name: "",
    streaming: true,
  });
  qaHistory.value.push(historyItem);
  qaInput.value = "";
  const timeoutMs = Math.max(15000, Math.min(900000, getQATimeoutSec() * 1000 + 10000));
  const controller = new AbortController();
  qaAbortController = controller;
  const timer = window.setTimeout(() => controller.abort(), timeoutMs);
  try {
    const resp = await fetch(`${api.baseURL}/knowledge-bases/${currentBaseID.value}/qa-test-stream`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
        ...(store.token ? { Authorization: `Bearer ${store.token}` } : {}),
      },
      body: JSON.stringify({
        query,
        top_k: retrieveTopK.value,
      }),
      signal: controller.signal,
    });
    if (!resp.ok) {
      let message = t("pageKnowledge.toast.qaFailed");
      try {
        const errPayload = await resp.json();
        message = String(errPayload?.msg || errPayload?.message || message);
      } catch {
        const errText = await resp.text();
        if (String(errText || "").trim()) {
          message = String(errText).trim();
        }
      }
      throw new Error(message);
    }
    if (!resp.body) {
      throw new Error(t("pageKnowledge.toast.qaFailed"));
    }

    const reader = resp.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";
    let finished = false;
    while (!finished) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split("\n\n");
      buffer = parts.pop() || "";
      for (const chunk of parts) {
        const events = parseSSEChunk(chunk);
        for (const item of events) {
          if (item.event === "delta") {
            historyItem.answer = String(item.payload?.answer || historyItem.answer || "");
            continue;
          }
          if (item.event === "final") {
            historyItem.answer = String(item.payload?.answer || historyItem.answer || "");
            historyItem.sources = item.payload?.sources || [];
            historyItem.chunks = item.payload?.chunks || [];
            historyItem.model_provider = String(item.payload?.model_provider || "");
            historyItem.model_name = String(item.payload?.model_name || "");
            historyItem.streaming = false;
            finished = true;
            break;
          }
          if (item.event === "error") {
            throw new Error(String(item.payload?.message || t("pageKnowledge.toast.qaFailed")));
          }
        }
      }
    }

    if (!finished && buffer.trim()) {
      const tailEvents = parseSSEChunk(buffer);
      for (const item of tailEvents) {
        if (item.event === "final") {
          historyItem.answer = String(item.payload?.answer || historyItem.answer || "");
          historyItem.sources = item.payload?.sources || [];
          historyItem.chunks = item.payload?.chunks || [];
          historyItem.model_provider = String(item.payload?.model_provider || "");
          historyItem.model_name = String(item.payload?.model_name || "");
          historyItem.streaming = false;
          finished = true;
          break;
        }
      }
    }
    if (!finished) {
      historyItem.streaming = false;
      if (!String(historyItem.answer || "").trim()) {
        historyItem.answer = t("pageKnowledge.toast.qaFailed");
      }
    }
  } catch (err) {
    qaHistory.value = qaHistory.value.filter((item) => item !== historyItem);
    const message = String(err?.message || "");
    if (err?.name === "AbortError" || message.includes("timeout")) {
      ElMessage.error(t("pageKnowledge.toast.qaTimeout"));
    } else {
      ElMessage.error(message || t("pageKnowledge.toast.qaFailed"));
    }
  } finally {
    window.clearTimeout(timer);
    historyItem.streaming = false;
    if (qaAbortController === controller) {
      qaAbortController = null;
    }
    qaStreaming.value = false;
  }
}

async function regenerate(item) {
  if (qaStreaming.value) return;
  qaInput.value = item.question || "";
  await runQATest();
}

async function feedback(item, helpful) {
  try {
    const sourceDocs = (item.sources || []).map((s) => s.doc_name).filter(Boolean);
    await api.saveKnowledgeFeedback(currentBaseID.value, {
      question: item.question || "",
      answer: item.answer || "",
      helpful,
      source_docs: sourceDocs,
    });
    ElMessage.success(helpful ? t("pageKnowledge.toast.feedbackPositive") : t("pageKnowledge.toast.feedbackNegative"));
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.feedbackFailed"));
  }
}

function clearQAHistory() {
  cancelQATestStream();
  qaHistory.value = [];
}

function onBaseChanged() {
  cancelQATestStream();
  if (activeTab.value === "documents") loadDocuments();
  else if (activeTab.value === "chunks") loadChunks();
  else if (activeTab.value === "retrieve") retrieveResults.value = [];
}

async function loadAPIModels() {
  try {
    const resp = await api.listAPIModels();
    apiModels.value = resp?.data?.data?.data || [];
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.loadModelListFailed"));
  }
}

function openModelDialog() {
  void loadAPIModels();
  llmDialogVisible.value = true;
}

function openAPIModelForm(item = null) {
  if (!item) {
    apiModelForm.value = {
      id: 0,
      model_type: activeModelType.value,
      name: "",
      provider: activeModelType.value === "embedding" ? "ollama" : "openai",
      api_key: "",
      api_key_mask: "",
      base_url: activeModelType.value === "embedding" ? ollamaDefaultURL : "",
      model_name: "",
      dims: 384,
      top_k: 20,
      top_n: 5,
      timeout_sec: 60,
      temperature: 0.7,
      top_p: 0.9,
      max_tokens: 512,
      is_default: false,
      status: 1,
    };
  } else {
    apiModelForm.value = {
      id: Number(item.id || 0),
      model_type: item.model_type || "chat",
      name: item.name || "",
      provider: item.provider || "openai",
      api_key: String(item.api_key || ""),
      api_key_mask: item.api_key_mask || "",
      base_url: item.base_url || "",
      model_name: item.model_name || "",
      dims: Number(item.dims || 384),
      top_k: Number(item.top_k || 20),
      top_n: Number(item.top_n || 5),
      timeout_sec: Number(item.timeout_sec || 60),
      temperature: Number(item.temperature || 0.7),
      top_p: Number(item.top_p || 0.9),
      max_tokens: Number(item.max_tokens || 512),
      is_default: Boolean(item.is_default),
      status: Number(item.status || 0) === 1 ? 1 : 0,
    };
    api.getAPIModel(item.id).then((resp) => {
      const detail = resp?.data?.data?.item || {};
      if (Number(detail.id || 0) !== Number(apiModelForm.value.id || 0)) return;
      apiModelForm.value.api_key = String(detail.api_key || "");
    }).catch((err) => {
      ElMessage.error(err?.message || t("pageKnowledge.toast.loadModelDetailFailed"));
    });
  }
  apiModelFormVisible.value = true;
}

const ollamaDefaultURL = "http://localhost:11434";

function onProviderChange(provider) {
  if (provider === "ollama") {
    if (!apiModelForm.value.base_url || apiModelForm.value.base_url === ollamaDefaultURL) {
      apiModelForm.value.base_url = ollamaDefaultURL;
    }
  } else {
    if (apiModelForm.value.base_url === ollamaDefaultURL) {
      apiModelForm.value.base_url = "";
    }
  }
}

async function saveAPIModel() {
  savingAPIModel.value = true;
  try {
    const payload = {
      model_type: String(apiModelForm.value.model_type || "chat"),
      name: String(apiModelForm.value.name || "").trim(),
      provider: String(apiModelForm.value.provider || "").trim().toLowerCase(),
      api_key: String(apiModelForm.value.api_key || "").trim(),
      base_url: String(apiModelForm.value.base_url || "").trim(),
      model_name: String(apiModelForm.value.model_name || "").trim(),
      dims: Number(apiModelForm.value.dims || 384),
      top_k: Number(apiModelForm.value.top_k || 20),
      top_n: Number(apiModelForm.value.top_n || 5),
      timeout_sec: Number(apiModelForm.value.timeout_sec || 60),
      temperature: Number(apiModelForm.value.temperature || 0.7),
      top_p: Number(apiModelForm.value.top_p || 0.9),
      max_tokens: Number(apiModelForm.value.max_tokens || 512),
      is_default: Boolean(apiModelForm.value.is_default),
      status: Number(apiModelForm.value.status || 0) === 1 ? 1 : 0,
    };
    if (!payload.provider || !payload.model_name) {
      ElMessage.warning(t("pageKnowledge.toast.modelRequiredFields"));
      return;
    }
    if (apiModelForm.value.id) {
      await api.updateAPIModel(apiModelForm.value.id, payload);
      ElMessage.success(t("pageKnowledge.toast.modelUpdated"));
    } else {
      await api.createAPIModel(payload);
      ElMessage.success(t("pageKnowledge.toast.modelCreated"));
    }
    apiModelFormVisible.value = false;
    await loadAPIModels();
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.saveModelFailed"));
  } finally {
    savingAPIModel.value = false;
  }
}

async function testAPIModel(item) {
  try {
    const resp = await api.testAPIModel(item.id);
    const data = resp?.data?.data || {};
    const preview = String(data.preview || "").trim();
    const inferMS = Number(data.infer_ms || 0);
    ElMessage.success(`${t("pageKnowledge.toast.connectSuccess")}${inferMS > 0 ? `, ${t("pageKnowledge.word.elapsed")} ${inferMS}ms` : ""}${preview ? `, ${t("pageKnowledge.word.returnValue")}: ${preview}` : ""}`);
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.connectFailed"));
  }
}

async function setDefaultAPIModel(item) {
  try {
    await api.setDefaultAPIModel(item.id);
    ElMessage.success(t("pageKnowledge.toast.setDefaultSuccess"));
    await loadAPIModels();
  } catch (err) {
    ElMessage.error(err.message || t("pageKnowledge.toast.setDefaultFailed"));
  }
}

async function removeAPIModel(item) {
  try {
    await ElMessageBox.confirm(`${t("pageKnowledge.confirm.deleteModel")}「${item.model_name}」？`, t("pageKnowledge.confirm.title"), { type: "warning" });
    await api.deleteAPIModel(item.id);
    ElMessage.success(t("pageKnowledge.toast.modelDeleted"));
    await loadAPIModels();
  } catch (err) {
    if (err !== "cancel" && err !== "close") {
      ElMessage.error(err.message || t("pageKnowledge.toast.deleteFailed"));
    }
  }
}

async function triggerRebuild(item) {
  try {
    await ElMessageBox.confirm(t("pageKnowledge.confirm.rebuild"), t("pageKnowledge.confirm.title"), { type: "warning" });
    const resp = await api.triggerRebuild(item.id);
    rebuildTask.value = resp?.data?.data?.task || null;
    ElMessage.success(t("pageKnowledge.toast.rebuildStarted"));
    startRebuildPoll(item.id);
  } catch (err) {
    if (err !== "cancel" && err !== "close") {
      ElMessage.error(err.message || t("pageKnowledge.toast.rebuildFailed"));
    }
  }
}

function startRebuildPoll(configID) {
  stopRebuildPoll();
  rebuildPollTimer = setInterval(async () => {
    try {
      const resp = await api.getRebuildStatus(configID);
      const task = resp?.data?.data?.task;
      if (task) {
        rebuildTask.value = task;
        if (task.status === "completed" || task.status === "failed") {
          stopRebuildPoll();
        }
      } else {
        stopRebuildPoll();
      }
    } catch {
      stopRebuildPoll();
    }
  }, 2000);
}

function stopRebuildPoll() {
  if (rebuildPollTimer) {
    clearInterval(rebuildPollTimer);
    rebuildPollTimer = null;
  }
}

const rebuildTaskStatusType = computed(() => {
  if (!rebuildTask.value) return "info";
  const s = rebuildTask.value.status;
  if (s === "completed") return "success";
  if (s === "running") return "warning";
  if (s === "failed") return "danger";
  return "info";
});

const rebuildTaskProgressStatus = computed(() => {
  if (!rebuildTask.value) return "";
  const s = rebuildTask.value.status;
  if (s === "completed") return "success";
  if (s === "failed") return "exception";
  return "";
});

watch(activeTab, async (tab) => {
  if (!currentBaseID.value) return;
  if (tab === "documents") await loadDocuments();
  if (tab === "chunks") await loadChunks();
});

onMounted(async () => {
  await Promise.all([loadAppOptions(), loadBases(), loadAPIModels(), refreshKnowledgeBackendHealth(false)]);
  if (currentBaseID.value) {
    await Promise.all([loadDocuments(), loadChunks()]);
  }
});

onBeforeUnmount(() => {
  cancelQATestStream();
  stopRebuildPoll();
});

</script>

<style scoped>
.knowledge-page {
  position: relative;
  min-height: calc(100vh - 64px);
  padding: 24px;
  background: linear-gradient(160deg, #f7fbff 0%, #f9fafb 45%, #eff6ff 100%);
}

.kb-ambient {
  position: absolute;
  top: -120px;
  right: -120px;
  width: 340px;
  height: 340px;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(37, 99, 235, 0.14) 0%, rgba(37, 99, 235, 0) 70%);
  pointer-events: none;
}

.kb-header-card,
.kb-selector-card,
.kb-tabs-wrap,
.kb-panel {
  position: relative;
  z-index: 1;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(203, 213, 225, 0.5);
  border-radius: 16px;
  backdrop-filter: blur(6px);
  box-shadow: 0 12px 36px rgba(15, 23, 42, 0.06);
}

.kb-header-card {
  padding: 18px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.kb-title {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.kb-subtitle {
  margin: 6px 0 0;
  color: #64748b;
  font-size: 13px;
}

.header-actions {
  display: inline-flex;
  gap: 10px;
}

.kb-create-btn {
  border-radius: 12px;
  border-color: #bfdbfe;
  color: #1d4ed8;
  font-weight: 600;
}

.kb-delete-btn {
  border-radius: 12px;
  border-color: #fecaca;
  color: #b91c1c;
  font-weight: 600;
}

.kb-selector-card {
  margin-top: 14px;
  padding: 14px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.selector-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.selector-label {
  font-size: 13px;
  color: #64748b;
  white-space: nowrap;
}

.kb-selector {
  width: 320px;
}

.selector-stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.stat-pill {
  padding: 4px 10px;
  border-radius: 999px;
  background: #eff6ff;
  border: 1px solid #dbeafe;
  font-size: 12px;
  color: #1d4ed8;
  font-weight: 500;
}

.kb-tabs-wrap {
  margin-top: 14px;
  padding: 0 16px;
  display: flex;
  gap: 18px;
  border-bottom: 1px solid #e2e8f0;
}

.kb-tab {
  border: none;
  background: transparent;
  padding: 14px 2px 12px;
  color: #64748b;
  font-size: 14px;
  border-bottom: 2px solid transparent;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.kb-tab:hover {
  color: #2563eb;
}

.kb-tab.active {
  color: #1d4ed8;
  border-bottom-color: #1d4ed8;
}

.tab-icon {
  opacity: 0.9;
}

.kb-panel {
  margin-top: 12px;
  padding: 16px;
  min-height: 560px;
}

.kb-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.toolbar-btn {
  border-radius: 10px;
  border-color: #bfdbfe;
  color: #1d4ed8;
  font-weight: 600;
}

.toolbar-btn.primary {
  border-color: transparent;
}

.kb-search {
  margin-left: auto;
  width: 320px;
}

.doc-filter,
.topk-select,
.model-select {
  width: 180px;
}

.hidden-input {
  display: none;
}

.doc-list {
  border-top: 1px solid #f1f5f9;
}

.doc-item {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 12px;
  padding: 12px 8px;
  border-bottom: 1px solid #f1f5f9;
}

.doc-name {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.doc-meta {
  margin-top: 3px;
  display: inline-flex;
  gap: 8px;
  color: #64748b;
  font-size: 12px;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 3px 10px;
  font-size: 12px;
  border: 1px solid transparent;
}

.status-chip.ok {
  color: #166534;
  background: #dcfce7;
  border-color: #bbf7d0;
}

.status-chip.running {
  color: #1d4ed8;
  background: #dbeafe;
  border-color: #bfdbfe;
}

.status-chip.pending {
  color: #9a3412;
  background: #ffedd5;
  border-color: #fed7aa;
}

.status-chip.bad {
  color: #b91c1c;
  background: #fee2e2;
  border-color: #fecaca;
}

.doc-error {
  color: #dc2626;
  font-size: 12px;
}

.doc-more {
  display: inline-block;
  color: #64748b;
  font-size: 20px;
  cursor: pointer;
  line-height: 1;
  padding: 0 6px;
}

.danger-item {
  color: #dc2626;
}

.chunk-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chunk-card {
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #ffffff;
  padding: 12px 12px 10px;
}

.chunk-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.chunk-doc {
  font-size: 12px;
  color: #1d4ed8;
  background: #eff6ff;
  padding: 3px 8px;
  border-radius: 999px;
}

.chunk-seq {
  font-size: 12px;
  color: #64748b;
}

.chunk-actions {
  margin-left: auto;
  display: inline-flex;
  gap: 8px;
}

.text-btn {
  border: none;
  background: transparent;
  color: #2563eb;
  font-size: 12px;
  cursor: pointer;
}

.text-btn.danger {
  color: #dc2626;
}

.chunk-body {
  margin: 0;
  color: #1f2937;
  font-size: 14px;
  line-height: 1.62;
  white-space: pre-wrap;
}

.chunk-meta {
  margin-top: 8px;
  font-size: 12px;
  color: #64748b;
}

.edit-actions {
  margin-top: 8px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.retrieve-list {
  display: flex;
  flex-direction: column;
}

.retrieve-item {
  border-bottom: 1px solid #f1f5f9;
  padding: 12px 0;
}

.retrieve-line {
  display: flex;
  align-items: center;
  gap: 10px;
}

.retrieve-score {
  color: #1d4ed8;
  font-weight: 700;
}

.retrieve-doc {
  color: #334155;
  font-size: 13px;
}

.retrieve-body {
  margin-top: 6px;
  color: #334155;
  line-height: 1.65;
  white-space: pre-wrap;
}

.model-hint {
  font-size: 12px;
  color: #64748b;
}

.qa-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.qa-card {
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  background: #ffffff;
  padding: 12px;
}

.qa-question,
.qa-answer {
  white-space: pre-wrap;
  line-height: 1.6;
  color: #0f172a;
}

.qa-answer {
  margin-top: 8px;
  color: #111827;
}

.qa-sources {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 12px;
  color: #64748b;
}

.src-chip {
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid #dbeafe;
  background: #eff6ff;
  color: #1d4ed8;
}

.qa-meta {
  margin-top: 6px;
  font-size: 12px;
  color: #64748b;
}

.qa-actions {
  margin-top: 10px;
  display: flex;
  gap: 8px;
}

.icon-btn {
  border: 1px solid #dbeafe;
  background: #fff;
  color: #1d4ed8;
  border-radius: 8px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
}

.icon-btn.active {
  background: #eff6ff;
  border-color: #93c5fd;
}

.qa-input-row {
  margin-top: 14px;
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
}

.kb-footer {
  margin-top: 12px;
  color: #94a3b8;
  font-size: 12px;
}

:deep(.llm-config-dialog .el-dialog__body) {
  padding: 24px;
  overflow: visible !important;
}

.model-type-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  border-bottom: 2px solid #e5e7eb;
  padding-bottom: 0;
}

.model-type-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  color: #6b7280;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
}

.model-type-tab:hover {
  color: #3b82f6;
}

.model-type-tab.active {
  color: #3b82f6;
  border-bottom-color: #3b82f6;
  font-weight: 600;
}

.model-type-icon {
  font-size: 16px;
}

.model-status-cards {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.model-status-card {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #f9fafb;
  transition: all 0.2s;
}

.model-status-card.has-default {
  background: #f0fdf4;
  border-color: #86efac;
}

.model-status-icon {
  font-size: 22px;
}

.model-status-info {
  flex: 1;
  min-width: 0;
}

.model-status-label {
  font-size: 12px;
  color: #6b7280;
}

.model-status-value {
  font-size: 13px;
  font-weight: 600;
  color: #111827;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rebuild-progress {
  margin-top: 8px;
}

@media (max-width: 1024px) {
  .kb-selector-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .kb-search {
    width: 100%;
    margin-left: 0;
  }

  .kb-toolbar {
    flex-wrap: wrap;
  }

  .qa-input-row {
    grid-template-columns: 1fr;
  }

}
</style>
