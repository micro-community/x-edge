// Package quic provides a QUIC based transport
package quic

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"time"

	quic "github.com/lucas-clemente/quic-go"
	"github.com/micro/go-micro/v2/transport"
	utls "github.com/micro/go-micro/v2/util/tls"
)

type quicSocket struct {
	s  quic.Session
	st quic.Stream
}

type quicTransport struct {
	opts transport.Options
}

type quicClient struct {
	*quicSocket
	t    *quicTransport
	opts transport.DialOptions
}

type quicListener struct {
	l    quic.Listener
	t    *quicTransport
	opts transport.ListenOptions
}

func (q *quicClient) Close() error {
	return q.quicSocket.st.Close()
}

func (q *quicSocket) Recv(m *transport.Message) error {
	//return q.dec.Decode(&m)
	m.Body = make([]byte, 1024)
	q.st.Read(m.Body)
	//reader := bufio.NewReader(q.st)
	data, err := ioutil.ReadAll(q.st)
	m.Body = data
	return err
}

func (q *quicSocket) Send(m *transport.Message) error {
	// set the write deadline
	q.st.SetWriteDeadline(time.Now().Add(time.Second * 10))
	q.st.Write(m.Body)
	return nil

}

func (q *quicSocket) Close() error {
	return q.s.CloseWithError(quic.ErrorCode(0), "")
}

func (q *quicSocket) Local() string {
	return q.s.LocalAddr().String()
}

func (q *quicSocket) Remote() string {
	return q.s.RemoteAddr().String()
}

func (q *quicListener) Addr() string {
	return q.l.Addr().String()
}

func (q *quicListener) Close() error {
	return q.l.Close()
}

func (q *quicListener) Accept(fn func(transport.Socket)) error {
	for {
		s, err := q.l.Accept(context.TODO())
		if err != nil {
			return err
		}

		stream, err := s.AcceptStream(context.TODO())
		if err != nil {
			continue
		}

		go func() {
			fn(&quicSocket{
				s:  s,
				st: stream,
			})
		}()
	}
}

func (q *quicTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&q.opts)
	}
	return nil
}

func (q *quicTransport) Options() transport.Options {
	return q.opts
}

func (q *quicTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	var options transport.DialOptions
	for _, o := range opts {
		o(&options)
	}

	config := q.opts.TLSConfig
	if config == nil {
		config = &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"http/1.1"},
		}
	}
	s, err := quic.DialAddr(addr, config, &quic.Config{
		MaxIdleTimeout: time.Minute * 2,
		KeepAlive:      true,
	})
	if err != nil {
		return nil, err
	}

	st, err := s.OpenStreamSync(context.TODO())
	if err != nil {
		return nil, err
	}

	return &quicClient{
		&quicSocket{
			s:  s,
			st: st,
		},
		q,
		options,
	}, nil
}

func (q *quicTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	var options transport.ListenOptions
	for _, o := range opts {
		o(&options)
	}

	config := q.opts.TLSConfig
	if config == nil {
		cfg, err := utls.Certificate(addr)
		if err != nil {
			return nil, err
		}
		config = &tls.Config{
			Certificates: []tls.Certificate{cfg},
			NextProtos:   []string{"http/1.1"},
		}
	}

	l, err := quic.ListenAddr(addr, config, &quic.Config{KeepAlive: true})
	if err != nil {
		return nil, err
	}

	return &quicListener{
		l:    l,
		t:    q,
		opts: options,
	}, nil
}

func (q *quicTransport) String() string {
	return "quic"
}

//NewTransport returns a new quic transport
func NewTransport(opts ...transport.Option) transport.Transport {
	options := transport.Options{}

	for _, o := range opts {
		o(&options)
	}

	return &quicTransport{
		opts: options,
	}
}