package logging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/BekkkEvrika/pageSDK/logging/log"
	"github.com/gin-gonic/gin"
)

type Logger struct {
	gin.ResponseWriter
	ReqID   string
	Request *http.Request
	Body    string
}

func (w Logger) Write(b []byte) (int, error) {
	log.WriteLn("# " + w.ReqID + " URL: " + w.Request.URL.String())
	log.WriteLn("# " + w.ReqID + " METHOD: " + w.Request.Method)
	log.WriteLn("# " + w.ReqID + " BODY: " + w.Body)
	log.WriteLn("# " + w.ReqID + " RESPONSE: " + string(b))
	return w.ResponseWriter.Write(b)
}

var reqId atomic.Uint64

func LogMiddleware(c *gin.Context) {
	reqId.Add(1)
	log.WriteLn("REQUEST #" + fmt.Sprintf("%v", reqId.Load()))
	logAuthorizationHeader(fmt.Sprintf("%v", reqId.Load()), c.Request)
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	blw := &Logger{ResponseWriter: c.Writer, ReqID: fmt.Sprintf("%v", reqId.Load()), Request: c.Request, Body: string(bodyBytes)}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	c.Writer = blw
	c.Next()
}

func LogMiddlewareSecond(c *gin.Context) {
	reqId.Add(1)
	log.WriteLn("REQUEST FROM SECOND #" + fmt.Sprintf("%v", reqId.Load()))
	logAuthorizationHeader(fmt.Sprintf("%v", reqId.Load()), c.Request)
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	blw := &Logger{ResponseWriter: c.Writer, ReqID: fmt.Sprintf("%v", reqId.Load()), Request: c.Request, Body: string(bodyBytes)}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	c.Writer = blw
	c.Next()
}

func logAuthorizationHeader(reqID string, request *http.Request) {
	header := request.Header.Get("Authorization")
	if header == "" {
		return
	}
	log.WriteLn("# " + reqID + " AUTHORIZATION: " + maskAuthorizationHeader(header))
}

func maskAuthorizationHeader(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "<present non-bearer>"
	}
	token := parts[1]
	if len(token) <= 12 {
		return fmt.Sprintf("Bearer <present len=%d>", len(token))
	}
	return fmt.Sprintf("Bearer %s...%s len=%d", token[:8], token[len(token)-4:], len(token))
}
