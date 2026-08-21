/**
 * toast.js - 通知组件（单例）
 * 自动 3s 消失，同一时刻同类同文案消息去重（避免批量操作刷屏）。
 */

const DURATION_MS = 3000

let container = null
const shown = new Set() // 当前展示中的去重 key

function ensureContainer() {
  if (!container) {
    container = document.getElementById('toastContainer')
  }
  return container
}

/**
 * toast 通知单例
 * @param {string} msg 消息文案
 * @param {'success'|'error'|'info'} type 类型（决定左侧色条）
 */
export const toast = {
  show(msg, type = 'info') {
    const c = ensureContainer()
    if (!c) return

    // 同类同文案去重
    const key = `${type}:${msg}`
    if (shown.has(key)) return
    shown.add(key)

    const el = document.createElement('div')
    el.className = `toast toast--${type}`
    el.textContent = msg
    c.appendChild(el)

    // 到期淡出后移除
    setTimeout(() => {
      el.classList.add('fade-out')
      setTimeout(() => {
        el.remove()
        shown.delete(key)
      }, 300)
    }, DURATION_MS)
  },
}
