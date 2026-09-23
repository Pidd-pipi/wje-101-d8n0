<template>
  <div class="page">
    <h1>配方广场</h1>
    <SearchFilter @search="onSearch" @reset="onReset">
      <template #filters>
        <el-form-item label="器具">
          <el-select v-model="device" clearable placeholder="全部器具" style="width: 160px" @change="load">
            <el-option v-for="d in DEVICES" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
      </template>
    </SearchFilter>
    <el-row :gutter="16">
      <el-col v-for="r in recipes" :key="r.id" :xs="24" :sm="12" :md="8">
        <el-card class="recipe-card" shadow="hover">
          <h3>{{ r.name }} <el-tag size="small">{{ r.device }}</el-tag></h3>
          <div class="meta">{{ r.water_temp }}°C · {{ r.grind_size }} · 粉水比 {{ r.ratio }}</div>
          <ol>
            <li v-for="s in stepsOf(r)" :key="s.step_number">
              第{{ s.step_number }}步：{{ s.description }}（{{ s.duration_seconds }}s）
            </li>
          </ol>
          <div v-if="isLoggedIn" class="card-actions">
            <el-button v-if="isOwner(r)" size="small" type="primary" plain @click="openEdit(r)">编辑</el-button>
            <el-button size="small" @click="fork(r)">复制为我的配方</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <EmptyState v-if="!recipes.length" description="暂无配方" />
    <el-button v-if="isLoggedIn" type="primary" style="margin-top: 16px" @click="openCreate">分享我的配方</el-button>
    <el-dialog v-model="showForm" :title="editingId ? '编辑冲煮配方' : '分享冲煮配方'" width="520px">
      <el-form label-width="80px">
        <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="器具">
          <el-select v-model="form.device" style="width: 200px">
            <el-option v-for="d in DEVICES" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
        <el-form-item label="水温" required><el-input-number v-model="form.water_temp" :min="80" :max="100" /></el-form-item>
        <el-form-item label="研磨度"><el-input v-model="form.grind_size" /></el-form-item>
        <el-form-item label="粉水比"><el-input v-model="form.ratio" placeholder="1:15" /></el-form-item>
        <el-form-item label="步骤(JSON)" required><el-input v-model="form.steps" type="textarea" :rows="4" placeholder='[{"step_number":1,"description":"闷蒸","duration_seconds":30}]' /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showForm = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRecipe">{{ editingId ? '保存' : '发布' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import SearchFilter from '@/components/common/SearchFilter.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { listRecipes, createRecipe, updateRecipe, forkRecipe } from '@/api/recipe'
import { useAuth } from '@/hooks/useAuth'
import type { BrewRecipe, RecipeStep } from '@/types/api'

const { store, isLoggedIn, user } = useAuth()
const recipes = ref<BrewRecipe[]>([])
const device = ref('')
const keyword = ref('')
const showForm = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({ name: '', device: '手冲壶', water_temp: 92 as number | undefined, grind_size: '中细', ratio: '1:15', steps: '[]' })

const DEVICES = ['手冲壶', '法压壶', '意式机', '爱乐压', '冷萃壶']

onMounted(async () => {
  if (isLoggedIn.value && !user.value) {
    try {
      await store.fetchProfile()
    } catch {
      /* 资料加载失败不影响浏览 */
    }
  }
  await load()
})

async function load() {
  const res = await listRecipes({ page: 1, page_size: 20, device: device.value, keyword: keyword.value })
  recipes.value = res.list
}
function onSearch(kw: string) {
  keyword.value = kw
  load()
}
function onReset() {
  device.value = ''
  keyword.value = ''
  load()
}
function stepsOf(r: BrewRecipe): RecipeStep[] {
  try {
    return JSON.parse(r.steps || '[]')
  } catch {
    return []
  }
}
function isOwner(r: BrewRecipe) {
  return !!user.value && r.user_id === user.value.id
}
function resetForm() {
  editingId.value = null
  Object.assign(form, { name: '', device: '手冲壶', water_temp: 92, grind_size: '中细', ratio: '1:15', steps: '[]' })
}
function openCreate() {
  resetForm()
  showForm.value = true
}
function openEdit(r: BrewRecipe) {
  editingId.value = r.id
  Object.assign(form, {
    name: r.name,
    device: r.device || '手冲壶',
    water_temp: r.water_temp,
    grind_size: r.grind_size,
    ratio: r.ratio,
    steps: r.steps || '[]',
  })
  showForm.value = true
}
function validateForm(): string | null {
  if (!form.name.trim()) return '请填写配方名称'
  if (!form.water_temp) return '请填写水温（80-100°C）'
  const text = form.steps.trim()
  if (!text) return '请填写冲煮步骤'
  let steps: unknown
  try {
    steps = JSON.parse(text)
  } catch {
    return '步骤格式不正确：请填写 JSON 数组，如 [{"step_number":1,"description":"闷蒸","duration_seconds":30}]'
  }
  if (!Array.isArray(steps) || steps.length === 0) return '步骤不完整：请至少填写一步'
  for (let i = 0; i < steps.length; i++) {
    const s = steps[i] as Partial<RecipeStep> | null
    if (!s || !String(s.description ?? '').trim()) return `步骤不完整：第 ${i + 1} 步缺少描述`
  }
  return null
}
async function saveRecipe() {
  const reason = validateForm()
  if (reason) {
    ElMessage.warning(reason)
    return
  }
  saving.value = true
  try {
    const payload = { ...form, name: form.name.trim(), water_temp: form.water_temp }
    if (editingId.value) {
      await updateRecipe(editingId.value, payload)
      ElMessage.success('配方已更新')
    } else {
      await createRecipe(payload)
      ElMessage.success('配方已分享')
    }
    showForm.value = false
    await load()
  } finally {
    saving.value = false
  }
}
async function fork(r: BrewRecipe) {
  await forkRecipe(r.id)
  ElMessage.success(`已复制为我的配方：《${r.name}》，可继续编辑调整`)
  await load()
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.recipe-card { margin-bottom: 16px; }
.meta { color: #999; font-size: 12px; margin: 6px 0; }
.card-actions { margin-top: 10px; display: flex; gap: 8px; }
</style>
