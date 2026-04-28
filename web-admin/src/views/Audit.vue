<template>
  <div class="admin-console-page audit-console">
    <div class="console-hero">
      <div class="console-hero__copy">
        <span class="console-kicker">{{ t("pageAudit.title") }}</span>
        <h1>{{ t("pageAudit.title") }}</h1>
        <p>{{ t("pageAudit.subtitle") }}</p>
      </div>
      <div class="console-hero__actions">
        <el-button type="primary" class="hero-button" @click="loadLogs">{{ t("pageAudit.refresh") }}</el-button>
      </div>
    </div>

    <el-card class="console-panel filter-panel" shadow="never">
      <div class="panel-head">
        <div>
          <h2>{{ t("pageAudit.filter") }}</h2>
          <p>{{ t("pageAudit.panel.filtersDesc") }}</p>
        </div>
      </div>
      <div class="filter-grid">
        <el-input v-model="operator" :placeholder="t('pageAudit.operator')" clearable @keyup.enter="applyFilters" />
        <el-input v-model="action" :placeholder="t('pageAudit.actionInput')" clearable @keyup.enter="applyFilters" />
        <el-date-picker
          v-model="startTime"
          type="datetime"
          :placeholder="t('pageAudit.startTime')"
          clearable
        />
        <el-date-picker
          v-model="endTime"
          type="datetime"
          :placeholder="t('pageAudit.endTime')"
          clearable
        />
        <el-select v-model="result" clearable :placeholder="t('pageAudit.result')" @change="applyFilters">
          <el-option :label="t('pageAudit.success')" value="success" />
          <el-option :label="t('pageAudit.failed')" value="failed" />
        </el-select>
        <div class="filter-actions">
          <el-button type="primary" class="filter-submit" @click="applyFilters">{{ t("pageAudit.filter") }}</el-button>
        </div>
      </div>
    </el-card>

    <el-card class="console-panel table-panel" shadow="never">
      <div class="panel-head panel-head--split">
        <div>
          <h2>{{ t("pageAudit.panel.list") }}</h2>
          <p>{{ t("pageAudit.panel.listDesc") }}</p>
        </div>
        <div class="panel-badge">
          <span>{{ total }}</span>
          <small>LOG</small>
        </div>
      </div>

      <div class="table-shell">
        <el-table :data="rows" v-loading="loading" stripe class="admin-console-table">
          <el-table-column :label="t('pageAudit.time')" width="190">
            <template #default="{ row }">{{ formatDate(row.created_at || row.CreatedAt || row.createdAt) }}</template>
          </el-table-column>
          <el-table-column prop="operator" :label="t('pageAudit.operator')" width="130" />
          <el-table-column prop="operator_role" :label="t('pageAudit.role')" width="130" />
          <el-table-column prop="action" :label="t('pageAudit.action')" min-width="180" />
          <el-table-column prop="target_type" :label="t('pageAudit.targetType')" width="130" />
          <el-table-column prop="target_id" :label="t('pageAudit.targetId')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="result" :label="t('pageAudit.result')" width="110">
            <template #default="{ row }">
              <el-tag :type="row.result === 'success' ? 'success' : 'danger'" effect="light">{{ row.result }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="detail" :label="t('pageAudit.detail')" min-width="220" show-overflow-tooltip />
          <el-table-column :label="t('action.actions')" width="120" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="openDetail(row)">{{ t("pageAudit.viewDetail") }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="table-pagination">
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
    </el-card>

    <el-dialog v-model="detailVisible" :title="t('pageAudit.detailDialog')" width="720px">
      <template v-if="activeRow">
        <div class="audit-detail-grid">
          <div class="audit-detail-field">
            <label>{{ t("pageAudit.time") }}</label>
            <span>{{ formatDate(activeRow.created_at || activeRow.CreatedAt || activeRow.createdAt) }}</span>
          </div>
          <div class="audit-detail-field">
            <label>{{ t("pageAudit.operator") }}</label>
            <span>{{ activeRow.operator || "-" }}</span>
          </div>
          <div class="audit-detail-field">
            <label>{{ t("pageAudit.role") }}</label>
            <span>{{ activeRow.operator_role || "-" }}</span>
          </div>
          <div class="audit-detail-field">
            <label>{{ t("pageAudit.result") }}</label>
            <span>{{ activeRow.result || "-" }}</span>
          </div>
          <div class="audit-detail-field audit-detail-field--full">
            <label>{{ t("pageAudit.action") }}</label>
            <span>{{ activeRow.action || "-" }}</span>
          </div>
          <div class="audit-detail-field">
            <label>{{ t("pageAudit.targetType") }}</label>
            <span>{{ activeRow.target_type || "-" }}</span>
          </div>
          <div class="audit-detail-field">
            <label>{{ t("pageAudit.targetId") }}</label>
            <span>{{ activeRow.target_id || "-" }}</span>
          </div>
          <div class="audit-detail-field audit-detail-field--full">
            <label>{{ t("pageAudit.detail") }}</label>
            <pre>{{ activeRow.detail || "-" }}</pre>
          </div>
        </div>
      </template>
      <template #footer>
        <el-button type="primary" @click="detailVisible = false">{{ t("action.close") }}</el-button>
      </template>
    </el-dialog>
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
const detailVisible = ref(false);
const activeRow = ref(null);

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
  return date.toLocaleString(localeRef.value || "zh-CN");
}

function openDetail(row) {
  activeRow.value = row || null;
  detailVisible.value = true;
}

function applyFilters() {
  page.value = 1;
  loadLogs();
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
    const list = resp?.data?.data?.data || [];
    rows.value = list.map((item) => ({
      ...item,
      created_at: item.created_at ?? item.CreatedAt ?? item.createdAt ?? "",
    }));
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
.audit-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.audit-detail-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.audit-detail-field label {
  color: #64748b;
  font-size: 0.8125rem;
  font-weight: 600;
}

.audit-detail-field span,
.audit-detail-field pre {
  margin: 0;
  color: #0f172a;
  font-size: 0.875rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.audit-detail-field--full {
  grid-column: 1 / -1;
}

@media (max-width: 768px) {
  .audit-detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
