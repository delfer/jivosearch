package breaker_test

import (
	"context"
	"testing"
	"time"

	. "github.com/kamilsk/breaker"
)

func TestWithContext(t *testing.T) {
	t.Run("active breaker", func(t *testing.T) {
		var (
			ctx, cancel = context.WithTimeout(context.Background(), 5*delta)
			br, _       = WithContext(ctx)
		)
		defer cancel()
		start := time.Now()
		<-br.Done()

		checkDuration(t, start.Add(5*delta), time.Now())
		checkBreakerIsReleased(t, br)
	})
	t.Run("closed breaker", func(t *testing.T) {
		var (
			ctx, cancel = context.WithTimeout(context.Background(), -delta)
			br, _       = WithContext(ctx)
		)
		defer cancel()
		start := time.Now()
		<-br.Done()

		checkDuration(t, start, time.Now())
		checkBreakerIsReleased(t, br)
	})
	t.Run("released breaker", func(t *testing.T) {
		var (
			ctx, cancel = context.WithTimeout(context.Background(), time.Hour)
			br, _       = WithContext(ctx)
		)
		defer cancel()
		br.Close()
		start := time.Now()
		<-br.Done()

		checkDuration(t, start, time.Now())
		checkBreakerIsReleased(t, br)
	})
	t.Run("canceled parent", func(t *testing.T) {
		var (
			ctx, cancel = context.WithTimeout(context.Background(), time.Hour)
			br, _       = WithContext(ctx)
		)
		cancel()
		start := time.Now()
		<-br.Done()

		checkDuration(t, start, time.Now())
		checkBreakerIsReleased(t, br)
	})
}
