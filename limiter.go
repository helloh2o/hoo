package hoo

import (
	"io"
	"golang.org/x/time/rate"
	"time"
	"fmt"
)

type reader struct {
	r       io.Reader
	limiter *rate.Limiter
}

func NewReader(r io.Reader, l *rate.Limiter) io.Reader {
	return &reader{
		r:       r,
		limiter: l,
	}
}

func (r *reader) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	if n <= 0 || err != nil {
		return n, err
	}
	now := time.Now()
	rv := r.limiter.ReserveN(now, n)
	if !rv.OK() {
		return 0, fmt.Errorf("%s", "Exceeds limiter's burst")
	}
	delay := rv.DelayFrom(now)
	time.Sleep(delay)
	return n, err
}
