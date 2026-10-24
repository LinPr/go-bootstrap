package resty

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"resty.dev/v3"
)

type slogLogger struct {
	LogBlackList LogBlackList

	requestHeader  bool
	responseHeader bool
}

type Option func(*slogLogger)

// LogBlackList is used to specify messages that should be ignored in the logs.
type LogBlackList struct {
	Message []string
}

func WithLogBlackList(blackList LogBlackList) Option {
	return func(l *slogLogger) {
		l.LogBlackList = blackList
	}
}

func WithoutRequestHeader() Option {
	return func(l *slogLogger) {
		l.requestHeader = false
	}
}

func NewSlogLogger(opts ...Option) resty.Logger {
	logger := &slogLogger{
		requestHeader:  true,
		responseHeader: true,
	}
	for _, opt := range opts {
		opt(logger)
	}
	return logger
}

func (l *slogLogger) transform(format string, args ...any) (string, []any) {
	msg := fmt.Sprintf(format, args...)
	data := &restyLog{}
	err := json.Unmarshal([]byte(msg), data)
	if err != nil {
		slog.Error("Failed to unmarshal resty log JSON: " + err.Error())
	}

	logmsg := fmt.Sprintf("http outgoing: %s %s://%s%s %s %d", data.Method, data.Proto, data.Host, data.URI, data.Action, data.StatusCode)
	logargMap := map[string]any{
		"action":          data.Action,
		"curl_cmd":        data.CurlCmd,
		"method":          data.Method,
		"url":             fmt.Sprintf("%s://%s%s", data.Proto, data.Host, data.URI),
		"status_code":     data.StatusCode,
		"status":          data.Status,
		"request_header":  data.RequestHeaders,
		"request_body":    data.RequestBody,
		"response_header": data.ResponseHeader,
		"response_body":   data.ResponseBody,
	}

	if !l.requestHeader {
		logargMap["request_header"] = nil
	}
	if !l.responseHeader {
		logargMap["response_header"] = nil
	}

	logargs := make([]any, 0, len(logargMap)*2)
	for k, v := range logargMap {
		if v == nil {
			continue
		}
		logargs = append(logargs, k, v)
	}
	return logmsg, logargs
}

func (l *slogLogger) verifyShouldLog(logmsg string, logargs []any) bool {
	for _, msg := range l.LogBlackList.Message {
		if strings.Contains(logmsg, msg) {
			return false
		}
	}
	return true
}

func (l *slogLogger) Errorf(format string, args ...any) {
	logmsg, logargs := l.transform(format, args...)
	if !l.verifyShouldLog(logmsg, logargs) {
		return
	}
	slog.Error(logmsg, logargs...)
}

func (l *slogLogger) Warnf(format string, args ...any) {
	logmsg, logargs := l.transform(format, args...)
	if !l.verifyShouldLog(logmsg, logargs) {
		return
	}
	slog.Warn(logmsg, logargs...)
}

func (l *slogLogger) Debugf(format string, args ...any) {
	logmsg, logargs := l.transform(format, args...)
	if !l.verifyShouldLog(logmsg, logargs) {
		return
	}
	slog.Debug(logmsg, logargs...)
}

type restyLog struct {
	Method         string `json:"method"`
	Proto          string `json:"proto"`
	Host           string `json:"host"`
	URI            string `json:"uri"`
	RequestHeaders string `json:"request_headers"`
	RequestBody    string `json:"request_body"`
	Status         string `json:"status"`
	StatusCode     int    `json:"status_code"`
	ResponseHeader string `json:"response_header"`
	ResponseBody   string `json:"response_body"`
	CurlCmd        string `json:"curl_cmd"`
	Action         string `json:"Action"`
}

func CustomLogFormatter(dl *resty.DebugLog) string {
	action := &restyLog{}
	_ = json.Unmarshal([]byte(dl.Request.Body), action)

	data := &restyLog{
		Method:         dl.Request.Method,
		Proto:          dl.Request.Proto,
		Host:           dl.Request.Host,
		URI:            dl.Request.URI,
		RequestHeaders: formatHttpHeader(dl.Request.Header),
		RequestBody:    dl.Request.Body,
		Status:         dl.Response.Status,
		StatusCode:     dl.Response.StatusCode,
		ResponseHeader: formatHttpHeader(dl.Response.Header),
		ResponseBody:   dl.Response.Body,
		CurlCmd:        dl.Request.CurlCmd,
		Action:         action.Action,
	}
	j, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to encode JSON: " + err.Error())
	}
	return string(j)
}

func formatHttpHeader(header http.Header) string {
	formattedHeader := make(map[string]string)
	for key, values := range header {
		formattedHeader[key] = strings.Join(values, ", ")
	}
	headerString, err := json.MarshalIndent(formattedHeader, "", "  ")
	if err != nil {
		return ""
	}
	return string(headerString)
}
