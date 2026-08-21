/**
 * statCard.js - 指标卡组件（指标条专用）
 */

/**
 * createStatCard 创建指标卡
 * @param {{label: string, tone?: 'normal'|'warn'|'danger'}} props
 * @returns {{el: HTMLElement, update: (value: string|number) => void}}
 */
export function createStatCard(props) {
  const el = document.createElement('div')
  el.className = 'stat-card' + (props.tone ? ` stat-card--${props.tone}` : '')

  const label = document.createElement('div')
  label.className = 'stat-card__label'
  label.textContent = props.label

  const value = document.createElement('div')
  value.className = 'stat-card__value'

  el.appendChild(label)
  el.appendChild(value)

  return {
    el,
    /** update 更新展示值（数字或字符串） */
    update(v) {
      value.textContent = v
    },
  }
}
