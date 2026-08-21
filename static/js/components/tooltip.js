/**
 * tooltip.js - 悬浮提示组件
 * attachTooltip 给任意元素挂载 Tooltip，悬停显示/移出隐藏；
 * 渲染内容由 renderFn 提供（返回 DOM 节点），数据取本地 store，无网络请求。
 */

const OFFSET = 12 // 跟随光标的偏移量

/**
 * attachTooltip 挂载 Tooltip
 * @param {HTMLElement} target 挂载元素
 * @param {() => HTMLElement|null} renderFn 渲染函数，返回 null 时不显示
 */
export function attachTooltip(target, renderFn) {
  let el = null
  let rafId = null

  function show(e) {
    const content = renderFn()
    if (!content) return

    el = document.createElement('div')
    el.className = 'tooltip'
    el.appendChild(content)
    document.body.appendChild(el)
    move(e)
  }

  /** move 跟随光标定位，靠近右/下边缘时翻转到另一侧 */
  function move(e) {
    if (!el) return
    // 用 rAF 合并高频 mousemove，避免重复触发回流
    if (rafId) cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(() => {
      rafId = null
      if (!el) return
      const x = Math.min(e.clientX + OFFSET, window.innerWidth - el.offsetWidth - 8)
      const y = Math.min(e.clientY + OFFSET, window.innerHeight - el.offsetHeight - 8)
      el.style.left = x + 'px'
      el.style.top = y + 'px'
    })
  }

  function hide() {
    if (rafId) cancelAnimationFrame(rafId)
    el && el.remove()
    el = null
  }

  target.addEventListener('mouseenter', show)
  target.addEventListener('mousemove', move)
  target.addEventListener('mouseleave', hide)
}
