/**
 * store.js - 全局状态存储（发布-订阅）
 *
 * WS / HTTP 更新数据后统一走这里，视图通过 subscribe 感知变化。
 * 单选（selectedId）与多选（selectedIds）互斥：进入多选清空单选，退出多选恢复列表态。
 */

const TRAIL_MAX = 50 // 每台机器人实时尾迹最多保留的点数（环形缓冲）
export const STALE_MS = 10000 // 超过该时长未更新视为"数据过期"

/** 工具枚举：取点（默认）/ 框选 / 拖动 */
export const TOOLS = {
  PICK: 'pick',
  RECT: 'rect',
  PAN: 'pan',
}

// 内部状态（外部只读，修改必须走 setter 以触发通知）
const state = {
  robots: {}, // id -> { id, x, y, battery, state, speed, task_id?, ts(最后更新), trail:[] }
  wsConnected: false,
  selectedId: null, // 单选
  selectedIds: new Set(), // 多选
  multiSelect: false,
  keyword: '', // 列表搜索关键字
  tool: TOOLS.PICK,
  viewMode: 'card', // 列表视图：card / table
  showTrail: true, // 实时尾迹开关
  localTargets: {}, // id -> {x, y} 本地下发的目标点标记
  historyTrack: null, // { id, points:[{ts,x,y}] } 历史轨迹（单选详情态加载）
  viewport: { dx: 0, dy: 0 }, // 地图视口偏移（世界坐标）
}

const listeners = new Set()

/** subscribe 订阅状态变化，返回取消订阅函数 */
export function subscribe(fn) {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

/** notify 通知所有订阅者（传入变更类型便于视图做增量优化） */
function notify(event) {
  listeners.forEach((fn) => fn(state, event))
}

/** getState 获取当前状态快照（只读引用） */
export function getState() {
  return state
}

/* ========== 设备数据（WS / HTTP 入口）========== */

/** upsertRobot 写入/更新一台设备状态，并维护实时尾迹缓冲 */
export function upsertRobot(d) {
  const prev = state.robots[d.id]
  const trail = prev ? prev.trail : []
  // 坐标有变化才追加尾迹点，避免静止设备塞满缓冲
  if (prev && (prev.x !== d.x || prev.y !== d.y)) {
    trail.push({ x: d.x, y: d.y })
    if (trail.length > TRAIL_MAX) trail.shift() // FIFO 淘汰
  }
  state.robots[d.id] = { ...d, ts: Date.now(), trail }
}

/** loadRobots 用 HTTP 设备列表同步快照（页面加载与 WS 重连时）：
 *  新设备写入、已有设备刷新基础字段、列表中消失的设备移除（对账，防幽灵数据） */
export function loadRobots(list) {
  const seen = new Set()
  list.forEach((d) => {
    seen.add(d.id)
    const prev = state.robots[d.id]
    upsertRobot({ ...d, trail: prev ? prev.trail : [] })
  })
  // HTTP 快照里没有的设备说明已离线，清理其关联状态
  Object.keys(state.robots).forEach((id) => {
    if (!seen.has(id)) removeRobot(id)
  })
  notify('robots')
}

/** removeRobot 设备离线：移除数据并清理所有关联选择状态 */
export function removeRobot(id) {
  delete state.robots[id]
  delete state.localTargets[id]
  if (state.selectedId === id) state.selectedId = null
  state.selectedIds.delete(id)
  if (state.historyTrack && state.historyTrack.id === id) state.historyTrack = null
  notify('robots')
}

/** setWsConnected WS 连接状态变化 */
export function setWsConnected(v) {
  state.wsConnected = v
  notify('ws')
}

/* ========== 选择与模式 ========== */

/** selectRobot 单选某台设备（多选模式下不生效） */
export function selectRobot(id) {
  if (state.multiSelect) return
  state.selectedId = id
  notify('select')
}

/** clearSelection 清空单选（返回列表态） */
export function clearSelection() {
  state.selectedId = null
  state.historyTrack = null
  notify('select')
}

/** toggleChecked 多选模式下勾选/取消勾选某台设备 */
export function toggleChecked(id) {
  if (state.selectedIds.has(id)) state.selectedIds.delete(id)
  else state.selectedIds.add(id)
  notify('select')
}

/** setCheckedIds 整体替换多选集合（框选/全选用） */
export function setCheckedIds(ids) {
  state.selectedIds = new Set(ids)
  notify('select')
}

/** enterMultiSelect 进入多选模式（清空单选） */
export function enterMultiSelect() {
  state.multiSelect = true
  state.selectedId = null
  state.historyTrack = null
  notify('mode')
}

/** exitMultiSelect 退出多选模式并清空选择 */
export function exitMultiSelect() {
  state.multiSelect = false
  state.selectedIds.clear()
  notify('mode')
}

/* ========== UI 偏好状态 ========== */

export function setKeyword(kw) {
  state.keyword = kw
  notify('keyword')
}

export function setTool(tool) {
  state.tool = tool
  notify('tool')
}

export function setViewMode(mode) {
  state.viewMode = mode
  notify('viewMode')
}

export function toggleTrail() {
  state.showTrail = !state.showTrail
  notify('trail')
}

/** setViewport 更新地图视口偏移（拖动平移，不发通知——rAF 重绘自然消费） */
export function setViewport(dx, dy) {
  state.viewport = { dx, dy }
}

export function resetViewport() {
  state.viewport = { dx: 0, dy: 0 }
  notify('viewport')
}

/** setLocalTarget 记录本地下发的目标点（指令下发成功后调用） */
export function setLocalTarget(id, x, y) {
  state.localTargets[id] = { x, y }
  notify('target')
}

/** clearLocalTarget 机器人到达（转 IDLE）后清除目标标记 */
export function clearLocalTarget(id) {
  delete state.localTargets[id]
}

/** setHistoryTrack 设置/清除历史轨迹（详情面板加载） */
export function setHistoryTrack(track) {
  state.historyTrack = track
  notify('track')
}
