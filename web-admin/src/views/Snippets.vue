<template>
  <div class="p-8 snippets-page">
    <div class="head">
      <div>
        <h1 class="page-title">{{ t('pageSnippets.title') }}</h1>
        <p class="page-subtitle">{{ t('pageSnippets.subtitle') }}</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">{{ t('pageSnippets.addSnippet') }}</el-button>
    </div>

    <div class="snippet-filters">
      <el-select v-model="categoryFilter" clearable :placeholder="t('pageSnippets.filterCategory')" @change="loadSnippets">
        <el-option v-for="cat in categoryOptions" :key="cat" :label="cat" :value="cat" />
      </el-select>
    </div>

    <el-table :data="snippets" v-loading="loading" stripe>
      <el-table-column prop="title" :label="t('pageSnippets.titleCol')" min-width="160" />
      <el-table-column prop="category" :label="t('pageSnippets.category')" width="120">
        <template #default="{ row }">
          <el-tag>{{ row.category }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="content" :label="t('pageSnippets.content')" min-width="320" show-overflow-tooltip />
      <el-table-column prop="usage_count" :label="t('pageSnippets.usageCount')" width="120" align="center" />
      <el-table-column :label="t('pageSnippets.actions')" width="220" align="center">
        <template #default="{ row }">
          <el-button type="primary" link @click="useSnippet(row)">{{ t('pageSnippets.use') }}</el-button>
          <el-button type="primary" link @click="openEditDialog(row)">{{ t('pageSnippets.edit') }}</el-button>
          <el-button type="danger" link @click="removeSnippet(row)">{{ t('pageSnippets.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? t('pageSnippets.dialogAdd') : t('pageSnippets.dialogEdit')" width="560px">
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('pageSnippets.titleCol')">
          <el-input v-model="form.title" maxlength="80" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('pageSnippets.category')">
          <el-select v-model="form.category" filterable allow-create default-first-option clearable>
            <el-option v-for="cat in categoryOptions" :key="cat" :label="cat" :value="cat" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('pageSnippets.content')">
          <el-input v-model="form.content" type="textarea" :rows="6" maxlength="2000" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('pageSnippets.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="saveSnippet">{{ t('pageSnippets.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/script/api";
import { t } from "@/script/i18n-text";

const route = useRoute();
const loading = ref(false);
const saving = ref(false);
const snippets = ref([]);
const categoryFilter = ref("");
const dialogVisible = ref(false);
const dialogMode = ref("create");
const defaultSnippetCategories = [t("pageSnippets.catGreeting"), t("pageSnippets.catFaq"), t("pageSnippets.catClosing"), t("pageSnippets.other")];
const form = ref({
  id: 0,
  title: "",
  category: t("pageSnippets.catGreeting"),
  content: "",
});
const categoryOptions = computed(() => {
  const set = new Set(defaultSnippetCategories);
  for (const item of snippets.value) {
    const cat = String(item?.category || "").trim();
    if (cat) set.add(cat);
  }
  const activeFilter = String(categoryFilter.value || "").trim();
  if (activeFilter) set.add(activeFilter);
  const editingCategory = String(form.value?.category || "").trim();
  if (editingCategory) set.add(editingCategory);
  return Array.from(set);
});

async function loadSnippets() {
  loading.value = true;
  try {
    const resp = await api.listQuickReplies({ category: categoryFilter.value || undefined });
    snippets.value = resp?.data?.data?.data || [];
  } catch (error) {
    ElMessage.error(error.message || t("pageSnippets.loadFailed"));
  } finally {
    loading.value = false;
  }
}

function openCreateDialog() {
  dialogMode.value = "create";
  form.value = {
    id: 0,
    title: "",
    category: t("pageSnippets.catGreeting"),
    content: "",
  };
  dialogVisible.value = true;
}

function openEditDialog(item) {
  dialogMode.value = "update";
  form.value = {
    id: item.id,
    title: item.title,
    category: item.category,
    content: item.content,
  };
  dialogVisible.value = true;
}

async function saveSnippet() {
  if (!form.value.title.trim() || !form.value.content.trim()) {
    ElMessage.warning(t("pageSnippets.fillTitleContent"));
    return;
  }
  saving.value = true;
  try {
    const category = String(form.value.category || "").trim() || t("pageSnippets.other");
    if (dialogMode.value === "create") {
      await api.createQuickReply({ title: form.value.title.trim(), category, content: form.value.content.trim() });
    } else {
      await api.updateQuickReply({ id: form.value.id, title: form.value.title.trim(), category, content: form.value.content.trim() });
    }
    dialogVisible.value = false;
    ElMessage.success(t("pageSnippets.saveSuccess"));
    await loadSnippets();
  } catch (error) {
    ElMessage.error(error.message || t("pageSnippets.saveFailed"));
  } finally {
    saving.value = false;
  }
}

async function removeSnippet(item) {
  try {
    await ElMessageBox.confirm(t("pageSnippets.deleteConfirm").replace("{title}", item.title), t("pageSnippets.deleteTitle"), { type: "warning" });
    await api.deleteQuickReply(item.id);
    ElMessage.success(t("pageSnippets.deleteSuccess"));
    await loadSnippets();
  } catch (error) {
    if (error !== "cancel") ElMessage.error(error.message || t("pageSnippets.deleteFailed"));
  }
}

async function useSnippet(item) {
  await api.useQuickReply(item.id).catch(() => {});
  localStorage.setItem("kefu_snippet_to_inbox", item.content || "");
  ElMessage.success(t("pageSnippets.inserted"));
  await loadSnippets();
}

onMounted(() => {
  const category = String(route.query.category || "").trim();
  if (category) categoryFilter.value = category;
  loadSnippets();
});
</script>

<style scoped>
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.head h1 {
  margin: 0;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.page-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  font-weight: 400;
  color: #6b7280;
}

.snippet-filters {
  margin-bottom: 12px;
}
</style>
