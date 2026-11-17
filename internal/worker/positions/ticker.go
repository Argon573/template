package positions

import "time"

type realTicker struct {
	t *time.Ticker
}

func (t *realTicker) C() <-chan time.Time { return t.t.C }
func (t *realTicker) Stop()               { t.t.Stop() }

type realTickerFactory struct{}

func (realTickerFactory) NewTicker(d time.Duration) ticker {
	return &realTicker{t: time.NewTicker(d)}
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }
