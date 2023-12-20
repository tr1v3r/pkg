package fetch

// RespCode 漏洞响应状态码
type RespCode int

// 定义一些 API 的响应码，语义上尽量与 HTTP 状态码对齐
const (
	CodeOK        RespCode = 200 // 正常
	CodeBadReq    RespCode = 400 // 请求格式异常
	CodeNeedAuth  RespCode = 401 // 需要登陆
	CodeForbidden RespCode = 403 // 无权限访问
	CodeServerErr RespCode = 500 // 服务端异常
)

// Pagination 分页的基本信息
type Pagination struct {
	Page     int `json:"page" example:"1"`
	Total    int `json:"total" example:"1024"`
	PageSize int `json:"page_size" example:"20"`
}

// JSONResult 表示 API 返回的数据
type JSONResult struct {
	Code RespCode `json:"code" example:"200"`
	Msg  string   `json:"msg" example:""`

	Data interface{} `json:"data,omitempty"`

	// Pagination 用于存放分页数据，对于不需要分页的 API，将不会出现这个字段
	Pagination *Pagination `json:"pagination,omitempty"`
}
