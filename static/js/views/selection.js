/**
 * selection.js - 地图手势（框选 / 拖动平移）
 *
 * 只负责监听与手势状态，渲染交给 map.js：
 * - 框选工具：拖动更新选区矩形（map.setSelectionRect），松开回调矩形内的机器人 id 集合
 * - 拖动工具：按住拖动只更新视口偏移（store.setViewport），rAF 重绘自然跟手
 */

import { TOOLS, setViewport } from '../store.js'

const MIN_DRAG_PX = 5 // 小于该像素距离视为误触，不触发框选

/**
 * createSelection 绑定地图手势
 * @param {HTMLCanvasElement} canvas 画布
 * @param {{toWorld: Function, getScale: Function, setSelectionRect: Function}} map 地图实例
 * @param {{onRectSelect: (ids: string[]) => void}} props
 * @returns {{update: (state: object) => void}}
 */
export function createSelection(canvas, map, props) {
  let state = null
  let dragging = null // {startPx:[x,y], lastPx:[x,y], type:'rect'|'pan'}

  function canvasPos(e) {
    const rect = canvas.getBoundingClientRect()
    return [e.clientX - rect.left, e.clientY - rect.top]
  }

  canvas.addEventListener('mousedown', (e) => {
    if (!state) return
    if (state.tool === TOOLS.RECT) {
      dragging = { startPx: canvasPos(e), lastPx: null, type: 'rect' }
    } else if (state.tool === TOOLS.PAN) {
      dragging = { startPx: canvasPos(e), lastPx: canvasPos(e), type: 'pan' }
      canvas.style.cursor = 'grabbing'
    }
    e.preventDefault() // 防止拖动时选中文本
  })

  canvas.addEventListener('mousemove', (e) => {
    if (!dragging) return
    const px = canvasPos(e)

    if (dragging.type === 'rect') {
      dragging.lastPx = px
      // 选区实时预览：像素矩形转世界坐标交给地图绘制
      const w1 = map.toWorld(dragging.startPx[0], dragging.startPx[1])
      const w2 = map.toWorld(px[0], px[1])
      map.setSelectionRect({ x1: w1[0], y1: w1[1], x2: w2[0], y2: w2[1] })
    } else if (dragging.type === 'pan') {
      // 拖动平移：像素位移换算成世界坐标视口偏移（y 轴翻转）
      const dpx = px[0] - dragging.lastPx[0]
      const dpy = px[1] - dragging.lastPx[1]
      const s = map.getScale()
      const { dx, dy } = state.viewport
      setViewport(dx + dpx / s, dy - dpy / s)
      dragging.lastPx = px
    }
  })

  function finish(e) {
    if (!dragging || !state) {
      dragging = null
      canvas.style.cursor = state && state.tool === TOOLS.PAN ? 'grab' : 'crosshair'
      return
    }
    const px = e ? canvasPos(e) : dragging.startPx

    if (dragging.type === 'rect') {
      map.setSelectionRect(null)
      // 误触保护：拖动距离过小不触发框选
      if (Math.hypot(px[0] - dragging.startPx[0], px[1] - dragging.startPx[1]) >= MIN_DRAG_PX) {
        const w1 = map.toWorld(dragging.startPx[0], dragging.startPx[1])
        const w2 = map.toWorld(px[0], px[1])
        const x1 = Math.min(w1[0], w2[0])
        const x2 = Math.max(w1[0], w2[0])
        const y1 = Math.min(w1[1], w2[1])
        const y2 = Math.max(w1[1], w2[1])
        // 矩形内所有机器人进入多选
        const ids = Object.values(state.robots)
          .filter((r) => r.x >= x1 && r.x <= x2 && r.y >= y1 && r.y <= y2)
          .map((r) => r.id)
        props.onRectSelect(ids)
      }
    }

    dragging = null
    canvas.style.cursor = state.tool === TOOLS.PAN ? 'grab' : 'crosshair'
  }

  canvas.addEventListener('mouseup', finish)
  canvas.addEventListener('mouseleave', (e) => finish(e))

  return {
    /** update 按当前工具刷新光标样式 */
    update(s) {
      state = s
      if (dragging) return // 手势进行中不打断
      canvas.style.cursor =
        s.tool === TOOLS.PAN ? 'grab' : s.tool === TOOLS.RECT ? 'crosshair' : 'crosshair'
    },
  }
}
