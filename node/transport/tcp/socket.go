//Package tcp provides a TCP transport
package tcp

import (
	"bufio"
	"errors"
	"net"
	"time"

	nts "github.com/micro-community/x-edge/node/transport"
	"github.com/micro/go-micro/v2/transport"
	"github.com/micro/go-micro/v2/util/log"
)

type tcpSocket struct {
	conn          net.Conn
	encBuf        *bufio.Writer
	timeout       time.Duration
	dataExtractor nts.DataExtractor
}

func (t *tcpSocket) Local() string {
	return t.conn.LocalAddr().String()
}

func (t *tcpSocket) Remote() string {
	return t.conn.RemoteAddr().String()
}

func (t *tcpSocket) Recv(m *transport.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}
	//寻找确定disconnected的错误，t.conn代表一个实际的连接
	//替代NEWScanner的错误
	//scanner disconnected的错误
	scanner := bufio.NewScanner(t.conn)

	scanner.Split(t.dataExtractor)

	if scanner.Scan() {
		m.Body = scanner.Bytes()
		return nil
	} else {
		log.Errorf("Scan fail ", scanner.Err().Error())
	}

	return errorTransportDataExtract
}

func (t *tcpSocket) Send(m *transport.Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		t.conn.SetDeadline(time.Now().Add(t.timeout))
	}

	writer := bufio.NewWriter(t.conn)
	writer.Write(m.Body)
	return writer.Flush()

	//_, err := t.conn.Write(m.Body)
	//return err
}

func (t *tcpSocket) Close() error {
	return t.conn.Close()
}
