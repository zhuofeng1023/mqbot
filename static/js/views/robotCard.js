/**
 * robotCard.js - 机器人卡片（卡片视图单元）
 * 组合 StateBadge + BatteryBar + Checkbox；节点一次创建、增量更新（不重建 DOM）。
 */

import { createStateBadge } from '../components/stateBadge.js'
import { createBatteryBar } from '../components/batteryBar.js'
import { createCheckbox } from '../components/checkbox.js'
import { STALE_MS } from '../store.js'

/**
 * createRobotCard 创建机器人卡片
 * @param {{id: string, onClick: (id: string) => void, onCheck: (id: string) => void}} props
 * @returns {{el: HTMLElement, update: (robot: object, ui: {selected: boolean,
 *            checked: boolean, multiSelect: boolean}) => void}}
 */
export function createRobotCard(props) {
  const el = document.createElement('div')
  el.className = 'robot-card'

  const badge = createStateBadge()
  const battery = createBatteryBar()

  // 多选模式下显示的复选框（默认隐藏，由 CSS .multi 控制）
  const checkbox = createCheckbox({
    onChange: () => props.onCheck(props.id),
  })
  checkbox.el.className += ' robot-card__check'

  // 数据过期标记（右上角）
  const stale = document.createElement('span')
  stale.className = 'robot-card__stale'
  stale.textContent = '过期'
  stale.style.display = 'none'

  const header = document.createElement('div')
  header.className = 'robot-card__header'
  const idEl = document.createElement('span')
  idEl.className = 'robot-card__id'
  header.appendChild(idEl)
  header.appendChild(badge.el)

  const coords = document.createElement('div')
  coords.className = 'robot-card__coords'

  const batteryRow = document.createElement('div')
  batteryRow.className = 'robot-card__battery'
  batteryRow.appendChild(battery.el)

  el.appendChild(checkbox.el)
  el.appendChild(stale)
  el.appendChild(header)
  el.appendChild(coords)
  el.appendChild(batteryRow)

  let boundId = null

  return {
    el,
    /** update 增量刷新卡片内容与选中/勾选态 */
    update(robot, ui) {
      boundId = robot.id
      idEl.textContent = robot.id
      badge.update(robot.state)
      coords.textContent = `x: ${robot.x.toFixed(2)}  y: ${robot.y.toFixed(2)}`
      battery.update(robot.battery)
      checkbox.update({ checked: ui.checked })
      el.classList.toggle('multi', ui.multiSelect)
      el.classList.toggle('selected', ui.selected)
      el.classList.toggle('checked', ui.multiSelect && ui.checked)
      // 卡片点击：多选=勾选切换，单选=进入详情
      el.onclick = () => {
        if (ui.multiSelect) props.onCheck(boundId)
        else props.onClick(boundId)
      }
      // 超过 STALE_MS 未更新显示"过期"灰标（区分"在线但没动"与"数据断流"）
      stale.style.display = Date.now() - robot.ts > STALE_MS ? '' : 'none'
    },
  }
}
