/**
 * main.js - 入口：装配组件、事件接线、渲染节流
 *
 * 数据流：WS / HTTP -> store -> subscribe -> scheduleRender -> 各视图 update。
 * 单选态右侧栏显示 DetailPanel，多选态显示 BatchActionBar，默认 RobotList。
 */

import * as api from './api.js'
import { connectWS } from './ws.js'
import {
  subscribe, getState, upsertRobot, loadRobots, removeRobot, setWsConnected,
  selectRobot, clearSelection, toggleChecked, enterMultiSelect, exitMultiSelect,
  setCheckedIds, setTool, toggleTrail, resetViewport, setLocalTarget, clearLocalTarget,
  setHistoryTrack, TOOLS,
} from './store.js'
import { toast } from './components/toast.js'
import { createDashboard } from './views/dashboard.js'
import { createMapToolbar } from './views/mapToolbar.js'
import { createMap } from './views/map.js'
import { createSelection } from './views/selection.js'
import { createRobotList } from './views/robotList.js'
import { createDetailPanel } from './views/detailPanel.js'
import { createBatchActionBar } from './views/batchActionBar.js'

/* ========== 组件装配 ========== */

const dashboard = createDashboard(document.getElementById('dashboard'))

const mapArea = document.getElementById('mapArea')
const mapToolbar = createMapToolbar({
  onToolChange: setTool,
  onToggleTrail: toggleTrail,
  onResetView: resetViewport,
})
mapArea.appendChild(mapToolbar.el)
mapArea.appendChild(mapToolbar.hintEl)

const map = createMap(document.getElementById('map'), {
  // 取点工具：已单选机器人则回填目标坐标，未选中仅更新坐标提示
  onCanvasClick(x, y) {
    const s = getState()
    if (s.selectedId && !s.multiSelect) detailPanel.setTarget(x, y)
  },
  onRobotClick(id) {
    const s = getState()
    if (s.multiSelect) toggleChecked(id)
    else selectRobot(id)
  },
})

const selection = createSelection(map.el, map, {
  onRectSelect(ids) {
    if (ids.length === 0) {
      toast.show('选区内无机器人', 'info')
      return
    }
    enterMultiSelect()
    setCheckedIds(ids)
  },
})

const sidebar = document.getElementById('sidebar')
const robotList = createRobotList(sidebar, {
  onSelect: selectRobot,
  onCheck: toggleChecked,
})
const detailPanel = createDetailPanel(sidebar, {
  onMove: sendMove,
  onStop: sendStop,
  onBack: clearSelection,
  onLoadTrack: toggleHistoryTrack,
})
const batchBar = createBatchActionBar(sidebar, {
  onBatchMove: sendBatchMove,
  onBatchStop: sendBatchStop,
  onExit: exitMultiSelect,
})

// WS 断开遮罩：保留最后数据但提示可能过期，不白屏
const staleMask = document.createElement('div')
staleMask.className = 'map-stale-mask'
staleMask.innerHTML = '<div class="map-stale-mask__text">连接已断开，数据可能过期，正在重连...</div>'
staleMask.style.display = 'none'
mapArea.appendChild(staleMask)

/* ========== 渲染节流（≤1 次/100ms）========== */

const RENDER_INTERVAL = 100
let renderPending = false
let lastRenderTime = 0

function scheduleRender() {
  if (renderPending) return
  renderPending = true
  const elapsed = Date.now() - lastRenderTime
  if (elapsed >= RENDER_INTERVAL) {
    renderPending = false
    lastRenderTime = Date.now()
    render()
  } else {
    setTimeout(() => {
      renderPending = false
      lastRenderTime = Date.now()
      render()
    }, RENDER_INTERVAL - elapsed)
  }
}

/** render 全量刷新各视图；右侧栏按单选/多选/默认三态切换 */
function render() {
  const s = getState()
  dashboard.update(s.robots)
  map.update(s)
  selection.update(s)
  mapToolbar.update(s)

  robotList.el.style.display = !s.multiSelect && !s.selectedId ? '' : 'none'
  detailPanel.el.style.display = !s.multiSelect && s.selectedId ? '' : 'none'
  batchBar.el.style.display = s.multiSelect ? '' : 'none'

  if (!s.multiSelect && s.selectedId) detailPanel.update(s)
  if (s.multiSelect) batchBar.update(s)
  robotList.update(s)
  staleMask.style.display = s.wsConnected ? 'none' : ''
}

subscribe(scheduleRender)

/* ========== 数据接入 ========== */

connectWS(
  (msg) => {
    if (!msg.id) return
    if (msg.state === 'OFFLINE') {
      removeRobot(msg.id)
      return
    }
    upsertRobot(msg)
    // 机器人转 IDLE 视为到达目标，清除本地下发标记
    if (msg.state === 'IDLE') clearLocalTarget(msg.id)
    scheduleRender()
  },
  (connected) => {
    setWsConnected(connected)
    document.getElementById('wsDot').classList.toggle('connected', connected)
    document.getElementById('wsLabel').textContent = connected ? '已连接' : '已断开'
    // 重连成功后重新拉取列表：断线期间的 WS 广播已丢失，
    // 且静止设备不再周期上报，必须靠 HTTP 同步一次 registry 快照
    if (connected) {
      api
        .list()
        .then((devices) => loadRobots(devices))
        .catch(() => {}) // 拉取失败不提示（连接可能刚建立）
    }
  },
)

// 页面加载：HTTP 拉一次设备列表作为初始数据（此后由 WS 推流维持）
api
  .list()
  .then((devices) => loadRobots(devices))
  .catch((e) => toast.show(`加载设备列表失败: ${e.message}`, 'error'))

/* ========== 指令下发 ========== */

/** sendMove 单机移动指令：loading 态防重复提交，成功记录目标标记 */
async function sendMove(x, y) {
  const id = getState().selectedId
  if (!id) return
  detailPanel.setBusy('move', true)
  try {
    await api.move(id, x, y)
    setLocalTarget(id, x, y)
    toast.show(`指令已下发 ${id}`, 'success') // 只代表"已下发"，非"移动成功"
  } catch (e) {
    toast.show(`下发失败: ${e.message}`, 'error')
  } finally {
    detailPanel.setBusy('move', false)
  }
}

/** sendStop 单机紧急停止 */
async function sendStop() {
  const id = getState().selectedId
  if (!id) return
  detailPanel.setBusy('stop', true)
  try {
    await api.stop(id)
    toast.show(`停止指令已下发 ${id}`, 'success')
  } catch (e) {
    toast.show(`停止失败: ${e.message}`, 'error')
  } finally {
    detailPanel.setBusy('stop', false)
  }
}

/** sendBatchMove 批量移动：allSettled 汇总成功/失败 */
async function sendBatchMove(x, y) {
  const ids = [...getState().selectedIds]
  const { ok, fail } = await api.batchMove(ids, x, y)
  ids.forEach((id) => ok.includes(id) && setLocalTarget(id, x, y))
  reportBatchResult('移动', ok, fail)
}

/** sendBatchStop 批量停止 */
async function sendBatchStop() {
  const ids = [...getState().selectedIds]
  const { ok, fail } = await api.batchStop(ids)
  reportBatchResult('停止', ok, fail)
}

/** reportBatchResult 批量结果 Toast 汇总（成功 N / 失败 M + 失败 ID） */
function reportBatchResult(action, ok, fail) {
  if (fail.length === 0) {
    toast.show(`${action}指令已下发 ${ok.length} 台`, 'success')
  } else {
    toast.show(
      `${action}: 成功 ${ok.length} 台 / 失败 ${fail.length} 台（${fail.map((f) => f.id).join(', ')}）`,
      'error',
    )
  }
}

/* ========== 历史轨迹（详情面板入口）========== */

/** toggleHistoryTrack 加载/清除近 1 小时历史轨迹（GET /devices/:id/track） */
async function toggleHistoryTrack() {
  const s = getState()
  const id = s.selectedId
  if (!id) return

  // 已加载则清除
  if (s.historyTrack && s.historyTrack.id === id) {
    setHistoryTrack(null)
    return
  }

  detailPanel.setTrackLoading(true)
  try {
    const to = Date.now()
    const from = to - 3600 * 1000
    const points = await api.track(id, from, to, '10s')
    if (points.length === 0) {
      toast.show('该时段无轨迹数据', 'info')
    } else {
      setHistoryTrack({ id, points })
      toast.show(`已加载 ${points.length} 个轨迹点（近 1 小时）`, 'success')
    }
  } catch (e) {
    toast.show(`轨迹查询失败: ${e.message}`, 'error')
  } finally {
    detailPanel.setTrackLoading(false)
  }
}

/* ========== 快捷键 ========== */

document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return
  const s = getState()
  if (s.multiSelect) exitMultiSelect() // ESC：先退多选
  else if (s.tool !== TOOLS.PICK) setTool(TOOLS.PICK) // 再回取点工具
  else clearSelection() // 最后退详情
})

/* 首帧渲染 */
render()
