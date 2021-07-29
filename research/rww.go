package research

import "net/http"

type ResponseWriterWrapper struct {
	status int
	http.ResponseWriter
}

func (rww *ResponseWriterWrapper) Status() int {
	return rww.status
}

func (rww *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rww.status = statusCode
	rww.ResponseWriter.WriteHeader(statusCode)
}

func NewResponseWriterWrapper(rww http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{http.StatusOK, rww}
}
