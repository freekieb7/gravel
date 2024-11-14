package http

import (
	"encoding/json"
	"log"
	"net/http"
)

type Response struct {
	http.ResponseWriter
}

func (response *Response) WithStatus(status int) *Response {
	response.WriteHeader(status)
	return response
}

func (response *Response) WithJson(data any) *Response {
	response.Header().Set("Content-Type", "application/json")

	if vStr, ok := data.(string); ok {
		response.Write([]byte(vStr))
	} else if err := json.NewEncoder(response).Encode(data); err != nil {
		log.Fatalf("response: encoding data to json failed")
	}

	return response
}

func (response *Response) WithText(data string) *Response {
	response.Header().Set("Content-Type", "plain/text")
	response.Write([]byte(data))
	return response
}
