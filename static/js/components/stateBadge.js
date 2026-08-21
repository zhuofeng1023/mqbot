/**
 * stateBadge.js - 状态徽章组件
 * 状态色唯一权威来源是 tokens.css 的 --state-* 变量，颜色同时配文字（可达性要求）。
 */

/** 状态中文名（Tooltip/详情用；徽章本身显示英文枚举保持 mono 风格） */
export const STATE_NAMES = {
  IDLE: '空闲',
  MOVING: '执行中',
  CHARGING: '充电中',
  ERROR: '异常',
  OFFLINE: '离线',
}

/**
 * createStateBadge 创建状态徽章
 * @returns {{el: HTMLElement, update: (state: string) => void}}
 */
export function createStateBadge() {
  const el = document.createElement('span')
  el.className = 'badge'

  return {
    el,
    /** update 更新状态（未知状态按 OFFLINE 兜底） */
    update(state) {
      const s = STATE_NAMES[state] ? state : 'OFFLINE'
      el.className = `badge badge--${s}`
      el.textContent = s
    },
  }
}
