<template>
  <div class="admin-console-page apps-console">
    <div class="console-hero">
      <div class="console-hero__copy">
        <span class="console-kicker">{{ t("pageApps.title") }}</span>
        <h1>{{ t("pageApps.title") }}</h1>
        <p>{{ t("pageApps.subtitle") }}</p>
      </div>
      <div class="console-hero__actions">
        <el-button type="primary" class="hero-button" @click="loadApps">{{ t("action.refresh") }}</el-button>
        <el-button type="primary" class="hero-button" @click="openDialog()">
          <el-icon><Plus /></el-icon>
          {{ t("pageApps.addApp") }}
        </el-button>
      </div>
    </div>

    <div class="console-overview">
      <article v-for="item in overviewCards" :key="item.key" class="overview-card" :class="item.tone">
        <span class="overview-card__label">{{ item.label }}</span>
        <strong class="overview-card__value">{{ item.value }}</strong>
      </article>
    </div>

    <el-card class="console-panel filter-panel" shadow="never">
      <div class="panel-head">
        <div>
          <h2>{{ t("pageApps.panel.filters") }}</h2>
          <p>{{ t("pageApps.panel.filtersDesc") }}</p>
        </div>
      </div>
      <div class="filter-grid">
        <el-input
          v-model="searchKeyword"
          :placeholder="t('pageApps.searchPlaceholder')"
          clearable
          @clear="applyFilters"
          @keyup.enter="applyFilters"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-select
          v-model="statusFilter"
          :placeholder="t('pageApps.statusFilter')"
          clearable
          @change="applyFilters"
        >
          <el-option :label="t('pageApps.all')" value="" />
          <el-option :label="t('pageApps.enabled')" :value="1" />
          <el-option :label="t('pageApps.disabled')" :value="0" />
        </el-select>
        <div class="filter-actions">
          <el-button type="primary" class="filter-submit" @click="applyFilters">{{ t("pageApps.search") }}</el-button>
        </div>
      </div>
    </el-card>

    <el-card class="console-panel table-panel" shadow="never">
      <div class="panel-head panel-head--split">
        <div>
          <h2>{{ t("pageApps.panel.list") }}</h2>
          <p>{{ t("pageApps.panel.listDesc") }}</p>
        </div>
        <div class="panel-badge">
          <span>{{ total }}</span>
          <small>APP</small>
        </div>
      </div>

      <div class="table-shell">
        <el-table :data="apps" v-loading="loading" stripe class="admin-console-table">
          <el-table-column prop="name" :label="t('pageApps.appName')" min-width="140" />
          <el-table-column prop="app_id" label="AppID" min-width="180" />
          <el-table-column prop="logo" :label="t('pageApps.logo')" width="100" align="center">
            <template #default="{ row }">
              <el-image
                v-if="row.logo"
                :src="row.logo"
                fit="cover"
                class="app-logo"
              />
              <div v-else class="app-logo app-logo--empty">{{ t("pageApps.emptyLogo") }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="allow_domain" :label="t('pageApps.allowDomain')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="contact" :label="t('pageApps.contact')" min-width="140" show-overflow-tooltip />
          <el-table-column prop="status" :label="t('pageApps.status')" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="light">
                {{ row.status === 1 ? t("pageApps.enabled") : t("pageApps.disabled") }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('pageApps.createdAt')" width="180">
            <template #default="{ row }">{{ formatDate(row.created_at || row.CreatedAt || row.createdAt) }}</template>
          </el-table-column>
          <el-table-column :label="t('pageApps.actions')" width="200" fixed="right">
            <template #default="{ row }">
              <el-button type="success" link size="small" @click="showEmbedCode(row)">{{ t("pageApps.embedCode") }}</el-button>
              <el-button type="primary" link size="small" @click="openDialog(row)">{{ t("pageApps.editApp") }}</el-button>
              <el-button :type="row.status === 1 ? 'warning' : 'success'" link size="small" @click="toggleStatus(row)">
                {{ row.status === 1 ? t("pageApps.disabled") : t("pageApps.enabled") }}
              </el-button>
              <el-button type="danger" link size="small" @click="deleteApp(row)">{{ t("pageApps.deleteApp") }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="table-pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadApps"
          @current-change="loadApps"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" @close="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('pageApps.appName')" prop="name" class="mr-8">
          <el-input v-model="form.name" :placeholder="t('pageApps.appNameInput')" />
        </el-form-item>
        <el-form-item label="AppID" prop="app_id" class="mr-8">
          <el-input v-model="form.app_id" :placeholder="t('pageApps.appIdInput')" :disabled="!!form.id" />
          <div v-if="!form.id" class="mt-1 text-xs text-slate-500">{{ t("pageApps.appIdAuto") }}</div>
        </el-form-item>
        <el-form-item :label="t('pageApps.logo')" prop="logo" class="mr-8">
          <el-input v-model="form.logo" :placeholder="t('pageApps.logoInput')" />
        </el-form-item>
        <el-form-item :label="t('pageApps.allowDomain')" prop="allow_domain" class="mr-8">
          <el-input v-model="form.allow_domain" :placeholder="t('pageApps.allowDomainInput')" />
        </el-form-item>
        <el-form-item :label="t('pageSettings.welcomeMsg')" prop="welcome_msg" class="mr-8">
          <el-input v-model="form.welcome_msg" type="textarea" :rows="3" :placeholder="t('pageApps.welcomeInput')" />
        </el-form-item>
        <el-form-item :label="t('pageApps.contact')" prop="contact" class="mr-8">
          <el-input v-model="form.contact" :placeholder="t('pageApps.contactInput')" />
        </el-form-item>
        <el-form-item :label="t('pageApps.status')" prop="status">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">{{ t("pageApps.enabled") }}</el-radio>
            <el-radio :value="0">{{ t("pageApps.disabled") }}</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("pageApps.cancel") }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">{{ t("pageApps.confirm") }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="codeDialogVisible" :title="t('pageApps.embedCodeTitle')" width="760px">
      <pre class="code-block">{{ embedCode }}</pre>
      <template #footer>
        <el-button @click="copyEmbedCode">{{ t("pageApps.copyCode") }}</el-button>
        <el-button type="primary" @click="codeDialogVisible = false">{{ t("pageApps.close") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Search, Plus } from "@element-plus/icons-vue";
import api from "@/script/api";
import { t } from "@/script/i18n-text";
import { localeRef } from "@/script/i18n";

const loading = ref(false);
const submitting = ref(false);
const dialogVisible = ref(false);
const dialogTitle = ref(t("pageApps.dialogAdd"));
const isUpdateMode = ref(false);
const searchKeyword = ref("");
const statusFilter = ref("");
const currentPage = ref(1);
const pageSize = ref(10);
const total = ref(0);
const apps = ref([]);
const formRef = ref(null);
const codeDialogVisible = ref(false);
const embedCode = ref("");

const form = ref({
  id: null,
  name: "",
  app_id: "",
  logo: "",
  allow_domain: "",
  welcome_msg: "",
  contact: "",
  status: 1,
});

const rules = {
  name: [{ required: true, message: t("pageApps.appNameInput"), trigger: "blur" }],
  status: [{ required: true, message: t("pageApps.statusFilter"), trigger: "change" }],
};

const overviewCards = computed(() => {
  const rows = apps.value || [];
  return [
    { key: "in-view", label: t("pageApps.overview.inView"), value: rows.length, tone: "" },
    { key: "enabled", label: t("pageApps.overview.enabled"), value: rows.filter((item) => item.status === 1).length, tone: "overview-card--success" },
    { key: "disabled", label: t("pageApps.overview.disabled"), value: rows.filter((item) => item.status !== 1).length, tone: "overview-card--warning" },
    { key: "contact", label: t("pageApps.overview.contactReady"), value: rows.filter((item) => String(item.contact || "").trim()).length, tone: "overview-card--info" },
  ];
});

function parseDateValue(value) {
  if (value === null || value === undefined || value === "") return null;
  const normalized = typeof value === "string" && /^\d+$/.test(value) ? Number(value) : value;
  if (typeof normalized === "number" && Number.isFinite(normalized)) {
    const millis = normalized > 1e12 ? normalized : normalized * 1000;
    const date = new Date(millis);
    return Number.isNaN(date.getTime()) ? null : date;
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

function formatDate(value) {
  const date = parseDateValue(value);
  if (!date) return "-";
  return date.toLocaleString(localeRef.value || "zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function applyFilters() {
  currentPage.value = 1;
  loadApps();
}

const loadApps = async () => {
  loading.value = true;
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value,
    };
    if (searchKeyword.value) params.keyword = searchKeyword.value;
    if (statusFilter.value !== "") params.status = statusFilter.value;
    const response = await api.listApps(params);
    const list = response?.data?.data?.data || [];
    apps.value = list.map((item) => ({
      ...item,
      created_at: item.created_at ?? item.CreatedAt ?? item.createdAt ?? "",
    }));
    total.value = Number(response?.data?.data?.total || 0);
  } catch (error) {
    ElMessage.error(t("pageApps.loadFailed"));
    console.error(error);
  } finally {
    loading.value = false;
  }
};

const openDialog = (row = null) => {
  if (row) {
    dialogTitle.value = t("pageApps.dialogEdit");
    isUpdateMode.value = true;
    form.value = { ...row };
  } else {
    dialogTitle.value = t("pageApps.dialogAdd");
    isUpdateMode.value = false;
    resetForm();
  }
  dialogVisible.value = true;
};

const resetForm = () => {
  form.value = {
    id: null,
    name: "",
    app_id: "",
    logo: "",
    allow_domain: "",
    welcome_msg: "",
    contact: "",
    status: 1,
  };
  formRef.value?.clearValidate();
};

const submitForm = async () => {
  if (!formRef.value) return;
  await formRef.value.validate();
  submitting.value = true;
  try {
    if (isUpdateMode.value) {
      await api.updateApp(form.value.app_id, form.value);
      ElMessage.success(t("pageApps.updateSuccess"));
    } else {
      await api.createApp(form.value);
      ElMessage.success(t("pageApps.createSuccess"));
    }
    dialogVisible.value = false;
    loadApps();
  } catch (error) {
    ElMessage.error(error.message || t("pageApps.opFailed"));
    console.error(error);
  } finally {
    submitting.value = false;
  }
};

const toggleStatus = async (row) => {
  const newStatus = row.status === 1 ? 0 : 1;
  const action = newStatus === 1 ? t("pageApps.enabled") : t("pageApps.disabled");
  try {
    await ElMessageBox.confirm(
      t("pageApps.toggleConfirm").replace("{action}", action),
      t("pageApps.hint"),
      { type: "warning" }
    );
    await api.updateApp(row.app_id, { status: newStatus });
    ElMessage.success(t("pageApps.toggleSuccess").replace("{action}", action));
    loadApps();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(t("pageApps.opFailed"));
      console.error(error);
    }
  }
};

const deleteApp = async (row) => {
  try {
    await ElMessageBox.confirm(
      t("pageApps.deleteConfirm").replace("{name}", row.name),
      t("pageApps.warning"),
      { type: "warning", confirmButtonText: t("pageApps.deleteConfirmBtn"), confirmButtonClass: "el-button--danger" }
    );
    await api.deleteApp(row.app_id);
    ElMessage.success(t("pageApps.deleteSuccess"));
    loadApps();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(t("pageApps.deleteFailed"));
      console.error(error);
    }
  }
};

const showEmbedCode = (row) => {
  const host = window.location.origin;
  embedCode.value = `<script\n  src="${host}/sdk/widget.min.js"\n  data-kefu-appid="${row.app_id}"\n  data-kefu-api-base-url="${host}"\n><\\/script>`;
  codeDialogVisible.value = true;
};

const copyEmbedCode = async () => {
  try {
    await navigator.clipboard.writeText(embedCode.value);
    ElMessage.success(t("pageApps.copied"));
  } catch {
    ElMessage.error(t("pageApps.copyFailed"));
  }
};

onMounted(() => {
  loadApps();
});
</script>

<style scoped>
.app-logo {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.75rem;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.08), rgba(14, 165, 233, 0.08));
}

.app-logo--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  font-size: 0.75rem;
}

.code-block {
  max-height: 360px;
  overflow: auto;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 12px;
  padding: 12px;
  font-size: 12px;
  line-height: 1.5;
}
</style>
