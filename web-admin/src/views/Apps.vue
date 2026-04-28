<template>
  <div class="p-6">
    <div class="mb-6">
      <h2 class="text-2xl font-bold text-gray-800">{{ t("pageApps.title") }}</h2>
      <p class="text-gray-600 mt-1">{{ t("pageApps.subtitle") }}</p>
    </div>

    <div class="bg-white rounded-lg shadow">
      <div class="p-4 border-b border-gray-200 flex justify-between items-center">
        <div class="flex items-center gap-4">
          <el-input
            v-model="searchKeyword"
            :placeholder="t('pageApps.searchPlaceholder')"
            clearable
            style="width: 300px"
            @clear="loadApps"
            @keyup.enter="loadApps"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" @click="loadApps">{{ t("pageApps.search") }}</el-button>
          <el-select
            v-model="statusFilter"
            :placeholder="t('pageApps.statusFilter')"
            clearable
            style="width: 150px"
            @change="loadApps"
          >
            <el-option :label="t('pageApps.all')" value="" />
            <el-option :label="t('pageApps.enabled')" :value="1" />
            <el-option :label="t('pageApps.disabled')" :value="0" />
          </el-select>
        </div>
        <el-button type="primary" @click="openDialog()">
          <el-icon class="mr-1"><Plus /></el-icon>
          {{ t("pageApps.addApp") }}
        </el-button>
      </div>

      <el-table :data="apps" v-loading="loading" stripe>
        <el-table-column prop="name" :label="t('pageApps.appName')" width="100" align="center" />
        <el-table-column prop="app_id" label="AppID" width="180" align="center" />
        <el-table-column prop="logo" :label="t('pageApps.logo')" width="100" align="center">
          <template #default="{ row }">
            <el-image
              v-if="row.logo"
              :src="row.logo"
              fit="cover"
              style="width: 40px; height: 40px; border-radius: 4px"
            />
            <div
              v-else
              class="w-10 h-10 bg-gray-200 rounded flex items-center justify-center text-gray-400 text-xs"
            >
              {{ t("pageApps.emptyLogo") }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="allow_domain" :label="t('pageApps.allowDomain')" min-width="120" align="center" show-overflow-tooltip />
        <el-table-column prop="contact" :label="t('pageApps.contact')" width="120" align="center" />
        <el-table-column prop="status" :label="t('pageApps.status')" width="50" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? t("pageApps.enabled") : t("pageApps.disabled") }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('pageApps.createdAt')" width="120" align="center">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('pageApps.actions')" width="120" align="center">
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

      <div class="p-4 border-t border-gray-200 flex justify-end">
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
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item :label="t('pageApps.appName')" prop="name" class="mr-8">
          <el-input v-model="form.name" :placeholder="t('pageApps.appNameInput')" />
        </el-form-item>
        <el-form-item label="AppID" prop="app_id" class="mr-8">
          <el-input v-model="form.app_id" :placeholder="t('pageApps.appIdInput')" :disabled="!!form.id" />
          <div v-if="!form.id" class="text-xs text-gray-500 mt-1">{{ t("pageApps.appIdAuto") }}</div>
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
        <el-button class="mr-8!" type="primary" @click="submitForm" :loading="submitting">{{ t("pageApps.confirm") }}</el-button>
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
import { ref, onMounted } from "vue";
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
    apps.value = response.data?.data?.data || [];
    total.value = response.data?.data?.total || 0;
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

const formatDate = (dateString) => {
  if (!dateString) return "-";
  const date = new Date(dateString);
  return date.toLocaleString(localeRef.value || "zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
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
:deep(.el-table) {
  font-size: 14px;
}

:deep(.el-table .cell) {
  padding: 8px 0;
}

.code-block {
  max-height: 360px;
  overflow: auto;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  padding: 12px;
  font-size: 12px;
  line-height: 1.5;
}
</style>
