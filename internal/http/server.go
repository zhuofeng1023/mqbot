package http

import (
	"fmt"
	"sync"

	"github.com/eclipse/paho.golang/paho"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zhuofeng1023/mqbot/internal/config"
	"github.com/zhuofeng1023/mqbot/internal/hub"
	"github.com/zhuofeng1023/mqbot/internal/mqtt"
)

type Server struct {
	Router     *gin.Engine
	MqttClient *paho.Client
	Requester  *mqtt.Requester
	Registry   *hub.DeviceRegistry
	wsConns    map[*websocket.Conn]chan []byte
	msgBuffer  [][]byte // 消息缓冲区
	mu         sync.Mutex
	cfg        *config.HTTPConfig
}

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
		Router:   r,
		Registry: hub.NewDeviceRegistry(),
		wsConns:  make(map[*websocket.Conn]chan []byte),
		cfg:      cfg,
	}
	return s
}

// 启动服务
func (s *Server) Start() error {
	s.registerRoutes()
	return s.Router.Run(fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port))
}
