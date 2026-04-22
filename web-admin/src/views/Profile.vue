<template>
  <div class="p-8">
    <h1 class="text-2xl font-bold text-gray-800 mb-6">{{ t('pageProfile.title') }}</h1>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <el-card class="lg:col-span-1" v-loading="loading">
        <div class="text-center">
          <el-avatar :size="100" :src="userInfo.avatar">
            {{ userInfo.name?.charAt(0) || "U" }}
          </el-avatar>
          <h2 class="text-xl font-bold text-gray-800 mt-4">{{ userInfo.name }}</h2>
          <p class="text-gray-500">{{ userInfo.role === "admin" ? t('role.admin') : t('pageProfile.roleAgent') }}</p>
          <div class="flex justify-center gap-4 mt-4">
            <div class="text-center">
              <p class="text-2xl font-bold text-blue-600">{{ profileSummary.sessions_today }}</p>
              <p class="text-sm text-gray-500">{{ t('pageProfile.todaySessions') }}</p>
            </div>
            <div class="text-center">
              <p class="text-2xl font-bold text-green-600">{{ profileSummary.rating }}</p>
              <p class="text-sm text-gray-500">{{ t('pageProfile.rating') }}</p>
            </div>
          </div>
        </div>
      </el-card>

      <el-card class="lg:col-span-2">
        <template #header><span class="font-bold">{{ t('pageProfile.editProfile') }}</span></template>
        <el-form :model="form" label-width="100px">
          <el-form-item :label="t('pageProfile.username')">
            <el-input v-model="form.username" disabled />
          </el-form-item>
          <el-form-item :label="t('pageProfile.email')">
            <el-input v-model="form.email" />
          </el-form-item>
          <el-form-item :label="t('pageProfile.phone')">
            <el-input v-model="form.phone" />
          </el-form-item>
          <el-form-item :label="t('pageProfile.avatarUrl')">
            <el-input v-model="form.avatar" />
          </el-form-item>
          <el-form-item :label="t('pageProfile.bio')">
            <el-input v-model="form.bio" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="savingProfile" @click="saveProfile">{{ t('pageProfile.saveChanges') }}</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <el-card class="mt-6">
      <template #header><span class="font-bold">{{ t('pageProfile.changePassword') }}</span></template>
      <el-form :model="passwordForm" label-width="100px">
        <el-form-item :label="t('pageProfile.currentPassword')">
          <el-input v-model="passwordForm.currentPassword" type="password" show-password />
        </el-form-item>
        <el-form-item :label="t('pageProfile.newPassword')">
          <el-input v-model="passwordForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item :label="t('pageProfile.confirmPassword')">
          <el-input v-model="passwordForm.confirmPassword" type="password" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="savingPassword" @click="changePassword">{{ t('pageProfile.changePassword') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";

const loading = ref(false);
const savingProfile = ref(false);
const savingPassword = ref(false);
const userInfo = ref({
  id: 0,
  name: "",
  username: "",
  email: "",
  phone: "",
  avatar: "",
  bio: "",
  role: "",
});
const profileSummary = ref({ sessions_today: 0, rating: 0 });
const form = ref({ username: "", email: "", phone: "", avatar: "", bio: "" });
const passwordForm = ref({ currentPassword: "", newPassword: "", confirmPassword: "" });

async function loadProfile() {
  loading.value = true;
  try {
    const [infoResp, summaryResp] = await Promise.all([
      api.getUserInfo(),
      api.getProfileSummary().catch(() => ({ data: { data: {} } })),
    ]);

    const user = infoResp?.data?.data?.user || {};
    userInfo.value = {
      id: user.ID || 0,
      name: user.username || "",
      username: user.username || "",
      email: user.email || "",
      phone: user.phone || "",
      avatar: user.avatar || "",
      bio: user.bio || "",
      role: user.role || "",
    };
    form.value = {
      username: userInfo.value.username,
      email: userInfo.value.email,
      phone: userInfo.value.phone,
      avatar: userInfo.value.avatar,
      bio: userInfo.value.bio,
    };

    profileSummary.value = {
      sessions_today: Number(summaryResp?.data?.data?.sessions_today || 0),
      rating: Number(summaryResp?.data?.data?.rating || 0),
    };
  } catch (error) {
    ElMessage.error(error.message || t("pageProfile.loadFailed"));
  } finally {
    loading.value = false;
  }
}

async function saveProfile() {
  savingProfile.value = true;
  try {
    await api.updateProfile({
      avatar: form.value.avatar,
      email: form.value.email,
      phone: form.value.phone,
      bio: form.value.bio,
    });
    ElMessage.success(t("pageProfile.saveSuccess"));
    await loadProfile();
  } catch (error) {
    ElMessage.error(error.message || t("pageProfile.saveFailed"));
  } finally {
    savingProfile.value = false;
  }
}

async function changePassword() {
  if (!passwordForm.value.currentPassword) {
    ElMessage.error(t("pageProfile.inputCurrentPassword"));
    return;
  }
  if (!passwordForm.value.newPassword || passwordForm.value.newPassword.length < 8) {
    ElMessage.error(t("pageProfile.newPasswordTooShort"));
    return;
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    ElMessage.error(t("pageProfile.passwordMismatch"));
    return;
  }

  savingPassword.value = true;
  try {
    await api.changePassword(passwordForm.value.currentPassword, passwordForm.value.newPassword);
    ElMessage.success(t("pageProfile.passwordChanged"));
    passwordForm.value = { currentPassword: "", newPassword: "", confirmPassword: "" };
  } catch (error) {
    ElMessage.error(error.message || t("pageProfile.passwordChangeFailed"));
  } finally {
    savingPassword.value = false;
  }
}

onMounted(loadProfile);
</script>
