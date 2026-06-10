package logging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"

	"github.com/behzod/pageSDK/logging/log"
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
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	blw := &Logger{ResponseWriter: c.Writer, ReqID: fmt.Sprintf("%v", reqId.Load()), Request: c.Request, Body: string(bodyBytes)}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	c.Writer = blw
	c.Next()
}
