/**
 * api.js — fetch 封装与后端 REST 接口
 *
 * 统一解析 hub 的响应信封 {code, msg, data}：
 * code === 0 视为成功返回 data；非 0 抛 ApiError（携带业务码与消息，
 * 码表见 internal/pkg/errcode/code.go）。
 */

const API_BASE = '/api/v1' // 与后端 http.api.prefix 一致
const TIMEOUT_MS = 10000 // 请求超时

/** ApiError 业务错误：code 为后端业务码，msg 为用户可读消息 */
export class ApiError extends Error {
  constructor(code, msg) {
    super(msg)
    this.code = code
  }
}

/**
 * request 底层 fetch 封装
 * @param {string} path 相对 /api/v1 的路径
 * @param {object} options fetch 配置（method/body 等）
 * @returns {Promise<any>} 成功时的 data 字段
 */
async function request(path, options = {}) {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), TIMEOUT_MS)

  let res
  try {
    res = await fetch(`${API_BASE}${path}`, { ...options, signal: ctrl.signal })
  } catch (e) {
    throw new ApiError(-1, e.name === 'AbortError' ? '请求超时' : `网络错误: ${e}`)
  } finally {
    clearTimeout(timer)
  }

  // 非 JSON 响应（如网关 502）统一兜底
  let json
  try {
    json = await res.json()
  } catch (_) {
    throw new ApiError(-1, `服务异常 (HTTP ${res.status})`)
  }
  if (json.code !== 0) throw new ApiError(json.code, json.msg || '未知错误')
  return json.data
}

/* ========== 单机接口 ========== */

/** 获取所有设备列表：[]Device{id,state,battery,x,y,speed,last_seen,online} */
export function list() {
  return request('/robots/')
}

/** 下发移动指令。成功仅代表"已下发"，不代表机器人已移动 */
export function move(id, x, y) {
  return request(`/robots/${encodeURIComponent(id)}/move`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ x, y }),
  })
}

/** 下发紧急停止指令 */
export function stop(id) {
  return request(`/robots/${encodeURIComponent(id)}/stop`, { method: 'POST' })
}

/**
 * 查询设备历史轨迹（TDengine 降采样）
 * @param {string} id 设备 ID
 * @param {number} from 起始毫秒时间戳
 * @param {number} to 截止毫秒时间戳
 * @param {string} interval 聚合窗口，如 "10s"
 * @returns {Promise<{ts,x,y}[]>} 轨迹点
 */
export function track(id, from, to, interval = '10s') {
  return request(
    `/devices/${encodeURIComponent(id)}/track?from=${from}&to=${to}&interval=${interval}`,
  )
}

/* ========== 批量接口（复用单机接口，无需后端改动）========== */

/**
 * 批量下发移动指令：Promise.allSettled 并发调用单机接口
 * @returns {Promise<{ok: string[], fail: {id, err}[]}>} 成功/失败汇总
 */
export async function batchMove(ids, x, y) {
  const results = await Promise.allSettled(ids.map((id) => move(id, x, y)))
  return settle(results, ids)
}

/** 批量下发停止指令 */
export async function batchStop(ids) {
  const results = await Promise.allSettled(ids.map((id) => stop(id)))
  return settle(results, ids)
}

/** 汇总 allSettled 结果为成功/失败列表 */
function settle(results, ids) {
  const ok = []
  const fail = []
  results.forEach((r, i) => {
    if (r.status === 'fulfilled') ok.push(ids[i])
    else fail.push({ id: ids[i], err: r.reason instanceof ApiError ? r.reason.message : String(r.reason) })
  })
  return { ok, fail }
}
