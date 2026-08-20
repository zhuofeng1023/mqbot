package errcode

import "net/http"

// AppError 定义业务错误结构体
type AppError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	HTTP int    `json:"-"` // 对应的 HTTP 状态码，在错误码定义处写死，调用方不需要关心
}

// Error 获取错误消息
func (e *AppError) Error() string {
	return e.Msg
}

// Code: 0 Msg: "success" 表示请求成功
var SUCCESS = &AppError{Code: 0, Msg: "success", HTTP: http.StatusOK}

// 通用错误 (400xx)
// 参数/格式类问题属于预期内失败，HTTP 用 200，前端按业务码处理
var (
	// Code: "40000" Msg: "参数错误"
	ERR_BAD_REQUEST = &AppError{Code: 40000, Msg: "参数错误", HTTP: http.StatusOK}

	// Code: "40001" Msg: "请求数据解析失败"
	ERR_INVALID_JSON = &AppError{Code: 40001, Msg: "请求数据解析失败", HTTP: http.StatusOK}

	// Code: "40002" Msg: "无效的时间范围"
	ERR_INVALID_TIME_RANGE = &AppError{Code: 40002, Msg: "无效的时间范围", HTTP: http.StatusOK}

	// Code: "40100" Msg: "未授权或 Token 失效"
	ERR_UNAUTHORIZED = &AppError{Code: 40100, Msg: "未授权/Token失效", HTTP: http.StatusUnauthorized}

	// Code: "40400" Msg: "请求的资源未找到"
	ERR_NOT_FOUND = &AppError{Code: 40400, Msg: "资源未找到", HTTP: http.StatusNotFound}
)

// 服务器错误 (500xx)
// 预期外的故障，HTTP 返回真实错误码，方便网关/监控/前端拦截器统一兜底
var (
	// Code: "50000" Msg: "服务器内部错误"
	ERR_INTERNAL_SERVER = &AppError{Code: 50000, Msg: "服务器内部错误", HTTP: http.StatusInternalServerError}

	// Code: "50001" Msg: "存储组件未启用"
	ERR_STORAGE_DISABLE = &AppError{Code: 50001, Msg: "存储组件未启用", HTTP: http.StatusServiceUnavailable}

	// Code: "50002" Msg: "历史数据查询失败"
	ERR_QUERY_FAILED = &AppError{Code: 50002, Msg: "历史数据查询失败", HTTP: http.StatusInternalServerError}
)

// 业务错误 (600xx)
// 业务规则上的失败，HTTP 用 200，前端按业务码提示用户
var (
	// Code: "60001" Msg: "设备不存在"
	ERR_DEVICE_NOT_FOUND = &AppError{Code: 60001, Msg: "设备不存在", HTTP: http.StatusOK}

	// Code: "60002" Msg: "设备离线"
	ERR_DEVICE_OFFLINE = &AppError{Code: 60002, Msg: "设备离线", HTTP: http.StatusOK}

	// Code: "60003" Msg: "无效的指令"
	ERR_INVALID_COMMAND = &AppError{Code: 60003, Msg: "无效的指令", HTTP: http.StatusOK}

	// Code: "60004" Msg: "指令下发失败"
	ERR_COMMAND_FAILED = &AppError{Code: 60004, Msg: "指令下发失败", HTTP: http.StatusOK}

	// Code: "60005" Msg: "设备响应超时或离线"
	ERR_DEVICE_TIMEOUT = &AppError{Code: 60005, Msg: "设备响应超时或离线", HTTP: http.StatusOK}

	// Code: "60006" Msg: "设备响应数据解析失败"
	ERR_DEVICE_RESPONSE_INVALID = &AppError{Code: 60006, Msg: "设备响应数据解析失败", HTTP: http.StatusOK}
)
