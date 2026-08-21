/**
 * batchActionBar.js - 批量操作栏（多选态右侧栏）
 *
 * 已选数量 + ID 列表 + 批量移动（坐标表单 + 确认）+ 批量停止（二次确认）+ 退出多选。
 */

import { createConfirmDialog } from '../components/confirmDialog.js'

/**
 * createBatchActionBar 创建批量操作栏
 * @param {HTMLElement} mount 挂载容器
 * @param {{onBatchMove: (x: number, y: number) => void,
 *          onBatchStop: () => void, onExit: () => void}} props
 * @returns {{el: HTMLElement, update: (state: object) => void}}
 */
export function createBatchActionBar(mount, props) {
  const el = document.createElement('div')
  el.className = 'batch-bar'

  /* -- 已选统计 -- */
  const count = document.createElement('div')
  count.className = 'batch-bar__count'

  const ids = document.createElement('div')
  ids.className = 'batch-bar__ids'

  /* -- 批量移动表单 -- */
  const formTitle = document.createElement('div')
  formTitle.className = 'section-title'
  formTitle.textContent = '批量移动（同一目标点）'

  const formRow = document.createElement('div')
  formRow.className = 'form-row'
  const mkInput = (label) => {
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
  const inputX = mkInput('X')
  const inputY = mkInput('Y')
  formRow.append(inputX.wrap, inputY.wrap)

  /* -- 操作按钮 -- */
  const actions = document.createElement('div')
  actions.className = 'batch-bar__actions'

  const moveBtn = document.createElement('button')
  moveBtn.className = 'btn btn--primary'
  moveBtn.textContent = '批量移动'

  const stopBtn = document.createElement('button')
  stopBtn.className = 'btn btn--danger'
  stopBtn.textContent = '批量停止'

  const exitBtn = document.createElement('button')
  exitBtn.className = 'btn btn--ghost'
  exitBtn.textContent = '退出多选'

  actions.append(moveBtn, stopBtn, exitBtn)
  el.append(count, ids, formTitle, formRow, actions)
  mount.appendChild(el)

  let selected = [] // 当前选中设备 id 快照

  // 批量移动：坐标校验 + 二次确认
  moveBtn.addEventListener('click', () => {
    if (selected.length === 0) return
    const x = parseFloat(inputX.input.value)
    const y = parseFloat(inputY.input.value)
    if (Number.isNaN(x) || Number.isNaN(y)) return
    createConfirmDialog({
      title: '确认批量移动',
      message: `将向 ${selected.length} 台设备下发移动指令，目标 (${x}, ${y})。确认执行？`,
      confirmText: '下发',
      onConfirm: () => props.onBatchMove(x, y),
    })
  })

  // 批量停止：危险操作二次确认
  stopBtn.addEventListener('click', () => {
    if (selected.length === 0) return
    createConfirmDialog({
      title: '确认批量停止',
      message: `将向 ${selected.length} 台设备下发停止指令，不可撤销。确认执行？`,
      confirmText: '批量停止',
      danger: true,
      onConfirm: () => props.onBatchStop(),
    })
  })

  exitBtn.addEventListener('click', () => props.onExit())

  return {
    el,
    /** update 刷新已选集合（离线设备由 store 自动移出，计数即时减少） */
    update(state) {
      selected = [...state.selectedIds]
      count.textContent = `已选 ${selected.length} 台`
      ids.textContent = selected.join(' · ')
      const empty = selected.length === 0
      moveBtn.disabled = empty
      stopBtn.disabled = empty
    },
  }
}
