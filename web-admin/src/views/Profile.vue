<template>
  <div class="admin-console-page profile-console">
    <div class="console-hero">
      <div class="console-hero__copy">
        <span class="console-kicker">{{ t("pageProfile.title") }}</span>
        <h1>{{ t("pageProfile.title") }}</h1>
        <p>{{ t("pageProfile.subtitle") }}</p>
      </div>
    </div>

    <div class="console-overview">
      <article class="overview-card overview-card--info">
        <span class="overview-card__label">{{ t("pageProfile.overview.sessions") }}</span>
        <strong class="overview-card__value">{{ profileSummary.sessions_today }}</strong>
      </article>
      <article class="overview-card overview-card--success">
        <span class="overview-card__label">{{ t("pageProfile.overview.rating") }}</span>
        <strong class="overview-card__value">{{ profileSummary.rating }}</strong>
      </article>
    </div>

    <div class="profile-grid">
      <el-card class="console-panel profile-summary-panel" shadow="never" v-loading="loading">
        <div class="profile-summary">
          <el-avatar :size="96" :src="userInfo.avatar" class="profile-avatar">
            {{ userInfo.name?.charAt(0) || "U" }}
          </el-avatar>
          <h2>{{ userInfo.name }}</h2>
          <p>{{ userInfo.role === "admin" ? t("role.admin") : t("pageProfile.roleAgent") }}</p>
        </div>
      </el-card>

      <el-card class="console-panel" shadow="never">
        <div class="panel-head">
          <div>
            <h2>{{ t("pageProfile.panel.profile") }}</h2>
            <p>{{ t("pageProfile.panel.profileDesc") }}</p>
          </div>
        </div>
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
            <el-button type="primary" :loading="savingProfile" @click="saveProfile">{{ t("pageProfile.saveChanges") }}</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <el-card class="console-panel" shadow="never">
      <div class="panel-head">
        <div>
          <h2>{{ t("pageProfile.panel.security") }}</h2>
          <p>{{ t("pageProfile.panel.securityDesc") }}</p>
        </div>
      </div>
      <el-form :model="passwordForm" label-width="100px" class="password-form">
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
          <el-button type="primary" :loading="savingPassword" @click="changePassword">{{ t("pageProfile.changePassword") }}</el-button>
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

<style scoped>
.profile-grid {
  display: grid;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  gap: 1.5rem;
  margin-bottom: 1.5rem;
}

.profile-summary-panel {
  align-self: start;
}

.profile-summary {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 0.75rem;
}

.profile-avatar {
  box-shadow: 0 18px 36px rgba(99, 102, 241, 0.18);
}

.profile-summary h2 {
  margin: 0;
  color: #0f172a;
  font-size: 1.375rem;
  font-weight: 700;
}

.profile-summary p {
  margin: 0;
  color: #64748b;
}

.password-form {
  max-width: 720px;
}

@media (max-width: 900px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }
}
</style>
