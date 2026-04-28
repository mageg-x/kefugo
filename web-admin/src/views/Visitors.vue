<template>
  <div class="admin-console-page visitors-console">
    <div class="console-hero">
      <div class="console-hero__copy">
        <span class="console-kicker">{{ t("pageVisitors.title") }}</span>
        <h1>{{ t("pageVisitors.title") }}</h1>
        <p>{{ t("pageVisitors.subtitle") }}</p>
      </div>
      <div class="console-hero__actions">
        <el-button type="primary" class="hero-button" @click="loadVisitors">{{ t("action.refresh") }}</el-button>
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
          <h2>{{ t("pageVisitors.panel.filters") }}</h2>
          <p>{{ t("pageVisitors.panel.filtersDesc") }}</p>
        </div>
        <el-button type="primary" @click="downloadSessions">{{ t("pageVisitors.exportCsv") }}</el-button>
      </div>
      <div class="filter-grid">
        <el-input v-model="searchQuery" :placeholder="t('pageVisitors.search')" clearable @keyup.enter="applyFilters" />
        <el-select v-model="statusFilter" :placeholder="t('pageVisitors.statusFilter')" clearable @change="applyFilters">
          <el-option :label="t('pageVisitors.all')" value="" />
          <el-option :label="t('pageVisitors.online')" value="online" />
          <el-option :label="t('pageVisitors.offline')" value="offline" />
        </el-select>
        <div class="filter-actions">
          <el-button type="primary" class="filter-submit" @click="applyFilters">{{ t("pageVisitors.query") }}</el-button>
        </div>
      </div>
    </el-card>

    <el-card class="console-panel table-panel" shadow="never" v-loading="loading">
      <div class="panel-head panel-head--split">
        <div>
          <h2>{{ t("pageVisitors.panel.list") }}</h2>
          <p>{{ t("pageVisitors.panel.listDesc") }}</p>
        </div>
        <div class="panel-badge">
          <span>{{ total }}</span>
          <small>{{ t("pageVisitors.visitorId") }}</small>
        </div>
      </div>

      <div class="table-shell">
        <el-table :data="visitors" stripe class="admin-console-table">
          <el-table-column prop="id" :label="t('pageVisitors.visitorId')" min-width="220" />
          <el-table-column prop="name" :label="t('pageVisitors.visitorName')" min-width="150" />
          <el-table-column prop="status" :label="t('pageVisitors.status')" width="110">
            <template #default="{ row }">
              <el-tag :type="row.status === 'online' ? 'success' : 'info'" size="small" effect="light">
                {{ row.status === "online" ? t("pageVisitors.online") : t("pageVisitors.offline") }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sessions" :label="t('pageVisitors.sessions')" width="110" />
          <el-table-column prop="ip" label="IP" min-width="140" />
          <el-table-column prop="device" :label="t('pageVisitors.device')" width="120" />
          <el-table-column prop="geo" :label="t('pageVisitors.region')" width="120" />
          <el-table-column prop="user_agent" :label="t('pageVisitors.userAgent')" min-width="240" show-overflow-tooltip />
          <el-table-column prop="last_visit" :label="t('pageVisitors.lastVisit')" width="180">
            <template #default="{ row }">{{ formatTime(row.last_visit) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="table-pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadVisitors"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";
import { localeRef } from "@/script/i18n";

const searchQuery = ref("");
const statusFilter = ref("");
const currentPage = ref(1);
const pageSize = ref(10);
const total = ref(0);
const visitors = ref([]);
const loading = ref(false);

const overviewCards = computed(() => {
  const rows = visitors.value || [];
  const onlineCount = rows.filter((item) => item.status === "online").length;
  const offlineCount = rows.filter((item) => item.status !== "online").length;
  const sessionTotal = rows.reduce((sum, item) => sum + Number(item.sessions || 0), 0);
  return [
    { key: "in-view", label: t("pageVisitors.overview.inView"), value: rows.length, tone: "" },
    { key: "online", label: t("pageVisitors.overview.online"), value: onlineCount, tone: "overview-card--success" },
    { key: "offline", label: t("pageVisitors.overview.offline"), value: offlineCount, tone: "overview-card--warning" },
    { key: "sessions", label: t("pageVisitors.overview.sessions"), value: sessionTotal, tone: "overview-card--info" },
  ];
});

function formatTime(ts) {
  if (!ts) return "-";
  const d = new Date(Number(ts) * 1000);
  return Number.isNaN(d.getTime()) ? "-" : d.toLocaleString(localeRef.value || "zh-CN");
}

function applyFilters() {
  currentPage.value = 1;
  loadVisitors();
}

async function loadVisitors() {
  loading.value = true;
  try {
    const resp = await api.listVisitors({
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchQuery.value || undefined,
      status: statusFilter.value || undefined,
    });
    visitors.value = resp?.data?.data?.data || [];
    total.value = Number(resp?.data?.data?.total || 0);
  } catch (error) {
    ElMessage.error(error.message || t("pageVisitors.loadFailed"));
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
    ElMessage.error(error.message || t("pageVisitors.exportFailed"));
  }
}

onMounted(loadVisitors);
</script>
