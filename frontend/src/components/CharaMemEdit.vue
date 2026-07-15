<script setup>
import { ref } from 'vue'
import { CharaAttach, CharaDetach, CharaGetAll, CharaSetOne } from '../../wailsjs/go/main/App'

const emit = defineEmits(['status'])

const connected = ref(false)
const pid = ref(0)
const list = ref([])        // [{ index, name, count }]
const edits = ref({})       // index -> string(输入框值)
const loading = ref(false)
const phase = ref('')       // 当前阶段提示文案
const errMsg = ref('')      // 常驻错误信息
const savingIndex = ref(-1)

function fillList(data) {
  list.value = data
  const m = {}
  data.forEach(c => { m[c.index] = String(c.count) })
  edits.value = m
  if (!data.length) errMsg.value = '已连接，但未读取到角色数据（请确认已进入游戏存档）'
}

// 连接：优先复用已有进程连接（例如已在「杂项」里连接过），
// 只有在后端尚未连接时才 CharaAttach，避免拆掉正常连接后重新全量扫描内存。
async function connect() {
  loading.value = true
  errMsg.value = ''
  phase.value = '读取角色次数中...'
  try {
    let data
    try {
      data = (await CharaGetAll()) || []       // 复用已有连接，秒读
    } catch (e1) {
      // 后端未连接 → 附加进程（首次会扫描内存定位列表）后再读
      phase.value = '连接进程并扫描游戏内存，首次可能需要 10~60 秒，请耐心等待...'
      const info = await CharaAttach()
      pid.value = info.pid
      data = (await CharaGetAll()) || []
    }
    connected.value = true
    fillList(data)
    emit('status', pid.value ? ('已连接 PID ' + pid.value) : '已读取角色次数', 'success')
  } catch (e) {
    connected.value = false
    errMsg.value = String(e)
    emit('status', String(e), 'error')
  } finally {
    loading.value = false
    phase.value = ''
  }
}

// 刷新：始终复用当前连接，不重新附加
async function read() {
  loading.value = true
  errMsg.value = ''
  phase.value = '读取角色次数中...'
  try {
    fillList((await CharaGetAll()) || [])
  } catch (e) {
    errMsg.value = String(e)
    emit('status', String(e), 'error')
  } finally {
    loading.value = false
    phase.value = ''
  }
}

async function save(c) {
  const v = parseInt(edits.value[c.index])
  if (isNaN(v) || v < 0) { emit('status', '请输入有效数值', 'error'); return }
  savingIndex.value = c.index
  try {
    await CharaSetOne(c.index, v)
    c.count = v
    emit('status', c.name + ' 已改为 ' + v, 'success')
  } catch (e) {
    emit('status', String(e), 'error')
  } finally {
    savingIndex.value = -1
  }
}

async function disconnect() {
  try { await CharaDetach() } catch (e) { /* ignore */ }
  connected.value = false
  pid.value = 0
  list.value = []
  edits.value = {}
  emit('status', '已断开进程', 'success')
}
</script>

<template>
  <div class="root">
    <div class="section">
      <div class="header">
        <span class="title">角色使用次数（修改）</span>
        <span class="hint">游戏运行中使用，读写进程内存 · 修改后需对应角色结算一局生效</span>
      </div>

      <div class="conn-row">
        <button v-if="!connected" class="btn-primary" @click="connect" :disabled="loading">
          {{ loading ? '连接中...' : '连接游戏进程' }}
        </button>
        <template v-else>
          <span class="pid-badge">{{ pid ? ('已连接 · PID ' + pid) : '已连接' }}</span>
          <button class="btn-refresh" @click="read" :disabled="loading">刷新次数</button>
          <button class="btn-detach" @click="disconnect">断开</button>
        </template>
      </div>

      <div v-if="errMsg" class="err-box">{{ errMsg }}</div>
      <div v-if="loading" class="empty">{{ phase || '读取中...' }}</div>
      <template v-else-if="connected && list.length">
        <div class="table">
          <div class="row row-head">
            <span class="col-name">角色</span>
            <span class="col-count">当前</span>
            <span class="col-edit">修改为</span>
            <span class="col-act"></span>
          </div>
          <div v-for="c in list" :key="c.index" class="row">
            <span class="col-name">{{ c.name }}</span>
            <span class="col-count">{{ c.count }}</span>
            <input v-model="edits[c.index]" type="number" min="0" class="edit-input" placeholder="数值" />
            <button class="btn-save" @click="save(c)"
              :disabled="savingIndex === c.index || edits[c.index] === '' || isNaN(parseInt(edits[c.index]))">
              {{ savingIndex === c.index ? '写入...' : '应用' }}
            </button>
          </div>
        </div>
      </template>
      <div v-else-if="connected" class="empty">未读取到角色数据，请先进入游戏存档后点「刷新次数」</div>
      <div v-else class="empty">请先启动游戏并进入存档，然后点「连接游戏进程」（建议以管理员身份运行本工具）</div>
    </div>
  </div>
</template>

<style scoped>
.root { display:flex; flex-direction:column; gap:10px; width:100%; max-width:720px; margin:0 auto; padding-bottom:40px; }
.section { border-radius:12px; padding:14px 16px; background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.06); display:flex; flex-direction:column; gap:12px; }
.header { display:flex; align-items:center; justify-content:space-between; gap:10px; }
.title { font-size:0.88rem; font-weight:600; color:rgba(255,255,255,0.65); letter-spacing:1px; }
.hint { font-size:0.68rem; color:rgba(255,255,255,0.25); text-align:right; }

.conn-row { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
.btn-primary { padding:8px 18px; border-radius:8px; border:1px solid rgba(103,232,249,0.35); background:rgba(103,232,249,0.12); color:#67e8f9; font-size:0.82rem; font-weight:600; cursor:pointer; transition:background 0.2s; }
.btn-primary:not(:disabled):hover { background:rgba(103,232,249,0.22); }
.btn-primary:disabled { opacity:0.4; cursor:not-allowed; }
.pid-badge { font-size:0.74rem; color:#4ade80; background:rgba(34,197,94,0.15); padding:4px 12px; border-radius:20px; font-weight:600; }
.btn-refresh, .btn-detach { padding:6px 14px; border-radius:6px; border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05); color:rgba(255,255,255,0.5); font-size:0.78rem; cursor:pointer; transition:background 0.2s; }
.btn-refresh:hover { background:rgba(255,255,255,0.1); color:rgba(255,255,255,0.7); }
.btn-detach:hover { background:rgba(239,68,68,0.15); color:#f87171; border-color:rgba(239,68,68,0.3); }

.table { display:flex; flex-direction:column; background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.06); border-radius:12px; overflow:hidden; }
.row { display:flex; align-items:center; padding:7px 14px; gap:10px; border-bottom:1px solid rgba(255,255,255,0.02); }
.row:hover { background:rgba(255,255,255,0.02); }
.row-head { background:rgba(255,255,255,0.03); border-bottom:1px solid rgba(255,255,255,0.05); font-size:0.7rem; color:rgba(255,255,255,0.3); font-weight:600; }
.col-name { flex:1; font-size:0.8rem; color:rgba(255,255,255,0.6); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.col-count { width:56px; text-align:right; font-size:0.8rem; color:#67e8f9; font-family:'Courier New',monospace; flex-shrink:0; }
.col-edit { width:90px; flex-shrink:0; }
.col-act { width:56px; flex-shrink:0; display:flex; justify-content:flex-end; }
.edit-input { width:90px; box-sizing:border-box; padding:5px 8px; border-radius:6px; border:1px solid rgba(255,255,255,0.15); background:rgba(255,255,255,0.07); color:#fff; font-size:0.82rem; outline:none; }
.edit-input:focus { border-color:rgba(255,255,255,0.4); background:rgba(255,255,255,0.12); }
.edit-input::-webkit-outer-spin-button, .edit-input::-webkit-inner-spin-button { -webkit-appearance:none; margin:0; }
.btn-save { padding:5px 12px; border-radius:6px; border:1px solid rgba(165,180,252,0.3); background:rgba(165,180,252,0.1); color:#a5b4fc; font-size:0.76rem; font-weight:600; cursor:pointer; white-space:nowrap; transition:background 0.2s; }
.btn-save:not(:disabled):hover { background:rgba(165,180,252,0.2); }
.btn-save:disabled { opacity:0.4; cursor:not-allowed; }
.empty { font-size:0.78rem; color:rgba(255,255,255,0.3); text-align:center; padding:12px 0; line-height:1.7; }
.err-box { font-size:0.76rem; color:#f87171; background:rgba(239,68,68,0.1); border:1px solid rgba(239,68,68,0.25); border-radius:8px; padding:8px 12px; line-height:1.6; }
</style>
