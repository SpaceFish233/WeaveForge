<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListSettings, GetSetting, UploadWorldSetting, UpdateSetting, DeleteSetting } from '../../wailsjs/go/main/App'

interface SettingItem { id: string; title: string; type: string }
interface SettingDetail { id: string; title: string; content: string; type: string }

const settings = ref<SettingItem[]>([])
const selectedSetting = ref<SettingDetail | null>(null)
const editing = ref(false)

const formTitle = ref('')
const formContent = ref('')
const formType = ref('世界观')
const settingTypes = ['世界观', '势力', '地理', '历史', '魔法/能力', '科技', '其他']

async function loadSettings() {
  try { settings.value = await ListSettings() } catch (e) { console.error(e) }
}

async function handleSelect(id: string) {
  editing.value = false
  try {
    const s = await GetSetting(id)
    selectedSetting.value = s
    formTitle.value = s.title; formContent.value = s.content; formType.value = s.type
  } catch (e) { console.error(e) }
}

function handleNew() { selectedSetting.value = null; editing.value = true; formTitle.value = ''; formContent.value = ''; formType.value = '世界观' }
function handleEdit() { editing.value = true }

async function handleSave() {
  if (!formTitle.value.trim()) return
  try {
    if (selectedSetting.value) await UpdateSetting(selectedSetting.value.id, formTitle.value, formContent.value, formType.value)
    else await UploadWorldSetting(formTitle.value, formContent.value, formType.value)
    editing.value = false; await loadSettings(); selectedSetting.value = null
  } catch (e) { console.error(e) }
}

async function handleDelete(id: string) {
  try {
    await DeleteSetting(id)
    if (selectedSetting.value?.id === id) selectedSetting.value = null
    await loadSettings()
  } catch (e) { console.error(e) }
}

function handleCancel() {
  editing.value = false
  if (selectedSetting.value) { formTitle.value = selectedSetting.value.title; formContent.value = selectedSetting.value.content; formType.value = selectedSetting.value.type }
}

function groupedSettings() {
  const g: Record<string, SettingItem[]> = {}
  for (const s of settings.value) { if (!g[s.type]) g[s.type] = []; g[s.type].push(s) }
  return g
}

onMounted(loadSettings)
</script>

<template>
  <div class="settings-layout">
    <div class="settings-sidebar">
      <div class="sidebar-header">
        <h3>设定列表</h3>
        <button class="btn-primary btn-sm" @click="handleNew">＋ 新增</button>
      </div>
      <div class="setting-tree">
        <div v-for="(items, type) in groupedSettings()" :key="type" class="type-group">
          <div class="type-label">{{ type }}</div>
          <div v-for="item in items" :key="item.id" class="setting-item"
            :class="{ active: selectedSetting?.id === item.id }" @click="handleSelect(item.id)">
            <span class="item-title">{{ item.title }}</span>
            <button class="btn-delete" @click.stop="handleDelete(item.id)" title="删除">×</button>
          </div>
        </div>
        <div v-if="settings.length === 0" class="tree-empty">暂无设定，点击「新增」添加</div>
      </div>
    </div>
    <div class="settings-detail">
      <div v-if="!selectedSetting && !editing" class="empty-detail"><p>选择左侧设定查看详情，或新增设定</p></div>
      <div v-else class="detail-form">
        <div class="form-field">
          <label>标题</label>
          <input v-model="formTitle" :disabled="!editing" placeholder="设定标题" />
        </div>
        <div class="form-field">
          <label>类型</label>
          <select v-model="formType" :disabled="!editing">
            <option v-for="t in settingTypes" :key="t" :value="t">{{ t }}</option>
          </select>
        </div>
        <div class="form-field">
          <label>内容</label>
          <textarea v-model="formContent" :disabled="!editing" placeholder="设定详细内容..." rows="20"></textarea>
        </div>
        <div v-if="editing" class="form-actions">
          <button class="btn-primary" @click="handleSave">保存</button>
          <button class="btn-secondary" @click="handleCancel">取消</button>
        </div>
        <div v-else class="form-actions">
          <button class="btn-primary" @click="handleEdit">编辑</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-layout { height: 100%; display: grid; grid-template-columns: 280px 1fr; overflow: hidden; background: #0d1117; }
.settings-sidebar { background: #161b22; border-right: 1px solid #21262d; display: flex; flex-direction: column; }
.sidebar-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #21262d; }
.sidebar-header h3 { font-size: 13px; font-weight: 600; color: #c9d1d9; margin: 0; }
.setting-tree { flex: 1; overflow-y: auto; padding: 8px 0; }
.type-group { margin-bottom: 8px; }
.type-label { padding: 4px 16px; font-size: 11px; font-weight: 600; color: #8b949e; text-transform: uppercase; letter-spacing: 0.5px; }
.setting-item { display: flex; align-items: center; justify-content: space-between; padding: 6px 16px 6px 24px; cursor: pointer; border-left: 3px solid transparent; transition: background 0.15s; }
.setting-item:hover { background: #1c2128; }
.setting-item.active { background: #1f2937; border-left-color: #58a6ff; }
.item-title { flex: 1; font-size: 13px; color: #c9d1d9; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.btn-delete { opacity: 0; width: 20px; height: 20px; border: none; background: transparent; color: #f85149; cursor: pointer; font-size: 16px; border-radius: 4px; }
.setting-item:hover .btn-delete { opacity: 1; }
.btn-delete:hover { background: rgba(248,81,73,0.1); }
.tree-empty { padding: 24px 16px; text-align: center; color: #484f58; font-size: 13px; }
.settings-detail { overflow-y: auto; padding: 24px 32px; }
.empty-detail { display: flex; align-items: center; justify-content: center; height: 100%; color: #484f58; }
.detail-form { max-width: 800px; }
.form-field { margin-bottom: 16px; }
.form-field label { display: block; font-size: 12px; font-weight: 600; color: #8b949e; margin-bottom: 6px; }
.form-field input, .form-field select, .form-field textarea { width: 100%; padding: 8px 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 14px; font-family: inherit; outline: none; transition: border-color 0.15s; }
.form-field input:focus, .form-field select:focus, .form-field textarea:focus { border-color: #58a6ff; }
.form-field input:disabled, .form-field select:disabled, .form-field textarea:disabled { background: #161b22; color: #8b949e; cursor: not-allowed; }
.form-field textarea { resize: vertical; line-height: 1.6; }
.form-actions { display: flex; gap: 8px; align-items: center; margin-top: 16px; }
.btn-primary { padding: 8px 16px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 600; transition: background 0.15s; }
.btn-primary:hover { background: #2ea043; }
.btn-sm { padding: 4px 10px; font-size: 11px; }
.btn-secondary { padding: 8px 16px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-secondary:hover { background: #30363d; }
</style>
