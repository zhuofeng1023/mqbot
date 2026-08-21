/**
 * dashboard.js - 指标条（4 张 StatCard 横排）
 * 在线数 / 平均电量 / 执行中任务数 / 异常数，全部由前端 robots 数据聚合。
 */

import { createStatCard } from '../components/statCard.js'

/**
 * createDashboard 创建指标条
 * @param {HTMLElement} mount 挂载容器
 * @returns {{update: (robots: object) => void}}
 */
export function createDashboard(mount) {
  const cards = {
    online: createStatCard({ label: '在线' }),
    battery: createStatCard({ label: '平均电量' }),
    moving: createStatCard({ label: '执行中' }),
    error: createStatCard({ label: '异常', tone: 'danger' }),
  }
  Object.values(cards).forEach((c) => mount.appendChild(c.el))

  return {
    /** update 用当前 robots 聚合并刷新指标（由渲染节流统一驱动） */
    update(robots) {
      const list = Object.values(robots)
      const online = list.filter((r) => r.state !== 'OFFLINE')
      const moving = list.filter((r) => r.state === 'MOVING')
      const error = list.filter((r) => r.state === 'ERROR')
      const avgBattery = online.length
        ? Math.round(online.reduce((s, r) => s + (r.battery || 0), 0) / online.length)
        : 0

      cards.online.update(online.length)
      // 平均电量低于阈值时整卡转警告色
      cards.battery.update(avgBattery + '%')
      cards.battery.el.className =
        'stat-card' + (online.length && avgBattery < 20 ? ' stat-card--warn' : '')
      cards.moving.update(moving.length)
      cards.error.update(error.length)
    },
  }
}
