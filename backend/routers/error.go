package routers

type ErrorResponse struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
}

// OK 成功
func OK() ErrorResponse {
	return ErrorResponse{ErrCode: 0, ErrMsg: "OK"}
}

// DefaultError 默认报错
func DefaultError() ErrorResponse {
	return ErrorResponse{ErrCode: 1000, ErrMsg: "出错"}
}

// ParamsError 缺少必要参
func ParamsError() ErrorResponse {
	return ErrorResponse{ErrCode: 1001, ErrMsg: "缺少参数"}
}

// NotFoundError 没有找到对应的doc
func NotFoundError() ErrorResponse {
	return ErrorResponse{ErrCode: 1002, ErrMsg: "Not Found Object."}
}

// LoginError 登录失败
func LoginError() ErrorResponse {
	return ErrorResponse{ErrCode: 1005, ErrMsg: "登录出错"}
}

// AuthError 验证失败
func AuthError() ErrorResponse {
	return ErrorResponse{ErrCode: 1006, ErrMsg: "账号验证失败"}
}

// PermissionError 没有操作权限
func PermissionError() ErrorResponse {
	return ErrorResponse{ErrCode: 1008, ErrMsg: "没有权限"}
}

// RequireError 参数错误
func RequireError() ErrorResponse {
	return ErrorResponse{ErrCode: 1009, ErrMsg: "参数错误"}
}

// NewError 自定义错误
func NewError(code int, msg string) ErrorResponse {
	return ErrorResponse{ErrCode: code, ErrMsg: msg}
}
