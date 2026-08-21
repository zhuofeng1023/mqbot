/**
 * mapToolbar.js - 地图工具栏
 * 三个互斥工具（取点/框选/拖动）+ 轨迹开关 + 重置视图，激活态高亮。
 */

import { TOOLS } from '../store.js'
import { attachTooltip } from '../components/tooltip.js'

// 工具定义：id / 图标 / 提示
const TOOL_DEFS = [
  { id: TOOLS.PICK, icon: '✛', tip: '取点：点击地图取目标坐标' },
  { id: TOOLS.RECT, icon: '⛶', tip: '框选：拖动矩形批量选机器人' },
  { id: TOOLS.PAN, icon: '✥', tip: '拖动：按住拖动平移画布' },
]

const TOOL_HINTS = {
  [TOOLS.PICK]: '取点模式：点击地图取目标坐标，ESC 切回',
  [TOOLS.RECT]: '框选模式：拖动选择机器人，ESC 退出',
  [TOOLS.PAN]: '拖动模式：按住拖动平移画布，ESC 退出',
}

/**
 * createMapToolbar 创建工具栏
 * @param {{onToolChange: (tool: string) => void, onToggleTrail: () => void,
 *          onResetView: () => void}} props
 * @returns {{el: HTMLElement, hintEl: HTMLElement, update: (state: object) => void}}
 */
export function createMapToolbar(props) {
  const el = document.createElement('div')
  el.className = 'map-toolbar'

  const toolBtns = {}

  TOOL_DEFS.forEach((def) => {
    const btn = document.createElement('button')
    btn.className = 'map-toolbar__btn'
    btn.textContent = def.icon
    btn.addEventListener('click', () => props.onToolChange(def.id))
    attachTooltip(btn, () => {
      const t = document.createElement('span')
      t.textContent = def.tip
      return t
    })
    el.appendChild(btn)
    toolBtns[def.id] = btn
  })

  // 分隔线
  const sep = document.createElement('div')
  sep.className = 'map-toolbar__sep'
  el.appendChild(sep)

  // 轨迹开关
  const trailBtn = document.createElement('button')
  trailBtn.className = 'map-toolbar__btn'
  trailBtn.textContent = '〜'
  trailBtn.addEventListener('click', () => props.onToggleTrail())
  attachTooltip(trailBtn, () => {
    const t = document.createElement('span')
    t.textContent = '实时尾迹开关'
    return t
  })
  el.appendChild(trailBtn)

  // 重置视图
  const resetBtn = document.createElement('button')
  resetBtn.className = 'map-toolbar__btn'
  resetBtn.textContent = '⟳'
  resetBtn.addEventListener('click', () => props.onResetView())
  attachTooltip(resetBtn, () => {
    const t = document.createElement('span')
    t.textContent = '重置视图（回到初始居中）'
    return t
  })
  el.appendChild(resetBtn)

  // 当前模式提示（地图顶部居中）
  const hintEl = document.createElement('div')
  hintEl.className = 'map-hint'

  return {
    el,
    hintEl,
    /** update 按全局状态刷新激活态与提示文案 */
    update(state) {
      Object.entries(toolBtns).forEach(([id, btn]) => {
        btn.classList.toggle('active', state.tool === id)
      })
      trailBtn.classList.toggle('active', state.showTrail)
      hintEl.textContent = state.tool === TOOLS.PICK ? '' : TOOL_HINTS[state.tool]
    },
  }
}
