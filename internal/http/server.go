package http

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zhuofeng1023/mqbot/internal/config"
	"github.com/zhuofeng1023/mqbot/internal/database"
	"github.com/zhuofeng1023/mqbot/internal/hub"
	"github.com/zhuofeng1023/mqbot/internal/mqtt"
)

// Server 是 hub 的 HTTP/WebSocket 服务，聚合路由、MQTT 客户端、设备注册表等组件
type Server struct {
	cfg     *config.HTTPConfig              //  HTTP配置，包含服务器相关配置
	Router  *gin.Engine                     //  Gin引擎，用于HTTP路由处理
	wsConns map[*websocket.Conn]chan []byte //  WebSocket连接映射，存储连接和对应的通道

	MqttClient mqtt.Publisher       //  MQTT客户端，用于MQTT通信（实现 Publish 的连接管理器）
	Requester  *mqtt.Requester     //  MQTT请求响应管理器，用于处理MQTT请求
	Registry   *hub.DeviceRegistry //  设备注册表，用于管理设备连接
	msgBuffer  [][]byte            // 消息缓冲区 ，用于临时存储消息
	mu         sync.Mutex          //  互斥锁，用于并发控制

	MsgCount  atomic.Int64 // 收到消息总数
	WsClients atomic.Int64 // WS 连接数
	startTime time.Time    // 启动时间

	Storage *database.Storage
}

// NewServer 创建 HTTP 服务实例，初始化路由、CORS 与设备注册表
func NewServer(cfg *config.HTTPConfig) *Server {
	r := gin.Default()

	// 启用 CORS
	if cfg.CORS.Enabled {
		r.Use(cors.New(cors.Config{
			AllowOrigins: cfg.CORS.AllowedOrigins,
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
		}))
	}

	s := &Server{
		Router:    r,
		Registry:  hub.NewDeviceRegistry(),
		wsConns:   make(map[*websocket.Conn]chan []byte),
		cfg:       cfg,
		startTime: time.Now(),
	}
	return s
}

// Start 启动 HTTP 服务
func (s *Server) Start() error {
	s.registerRoutes()
	return s.Router.Run(fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port))
}
