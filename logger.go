package gee

import (
	"log"
	"time"
)

func Logger() HandlerFunc {
	return func(ctx *Context) {
		// start timer
		t := time.Now()
		ctx.Next()
		// Calculate resolution time
		log.Printf("[%d] %s in %v", ctx.StatusCode, ctx.Req.RequestURI, time.Since(t))
	}
}
