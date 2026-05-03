package response

import "net/http"

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(message string) APIResponse {
	return APIResponse{Code: http.StatusOK, Message: message}
}

func OKWithData(data interface{}) APIResponse {
	return APIResponse{Code: http.StatusOK, Message: "ok", Data: data}
}

func Error(code int, message string) APIResponse {
	return APIResponse{Code: code, Message: message}
}
