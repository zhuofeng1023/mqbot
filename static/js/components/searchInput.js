/**
 * searchInput.js - 搜索框组件（200ms 输入防抖）
 */

const DEBOUNCE_MS = 200

/**
 * createSearchInput 创建搜索框
 * @param {{placeholder?: string, onInput?: (keyword: string) => void}} props
 * @returns {{el: HTMLElement}}
 */
export function createSearchInput(props = {}) {
  const el = document.createElement('div')
  el.className = 'search'

  const icon = document.createElement('span')
  icon.className = 'search__icon'
  icon.textContent = '🔍'

  const input = document.createElement('input')
  input.className = 'search__input'
  input.type = 'text'
  input.placeholder = props.placeholder || '搜索 ID...'

  el.appendChild(icon)
  el.appendChild(input)

  // 防抖：输入停止 200ms 后才回调，不逐键触发过滤
  let timer = null
  input.addEventListener('input', () => {
    clearTimeout(timer)
    timer = setTimeout(() => props.onInput && props.onInput(input.value.trim()), DEBOUNCE_MS)
  })

  return { el }
}
