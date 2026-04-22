<template>
  <div class="p-8">
    <h1 class="text-2xl font-bold text-gray-800 mb-6">{{ t('pageStaff.title') }}</h1>

    <div class="mb-6">
      <el-button type="primary" @click="openAddDialog">
        <template #icon>
          <el-icon><Plus /></el-icon>
        </template>
        {{ t('pageStaff.addStaff') }}
      </el-button>
    </div>

    <el-card v-loading="loading">
      <div class="mb-4 flex gap-2">
        <el-button type="success" :disabled="selectedIDs.length === 0" @click="batchSetActive(true)">
          {{ t('pageStaff.batchEnable') }}
        </el-button>
        <el-button type="warning" :disabled="selectedIDs.length === 0" @click="batchSetActive(false)">
          {{ t('pageStaff.batchDisable') }}
        </el-button>
      </div>
      <el-table :data="staffList" stripe @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" :label="t('pageStaff.name')" align="center" />
        <el-table-column prop="role" :label="t('pageStaff.role')" align="center">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'" size="small">
              {{ row.role === 'admin' ? t('role.admin') : t('role.agent') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('pageStaff.status')" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'info'" size="small">
              {{ row.status === 'online' ? t('pageStaff.online') : t('pageStaff.offline') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sessions" :label="t('pageStaff.todaySessions')" width="100" align="center" />
        <el-table-column prop="rating" :label="t('pageStaff.rating')" width="100" align="center">
          <template #default="{ row }">
            <el-rate v-model="row.rating" disabled show-score text-color="#ff9900" />
          </template>
        </el-table-column>
        <el-table-column :label="t('pageStaff.actions')" width="200" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleEdit(row)">{{ t('pageStaff.edit') }}</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('pageStaff.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="showAddDialog" :title="isEditMode ? t('pageStaff.dialogEdit') : t('pageStaff.dialogAdd')" width="500px">
      <el-form :model="newStaff" label-width="80px">
        <el-form-item :label="t('pageStaff.username')" class="mr-8">
          <el-input v-model="newStaff.username" />
        </el-form-item>
        <el-form-item :label="t('pageStaff.password')" class="mr-8">
          <el-input v-model="newStaff.password" type="password" :placeholder="isEditMode ? t('pageStaff.passwordEditPlaceholder') : ''" />
        </el-form-item>
        <el-form-item :label="t('pageStaff.avatar')" class="mr-8">
          <el-input v-model="newStaff.avatar" />
        </el-form-item>
        <el-form-item :label="t('pageStaff.role')" class="mr-8">
          <el-select v-model="newStaff.role">
            <el-option :label="t('role.agent')" value="agent" />
            <el-option :label="t('role.admin')" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageStaff.active')" class="mr-8">
          <el-radio-group v-model="newStaff.active">
            <el-radio :label="true">{{ t('pageStaff.enable') }}</el-radio>
            <el-radio :label="false">{{ t('pageStaff.disable') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('pageStaff.serviceApps')" class="mr-8">
          <el-select v-model="newStaff.apps" multiple :placeholder="t('pageStaff.serviceAppsPlaceholder')">
            <el-option :label="t('pageStaff.allApps')" value="all" />
            <el-option v-for="app in appList" :key="app.ID" :label="app.name" :value="app.app_id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">{{ t('action.cancel') }}</el-button>
        <el-button type="primary" @click="handleAdd">{{ t('action.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { Plus } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import api from "../script/api.js";
import { t } from "@/script/i18n-text";

const showAddDialog = ref(false);
const loading = ref(false);
const isEditMode = ref(false);
const appList = ref([]);
const newStaff = ref({
  id: null,
  username: "",
  password: "",
  avatar: "",
  role: "agent",
  active: true,
  apps: [],
});

const staffList = ref([]);
const selectedIDs = ref([]);

const fetchAppList = async () => {
  try {
    const response = await api.listApps({ page: 1, page_size: 100 });
    if (response.data && response.data.data) {
      appList.value = response.data.data.data || response.data.data;
    }
  } catch (error) {
    console.error("fetch app list failed:", error);
  }
};

const fetchStaffList = async () => {
  loading.value = true;
  try {
    const [response, statsResp] = await Promise.all([
      api.listStaff(),
      api.listUserStats().catch(() => ({ data: { data: { data: [] } } })),
    ]);
    const statsRows = statsResp?.data?.data?.data || [];
    const sessionsMap = new Map(statsRows.map((item) => [item.username, Number(item.sessions || 0)]));
    if (response.data && response.data.data && response.data.data.users) {
      staffList.value = response.data.data.users.map((user) => ({
        id: user.ID,
        name: user.username,
        username: user.username,
        role: user.role,
        status: user.status === 1 ? "online" : "offline",
        sessions: sessionsMap.get(user.username) || 0,
        rating: Number(user.rating || 0),
        avatar: user.avatar || "",
        active: !!user.active,
        apps: user.apps ? JSON.parse(user.apps) : [],
      }));
      selectedIDs.value = [];
    }
  } catch (error) {
    console.error("fetch staff list failed:", error);
  } finally {
    loading.value = false;
  }
};

const handleAdd = async () => {
  try {
    const finalPassword = String(newStaff.value.password || "").trim();

    if (!newStaff.value.username || !String(newStaff.value.username).trim()) {
      ElMessage.warning(t("pageStaff.inputUsername"));
      return;
    }
    if (!isEditMode.value && !finalPassword) {
      ElMessage.warning(t("pageStaff.inputPassword"));
      return;
    }
    if (finalPassword && finalPassword.length < 8) {
      ElMessage.warning(t("pageStaff.passwordTooShort"));
      return;
    }

    const data = {
      username: String(newStaff.value.username || "").trim(),
      password: finalPassword,
      avatar: newStaff.value.avatar,
      role: newStaff.value.role,
      active: newStaff.value.active,
      apps: newStaff.value.apps.length > 0 ? newStaff.value.apps : ["all"],
    };

    if (isEditMode.value) {
      data.id = newStaff.value.id;
      await api.updateStaff(data);
      ElMessage.success(finalPassword ? t("pageStaff.updatedWithPassword") : t("pageStaff.updated"));
    } else {
      await api.createStaff(data);
      ElMessage.success(t("pageStaff.created"));
    }

    showAddDialog.value = false;
    resetForm();
    await fetchStaffList();
  } catch (error) {
    console.error("staff save failed:", error);
    ElMessage.error(error?.message || t("pageStaff.saveFailed"));
  }
};

const handleEdit = (row) => {
  isEditMode.value = true;
  newStaff.value = {
    id: row.id,
    username: row.username,
    password: "",
    avatar: row.avatar,
    role: row.role,
    active: !!row.active,
    apps: row.apps || [],
  };
  showAddDialog.value = true;
};

const resetForm = () => {
  newStaff.value = {
    id: null,
    username: "",
    password: "",
    avatar: "",
    role: "agent",
    active: true,
    apps: [],
  };
  isEditMode.value = false;
};

const openAddDialog = () => {
  resetForm();
  showAddDialog.value = true;
};

const handleDelete = async (row) => {
  try {
    await api.deleteStaff(row.id);
    await fetchStaffList();
  } catch (error) {
    console.error("delete staff failed:", error);
  }
};

const batchSetActive = async (active) => {
  if (!selectedIDs.value.length) return;
  try {
    await api.batchSetStaffActive(selectedIDs.value, active);
    await fetchStaffList();
  } catch (error) {
    console.error("batch set active failed:", error);
  }
};

const onSelectionChange = (rows) => {
  selectedIDs.value = (rows || [])
    .filter((item) => item.role !== "admin")
    .map((item) => item.id)
    .filter(Boolean);
};

onMounted(() => {
  fetchAppList();
  fetchStaffList();
});
</script>
