/**
 * confirmDialog.js - 确认弹窗组件（危险操作二次确认）
 */

/**
 * createConfirmDialog 创建确认弹窗
 * @param {{title: string, message: string, confirmText?: string, danger?: boolean,
 *          onConfirm: () => void, onCancel?: () => void}} props
 * @returns {{el: HTMLElement, destroy: () => void}}
 */
export function createConfirmDialog(props) {
  const mask = document.createElement('div')
  mask.className = 'dialog-mask'

  const dialog = document.createElement('div')
  dialog.className = 'dialog'

  const title = document.createElement('div')
  title.className = 'dialog__title'
  title.textContent = props.title

  const message = document.createElement('div')
  message.className = 'dialog__message'
  message.textContent = props.message

  const actions = document.createElement('div')
  actions.className = 'dialog__actions'

  const btnCancel = document.createElement('button')
  btnCancel.className = 'btn btn--ghost'
  btnCancel.textContent = '取消'

  const btnConfirm = document.createElement('button')
  btnConfirm.className = 'btn ' + (props.danger ? 'btn--danger' : 'btn--primary')
  btnConfirm.textContent = props.confirmText || '确认'

  actions.appendChild(btnCancel)
  actions.appendChild(btnConfirm)
  dialog.appendChild(title)
  dialog.appendChild(message)
  dialog.appendChild(actions)
  mask.appendChild(dialog)

  function destroy() {
    document.removeEventListener('keydown', onKeydown)
    mask.remove()
  }

  function onKeydown(e) {
    if (e.key === 'Escape') {
      destroy()
      props.onCancel && props.onCancel()
    }
  }

  btnCancel.addEventListener('click', () => {
    destroy()
    props.onCancel && props.onCancel()
  })
  btnConfirm.addEventListener('click', () => {
    destroy()
    props.onConfirm()
  })
  // 点击遮罩空白处视为取消
  mask.addEventListener('click', (e) => {
    if (e.target === mask) {
      destroy()
      props.onCancel && props.onCancel()
    }
  })
  document.addEventListener('keydown', onKeydown)

  document.body.appendChild(mask)

  return { el: mask, destroy }
}
