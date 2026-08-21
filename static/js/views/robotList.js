/**
 * robotList.js - 机器人列表视图
 *
 * 头部 = 搜索框 + 卡片/表格视图切换 + 多选按钮；
 * 表格视图支持按 ID/电量/状态排序与"仅看低电量(<20%)"筛选；
 * 卡片视图沿用 idsKey diff 增量更新（不重建 DOM，避免打断点击）。
 */

import { createSearchInput } from '../components/searchInput.js'
import { createCheckbox } from '../components/checkbox.js'
import { createRobotCard } from './robotCard.js'
import { setKeyword, setViewMode, enterMultiSelect, setCheckedIds } from '../store.js'

// 表格排序：电量/状态有语义顺序，ID 按字典序
const STATE_ORDER = { ERROR: 0, CHARGING: 1, MOVING: 2, IDLE: 3, OFFLINE: 4 }

/**
 * createRobotList 创建机器人列表
 * @param {HTMLElement} mount 挂载容器
 * @param {{onSelect: (id: string) => void, onCheck: (id: string) => void,
 *          onExitMulti: () => void}} props
 * @returns {{el: HTMLElement, update: (state: object) => void}}
 */
export function createRobotList(mount, props) {
  const el = document.createElement('div')
  el.className = 'robot-list'

  let sort = { key: 'id', dir: 1 } // 表格排序状态（本地）
  let lowOnly = false // 仅看低电量筛选（本地）

  /* -- 头部：搜索 + 工具按钮 -- */
  const header = document.createElement('div')
  header.className = 'robot-list__header'

  const search = createSearchInput({
    placeholder: '搜索 ID...',
    onInput: (kw) => setKeyword(kw), // 写回 store，全选范围随之变化
  })
  header.appendChild(search.el)

  const tools = document.createElement('div')
  tools.className = 'list-tools'

  const cardBtn = mkToolBtn('▦', '卡片视图')
  const tableBtn = mkToolBtn('☰', '表格视图')
  const multiBtn = mkToolBtn('☑', '多选模式')
  tools.append(cardBtn, tableBtn, multiBtn)
  header.appendChild(tools)

  function mkToolBtn(icon, title) {
    const b = document.createElement('button')
    b.className = 'list-tools__btn'
    b.textContent = icon
    b.title = title
    return b
  }

  cardBtn.addEventListener('click', () => setViewMode('card'))
  tableBtn.addEventListener('click', () => setViewMode('table'))
  multiBtn.addEventListener('click', () => enterMultiSelect())

  /* -- 全选行（多选模式显示）-- */
  const checkallRow = document.createElement('div')
  checkallRow.className = 'list-checkall'
  checkallRow.style.display = 'none'

  const checkallLabel = document.createElement('span')
  checkallLabel.textContent = '全选'
  const checkallState = document.createElement('span')

  /* -- 低电量筛选（表格视图）-- */
  const lowFilter = document.createElement('label')
  lowFilter.className = 'list-filter'
  const lowText = document.createElement('span')
  lowText.textContent = '低电量<20%'
  lowFilter.appendChild(lowText)

  /* -- 列表主体 -- */
  const body = document.createElement('div')
  body.className = 'robot-list__body'

  el.append(header, checkallRow, body)
  mount.appendChild(el)

  // 卡片模式增量更新的节点缓存
  const cardNodes = {} // id -> robotCard 实例
  let lastIdsKey = ''

  /** filterRobots 按 keyword（ID 子串不区分大小写）过滤 */
  function filterRobots(state) {
    const kw = state.keyword.toLowerCase()
    let list = Object.values(state.robots).filter((r) => r.id.toLowerCase().includes(kw))
    if (lowOnly) list = list.filter((r) => r.battery < 20)
    return list
  }

  /** sortRobots 按当前排序键排序 */
  function sortRobots(list) {
    const { key, dir } = sort
    return list.slice().sort((a, b) => {
      if (key === 'battery') return (a.battery - b.battery) * dir
      if (key === 'state') return (STATE_ORDER[a.state] - STATE_ORDER[b.state]) * dir
      return a.id.localeCompare(b.id) * dir
    })
  }

  /** update 渲染列表（由 main 的渲染节流统一驱动） */
  function update(state) {
    const visible = sortRobots(filterRobots(state))

    cardBtn.classList.toggle('active', state.viewMode === 'card')
    tableBtn.classList.toggle('active', state.viewMode === 'table')
    multiBtn.classList.toggle('active', state.multiSelect)

    // 全选行：多选模式显示，三态（未选/半选/全选）
    checkallRow.style.display = state.multiSelect ? '' : 'none'
    if (state.multiSelect) {
      renderCheckall(state, visible)
    }

    if (state.viewMode === 'card') renderCards(state, visible)
    else renderTable(state, visible)
  }

  /** renderCheckall 全选框三态逻辑，作用范围 = 过滤后的可见集合 */
  function renderCheckall(state, visible) {
    checkallRow.innerHTML = ''
    const visibleIds = visible.map((r) => r.id)
    const checkedCount = visibleIds.filter((id) => state.selectedIds.has(id)).length
    const all = visibleIds.length > 0 && checkedCount === visibleIds.length
    const some = checkedCount > 0 && !all

    const cb = createCheckbox({
      checked: all,
      indeterminate: some,
      onChange: (v) => {
        // 全选只作用于当前搜到的集合：勾选合并已有选择，取消则移除可见集合
        if (v) setCheckedIds([...new Set([...state.selectedIds, ...visibleIds])])
        else {
          const rest = [...state.selectedIds].filter((id) => !visibleIds.includes(id))
          setCheckedIds(rest)
        }
      },
    })
    checkallRow.append(cb.el, checkallLabel, checkallState)
    checkallState.textContent = `${checkedCount}/${visibleIds.length}`
  }

  /** renderCards 卡片视图：idsKey diff 增量更新 */
  function renderCards(state, visible) {
    const ids = visible.map((r) => r.id)
    const idsKey = ids.join(',')

    // id 集合变化（增减/过滤）时才重建 DOM 结构
    if (idsKey !== lastIdsKey) {
      for (const k in cardNodes) delete cardNodes[k]
      if (ids.length === 0) {
        body.innerHTML = ''
        body.appendChild(renderEmpty(state))
      } else {
        body.innerHTML = ''
        ids.forEach((id) => {
          const card = createRobotCard({
            id,
            onClick: props.onSelect,
            onCheck: props.onCheck,
          })
          body.appendChild(card.el)
          cardNodes[id] = card
        })
      }
      lastIdsKey = idsKey
    }

    // 动态字段增量更新
    visible.forEach((r) => {
      const card = cardNodes[r.id]
      if (!card) return
      card.update(r, {
        selected: state.selectedId === r.id,
        checked: state.selectedIds.has(r.id),
        multiSelect: state.multiSelect,
      })
    })
  }

  /** renderTable 表格视图：直接重建（设备少，成本可忽略） */
  function renderTable(state, visible) {
    body.innerHTML = ''

    // 工具行：低电量筛选（点击切换，激活高亮）
    lowFilter.style.cursor = 'pointer'
    lowText.style.color = lowOnly ? 'var(--yellow)' : 'var(--text-muted)'
    lowFilter.onclick = () => {
      lowOnly = !lowOnly
      update(state)
    }
    body.appendChild(lowFilter)

    if (visible.length === 0) {
      body.appendChild(renderEmpty(state))
      return
    }

    const table = document.createElement('table')
    table.className = 'robot-table'

    // 表头：点击切换排序
    const thead = document.createElement('thead')
    const tr = document.createElement('tr')
    const cols = [
      { key: 'id', label: 'ID' },
      { key: 'state', label: '状态' },
      { key: 'battery', label: '电量' },
      { key: 'pos', label: '坐标' },
    ]
    cols.forEach((c) => {
      const th = document.createElement('th')
      // 当前排序列显示方向箭头
      th.textContent = sort.key === c.key ? c.label + (sort.dir > 0 ? ' ↑' : ' ↓') : c.label
      if (c.key !== 'pos') {
        th.addEventListener('click', () => {
          sort = sort.key === c.key ? { key: c.key, dir: -sort.dir } : { key: c.key, dir: 1 }
          update(state)
        })
      }
      tr.appendChild(th)
    })
    thead.appendChild(tr)
    table.appendChild(thead)

    const tbody = document.createElement('tbody')
    visible.forEach((r) => {
      const row = document.createElement('tr')
      if (state.selectedId === r.id) row.classList.add('selected')

      const mkTd = (text) => {
        const td = document.createElement('td')
        td.textContent = text
        return td
      }
      const stTd = mkTd(r.state || 'OFFLINE')
      stTd.style.color = `var(--state-${(r.state || 'OFFLINE').toLowerCase()})`
      const btTd = mkTd(Math.round(r.battery) + '%')
      if (r.battery < 20) btTd.style.color = 'var(--red)'

      row.append(mkTd(r.id), stTd, btTd, mkTd(`${r.x.toFixed(1)}, ${r.y.toFixed(1)}`))
      row.addEventListener('click', () => {
        if (state.multiSelect) props.onCheck(r.id)
        else props.onSelect(r.id)
      })
      tbody.appendChild(row)
    })
    table.appendChild(tbody)
    body.appendChild(table)
  }

  /** renderEmpty 空态：无设备 / 搜索无结果两种文案 */
  function renderEmpty(state) {
    const div = document.createElement('div')
    div.className = 'robot-list__empty'
    if (Object.keys(state.robots).length === 0) {
      div.textContent = '等待机器人连接...'
    } else {
      div.textContent = `无匹配 '${state.keyword}'，`
      const link = document.createElement('a')
      link.textContent = '清空搜索'
      link.addEventListener('click', () => {
        setKeyword('')
        // 直接清空搜索框内容（组件内部 input）
        const input = search.el.querySelector('input')
        if (input) input.value = ''
      })
      div.appendChild(link)
    }
    return div
  }

  return { el, update }
}
