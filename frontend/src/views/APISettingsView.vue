<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { GetConfig, UpdateConfig, TestLLMConnection, TestEmbeddingLocal, SelectExeFile, SelectGGUFFile, GetEmbedderEngine } from '../../wailsjs/go/main/App'
import { config } from '../../wailsjs/go/models'

type AppConfig = config.Config

const apiConfig = ref<AppConfig | null>(null)
const llmSaved = ref(false)
const embedSaved = ref(false)
const testLLMStatus = ref<'idle'|'testing'|'ok'|'fail'>('idle')
const testEmbedStatus = ref<'idle'|'testing'|'ok'|'fail'>('idle')
const testEmbedResult = ref('')
const activeEngine = ref('')

async function loadConfig() {
  try {
    apiConfig.value = await GetConfig()
    activeEngine.value = await GetEmbedderEngine()
  } catch (e) { console.error(e) }
}

async function handleTestLLM() {
  if (!apiConfig.value) return
  testLLMStatus.value = 'testing'
  try {
    await TestLLMConnection(apiConfig.value.llm.base_url, apiConfig.value.llm.api_key)
    testLLMStatus.value = 'ok'
  } catch { testLLMStatus.value = 'fail' }
  setTimeout(() => { if (testLLMStatus.value !== 'testing') testLLMStatus.value = 'idle' }, 3000)
}

async function handleSaveLLM() {
  if (!apiConfig.value) return
  try {
    await UpdateConfig(new config.Config({
      llm: new config.LLMConfig(apiConfig.value.llm),
      embedding: new config.EmbeddingConfig(apiConfig.value.embedding),
    }))
    llmSaved.value = true
    setTimeout(() => { llmSaved.value = false }, 2000)
  } catch (e) { console.error(e) }
}

async function handleSaveEmbedding() {
  if (!apiConfig.value) return
  try {
    await UpdateConfig(new config.Config({
      llm: new config.LLMConfig(apiConfig.value.llm),
      embedding: new config.EmbeddingConfig(apiConfig.value.embedding),
    }))
    embedSaved.value = true
    setTimeout(() => { embedSaved.value = false }, 2000)
  } catch (e) { console.error(e) }
}

function presetLLMProvider(p: string) {
  if (!apiConfig.value) return
  const c = apiConfig.value.llm
  if (p === 'openai') { c.base_url = 'https://api.deepseek.com/v1'; c.chat_model = 'deepseek-chat' }
  else if (p === 'anthropic') { c.base_url = 'https://api.anthropic.com/v1'; c.chat_model = 'claude-sonnet-4-20250514' }
}

async function handleSelectExe() {
  try {
    const path = await SelectExeFile()
    if (path && apiConfig.value) {
      apiConfig.value.embedding.server_path = path
    }
  } catch (e) { console.error(e) }
}

async function handleSelectGGUF() {
  try {
    const path = await SelectGGUFFile()
    if (path && apiConfig.value) {
      apiConfig.value.embedding.model_path = path
    }
  } catch (e) { console.error(e) }
}

async function handleTestEmbedding() {
  testEmbedStatus.value = 'testing'
  testEmbedResult.value = ''
  try {
    const result = await TestEmbeddingLocal()
    testEmbedStatus.value = 'ok'
    testEmbedResult.value = result
  } catch (e: any) {
    testEmbedStatus.value = 'fail'
    testEmbedResult.value = e?.message || String(e)
  }
  setTimeout(() => { if (testEmbedStatus.value !== 'testing') testEmbedStatus.value = 'idle' }, 5000)
}

onMounted(loadConfig)
</script>

<template>
  <div class="api-page">
    <div class="api-form" v-if="apiConfig">
      <h3>LLM 对话模型</h3>
      <p class="hint">用于剧情推演、润色、设定校验等。</p>

      <div class="form-field">
        <label>API 格式</label>
        <div class="provider-group">
          <button v-for="p in ['openai','anthropic']" :key="p"
            class="provider-btn" :class="{ active: apiConfig.llm.provider === p }"
            @click="apiConfig.llm.provider = p; presetLLMProvider(p)">
            {{ p === 'openai' ? 'OpenAI 格式' : 'Anthropic 格式' }}
          </button>
        </div>
      </div>

      <div class="form-field">
        <label>API Base URL</label>
        <input v-model="apiConfig.llm.base_url" placeholder="https://api.deepseek.com/v1" />
      </div>

      <div class="form-field">
        <label>API Key</label>
        <div class="key-row">
          <input v-model="apiConfig.llm.api_key" type="password" placeholder="sk-..." />
          <button class="btn-test" :class="testLLMStatus" @click="handleTestLLM" :disabled="testLLMStatus==='testing'">
            {{ testLLMStatus === 'testing' ? '测试中' : testLLMStatus === 'ok' ? '✓ 正常' : testLLMStatus === 'fail' ? '✗ 失败' : '测试' }}
          </button>
        </div>
      </div>

      <div class="form-field">
        <label>模型名</label>
        <input v-model="apiConfig.llm.chat_model" placeholder="deepseek-chat / claude-sonnet-4-20250514" />
      </div>

      <div class="form-actions">
        <button class="btn-primary" @click="handleSaveLLM">保存配置</button>
        <span v-if="llmSaved" class="saved">✓ 已保存</span>
      </div>

      <div class="divider"></div>
      <h3>Embedding 本地模型</h3>
      <p class="hint">用 llama.cpp + GGUF 在本地运行 embedding 模型。</p>

      <div class="form-field">
        <label>引擎</label>
        <div class="provider-group">
          <button v-for="p in ['hash','llamacpp']" :key="p"
            class="provider-btn" :class="{ active: apiConfig.embedding.engine === p }"
            @click="apiConfig.embedding.engine = p">
            {{ p === 'hash' ? '哈希向量（内置）' : 'llama.cpp + GGUF' }}
          </button>
        </div>
        <p v-if="activeEngine" class="engine-status">
          当前运行: <span :class="{ warn: apiConfig.embedding.engine === 'llamacpp' && activeEngine !== 'llamacpp' }">{{ activeEngine === 'llamacpp' ? 'llama.cpp + GGUF' : '哈希向量（内置）' }}</span>
          <span v-if="apiConfig.embedding.engine === 'llamacpp' && activeEngine !== 'llamacpp'" class="warn-hint">（llama-server 未启动或启动失败，已回退到哈希引擎）</span>
        </p>
      </div>

      <template v-if="apiConfig.embedding.engine === 'llamacpp'">
      <div class="form-field">
        <label>llama-server 路径</label>
        <div class="path-row">
          <input v-model="apiConfig.embedding.server_path" placeholder="留空自动在 PATH 查找" />
          <button class="btn-browse" @click="handleSelectExe">浏览...</button>
        </div>
      </div>
      <div class="form-field">
        <label>GGUF 模型路径</label>
        <div class="path-row">
          <input v-model="apiConfig.embedding.model_path" placeholder="bge-small-zh-q5_k_m.gguf" />
          <button class="btn-browse" @click="handleSelectGGUF">浏览...</button>
        </div>
      </div>
      <div class="form-field">
        <label>服务端口</label>
        <input v-model.number="apiConfig.embedding.port" type="number" placeholder="18635" />
      </div>
      <div class="note-box">
        <p><strong>使用前：</strong></p>
        <p>1. 下载 <code>llama-server</code>：<a href="https://github.com/ggml-org/llama.cpp/releases" target="_blank">github.com/ggml-org/llama.cpp</a></p>
        <p>2. 下载 GGUF 模型（如 <code>bge-small-zh</code>）：<a href="https://huggingface.co/ChristianAzinn/bge-small-zh-Q5_K_M-GGUF" target="_blank">HuggingFace</a></p>
        <p>3. 填入上方路径后保存，重启应用即可生效。</p>
      </div>
      </template>

      <template v-else>
      <div class="note-box">
        <p>内置字符 n-gram 哈希向量引擎，零外部依赖，离线可用。</p>
      </div>
      </template>

      <div class="form-field">
        <label>Embedding 测试</label>
        <div class="test-row">
          <button class="btn-test" :class="testEmbedStatus" @click="handleTestEmbedding" :disabled="testEmbedStatus==='testing'">
            {{ testEmbedStatus === 'testing' ? '测试中' : testEmbedStatus === 'ok' ? '✓ 正常' : testEmbedStatus === 'fail' ? '✗ 失败' : '测试 Embedding' }}
          </button>
          <span v-if="testEmbedResult" class="test-result" :class="testEmbedStatus">{{ testEmbedResult }}</span>
        </div>
      </div>

      <div class="form-actions">
        <button class="btn-primary" @click="handleSaveEmbedding">保存配置</button>
        <span v-if="embedSaved" class="saved">✓ 已保存</span>
      </div>
    </div>
    <div v-else class="loading">加载中...</div>
  </div>
</template>

<style scoped>
.api-page { height: 100%; overflow-y: auto; padding: 32px 40px; background: #0d1117; }
.api-form { max-width: 620px; margin: 0 auto; }
h3 { font-size: 16px; font-weight: 600; color: #f0f6fc; margin: 0 0 4px; }
.hint { font-size: 12px; color: #8b949e; margin: 0 0 20px; }
.divider { height: 1px; background: #21262d; margin: 24px 0; }
.form-field { margin-bottom: 16px; }
.form-field label { display: block; font-size: 12px; font-weight: 600; color: #8b949e; margin-bottom: 6px; }
.form-field input { width: 100%; padding: 8px 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 14px; font-family: inherit; outline: none; }
.form-field input:focus { border-color: #58a6ff; }
.provider-group { display: flex; gap: 4px; }
.provider-btn { padding: 6px 16px; border: 1px solid #30363d; background: #0d1117; color: #8b949e; border-radius: 6px; cursor: pointer; font-size: 12px; font-family: inherit; transition: all 0.15s; }
.provider-btn:hover { border-color: #58a6ff; color: #c9d1d9; }
.provider-btn.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }
.key-row { display: flex; gap: 8px; }
.key-row input { flex: 1; }
.path-row { display: flex; gap: 8px; }
.path-row input { flex: 1; }
.btn-browse { flex-shrink: 0; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 11px; font-weight: 600; font-family: inherit; border: 1px solid #30363d; background: #21262d; color: #c9d1d9; white-space: nowrap; }
.btn-browse:hover { background: #30363d; }
.btn-test { flex-shrink: 0; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 11px; font-weight: 600; font-family: inherit; border: 1px solid #30363d; background: #21262d; color: #c9d1d9; white-space: nowrap; }
.btn-test:hover:not(:disabled) { background: #30363d; }
.btn-test:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-test.ok { background: rgba(63,185,80,0.15); border-color: #3fb950; color: #3fb950; }
.btn-test.fail { background: rgba(248,81,73,0.15); border-color: #f85149; color: #f85149; }
.test-row { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.test-result { font-size: 12px; font-family: monospace; white-space: pre-wrap; word-break: break-all; }
.test-result.ok { color: #3fb950; }
.test-result.fail { color: #f85149; }
.form-actions { display: flex; gap: 8px; align-items: center; margin-top: 24px; }
.btn-primary { padding: 8px 16px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 600; }
.btn-primary:hover { background: #2ea043; }
.saved { font-size: 13px; color: #3fb950; font-weight: 600; }
.engine-status { font-size: 11px; color: #8b949e; margin: 6px 0 0; }
.engine-status span { font-weight: 600; color: #c9d1d9; }
.engine-status span.warn { color: #d29922; }
.warn-hint { font-size: 11px; color: #d29922; margin-left: 4px; }
.loading { display: flex; align-items: center; justify-content: center; height: 100%; color: #484f58; }
.note-box { padding: 14px; background: #1c2128; border: 1px solid #30363d; border-radius: 8px; margin-bottom: 20px; }
.note-box p { margin: 0 0 8px; font-size: 12px; color: #8b949e; line-height: 1.5; }
.note-box p:last-child { margin: 0; }
.note-box code { padding: 2px 6px; background: #0d1117; border-radius: 4px; font-size: 12px; color: #58a6ff; }
.note-box a { color: #58a6ff; }
</style>
