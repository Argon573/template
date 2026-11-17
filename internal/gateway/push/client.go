package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	BaseURL string
	Timeout time.Duration
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

type pushRequest struct {
	RequestID   string            `json:"request_id"`
	RecipientID string            `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func (c *Client) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	ctx, span := otel.Tracer("github.com/Argon573/template/internal/gateway/push").Start(
		ctx,
		"push.RequestPosition",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("peer.service", "push-service"),
			attribute.String("trip.id", tripID.String()),
			attribute.String("push.kind", "REQUEST_POSITION"),
		),
	)
	defer span.End()

	body, err := json.Marshal(pushRequest{
		RequestID:   uuid.New().String(),
		RecipientID: driverID.String(),
		Kind:        "REQUEST_POSITION",
		Data: map[string]string{
			"trip_id": tripID.String(),
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("marshal push request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/push", bytes.NewReader(body))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("push transport: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	err = fmt.Errorf("push server error: status %d", resp.StatusCode)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		err = fmt.Errorf("push client error: status %d", resp.StatusCode)
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
