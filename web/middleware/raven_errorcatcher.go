package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	// "strings"
	"runtime"

	"github.com/getsentry/raven-go"
)

/*
Middleware that catches panics, and:
	- logs them
	- optionally reports them to sentry - pass in "" if you don't want this
	- sends a 500 response
You can also use ThrowError() to raise an error that this middleware will catch, for example
if you want an error to be reported to sentry
*/
func BuildErrorCatcher(sentryClient *raven.Client) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				err := recover()
				if err == nil {
					return
				}
				if sentryClient != nil {
					// Send the error to sentry
					const size = 1 << 12
					buf := make([]byte, size)
					n := runtime.Stack(buf, false)
					sentryClient.CaptureMessage(fmt.Sprintf("%v\nStacktrace:\n%s", err, buf[:n]), sentryClient.Tags)
				}

				switch err := err.(type) {
				case HttpError:
					log.Printf("Return response for error %s", err.Message)
					err.WriteResponse(w)
					return
				default:
					log.Printf("Panic: %v\n", err)
					debug.PrintStack()
					http.Error(w, http.StatusText(500), 500)
					return
				}
			}()

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(handler)
	}
}

/*
An error that encapsulates an HTTP status code and message.
Use ThrowError() to build an HttpError and raise it as a panic
*/
type HttpError struct {
	StatusCode int
	Message    string
}

func (h HttpError) Error() string {
	return h.Message
}

func (h HttpError) WriteResponse(w http.ResponseWriter) {
	http.Error(w, h.Message, h.StatusCode)
}

func MakeError(statusCode int, format string, params ...interface{}) error {
	return HttpError{statusCode, fmt.Sprintf(format, params...)}
}

func ThrowError(statusCode int, format string, params ...interface{}) {
	panic(MakeError(statusCode, format, params...))
}
