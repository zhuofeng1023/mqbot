/**
 * detailPanel.js - 机器人详情面板（单选态右侧栏）
 *
 * 实时数据 + 移动任务表单 + 紧急停止（二次确认）；
 * 底部显示最后上报时间与历史轨迹入口（调 GET /devices/:id/track）。
 */

import { createStateBadge } from '../components/stateBadge.js'
import { createBatteryBar } from '../components/batteryBar.js'
import { createConfirmDialog } from '../components/confirmDialog.js'

/**
 * createDetailPanel 创建详情面板
 * @param {HTMLElement} mount 挂载容器
 * @param {{onMove: (x: number, y: number) => void, onStop: () => void,
 *          onBack: () => void, onLoadTrack: () => void, onClearTrack: () => void}} props
 * @returns {{el: HTMLElement, update: (state: object) => void,
 *            setTarget: (x: number, y: number) => void}}
 */
export function createDetailPanel(mount, props) {
  const el = document.createElement('div')
  el.className = 'detail-panel'

  let robot = null // 当前展示的机器人（update 时刷新）
  let trackLoading = false // 历史轨迹按钮 loading 态

  /* -- 返回列表 -- */
  const backBtn = document.createElement('button')
  backBtn.className = 'btn btn--ghost detail-panel__back'
  backBtn.textContent = '← 返回列表'
  backBtn.addEventListener('click', () => props.onBack())

  /* -- 标题：ID + 状态徽章 -- */
  const title = document.createElement('div')
  title.className = 'detail-panel__id'
  const idEl = document.createElement('span')
  const badge = createStateBadge()
  title.append(idEl, badge.el)

  /* -- 实时数据网格 -- */
  const grid = document.createElement('div')
  grid.className = 'detail-panel__grid'
  const cells = {}
  ;[
    ['coords', '坐标'],
    ['battery', '电量'],
    ['speed', '速度'],
    ['task', '任务'],
  ].forEach(([key, label]) => {
    const item = document.createElement('div')
    item.className = 'detail-item'
    const l = document.createElement('div')
    l.className = 'detail-item__label'
    l.textContent = label
    const v = document.createElement('div')
    v.className = 'detail-item__value'
    item.append(l, v)
    grid.appendChild(item)
    cells[key] = v
  })
  const batteryBar = createBatteryBar()
  cells.battery.appendChild(batteryBar.el)

  // 离线提示（设备离线时替代数据网格下方操作）
  const offlineTip = document.createElement('div')
  offlineTip.className = 'detail-panel__offline'
  offlineTip.style.display = 'none'

  /* -- 任务下发表单 -- */
  const formTitle = document.createElement('div')
  formTitle.className = 'section-title'
  formTitle.textContent = '任务下发'

  const formRow = document.createElement('div')
  formRow.className = 'form-row'
  const inputX = mkInput('X')
  const inputY = mkInput('Y')
  formRow.append(inputX.wrap, inputY.wrap)

  function mkInput(label) {
    const wrap = document.createElement('div')
    const l = document.createElement('div')
    l.className = 'form-label'
    l.textContent = label
    const input = document.createElement('input')
    input.className = 'form-input'
    input.type = 'number'
    input.step = '0.1'
    input.placeholder = '0.0'
    wrap.append(l, input)
    return { wrap, input }
  }

  /* -- 操作按钮 -- */
  const actions = document.createElement('div')
  actions.className = 'detail-panel__actions'

  const moveBtn = document.createElement('button')
  moveBtn.className = 'btn btn--primary'
  moveBtn.textContent = '发送移动指令'

  const stopBtn = document.createElement('button')
  stopBtn.className = 'btn btn--danger'
  stopBtn.textContent = '紧急停止'

  actions.append(moveBtn, stopBtn)

  // 发送移动：表单校验后上抛（语义是"已下发"而非"移动成功"）
  moveBtn.addEventListener('click', () => {
    if (!robot) return
    const x = parseFloat(inputX.input.value)
    const y = parseFloat(inputY.input.value)
    if (Number.isNaN(x) || Number.isNaN(y)) return
    props.onMove(x, y)
  })

  // 紧急停止：危险操作二次确认后上抛
  stopBtn.addEventListener('click', () => {
    if (!robot) return
    createConfirmDialog({
      title: '确认停止',
      message: `确认停止 ${robot.id}？该指令不可撤销。`,
      confirmText: '紧急停止',
      danger: true,
      onConfirm: () => props.onStop(),
    })
  })

  /* -- 底部：最后上报时间 + 历史轨迹入口 -- */
  const footer = document.createElement('div')
  footer.className = 'detail-panel__footer'
  const lastseen = document.createElement('span')
  lastseen.className = 'detail-panel__lastseen'
  const trackBtn = document.createElement('button')
  trackBtn.className = 'btn btn--ghost'
  trackBtn.style.padding = '4px 10px'
  trackBtn.addEventListener('click', () => {
    if (trackLoading) return
    // 已加载则清除，未加载则请求
    props.onLoadTrack()
  })
  footer.append(lastseen, trackBtn)

  el.append(backBtn, title, grid, offlineTip, formTitle, formRow, actions, footer)
  mount.appendChild(el)

  /** setTarget 地图取点后回填表单坐标（方式B：地图快捷取点） */
  function setTarget(x, y) {
    inputX.input.value = x.toFixed(1)
    inputY.input.value = y.toFixed(1)
  }

  return {
    el,
    setTarget,

    /** update 增量刷新实时数据（robot 引用来自 store.robots） */
    update(state) {
      robot = state.selectedId ? state.robots[state.selectedId] : null
      if (!robot) return

      idEl.textContent = robot.id
      badge.update(robot.state)
      cells.coords.textContent = `${robot.x.toFixed(2)}, ${robot.y.toFixed(2)}`
      batteryBar.update(robot.battery)
      cells.speed.textContent = (robot.speed ?? 0).toFixed(1)
      cells.task.textContent = robot.task_id || '—'

      const offline = !robot.online && robot.state === 'OFFLINE'
      offlineTip.style.display = offline ? '' : 'none'
      offlineTip.textContent = '设备已离线，指令暂不可用'
      ;[moveBtn, stopBtn].forEach((b) => (b.disabled = offline))

      lastseen.textContent = robot.last_seen
        ? `上报: ${new Date(robot.last_seen).toLocaleTimeString()}`
        : ''

      // 历史轨迹按钮：未加载=加载 / 已加载=清除
      if (state.historyTrack && state.historyTrack.id === robot.id) {
        trackBtn.textContent = '清除轨迹'
      } else {
        trackBtn.textContent = trackLoading ? '加载中...' : '历史轨迹'
      }
      trackBtn.disabled = trackLoading
    },

    /** setTrackLoading 历史轨迹请求进行中的 loading 态 */
    setTrackLoading(v) {
      trackLoading = v
      trackBtn.textContent = v ? '加载中...' : '历史轨迹'
      trackBtn.disabled = v
    },

    /** setBusy 指令按钮 loading 态（kind: 'move' | 'stop'），防重复提交 */
    setBusy(kind, busy) {
      const btn = kind === 'move' ? moveBtn : stopBtn
      const orig = kind === 'move' ? '发送移动指令' : '紧急停止'
      btn.disabled = busy
      btn.textContent = busy ? '下发中...' : orig
    },
  }
}
