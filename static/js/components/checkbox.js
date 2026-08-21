/**
 * checkbox.js - 复选框组件（含半选态）
 * 纯 div 实现（不用原生 input），半选态用 CSS class indeterminate 表达。
 */

/**
 * createCheckbox 创建复选框
 * @param {{checked?: boolean, indeterminate?: boolean, onChange?: (checked: boolean) => void}} props
 * @returns {{el: HTMLElement, update: (props: {checked?: boolean, indeterminate?: boolean}) => void}}
 */
export function createCheckbox(props = {}) {
  const el = document.createElement('div')
  el.className = 'checkbox'
  el.setAttribute('role', 'checkbox')

  function render(checked, indeterminate) {
    el.classList.toggle('checked', !!checked && !indeterminate)
    el.classList.toggle('indeterminate', !!indeterminate)
    // 半选显示横线，全选显示对勾
    el.textContent = indeterminate ? '–' : checked ? '✓' : ''
    el.setAttribute('aria-checked', indeterminate ? 'mixed' : checked ? 'true' : 'false')
  }

  let current = { checked: !!props.checked, indeterminate: !!props.indeterminate }
  render(current.checked, current.indeterminate)

  el.addEventListener('click', (e) => {
    e.stopPropagation() // 阻止冒泡到卡片/行的点击选中
    // 点击永远从当前态切换到另一个确定态（半选视为未全选）
    const next = !(current.checked && !current.indeterminate)
    current = { checked: next, indeterminate: false }
    render(next, false)
    props.onChange && props.onChange(next)
  })

  return {
    el,
    /** update 外部状态变化时同步视图（不触发 onChange） */
    update(p = {}) {
      current = { checked: !!p.checked, indeterminate: !!p.indeterminate }
      render(current.checked, current.indeterminate)
    },
  }
}
