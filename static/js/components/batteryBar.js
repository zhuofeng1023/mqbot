/**
 * batteryBar.js - 电量条组件
 * 电量 < 20% 时自动加 ⚠ 图标（颜色之外的第二载体）。
 */

/**
 * createBatteryBar 创建电量条
 * @param {{showText?: boolean}} props showText=false 时隐藏百分比文字（紧凑场景）
 * @returns {{el: HTMLElement, update: (value: number) => void}}
 */
export function createBatteryBar(props = {}) {
  const el = document.createElement('div')
  el.className = 'battery'

  const bar = document.createElement('div')
  bar.className = 'battery__bar'
  const fill = document.createElement('div')
  fill.className = 'battery__fill'
  bar.appendChild(fill)

  const warn = document.createElement('span')
  warn.className = 'battery__warn'
  warn.textContent = '⚠'
  warn.style.display = 'none'

  el.appendChild(bar)

  if (props.showText !== false) {
    const text = document.createElement('span')
    text.className = 'battery__text'
    el.appendChild(text)

    return {
      el,
      /** update 更新电量值（0~100） */
      update(value) {
        const v = Math.max(0, Math.min(100, value))
        // 三档颜色：>50 绿 / 20~50 黄 / <20 红
        const color = v > 50 ? 'var(--green)' : v > 20 ? 'var(--yellow)' : 'var(--red)'
        fill.style.width = v + '%'
        fill.style.background = color
        text.textContent = Math.round(v) + '%'
        warn.style.display = v < 20 ? '' : 'none'
      },
    }
  }

  // 无文字模式（Tooltip 内使用）
  return {
    el,
    update(value) {
      const v = Math.max(0, Math.min(100, value))
      const color = v > 50 ? 'var(--green)' : v > 20 ? 'var(--yellow)' : 'var(--red)'
      fill.style.width = v + '%'
      fill.style.background = color
      warn.style.display = v < 20 ? '' : 'none'
    },
  }
}
