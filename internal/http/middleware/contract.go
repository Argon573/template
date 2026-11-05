package middleware

import "context"

type httpMetrics interface {
	IncActive(ctx context.Context, method, route string)
	DecActive(ctx context.Context, method, route string)
}
