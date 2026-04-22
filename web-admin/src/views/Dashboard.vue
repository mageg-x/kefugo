<template>
  <div class="p-8">
    <h1 class="text-2xl font-bold text-gray-800 mb-6">{{ t("pageDashboard.title") }}</h1>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <el-card class="bg-gradient-to-br from-blue-500 to-blue-600 text-white">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-blue-100 text-sm">{{ t("pageDashboard.todaySessions") }}</p>
            <p class="text-3xl font-bold mt-2">{{ dashboard.today_sessions }}</p>
          </div>
          <el-icon :size="40" class="opacity-80"><ChatDotRound /></el-icon>
        </div>
      </el-card>

      <el-card class="bg-gradient-to-br from-green-500 to-green-600 text-white">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-green-100 text-sm">{{ t("pageDashboard.resolved") }}</p>
            <p class="text-3xl font-bold mt-2">{{ dashboard.resolved_sessions }}</p>
          </div>
          <el-icon :size="40" class="opacity-80"><CircleCheck /></el-icon>
        </div>
      </el-card>

      <el-card class="bg-gradient-to-br from-orange-500 to-orange-600 text-white">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-orange-100 text-sm">{{ t("pageDashboard.pending") }}</p>
            <p class="text-3xl font-bold mt-2">{{ dashboard.pending_sessions }}</p>
          </div>
          <el-icon :size="40" class="opacity-80"><Clock /></el-icon>
        </div>
      </el-card>

      <el-card class="bg-gradient-to-br from-indigo-500 to-indigo-700 text-white">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-indigo-100 text-sm">{{ t("pageDashboard.onlineAgents") }}</p>
            <p class="text-3xl font-bold mt-2">{{ dashboard.online_agents }}</p>
          </div>
          <el-icon :size="40" class="opacity-80"><User /></el-icon>
        </div>
      </el-card>

      <el-card class="bg-gradient-to-br from-cyan-500 to-cyan-700 text-white">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-cyan-100 text-sm">{{ t("pageDashboard.activeSessions") }}</p>
            <p class="text-3xl font-bold mt-2">{{ dashboard.active_sessions }}</p>
          </div>
          <el-icon :size="40" class="opacity-80"><MessageBox /></el-icon>
        </div>
      </el-card>

      <el-card class="bg-gradient-to-br from-slate-500 to-slate-700 text-white">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-slate-100 text-sm">{{ t("pageDashboard.totalSessions") }}</p>
            <p class="text-3xl font-bold mt-2">{{ dashboard.total_sessions }}</p>
          </div>
          <el-icon :size="40" class="opacity-80"><DataLine /></el-icon>
        </div>
      </el-card>
    </div>

    <el-card v-loading="loading">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-bold">{{ t("pageDashboard.recentSessions") }}</span>
          <el-button size="small" @click="downloadSessions">{{ t("pageDashboard.exportCsv") }}</el-button>
        </div>
      </template>
      <el-table :data="dashboard.recent_sessions || []" stripe>
        <el-table-column prop="sid" :label="t('pageDashboard.sessionId')" min-width="220" />
        <el-table-column prop="visitor_id" :label="t('pageDashboard.visitor')" min-width="120" />
        <el-table-column prop="agent_id" :label="t('pageDashboard.agent')" min-width="120">
          <template #default="{ row }">
            {{ row.agent_id || "-" }}
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('pageDashboard.status')" width="120">
          <template #default="{ row }">
            <el-tag :type="tagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="time" :label="t('pageDashboard.time')" width="180">
          <template #default="{ row }">{{ formatTime(row.time) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { ChatDotRound, CircleCheck, Clock, User, MessageBox, DataLine } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";
import { localeRef } from "@/script/i18n";

const loading = ref(false);
const dashboard = ref({
  today_sessions: 0,
  resolved_sessions: 0,
  pending_sessions: 0,
  active_sessions: 0,
  total_sessions: 0,
  online_agents: 0,
  recent_sessions: [],
});

function statusLabel(status) {
  const m = {
    unassigned: t("pageDashboard.unassigned"),
    unread: t("pageDashboard.unread"),
    unreply: t("pageDashboard.pendingReply"),
    assigned: t("pageDashboard.activeSessions"),
    follow: t("pageDashboard.pendingFollow"),
    closed: t("pageDashboard.closed"),
  };
  return m[status] || status || "-";
}

function tagType(status) {
  if (status === "closed") return "success";
  if (status === "assigned") return "primary";
  if (status === "unread" || status === "unreply") return "warning";
  return "info";
}

function formatTime(ts) {
  if (!ts) return "-";
  const d = new Date(Number(ts) * 1000);
  return Number.isNaN(d.getTime()) ? "-" : d.toLocaleString(localeRef.value || "zh-CN");
}

async function loadDashboard() {
  loading.value = true;
  try {
    const resp = await api.getDashboard();
    dashboard.value = resp?.data?.data || dashboard.value;
  } catch (error) {
    ElMessage.error(error.message || t("pageDashboard.loadFailed"));
  } finally {
    loading.value = false;
  }
}

async function downloadSessions() {
  try {
    const resp = await api.exportSessions();
    const blob = new Blob([resp.data], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `sessions_${Date.now()}.csv`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  } catch (error) {
    ElMessage.error(error.message || t("pageDashboard.exportFailed"));
  }
}

onMounted(loadDashboard);
</script>
