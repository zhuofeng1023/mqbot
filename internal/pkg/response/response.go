package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhuofeng1023/mqbot/internal/pkg/errcode"
)

// Response 是 HTTP 接口统一的响应结构
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// Success 成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code: errcode.SUCCESS.Code,
		Msg:  errcode.SUCCESS.Msg,
		Data: data,
	})
}

// Fail 失败响应，HTTP 状态码取自错误码定义（未设置时缺省 200）
func Fail(c *gin.Context, apperr *errcode.AppError) {
	status := apperr.HTTP
	if status == 0 {
		status = http.StatusOK
	}
	c.JSON(status, Response{
		Code: apperr.Code,
		Msg:  apperr.Msg,
		Data: nil,
	})
}
