<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListCharacters, CreateCharacter, UpdateCharacter, GetCharacter, DeleteCharacter } from '../../wailsjs/go/main/App'
import { models } from '../../wailsjs/go/models'

type Character = models.Character
interface CharSummary { id: string; name: string; gender: string; race: string; role: string; personality: string; status: string; tags: string; avatar: string; created_at: string }

const chars = ref<CharSummary[]>([])
const editing = ref(false)
const avatarInput = ref<HTMLInputElement | null>(null)
const form = ref<Character>(new models.Character({ id: '', name: '', gender: '', race: '', personality: '', description: '', first_chapter: '', tags: '[]', avatar: '', role: '配角', status: 'active' }))
const roles = ['主角', '配角', '反派', '路人']
const statuses = ['active', 'inactive', 'deceased']

async function load() { try { chars.value = await ListCharacters() } catch (e) { console.error(e) } }
async function handleNew() { editing.value = true; form.value = new models.Character({ id: '', name: '', gender: '', race: '', personality: '', description: '', first_chapter: '', tags: '[]', avatar: '', role: '配角', status: 'active' }) }
async function handleEdit(id: string) {
  try { const c = await GetCharacter(id); form.value = new models.Character(c); editing.value = true } catch (e) { console.error(e) }
}
async function handleSave() {
  if (!form.value.name.trim()) return
  try {
    const c = new models.Character(form.value)
    if (form.value.id) { await UpdateCharacter(form.value.id, c) }
    else { await CreateCharacter(c) }
    editing.value = false; await load()
  } catch (e) { console.error(e) }
}
async function handleDelete(id: string) {
  try { await DeleteCharacter(id); await load() } catch (e) { console.error(e) }
}
function handleCancel() { editing.value = false }

function handleAvatarUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 2 * 1024 * 1024) {
    alert('头像文件大小不能超过 2MB')
    return
  }
  const reader = new FileReader()
  reader.onload = () => { form.value.avatar = reader.result as string }
  reader.readAsDataURL(file)
}

function removeAvatar() { form.value.avatar = '' }

function avatarSrc(c: { avatar?: string; name?: string }): string {
  return c.avatar || ''
}

onMounted(load)
</script>

<template>
  <div class="char-page">
    <div class="page-header">
      <h2>角色管理</h2>
      <button class="btn-primary" @click="handleNew">＋ 新建角色</button>
    </div>

    <div v-if="editing" class="edit-panel">
      <h3>{{ form.id ? '编辑角色' : '新建角色' }}</h3>
      <div class="avatar-row">
        <div class="avatar-preview" @click="avatarInput?.click()">
          <img v-if="form.avatar" :src="form.avatar" class="avatar-img" />
          <span v-else class="avatar-initials">{{ form.name?.charAt(0) || '?' }}</span>
          <div class="avatar-overlay">点击上传</div>
        </div>
        <div class="avatar-actions">
          <input ref="avatarInput" type="file" accept="image/*" style="display:none" @change="handleAvatarUpload" />
          <button v-if="form.avatar" class="btn-sm btn-danger" @click="removeAvatar">清除</button>
        </div>
      </div>
      <div class="form-grid">
        <div class="form-field"><label>姓名</label><input v-model="form.name" placeholder="角色名" /></div>
        <div class="form-field"><label>性别</label><input v-model="form.gender" placeholder="男/女/未知" /></div>
        <div class="form-field"><label>种族</label><input v-model="form.race" placeholder="人类/精灵/龙族" /></div>
        <div class="form-field"><label>角色定位</label>
          <div class="radio-group"> <button v-for="r in roles" :key="r" class="radio-btn" :class="{ active: form.role === r }" @click="form.role = r">{{ r }}</button> </div>
        </div>
        <div class="form-field"><label>状态</label>
          <div class="radio-group"> <button v-for="s in statuses" :key="s" class="radio-btn" :class="{ active: form.status === s }" @click="form.status = s">{{ s === 'active' ? '活跃' : s === 'inactive' ? '退场' : '已故' }}</button> </div>
        </div>
        <div class="form-field"><label>性格标签</label><input v-model="form.personality" placeholder="冷静/暴躁/狡诈…逗号分隔" /></div>
        <div class="form-field full"><label>外貌描述</label><textarea v-model="form.description" placeholder="外貌、特征、衣着…" rows="3"></textarea></div>
        <div class="form-field full"><label>背景故事</label><textarea v-model="form.first_chapter" placeholder="首次出场背景或角色背景故事…" rows="4"></textarea></div>
      </div>
      <div class="form-actions">
        <button class="btn-primary" @click="handleSave">保存</button>
        <button class="btn-secondary" @click="handleCancel">取消</button>
      </div>
    </div>

    <div v-else class="card-grid">
      <div v-for="c in chars" :key="c.id" class="char-card">
        <div class="card-avatar">
          <img v-if="c.avatar" :src="c.avatar" class="avatar-img" />
          <span v-else>{{ c.name.charAt(0) }}</span>
        </div>
        <div class="card-body">
          <div class="card-name">{{ c.name }}</div>
          <div class="card-tags">
            <span class="tag">{{ c.role }}</span>
            <span v-if="c.gender" class="tag">{{ c.gender }}</span>
            <span v-if="c.race" class="tag">{{ c.race }}</span>
            <span class="tag">{{ c.status === 'active' ? '活跃' : c.status === 'inactive' ? '退场' : '已故' }}</span>
          </div>
          <div v-if="c.personality" class="card-text">{{ c.personality }}</div>
          <div class="card-footer">
            <span class="card-date">{{ c.created_at }}</span>
            <div>
              <button class="btn-sm" @click="handleEdit(c.id)">编辑</button>
              <button class="btn-sm btn-danger" @click="handleDelete(c.id)">删除</button>
            </div>
          </div>
        </div>
      </div>
      <div v-if="chars.length === 0" class="empty">暂无角色，点击右上角新建</div>
    </div>
  </div>
</template>

<style scoped>
.char-page { height: 100%; overflow-y: auto; padding: 24px; background: #0d1117; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h2 { margin: 0; font-size: 18px; font-weight: 700; color: #f0f6fc; }
.btn-primary { padding: 8px 16px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 600; }
.btn-primary:hover { background: #2ea043; }
.btn-secondary { padding: 8px 16px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 13px; }

.edit-panel { background: #161b22; border: 1px solid #21262d; border-radius: 10px; padding: 24px; max-width: 700px; }
.edit-panel h3 { margin: 0 0 16px; font-size: 16px; color: #f0f6fc; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.form-field.full { grid-column: 1 / -1; }
.form-field label { display: block; font-size: 11px; font-weight: 600; color: #8b949e; margin-bottom: 4px; }
.form-field input, .form-field textarea { width: 100%; padding: 8px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 13px; font-family: inherit; outline: none; }
.form-field input:focus, .form-field textarea:focus { border-color: #58a6ff; }
.form-field textarea { resize: vertical; }
.radio-group { display: flex; gap: 4px; flex-wrap: wrap; }
.radio-btn { padding: 4px 10px; border: 1px solid #30363d; background: #0d1117; color: #8b949e; border-radius: 6px; cursor: pointer; font-size: 11px; font-family: inherit; }
.radio-btn.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }
.form-actions { display: flex; gap: 8px; margin-top: 16px; }

.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; }
.char-card { display: flex; gap: 12px; padding: 16px; background: #161b22; border: 1px solid #21262d; border-radius: 10px; }
.card-avatar { width: 48px; height: 48px; border-radius: 50%; background: #1f6feb; color: #fff; display: flex; align-items: center; justify-content: center; font-size: 20px; font-weight: 700; flex-shrink: 0; }
.card-body { flex: 1; min-width: 0; }
.card-name { font-size: 15px; font-weight: 600; color: #f0f6fc; margin-bottom: 6px; }
.card-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 6px; }
.tag { font-size: 10px; padding: 2px 6px; background: #21262d; border: 1px solid #30363d; border-radius: 8px; color: #8b949e; }
.card-text { font-size: 12px; color: #8b949e; line-height: 1.4; margin-bottom: 6px; }
.card-footer { display: flex; justify-content: space-between; align-items: center; }
.card-date { font-size: 10px; color: #484f58; }
.btn-sm { padding: 3px 8px; background: transparent; border: 1px solid #30363d; color: #c9d1d9; border-radius: 4px; cursor: pointer; font-size: 10px; font-family: inherit; margin-left: 4px; }
.btn-sm:hover { background: #1c2128; }
.btn-danger { color: #f85149; border-color: rgba(248,81,73,0.3); }
.btn-danger:hover { background: rgba(248,81,73,0.1); }
.empty { grid-column: 1 / -1; text-align: center; color: #484f58; font-size: 13px; padding: 40px; }

/* Avatar */
.avatar-row { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.avatar-preview { position: relative; width: 72px; height: 72px; border-radius: 50%; background: #1f6feb; display: flex; align-items: center; justify-content: center; cursor: pointer; overflow: hidden; border: 2px solid #30363d; }
.avatar-preview:hover .avatar-overlay { opacity: 1; }
.avatar-img { width: 100%; height: 100%; object-fit: cover; }
.avatar-initials { font-size: 28px; font-weight: 700; color: #fff; user-select: none; }
.avatar-overlay { position: absolute; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; font-size: 11px; color: #fff; opacity: 0; transition: opacity 0.15s; }
.avatar-actions { display: flex; gap: 4px; }
.card-avatar { width: 48px; height: 48px; border-radius: 50%; background: #1f6feb; color: #fff; display: flex; align-items: center; justify-content: center; font-size: 20px; font-weight: 700; flex-shrink: 0; overflow: hidden; }
.card-avatar img { width: 100%; height: 100%; object-fit: cover; }
</style>
