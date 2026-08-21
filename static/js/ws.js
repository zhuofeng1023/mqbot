/**
 * ws.js - WebSocket 封装
 *
 * 3s 自动重连；连接/断开状态通过 onState 回调上抛；
 * 消息以解析后的对象交给 onMessage，订阅方不关心传输细节。
 */

const WS_URL = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`
const RECONNECT_MS = 3000

/**
 * connectWS 建立 WebSocket 连接
 * @param {(msg: object) => void} onMessage 收到解析后的消息（hub 推送的 StatusBody）
 * @param {(connected: boolean) => void} onState 连接状态变化
 * @returns {{close: () => void}} 控制句柄（页面卸载时主动关闭，不再重连）
 */
export function connectWS(onMessage, onState) {
  let ws = null
  let closed = false // 用户主动关闭标志，主动关闭不触发重连

  function connect() {
    ws = new WebSocket(WS_URL)

    ws.onopen = () => onState(true)
    ws.onclose = () => {
      onState(false)
      if (!closed) setTimeout(connect, RECONNECT_MS) // 自动重连
    }
    ws.onerror = () => ws.close()
    ws.onmessage = (e) => {
      try {
        onMessage(JSON.parse(e.data))
      } catch (_) {
        // 非 JSON 消息直接忽略
      }
    }
  }

  connect()

  return {
    close() {
      closed = true
      ws && ws.close()
    },
  }
}
