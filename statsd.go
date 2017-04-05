package statsd

import (
	"errors"
	"time"
)

var (
	ErrInvalidNetwork = errors.New("invalid network")
)

type options struct {
	timeout       time.Duration
	flushPeriod   time.Duration
	maxPacketSize int
	errHandler    func(error)

	prefix string
}

type Option func(*options)

func ErrorHandler(h func(error)) Option {
	return func(o *options) {
		o.errHandler = h
	}
}

func FlushPeriod(d time.Duration) Option {
	return func(o *options) {
		o.flushPeriod = d
	}
}

func Prefix(s string) Option {
	return func(o *options) {
		o.prefix = s
	}
}

func MaxPacketSize(n int) Option {
	return func(o *options) {
		o.maxPacketSize = n
	}
}

func Timeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

type Client struct {
	opts options

	cc *clientConn
}

func New(network, addr string, opt ...Option) (*Client, error) {
	switch network {
	case "udp", "tcp", "tcp4", "tcp6":
	default:
		return nil, ErrInvalidNetwork
	}

	c := &Client{}
	for _, o := range opt {
		o(&c.opts)
	}

	if c.opts.timeout <= 0 {
		c.opts.timeout = time.Second * 5
	}
	if c.opts.flushPeriod <= 0 {
		c.opts.flushPeriod = time.Millisecond * 100
	}
	if c.opts.maxPacketSize <= 0 {
		c.opts.maxPacketSize = 1400
	}

	cc, err := newClientConn(network, addr, c)
	if err != nil {
		return c, err
	}

	c.cc = cc

	return c, nil
}

func (c *Client) Increment(bucket ...Field) {
	c.CountInt(1, bucket...)
}

func (c *Client) CountInt(n int, bucket ...Field) {
	c.send(encode(MetricTypeCount, c.opts.prefix, bucket, Int(n)))
}

func (c *Client) CountInt64(n int64, bucket ...Field) {
	c.send(encode(MetricTypeCount, c.opts.prefix, bucket, Int64(n)))
}

func (c *Client) GaugeInt(n int, bucket ...Field) {
	c.send(encode(MetricTypeGauge, c.opts.prefix, bucket, Int(n)))
}

func (c *Client) GaugeInt64(n int64, bucket ...Field) {
	c.send(encode(MetricTypeGauge, c.opts.prefix, bucket, Int64(n)))
}

func (c *Client) Timing(start time.Time, bucket ...Field) {
	c.send(encode(MetricTypeTiming, c.opts.prefix, bucket, Int64(time.Now().Sub(start).Nanoseconds()/int64(time.Millisecond))))
}

func (c *Client) send(b *buf) {
	if b == nil {
		return
	}
	c.cc.write(b.Bytes())
	freeBuf(b)
}