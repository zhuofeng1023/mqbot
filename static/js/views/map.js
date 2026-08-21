/**
 * map.js - 地图画布（MapCanvas）
 *
 * 网格坐标系（世界 -20~20）+ 机器人点位 + 实时尾迹 + 历史轨迹 + 选区 + 目标标记。
 * 网格预渲染到离屏 canvas，拖动平移只更新视口偏移、不重建网格；
 * 机器人层每帧（rAF）重绘，动画（脉冲/闪烁）由时间驱动。
 */

import { TOOLS } from '../store.js'
import { attachTooltip } from '../components/tooltip.js'
import { STATE_NAMES } from '../components/stateBadge.js'

const GRID = 40 // 世界坐标范围 -20 ~ 20
const STEP = 5 // 网格线间隔
const HIT_RADIUS = 12 // 机器人点位点击命中半径（像素）

// 状态点色：与 tokens.css --state-* 一致（Canvas 无法消费 CSS 变量）
const STATE_COLORS = {
  IDLE: '#3fb950',
  MOVING: '#58a6ff',
  CHARGING: '#d29922',
  ERROR: '#f85149',
  OFFLINE: '#7d8590',
}

/**
 * createMap 创建地图画布
 * @param {HTMLCanvasElement} canvas 画布元素
 * @param {{onCanvasClick: (x: number, y: number) => void,
 *          onRobotClick: (id: string) => void}} props
 * @returns {{el: HTMLCanvasElement, update: (state: object) => void,
 *            setSelectionRect: (rect: {x1,y1,x2,y2}|null) => void,
 *            toWorld: (px: number, py: number) => [number, number],
 *            getScale: () => number}}
 */
export function createMap(canvas, props) {
  const ctx = canvas.getContext('2d')

  let state = null // 全局状态引用（rAF 循环每帧读取，WS 更新自动可见）
  let selectionRect = null // 当前选区（世界坐标矩形），由 selection.js 驱动
  let lastMouse = null // 最近一次鼠标世界坐标（坐标提示用）

  /* ========== 尺寸与离屏网格 ========== */

  let scale = 1
  const off = document.createElement('canvas') // 离屏网格层
  const offCtx = off.getContext('2d')

  function resize() {
    const rect = canvas.parentElement.getBoundingClientRect()
    canvas.width = rect.width * devicePixelRatio
    canvas.height = rect.height * devicePixelRatio
    canvas.style.width = rect.width + 'px'
    canvas.style.height = rect.height + 'px'
    ctx.setTransform(devicePixelRatio, 0, 0, devicePixelRatio, 0, 0)
    scale = Math.min(rect.width, rect.height) / GRID
    renderGridLayer()
  }

  /** renderGridLayer 把网格+坐标轴+标签预渲染进离屏 canvas（resize 时重建） */
  function renderGridLayer() {
    const size = GRID * scale
    off.width = size
    off.height = size
    offCtx.clearRect(0, 0, size, size)

    // 网格线
    offCtx.strokeStyle = '#161b22'
    offCtx.lineWidth = 1
    for (let i = 0; i <= GRID; i += STEP) {
      offCtx.beginPath()
      offCtx.moveTo(i * scale, 0)
      offCtx.lineTo(i * scale, size)
      offCtx.stroke()
      offCtx.beginPath()
      offCtx.moveTo(0, i * scale)
      offCtx.lineTo(size, i * scale)
      offCtx.stroke()
    }

    // 坐标轴（穿过世界原点）
    const o = worldToGrid(0, 0)
    offCtx.strokeStyle = '#30363d'
    offCtx.lineWidth = 1.5
    offCtx.beginPath()
    offCtx.moveTo(0, o[1])
    offCtx.lineTo(size, o[1])
    offCtx.stroke()
    offCtx.beginPath()
    offCtx.moveTo(o[0], 0)
    offCtx.lineTo(o[0], size)
    offCtx.stroke()

    // 轴标签
    offCtx.font = '11px "JetBrains Mono"'
    offCtx.fillStyle = '#484f58'
    offCtx.textAlign = 'center'
    for (let i = -GRID / 2; i <= GRID / 2; i += STEP) {
      if (i === 0) continue
      const [lx, ly] = worldToGrid(i, 0)
      offCtx.fillText(i, lx, ly + 14)
      const [ax, ay] = worldToGrid(0, i)
      offCtx.fillText(i, ax - 14, ay + 4)
    }
    // 原点标记
    offCtx.textAlign = 'right'
    offCtx.fillText('O', o[0] - 6, o[1] + 14)
  }

  /** worldToGrid 世界坐标 -> 离屏网格图内像素（网格图以世界 (-20,20) 为左上角） */
  function worldToGrid(wx, wy) {
    return [(wx + GRID / 2) * scale, (GRID / 2 - wy) * scale]
  }

  /* ========== 坐标变换（含视口偏移）========== */

  /** toCanvas 世界坐标 -> 主画布 CSS 像素 */
  function toCanvas(wx, wy) {
    const w = canvas.clientWidth
    const h = canvas.clientHeight
    const { dx, dy } = state ? state.viewport : { dx: 0, dy: 0 }
    return [w / 2 + (wx + dx) * scale, h / 2 - (wy + dy) * scale]
  }

  /** toCanvas 内部使用：避免反复取 clientWidth */
  function toCanvasWith(w, h, wx, wy) {
    const { dx, dy } = state.viewport
    return [w / 2 + (wx + dx) * scale, h / 2 - (wy + dy) * scale]
  }

  /** toCanvas 像素 -> 世界坐标 */
  function toWorld(px, py) {
    const w = canvas.clientWidth
    const h = canvas.clientHeight
    const { dx, dy } = state ? state.viewport : { dx: 0, dy: 0 }
    return [(px - w / 2) / scale - dx, (h / 2 - py) / scale - dy]
  }

  /* ========== 交互 ========== */

  // 鼠标坐标提示（左下角）
  canvas.addEventListener('mousemove', (e) => {
    const rect = canvas.getBoundingClientRect()
    const [wx, wy] = toWorld(e.clientX - rect.left, e.clientY - rect.top)
    lastMouse = [wx, wy]
    const coordEl = document.getElementById('mapCoord')
    if (coordEl) coordEl.textContent = `x: ${wx.toFixed(1)}  y: ${wy.toFixed(1)}`
  })
  canvas.addEventListener('mouseleave', () => {
    lastMouse = null
    const coordEl = document.getElementById('mapCoord')
    if (coordEl) coordEl.textContent = '移动鼠标查看坐标'
  })

  // 点击：命中机器人点位则选中机器人，否则按当前工具取坐标
  canvas.addEventListener('click', (e) => {
    const rect = canvas.getBoundingClientRect()
    const px = e.clientX - rect.left
    const py = e.clientY - rect.top
    const id = hitRobot(px, py)
    if (id) {
      props.onRobotClick(id)
      return
    }
    if (state && state.tool === TOOLS.PICK) {
      const [wx, wy] = toWorld(px, py)
      props.onCanvasClick(wx, wy)
    }
  })

  /** hitRobot 返回命中的机器人 id（无命中返回 null） */
  function hitRobot(px, py) {
    if (!state) return null
    let best = null
    let bestDist = HIT_RADIUS
    for (const id in state.robots) {
      const r = state.robots[id]
      const [rx, ry] = toCanvas(r.x, r.y)
      const d = Math.hypot(rx - px, ry - py)
      if (d < bestDist) {
        bestDist = d
        best = id
      }
    }
    return best
  }

  // 悬停机器人点位 Tooltip：数据取本地 state，无网络请求
  attachTooltip(canvas, () => {
    if (!lastMouse || !state) return null
    const [px, py] = toCanvas(lastMouse[0], lastMouse[1])
    const id = hitRobot(px, py)
    if (!id) return null
    const r = state.robots[id]
    if (!r) return null

    const box = document.createElement('div')
    box.style.cssText = 'display:flex;flex-direction:column;gap:3px;font-family:var(--font-mono)'
    box.innerHTML =
      `<b>${id}</b>` +
      `<span>${STATE_NAMES[r.state] || r.state} · ${Math.round(r.battery)}%</span>` +
      `<span style="color:var(--text-muted)">x: ${r.x.toFixed(1)}  y: ${r.y.toFixed(1)}</span>`
    return box
  })

  /* ========== 绘制主循环 ========== */

  function draw() {
    const w = canvas.clientWidth
    const h = canvas.clientHeight
    if (w === 0) {
      requestAnimationFrame(draw)
      return
    }
    ctx.clearRect(0, 0, w, h)

    if (!state) {
      requestAnimationFrame(draw)
      return
    }

    // -- 网格层：按视口偏移平移绘制离屏图（拖动不重建网格） --
    const { dx, dy } = state.viewport
    const gx = w / 2 + dx * scale - (GRID / 2) * scale
    const gy = h / 2 - dy * scale - (GRID / 2) * scale
    ctx.drawImage(off, gx, gy)

    const now = performance.now()

    // -- 历史轨迹（紫色虚线，静态数据）--
    if (state.historyTrack) drawPolyline(state.historyTrack.points, '#bc8cff', [4, 4], w, h)

    // -- 实时尾迹（半透明实线，仅开关联通时绘制）--
    if (state.showTrail) {
      for (const id in state.robots) {
        const trail = state.robots[id].trail
        if (trail.length > 1) {
          const color = STATE_COLORS[state.robots[id].state] || STATE_COLORS.OFFLINE
          drawPolyline(trail, color + '66', [], w, h, id)
        }
      }
    }

    // -- 机器人点位 --
    for (const id in state.robots) {
      const r = state.robots[id]
      const [px, py] = toCanvasWith(w, h, r.x, r.y)
      const color = STATE_COLORS[r.state] || STATE_COLORS.OFFLINE
      let radius = 5

      // CHARGING：半径呼吸脉冲（1.5s 周期正弦）
      if (r.state === 'CHARGING') radius = 5 + 2 * Math.sin((now / 1500) * Math.PI * 2)
      // ERROR：透明度方波闪烁（0.5s）
      const alpha = r.state === 'ERROR' ? (Math.floor(now / 500) % 2 ? 0.35 : 1) : 1

      // 光晕
      const glow = ctx.createRadialGradient(px, py, 0, px, py, 16)
      glow.addColorStop(0, color + '40')
      glow.addColorStop(1, 'transparent')
      ctx.fillStyle = glow
      ctx.globalAlpha = alpha
      ctx.beginPath()
      ctx.arc(px, py, 16, 0, Math.PI * 2)
      ctx.fill()

      // 点位
      ctx.fillStyle = color
      ctx.beginPath()
      ctx.arc(px, py, radius, 0, Math.PI * 2)
      ctx.fill()

      // MOVING：朝向三角（由尾迹最后两点算方向）
      if (r.state === 'MOVING' && r.trail && r.trail.length > 0) {
        const prev = r.trail[r.trail.length - 1]
        const ang = Math.atan2(r.y - prev.y, r.x - prev.x)
        drawHeading(px, py, ang, color)
      }

      // 选中环：单选蓝环；多选（勾选集合内）蓝环 + 淡光晕
      if (state.selectedId === id) {
        drawRing(px, py, 10, '#58a6ff', 1.5)
      } else if (state.multiSelect && state.selectedIds.has(id)) {
        drawRing(px, py, 10, '#58a6ff', 1.5)
        drawRing(px, py, 14, 'rgba(88,166,255,0.25)', 4)
      }

      // 标签
      ctx.font = '11px "JetBrains Mono"'
      ctx.fillStyle = color
      ctx.textAlign = 'left'
      ctx.fillText(id, px + 12, py + 4)
      ctx.globalAlpha = 1
    }

    // -- 本地下发目标标记 ◇（机器人到达后由 store 清除）--
    for (const id in state.localTargets) {
      const t = state.localTargets[id]
      const [px, py] = toCanvasWith(w, h, t.x, t.y)
      ctx.strokeStyle = '#bc8cff'
      ctx.lineWidth = 1.5
      ctx.beginPath()
      ctx.moveTo(px, py - 6)
      ctx.lineTo(px + 6, py)
      ctx.lineTo(px, py + 6)
      ctx.lineTo(px - 6, py)
      ctx.closePath()
      ctx.stroke()
    }

    // -- 框选虚线矩形（世界坐标，随视口变换）--
    if (selectionRect) {
      const [x1, y1] = toCanvasWith(w, h, selectionRect.x1, selectionRect.y1)
      const [x2, y2] = toCanvasWith(w, h, selectionRect.x2, selectionRect.y2)
      const rx = Math.min(x1, x2)
      const ry = Math.min(y1, y2)
      const rw = Math.abs(x2 - x1)
      const rh = Math.abs(y2 - y1)
      ctx.setLineDash([5, 4])
      ctx.strokeStyle = '#58a6ff'
      ctx.fillStyle = 'rgba(88, 166, 255, 0.15)'
      ctx.lineWidth = 1
      ctx.fillRect(rx, ry, rw, rh)
      ctx.strokeRect(rx, ry, rw, rh)
      ctx.setLineDash([])
    }

    requestAnimationFrame(draw)
  }

  /** drawPolyline 画折线（尾迹 / 历史轨迹） */
  function drawPolyline(points, color, dash, w, h, robotId) {
    ctx.save()
    ctx.setLineDash(dash)
    ctx.strokeStyle = color
    ctx.lineWidth = 1.5
    ctx.beginPath()
    // 尾迹点的最后一个点用机器人当前坐标补齐，避免线落后于点位
    points.forEach((p, i) => {
      const [px, py] = toCanvasWith(w, h, p.x, p.y)
      i === 0 ? ctx.moveTo(px, py) : ctx.lineTo(px, py)
    })
    if (robotId && state.robots[robotId]) {
      const r = state.robots[robotId]
      const [px, py] = toCanvasWith(w, h, r.x, r.y)
      ctx.lineTo(px, py)
    }
    ctx.stroke()
    ctx.restore()
  }

  /** drawHeading 画 MOVING 朝向三角（画布 y 向下，角度需翻转） */
  function drawHeading(px, py, worldAngle, color) {
    const ang = -worldAngle // 世界坐标 y 向上，画布 y 向下
    ctx.fillStyle = color
    ctx.beginPath()
    // 以点位为圆心、半径 9 处画一个小三角指向运动方向
    const cx = px + Math.cos(ang) * 9
    const cy = py + Math.sin(ang) * 9
    const a1 = ang
    const a2 = ang + Math.PI * 0.75
    const a3 = ang - Math.PI * 0.75
    ctx.moveTo(cx, cy)
    ctx.lineTo(px + Math.cos(a2) * 6, py + Math.sin(a2) * 6)
    ctx.lineTo(px + Math.cos(a3) * 6, py + Math.sin(a3) * 6)
    ctx.closePath()
    ctx.fill()
  }

  function drawRing(px, py, r, color, width) {
    ctx.strokeStyle = color
    ctx.lineWidth = width
    ctx.beginPath()
    ctx.arc(px, py, r, 0, Math.PI * 2)
    ctx.stroke()
  }

  window.addEventListener('resize', resize)
  resize()
  requestAnimationFrame(draw)

  /* ========== 对外接口 ========== */

  return {
    el: canvas,
    /** update 接收全局状态引用（rAF 循环每帧读取最新值） */
    update(s) {
      state = s
    },
    /** setSelectionRect 设置/清除选区矩形（世界坐标） */
    setSelectionRect(rect) {
      selectionRect = rect
    },
    toWorld,
    getScale: () => scale,
  }
}
