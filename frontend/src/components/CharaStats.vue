<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { FindSaveFiles, GetCharacterStatsEditable, ApplyCharacterStats } from '../../wailsjs/go/main/App'

const emit = defineEmits(['status'])

const slots = ref([])
const list = ref([])        // [{ slot, name, count }]
const edits = reactive({})  // slot -> string(输入框值)
const savePath = ref('')
const loading = ref(false)
const saving = ref(false)
const sortDesc = ref(false)
const newSave = ref(false)
const error = ref('')
const outPath = ref('')

const sorted = computed(() => {
  if (!sortDesc.value) return list.value
  return [...list.value].sort((a, b) => b.count - a.count)
})

// 有改动（输入值与当前值不同且有效）的行
const changed = computed(() =>
  list.value
    .filter(c => {
      const v = edits[c.slot]
      return v !== undefined && v !== '' && !isNaN(parseInt(v)) && parseInt(v) !== c.count
    })
    .map(c => ({ slot: c.slot, name: c.name, count: parseInt(edits[c.slot]) }))
)

async function scanSaves() {
  slots.value = await FindSaveFiles() || []
}

async function load(path) {
  loading.value = true
  savePath.value = path
  error.value = ''
  outPath.value = ''
  try {
    list.value = await GetCharacterStatsEditable(path, newSave.value) || []
    Object.keys(edits).forEach(k => delete edits[k])
    list.value.forEach(c => { edits[c.slot] = String(c.count) })
  } catch (err) {
    list.value = []
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await scanSaves()
  if (savePath.value) await load(savePath.value)
}

function switchVersion(value) {
  if (newSave.value === value) return
  newSave.value = value
  if (savePath.value) load(savePath.value)
}

async function applyEdits() {
  if (!savePath.value) { emit('status', '请先加载存档', 'error'); return }
  const payload = changed.value
  if (!payload.length) { emit('status', '没有改动的角色次数', 'error'); return }
  saving.value = true
  error.value = ''
  try {
    outPath.value = await ApplyCharacterStats(savePath.value, '', payload)
    emit('status', '已写入 ' + payload.length + ' 个角色，输出到新存档', 'success')
  } catch (err) {
    error.value = String(err)
    emit('status', String(err), 'error')
  } finally {
    saving.value = false
  }
}

onMounted(scanSaves)
</script>

<template>
  <div class="root">
    <div class="section">
      <div class="header">
        <span class="title">角色次数统计</span>
        <span class="hint">显示存档角色任务次数(新-DLC更新后创建的存档/旧-DLC更新前创建并转换过来的存档)</span>
      </div>
      <div class="slots">
        <button v-for="s in slots" :key="s.index" class="slot-btn"
          :class="{ on: savePath === s.path }" @click="load(s.path)">
          {{ s.name }}
        </button>
        <button class="btn-refresh" @click="refresh">刷新</button>
      </div>
      <div class="version-row">
        <span class="version-label">存档版本</span>
        <div class="version-switch">
          <button :class="{ on: !newSave }" @click="switchVersion(false)">旧版转换存档</button>
          <button :class="{ on: newSave }" @click="switchVersion(true)">DLC更新后新建存档</button>
        </div>
        <button v-if="list.length" class="btn-sort" @click="sortDesc = !sortDesc">
          {{ sortDesc ? '恢复原序' : '按次数排序' }}
        </button>
      </div>

      <div v-if="loading" class="empty">解析中...</div>
      <div v-else-if="error" class="empty">{{ error }}</div>
      <template v-else-if="list.length">
        <div class="table">
          <div class="row row-head">
            <span class="col-name">角色</span>
            <span class="col-count">当前</span>
            <span class="col-edit">修改为</span>
          </div>
          <div v-for="c in sorted" :key="c.slot" class="row">
            <span class="col-name">{{ c.name }}</span>
            <span class="col-count">{{ c.count }}</span>
            <input v-model="edits[c.slot]" type="number" min="0" class="edit-input" placeholder="次数" />
          </div>
        </div>

        <div class="apply-bar">
          <span class="apply-info">{{ changed.length ? ('待写入 ' + changed.length + ' 个改动') : '修改上方输入框后写入新存档（原存档不变）' }}</span>
          <button class="btn-apply" @click="applyEdits" :disabled="saving || !changed.length">
            {{ saving ? '写入中...' : '应用写入到新存档' }}
          </button>
        </div>
        <div v-if="outPath" class="out-path">已保存：{{ outPath }}</div>
      </template>
      <div v-else-if="savePath" class="empty">未找到当前档案角色次数</div>
    </div>
  </div>
</template>

<style scoped>
.root { display:flex; flex-direction:column; gap:10px; width:100%; max-width:720px; margin:0 auto; padding-bottom:40px; }
.section { border-radius:12px; padding:14px 16px; background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.06); display:flex; flex-direction:column; gap:10px; }
.header { display:flex; align-items:center; justify-content:space-between; gap:10px; }
.title { font-size:0.88rem; font-weight:600; color:rgba(255,255,255,0.65); letter-spacing:1px; }
.hint { font-size:0.68rem; color:rgba(255,255,255,0.25); text-align:right; }
.slots { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.version-row { display:flex; align-items:center; gap:10px; }
.version-label { font-size:0.76rem; color:rgba(255,255,255,0.4); }
.version-switch { display:flex; border:1px solid rgba(255,255,255,0.12); border-radius:6px; overflow:hidden; }
.version-switch button { padding:6px 10px; border:0; border-right:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.03); color:rgba(255,255,255,0.45); font-size:0.72rem; cursor:pointer; }
.version-switch button:last-child { border-right:0; }
.version-switch button.on { background:rgba(103,232,249,0.12); color:#67e8f9; }
.slot-btn, .btn-refresh, .btn-sort { padding:6px 14px; border-radius:6px; border:1px solid rgba(255,255,255,0.12); background:rgba(255,255,255,0.05); color:rgba(255,255,255,0.5); font-size:0.78rem; cursor:pointer; transition:background 0.2s; }
.slot-btn { padding:8px 14px; }
.slot-btn:hover, .btn-refresh:hover, .btn-sort:hover { background:rgba(255,255,255,0.1); color:rgba(255,255,255,0.7); }
.slot-btn.on { border-color:rgba(103,232,249,0.4); background:rgba(103,232,249,0.1); color:#67e8f9; }
.batch-row { display:flex; gap:8px; align-items:center; }
.table { display:flex; flex-direction:column; background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.06); border-radius:12px; overflow:hidden; }
.row { display:flex; align-items:center; padding:7px 14px; gap:8px; border-bottom:1px solid rgba(255,255,255,0.02); }
.row:hover { background:rgba(255,255,255,0.02); }
.row-head { background:rgba(255,255,255,0.03); border-bottom:1px solid rgba(255,255,255,0.05); font-size:0.7rem; color:rgba(255,255,255,0.3); font-weight:600; }
.col-name { flex:1; font-size:0.8rem; color:rgba(255,255,255,0.6); overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.col-count { width:56px; text-align:right; font-size:0.8rem; color:#67e8f9; font-family:'Courier New',monospace; flex-shrink:0; }
.col-edit { width:96px; flex-shrink:0; display:flex; justify-content:flex-end; }
.edit-input { width:96px; box-sizing:border-box; padding:5px 8px; border-radius:6px; border:1px solid rgba(255,255,255,0.15); background:rgba(255,255,255,0.07); color:#fff; font-size:0.82rem; outline:none; }
.edit-input:focus { border-color:rgba(103,232,249,0.5); background:rgba(255,255,255,0.12); }
.edit-input::-webkit-outer-spin-button, .edit-input::-webkit-inner-spin-button { -webkit-appearance:none; margin:0; }
.row-head .col-edit { justify-content:flex-end; padding-right:4px; }
.apply-bar { display:flex; align-items:center; justify-content:space-between; gap:10px; margin-top:10px; padding:10px 14px; border-radius:10px; background:rgba(255,255,255,0.03); border:1px solid rgba(255,255,255,0.06); }
.apply-info { font-size:0.74rem; color:rgba(255,255,255,0.4); }
.btn-apply { padding:7px 16px; border-radius:8px; border:1px solid rgba(165,180,252,0.35); background:rgba(165,180,252,0.12); color:#a5b4fc; font-size:0.8rem; font-weight:600; cursor:pointer; white-space:nowrap; transition:background 0.2s; }
.btn-apply:not(:disabled):hover { background:rgba(165,180,252,0.22); }
.btn-apply:disabled { opacity:0.4; cursor:not-allowed; }
.out-path { margin-top:8px; font-size:0.72rem; color:#4ade80; font-family:'Courier New',monospace; word-break:break-all; }
.empty { font-size:0.78rem; color:rgba(255,255,255,0.3); text-align:center; padding:12px 0; }
</style>
