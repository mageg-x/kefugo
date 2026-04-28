<template>
  <div class="dashboard-container p-8">
    <div class="dashboard-header mb-8">
      <h1 class="text-page-title">{{ t("pageDashboard.title") }}</h1>
      <p class="text-secondary mt-2">{{ t("pageDashboard.subtitle") }}</p>
    </div>

    <div class="grid grid-cols-1 gap-5 mb-8 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <el-card class="stat-card stat-card--sessions" shadow="never">
        <div class="stat-card__inner">
          <div class="stat-card__icon">
            <el-icon :size="28"><ChatDotRound /></el-icon>
          </div>
          <div class="stat-card__content">
            <p class="stat-card__label">{{ t("pageDashboard.todaySessions") }}</p>
            <p class="stat-card__value">{{ dashboard.today_sessions }}</p>
          </div>
        </div>
        <div class="stat-card__glow"></div>
      </el-card>

      <el-card class="stat-card stat-card--resolved" shadow="never">
        <div class="stat-card__inner">
          <div class="stat-card__icon">
            <el-icon :size="28"><CircleCheck /></el-icon>
          </div>
          <div class="stat-card__content">
            <p class="stat-card__label">{{ t("pageDashboard.resolved") }}</p>
            <p class="stat-card__value">{{ dashboard.resolved_sessions }}</p>
          </div>
        </div>
        <div class="stat-card__glow"></div>
      </el-card>

      <el-card class="stat-card stat-card--pending" shadow="never">
        <div class="stat-card__inner">
          <div class="stat-card__icon">
            <el-icon :size="28"><Clock /></el-icon>
          </div>
          <div class="stat-card__content">
            <p class="stat-card__label">{{ t("pageDashboard.pending") }}</p>
            <p class="stat-card__value">{{ dashboard.pending_sessions }}</p>
          </div>
        </div>
        <div class="stat-card__glow"></div>
      </el-card>

      <el-card class="stat-card stat-card--online" shadow="never">
        <div class="stat-card__inner">
          <div class="stat-card__icon">
            <el-icon :size="28"><User /></el-icon>
          </div>
          <div class="stat-card__content">
            <p class="stat-card__label">{{ t("pageDashboard.onlineAgents") }}</p>
            <p class="stat-card__value">{{ dashboard.online_agents }}</p>
          </div>
        </div>
        <div class="stat-card__glow"></div>
      </el-card>

      <el-card class="stat-card stat-card--active" shadow="never">
        <div class="stat-card__inner">
          <div class="stat-card__icon">
            <el-icon :size="28"><MessageBox /></el-icon>
          </div>
          <div class="stat-card__content">
            <p class="stat-card__label">{{ t("pageDashboard.activeSessions") }}</p>
            <p class="stat-card__value">{{ dashboard.active_sessions }}</p>
          </div>
        </div>
        <div class="stat-card__glow"></div>
      </el-card>

      <el-card class="stat-card stat-card--total" shadow="never">
        <div class="stat-card__inner">
          <div class="stat-card__icon">
            <el-icon :size="28"><DataLine /></el-icon>
          </div>
          <div class="stat-card__content">
            <p class="stat-card__label">{{ t("pageDashboard.totalSessions") }}</p>
            <p class="stat-card__value">{{ dashboard.total_sessions }}</p>
          </div>
        </div>
        <div class="stat-card__glow"></div>
      </el-card>
    </div>

    <el-card class="recent-card" shadow="never" v-loading="loading">
      <template #header>
        <div class="recent-card__header">
          <div>
            <h2 class="text-card-title">{{ t("pageDashboard.recentSessions") }}</h2>
            <p class="text-tertiary mt-1">{{ t("pageDashboard.recentSubtitle") }}</p>
          </div>
          <el-button type="primary" size="default" class="export-btn" @click="downloadSessions">
            <template #icon>
              <el-icon><Download /></el-icon>
            </template>
            {{ t("pageDashboard.exportCsv") }}
          </el-button>
        </div>
      </template>
      <el-table :data="dashboard.recent_sessions || []" stripe class="admin-console-table">
        <el-table-column prop="sid" :label="t('pageDashboard.sessionId')" min-width="220" />
        <el-table-column prop="visitor_id" :label="t('pageDashboard.visitor')" min-width="120" />
        <el-table-column prop="agent_id" :label="t('pageDashboard.agent')" min-width="120">
          <template #default="{ row }">
            <span class="agent-name">{{ row.agent_id || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('pageDashboard.status')" width="140">
          <template #default="{ row }">
            <el-tag :type="tagType(row.status)" size="small" effect="light" round>{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="time" :label="t('pageDashboard.time')" width="180">
          <template #default="{ row }">
            <span class="time-cell">{{ formatTime(row.time) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { ChatDotRound, CircleCheck, Clock, User, MessageBox, DataLine, Download } from "@element-plus/icons-vue";
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
    assigned: t("pageDashboard.inProgress"),
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

<style scoped>
.dashboard-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.dashboard-header {
  margin-bottom: 2rem;
}

.stat-card {
  position: relative;
  overflow: hidden;
  border-radius: 1rem;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.stat-card:hover {
  transform: translateY(-4px);
}

.stat-card__inner {
  display: flex;
  align-items: center;
  gap: 1rem;
  position: relative;
  z-index: 1;
}

.stat-card__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 1rem;
  flex-shrink: 0;
}

.stat-card__content {
  flex: 1;
  min-width: 0;
}

.stat-card__label {
  font-size: 0.8125rem;
  font-weight: 500;
  opacity: 0.9;
  margin-bottom: 0.25rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stat-card__value {
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1.2;
  letter-spacing: -0.025em;
}

.stat-card__glow {
  position: absolute;
  top: -50%;
  right: -50%;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  opacity: 0.15;
  pointer-events: none;
}

.stat-card--sessions {
  background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.3);
}

.stat-card--sessions .stat-card__icon {
  background: rgba(255, 255, 255, 0.2);
}

.stat-card--sessions:hover {
  box-shadow: 0 8px 30px rgba(99, 102, 241, 0.4);
}

.stat-card--resolved {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(16, 185, 129, 0.3);
}

.stat-card--resolved .stat-card__icon {
  background: rgba(255, 255, 255, 0.2);
}

.stat-card--resolved:hover {
  box-shadow: 0 8px 30px rgba(16, 185, 129, 0.4);
}

.stat-card--pending {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(245, 158, 11, 0.3);
}

.stat-card--pending .stat-card__icon {
  background: rgba(255, 255, 255, 0.2);
}

.stat-card--pending:hover {
  box-shadow: 0 8px 30px rgba(245, 158, 11, 0.4);
}

.stat-card--online {
  background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(139, 92, 246, 0.3);
}

.stat-card--online .stat-card__icon {
  background: rgba(255, 255, 255, 0.2);
}

.stat-card--online:hover {
  box-shadow: 0 8px 30px rgba(139, 92, 246, 0.4);
}

.stat-card--active {
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(6, 182, 212, 0.3);
}

.stat-card--active .stat-card__icon {
  background: rgba(255, 255, 255, 0.2);
}

.stat-card--active:hover {
  box-shadow: 0 8px 30px rgba(6, 182, 212, 0.4);
}

.stat-card--total {
  background: linear-gradient(135deg, #64748b 0%, #475569 100%);
  color: white;
  box-shadow: 0 4px 20px rgba(100, 116, 139, 0.3);
}

.stat-card--total .stat-card__icon {
  background: rgba(255, 255, 255, 0.2);
}

.stat-card--total:hover {
  box-shadow: 0 8px 30px rgba(100, 116, 139, 0.4);
}

.recent-card {
  border-radius: 1rem;
  overflow: hidden;
}

.recent-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
}

.export-btn {
  background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
  border: none;
  border-radius: 0.75rem;
  font-weight: 500;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.export-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}

.agent-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.time-cell {
  color: var(--color-text-secondary);
  font-size: 0.8125rem;
}

:deep(.el-card) {
  border: none;
}

:deep(.el-table) {
  border-radius: 0.75rem;
}

:deep(.el-table th.el-table__cell) {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  padding: 1rem 0.75rem;
}

:deep(.el-table td.el-table__cell) {
  padding: 1rem 0.75rem;
}

:deep(.el-table__header-wrapper) {
  border-radius: 0.75rem 0.75rem 0 0;
}
</style>
