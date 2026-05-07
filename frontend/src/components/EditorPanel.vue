<script lang="ts" setup>
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Placeholder from '@tiptap/extension-placeholder'
import { ref, watch, nextTick, onBeforeUnmount, onMounted } from 'vue'

const props = defineProps<{
  content: string
  chapterId: string | null
}>()

const emit = defineEmits<{
  (e: 'update:content', content: string): void
  (e: 'markForeshadow', text: string): void
}>()

const editor = useEditor({
  content: props.content || '<p></p>',
  editable: true,
  extensions: [
    StarterKit.configure({ heading: { levels: [1, 2, 3, 4] } }),
    Underline,
    Placeholder.configure({ placeholder: '开始写作...' }),
  ],
  onUpdate: ({ editor }) => {
    emit('update:content', editor.getHTML())
  },
})

// Guard against re-entrant content updates
let suppressWatch = false
let lastEmitted = ''
watch(() => props.content, (html) => {
  if (!editor.value || suppressWatch) return
  const cur = editor.value.getHTML()
  if (html !== cur) {
    suppressWatch = true
    editor.value.commands.setContent(html || '<p></p>')
    nextTick(() => { suppressWatch = false })
  }
})

// When parent signals a chapter switch (chapterId changed), sync & focus
watch(() => props.chapterId, (newId, oldId) => {
  if (editor.value && newId && newId !== oldId) {
    suppressWatch = true
    editor.value.commands.setContent(props.content || '<p></p>')
    nextTick(() => {
      suppressWatch = false
      editor.value?.commands.focus('end')
    })
  }
})

// ─── Context menu ─────────────────────────────────────────────────
const menuVisible = ref(false)
const menuX = ref(0)
const menuY = ref(0)

function getSelectedText(): string {
  if (!editor.value) return ''
  const { from, to, empty } = editor.value.state.selection
  if (empty) return ''
  return editor.value.state.doc.textBetween(from, to, ' ')
}

function handleContextMenu(e: MouseEvent) {
  const text = getSelectedText()
  if (!text.trim()) {
    menuVisible.value = false
    return
  }
  e.preventDefault()
  menuX.value = e.clientX
  menuY.value = e.clientY
  menuVisible.value = true
}

function handleMarkForeshadow() {
  const text = getSelectedText()
  if (text.trim()) {
    emit('markForeshadow', text.trim())
  }
  menuVisible.value = false
}

function closeMenu() {
  menuVisible.value = false
}

onMounted(() => {
  document.addEventListener('click', closeMenu)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', closeMenu)
  editor.value?.destroy()
})
</script>

<template>
  <div class="editor-panel">
    <div class="editor-toolbar" v-if="editor">
      <button @click="editor.chain().focus().toggleBold().run()" :class="{ active: editor.isActive('bold') }" title="粗体">B</button>
      <button @click="editor.chain().focus().toggleItalic().run()" :class="{ active: editor.isActive('italic') }" title="斜体"><em>I</em></button>
      <button @click="editor.chain().focus().toggleUnderline().run()" :class="{ active: editor.isActive('underline') }" title="下划线"><u>U</u></button>
      <span class="separator"></span>
      <button @click="editor.chain().focus().toggleHeading({ level: 2 }).run()" :class="{ active: editor.isActive('heading', { level: 2 }) }" title="标题2">H2</button>
      <button @click="editor.chain().focus().toggleHeading({ level: 3 }).run()" :class="{ active: editor.isActive('heading', { level: 3 }) }" title="标题3">H3</button>
      <span class="separator"></span>
      <button @click="editor.chain().focus().toggleBulletList().run()" :class="{ active: editor.isActive('bulletList') }" title="无序列表">•</button>
      <button @click="editor.chain().focus().toggleOrderedList().run()" :class="{ active: editor.isActive('orderedList') }" title="有序列表">1.</button>
      <button @click="editor.chain().focus().toggleBlockquote().run()" :class="{ active: editor.isActive('blockquote') }" title="引用">"</button>
    </div>
    <div class="editor-content" @contextmenu="handleContextMenu">
      <EditorContent :editor="editor" />
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div v-if="menuVisible" class="context-menu" :style="{ left: menuX + 'px', top: menuY + 'px' }" @click.stop>
        <div class="menu-item" @click="handleMarkForeshadow">
          <span class="menu-icon">🔮</span>
          <span>设为伏笔</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.editor-panel { height: 100%; display: flex; flex-direction: column; background: #0d1117; }
.editor-toolbar { display: flex; align-items: center; gap: 2px; padding: 8px 16px; border-bottom: 1px solid #21262d; background: #161b22; flex-wrap: wrap; }
.editor-toolbar button { width: 32px; height: 32px; border: 1px solid transparent; background: transparent; color: #8b949e; border-radius: 4px; cursor: pointer; font-size: 13px; font-family: inherit; display: flex; align-items: center; justify-content: center; transition: all 0.15s; }
.editor-toolbar button:hover { background: #21262d; color: #c9d1d9; }
.editor-toolbar button.active { background: #1f2937; color: #58a6ff; border-color: #30363d; }
.separator { width: 1px; height: 20px; background: #21262d; margin: 0 4px; }
.editor-content { flex: 1; overflow-y: auto; padding: 0; display: flex; flex-direction: column; }
.editor-content :deep(.ProseMirror) {
  flex: 1;
  padding: 24px 32px;
  outline: none;
  min-height: 200px;
}
.editor-content :deep(.ProseMirror p) {
  margin: 0;
}
.editor-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  float: left;
  color: #484f58;
  pointer-events: none;
  height: 0;
}

/* Context menu */
.context-menu { position: fixed; z-index: 10000; background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 4px; min-width: 160px; box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
.menu-item { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; color: #c9d1d9; transition: background 0.1s; }
.menu-item:hover { background: #1c2128; }
.menu-icon { font-size: 14px; }
</style>
