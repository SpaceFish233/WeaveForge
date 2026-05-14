<script lang="ts" setup>
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Placeholder from '@tiptap/extension-placeholder'
import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from 'prosemirror-state'
import { Decoration, DecorationSet } from 'prosemirror-view'
import { ref, watch, nextTick, onBeforeUnmount, onMounted } from 'vue'

const props = defineProps<{
  content: string
  chapterId: string | null
  pendingReplacement?: { text: string; from: number; to: number } | null
}>()

const emit = defineEmits<{
  (e: 'update:content', content: string): void
  (e: 'markForeshadow', text: string): void
  (e: 'polish', text: string, from: number, to: number): void
  (e: 'replacementDone'): void
}>()

// ─── Search highlight plugin ───
interface SearchMatch { offset: number; length: number }
const searchMatches = ref<SearchMatch[]>([])
const activeMatchIndex = ref(-1)
const searchPluginKey = new PluginKey('searchHighlight')

function createSearchPlugin() {
  return new Plugin({
    key: searchPluginKey,
    props: {
      decorations(state) {
        if (searchMatches.value.length === 0) return DecorationSet.empty
        const doc = state.doc
        const decorations: Decoration[] = []
        // Build text offset → doc position map
        const textMap = buildTextMap(doc)
        searchMatches.value.forEach((m, i) => {
          const from = offsetToPos(textMap, m.offset)
          const to = offsetToPos(textMap, m.offset + m.length)
          if (from !== null && to !== null && from < to) {
            const isActive = i === activeMatchIndex.value
            const cls = isActive ? 'search-hl-active' : 'search-hl'
            decorations.push(Decoration.inline(from, to, { class: cls }))
          }
        })
        return DecorationSet.create(doc, decorations)
      },
    },
  })
}

// Build a mapping from plain text offset to ProseMirror document position
function buildTextMap(doc: any): { offset: number; pos: number }[] {
  const map: { offset: number; pos: number }[] = []
  let textOffset = 0
  doc.descendants((node: any, pos: number) => {
    if (node.isText) {
      map.push({ offset: textOffset, pos })
      textOffset += node.text.length
    }
  })
  return map
}

function offsetToPos(textMap: { offset: number; pos: number }[], targetOffset: number): number | null {
  // Binary search for the text map entry
  let lo = 0, hi = textMap.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (textMap[mid].offset <= targetOffset) lo = mid + 1
    else hi = mid - 1
  }
  if (hi < 0) return null
  const entry = textMap[hi]
  return entry.pos + (targetOffset - entry.offset)
}

const searchExtension = Extension.create({
  name: 'searchHighlight',
  addProseMirrorPlugins() {
    return [createSearchPlugin()]
  },
})

const editor = useEditor({
  content: props.content || '<p></p>',
  editable: true,
  extensions: [
    StarterKit.configure({ heading: { levels: [1, 2, 3, 4] } }),
    Underline,
    Placeholder.configure({ placeholder: '开始写作...' }),
    searchExtension,
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

// Handle pending replacement from polish flow
watch(() => props.pendingReplacement, (r) => {
  if (!r || !editor.value) return
  const { text, from, to } = r
  // Clamp positions to valid doc range
  const docSize = editor.value.state.doc.content.size
  const safeFrom = Math.max(0, Math.min(from, docSize))
  const safeTo = Math.max(safeFrom, Math.min(to, docSize))
  editor.value.chain()
    .focus()
    .deleteRange({ from: safeFrom, to: safeTo })
    .insertContentAt(safeFrom, text)
    .run()
  emit('replacementDone')
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

function handlePolish() {
  if (!editor.value) return
  const { from, to, empty } = editor.value.state.selection
  if (empty) return
  const text = editor.value.state.doc.textBetween(from, to, ' ')
  if (text.trim()) {
    emit('polish', text.trim(), from, to)
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

// ─── Search highlight API ───
function setSearchHighlights(matches: { offset: number; length: number }[], activeIdx: number) {
  searchMatches.value = matches
  activeMatchIndex.value = activeIdx
}

function clearSearchHighlights() {
  searchMatches.value = []
  activeMatchIndex.value = -1
}

function scrollToOffset(offset: number) {
  if (!editor.value) return
  const doc = editor.value.state.doc
  const textMap = buildTextMap(doc)
  const pos = offsetToPos(textMap, offset)
  if (pos !== null) {
    editor.value.chain().focus().setTextSelection({ from: pos, to: pos }).run()
    // Scroll into view
    const { node } = editor.value.view.domAtPos(pos)
    if (node instanceof HTMLElement) {
      node.scrollIntoView({ behavior: 'smooth', block: 'center' })
    } else if (node.parentElement) {
      node.parentElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }
}

defineExpose({ setSearchHighlights, clearSearchHighlights, scrollToOffset })
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
        <div class="menu-item" @click="handlePolish">
          <span class="menu-icon">✒</span>
          <span>润色</span>
        </div>
        <div class="menu-sep"></div>
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
.editor-content :deep(.ProseMirror .search-hl) {
  background: rgba(210, 153, 34, 0.35);
  border-radius: 2px;
}
.editor-content :deep(.ProseMirror .search-hl-active) {
  background: rgba(243, 156, 18, 0.6);
  border-radius: 2px;
  box-shadow: 0 0 0 1px rgba(243, 156, 18, 0.8);
}

/* Context menu */
.context-menu { position: fixed; z-index: 10000; background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 4px; min-width: 160px; box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
.menu-item { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; color: #c9d1d9; transition: background 0.1s; }
.menu-item:hover { background: #1c2128; }
.menu-icon { font-size: 14px; }
.menu-sep { height: 1px; background: #21262d; margin: 4px 8px; }
</style>
