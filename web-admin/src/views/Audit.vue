<template>
  <div class="p-8">
    <div class="head">
      <div>
        <h1>{{ t('pageAudit.title') }}</h1>
        <p>{{ t('pageAudit.subtitle') }}</p>
      </div>
      <el-button @click="loadLogs">{{ t('pageAudit.refresh') }}</el-button>
    </div>

    <div class="filters">
      <el-input v-model="operator" :placeholder="t('pageAudit.operator')" clearable @keyup.enter="loadLogs" />
      <el-input v-model="action" :placeholder="t('pageAudit.actionInput')" clearable @keyup.enter="loadLogs" />
      <el-date-picker v-model="startTime" type="datetime" :placeholder="t('pageAudit.startTime')" clearable style="width: 180px" />
      <el-date-picker v-model="endTime" type="datetime" :placeholder="t('pageAudit.endTime')" clearable style="width: 180px" />
      <el-select v-model="result" clearable :placeholder="t('pageAudit.result')" @change="loadLogs">
        <el-option :label="t('pageAudit.success')" value="success" />
        <el-option :label="t('pageAudit.failed')" value="failed" />
      </el-select>
      <el-button type="primary" @click="loadLogs">{{ t('pageAudit.filter') }}</el-button>
    </div>

    <el-table :data="rows" v-loading="loading" stripe>
      <el-table-column prop="CreatedAt" :label="t('pageAudit.time')" width="180">
        <template #default="{ row }">{{ formatDate(row.CreatedAt) }}</template>
      </el-table-column>
      <el-table-column prop="operator" :label="t('pageAudit.operator')" width="120" />
      <el-table-column prop="operator_role" :label="t('pageAudit.role')" width="120" />
      <el-table-column prop="action" :label="t('pageAudit.action')" min-width="180" />
      <el-table-column prop="target_type" :label="t('pageAudit.targetType')" width="120" />
      <el-table-column prop="target_id" :label="t('pageAudit.targetId')" min-width="180" />
      <el-table-column prop="result" :label="t('pageAudit.result')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.result === 'success' ? 'success' : 'danger'">{{ row.result }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="detail" :label="t('pageAudit.detail')" min-width="200" show-overflow-tooltip />
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadLogs"
        @current-change="loadLogs"
      />
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import api from "@/script/api";
import { ElMessage } from "element-plus";
import { t } from "@/script/i18n-text";
import { localeRef } from "@/script/i18n";

const rows = ref([]);
const loading = ref(false);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);
const operator = ref("");
const action = ref("");
const result = ref("");
const startTime = ref(null);
const endTime = ref(null);

function formatDate(value) {
  if (!value) return "-";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "-";
  return d.toLocaleString(localeRef.value || "zh-CN");
}

async function loadLogs() {
  loading.value = true;
  try {
    const resp = await api.listAuditLogs({
      page: page.value,
      page_size: pageSize.value,
      operator: operator.value || undefined,
      action: action.value || undefined,
      start_time: startTime.value ? Math.floor(new Date(startTime.value).getTime() / 1000) : undefined,
      end_time: endTime.value ? Math.floor(new Date(endTime.value).getTime() / 1000) : undefined,
      result: result.value || undefined,
    });
    rows.value = resp?.data?.data?.data || [];
    total.value = Number(resp?.data?.data?.total || 0);
  } catch (error) {
    ElMessage.error(error.message || t("pageAudit.loadFailed"));
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadLogs();
});
</script>

<style scoped>
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.head h1 {
  margin: 0;
  font-size: 24px;
}

.head p {
  margin: 4px 0 0;
  color: #6b7280;
  font-size: 13px;
}

.filters {
  display: grid;
  grid-template-columns: 180px 220px 180px 180px 140px 100px;
  gap: 10px;
  margin-bottom: 12px;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>
