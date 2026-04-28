<template>
  <div class="p-8">
    <h1 class="text-2xl font-bold text-gray-800 mb-6">{{ t('pageVisitors.title') }}</h1>

    <el-card class="mb-6">
      <div class="flex gap-4">
        <el-input v-model="searchQuery" :placeholder="t('pageVisitors.search')" clearable style="width: 300px" @keyup.enter="loadVisitors" />
        <el-select v-model="statusFilter" :placeholder="t('pageVisitors.statusFilter')" clearable style="width: 150px" @change="loadVisitors">
          <el-option :label="t('pageVisitors.all')" value="" />
          <el-option :label="t('pageVisitors.online')" value="online" />
          <el-option :label="t('pageVisitors.offline')" value="offline" />
        </el-select>
        <el-button type="primary" @click="loadVisitors">{{ t('pageVisitors.query') }}</el-button>
        <el-button @click="downloadSessions">{{ t('pageVisitors.exportCsv') }}</el-button>
      </div>
    </el-card>

    <el-card v-loading="loading">
      <el-table :data="visitors" stripe>
        <el-table-column prop="id" :label="t('pageVisitors.visitorId')" min-width="220" />
        <el-table-column prop="name" :label="t('pageVisitors.visitorName')" min-width="140" />
        <el-table-column prop="status" :label="t('pageVisitors.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'info'" size="small">
              {{ row.status === 'online' ? t('pageVisitors.online') : t('pageVisitors.offline') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sessions" :label="t('pageVisitors.sessions')" width="100" />
        <el-table-column prop="ip" :label="t('pageVisitors.ip')" min-width="130" />
        <el-table-column prop="device" :label="t('pageVisitors.device')" width="100" />
        <el-table-column prop="geo" :label="t('pageVisitors.region')" width="100" />
        <el-table-column prop="user_agent" :label="t('pageVisitors.userAgent')" min-width="220" show-overflow-tooltip />
        <el-table-column prop="last_visit" :label="t('pageVisitors.lastVisit')" width="180">
          <template #default="{ row }">{{ formatTime(row.last_visit) }}</template>
        </el-table-column>
      </el-table>

      <div class="flex justify-end mt-4">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="loadVisitors"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
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

function formatTime(ts) {
  if (!ts) return "-";
  const d = new Date(Number(ts) * 1000);
  return Number.isNaN(d.getTime()) ? "-" : d.toLocaleString(localeRef.value || "zh-CN");
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
