package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey int

const (
	sqlStartKey ctxKey = iota
	sqlTextKey
)

type sqlTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

func newSQLTracer() *sqlTracer {
	inner := otelpgx.NewTracer(
		otelpgx.WithMeterProvider(metricnoop.NewMeterProvider()),
		otelpgx.WithMetricOperationName(otelpgx.SQLOperationName),
	)

	duration, err := otel.Meter("github.com/Argon573/template/internal/infra/postgres").Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
	)
	if err != nil {
		otel.Handle(err)
	}

	return &sqlTracer{Tracer: inner, duration: duration}
}

func (t *sqlTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = context.WithValue(ctx, sqlStartKey, time.Now())
	ctx = context.WithValue(ctx, sqlTextKey, data.SQL)
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)

	if table := tableFromSQL(data.SQL); table != "" {
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.collection.name", table))
	}
	return ctx
}

func (t *sqlTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	sql, _ := ctx.Value(sqlTextKey).(string)
	op := otelpgx.SQLOperationName(ctx, sql)
	table := tableFromSQL(sql)

	attrs := []attribute.KeyValue{
		attribute.String("db.operation.name", op),
	}
	if table != "" {
		attrs = append(attrs, attribute.String("db.collection.name", table))
	}

	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		errType := errorTypeName(data.Err)
		attrs = append(attrs, attribute.String("error.type", errType))
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", errType))
	}

	if start, ok := ctx.Value(sqlStartKey).(time.Time); ok && t.duration != nil {
		t.duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	}

	t.Tracer.TraceQueryEnd(ctx, conn, data)
}

func errorTypeName(err error) string {
	return fmt.Sprintf("%T", err)
}

func tableFromSQL(sql string) string {
	s := strings.TrimSpace(sql)
	upper := strings.ToUpper(s)

	switch {
	case strings.HasPrefix(upper, "INSERT INTO "):
		return firstIdent(s[len("INSERT INTO "):])
	case strings.HasPrefix(upper, "UPDATE "):
		return firstIdent(s[len("UPDATE "):])
	case strings.HasPrefix(upper, "DELETE FROM "):
		return firstIdent(s[len("DELETE FROM "):])
	}

	if idx := strings.Index(upper, " FROM "); idx >= 0 {
		return firstIdent(s[idx+len(" FROM "):])
	}
	return ""
}

func firstIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimPrefix(s, "`")
	for i, r := range s {
		if r == ' ' || r == '\n' || r == '\t' || r == '(' || r == '"' || r == '`' || r == ',' {
			return s[:i]
		}
	}
	return s
}
