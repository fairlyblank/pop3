// Copyright 2013, fairlyblank
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package pop3

import (
	"fmt"
	"net"
	"bytes"
	"bufio"
	"strconv"
	"strings"
	"crypto/tls"
	"net/textproto"
)

type Client struct {
	addr       string
	rd         *textproto.Reader
	wt         *textproto.Writer
}

func Dial(addr string) (client *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, Pop3Error{"Dial", err.Error()}
	}
	return newClient(conn, addr)

}

func DialTLS(addr string) (client *Client, err error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		return nil, Pop3Error{"DialTLS", err.Error()}
	}
	return newClient(conn, addr)
}

func newClient(conn net.Conn, addr string) (*Client, error) {
	client := new(Client)

	client.addr = addr
	client.rd = textproto.NewReader(bufio.NewReader(conn))
	client.wt = textproto.NewWriter(bufio.NewWriter(conn))

	_, err := client.readLine()
	if err != nil {
		return nil, Pop3Error{"Read Greeting", err.Error()}
	}
	return client, nil
}

func (c *Client) AuthBasic(user, pass string) error {
	err := c.User(user)
	if err != nil {
		return err
	}

	err = c.Pass(pass)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) User(name string) error {
	_, _, err := c.command(USER, name, false)
	if err != nil {
		return Pop3Error{"USER", err.Error()}
	}
	return nil
}

func (c *Client) Pass(secret string) error {
	_, _, err := c.command(PASSWORD, secret, false)
	if err != nil {
		return Pop3Error{"PASS", err.Error()}
	}
	return nil
}

func (c *Client) Noop() error {
	_, _, err := c.command(NOOP, "", false)
	if err != nil {
		return Pop3Error{"NOOP", err.Error()}
	}
	return nil
}

func (c *Client) Reset() error {
	_, _, err := c.command(RESET, "", false)
	if err != nil {
		return Pop3Error{"RSET", err.Error()}
	}
	return nil
}

func (c *Client) Quit() error {
	_, _, err := c.command(QUIT, "", false)
	if err != nil {
		return Pop3Error{"QUIT", err.Error()}
	}
	return nil
}

func (c *Client) Delete(index int) error {
	if index <= 0 {
		return Pop3Error{"DELE", "invalid index"}
	}

	_, _, err := c.command(DELETE, strconv.Itoa(index), false)
	if err != nil {
		return Pop3Error{"DELE", err.Error()}
	}
	return nil
}

func (c *Client) Status() (int, int, error) {
	line, _, err := c.command(STATUS, "", false)
	if err != nil {
		return 0, 0, Pop3Error{"STAT", err.Error()}
	}

	fds := strings.Fields(line)
	if len(fds) < 2 {
		return 0, 0, Pop3Error{"STAT Response", line}
	}

	i1, err := strconv.Atoi(fds[0])
	if err != nil {
		return 0, 0, Pop3Error{"STAT Response", line}
	}
	i2, err := strconv.Atoi(fds[1])
	if err != nil {
		return 0, 0, Pop3Error{"STAT Response", line}
	}
	return i1, i2, nil
}

func (c *Client) List(index int) (int, int, error) {
	if index <= 0 {
		return 0, 0, Pop3Error{"LIST", "invalid index"}
	}

	line, _, err := c.command(LIST, strconv.Itoa(index), false)
	if err != nil {
		return 0, 0, Pop3Error{"LIST", err.Error()}
	}

	fds := strings.Fields(line)
	if len(fds) < 2 {
		return 0, 0, Pop3Error{"LIST Response", line}
	}

	i1, err := strconv.Atoi(fds[0])
	if err != nil {
		return 0, 0, Pop3Error{"LIST Response", line}
	}
	i2, err := strconv.Atoi(fds[1])
	if err != nil {
		return 0, 0, Pop3Error{"LIST Response", line}
	}
	return i1, i2, nil
}

func (c *Client) ListAll() ([]int, error) {
	_, bts, err := c.command(LIST, "", true)
	if err != nil {
		return nil, Pop3Error{"LIST", err.Error()}
	}

	lines := bytes.Split(bts, []byte("\n"))
	ret := make([]int, 0, 10)
	for i, line := range lines {
		if len(line) <= 0 {
			break
		}
		str := string(line)
		fds := strings.Fields(str)
		if len(fds) < 2 {
			return nil, Pop3Error{"LIST Response", str}
		}
		i1, err := strconv.Atoi(fds[0])
		if err != nil {
			return nil, Pop3Error{"LIST Response", str}
		}
		i2, err := strconv.Atoi(fds[1])
		if err != nil {
			return nil, Pop3Error{"LIST Response", str}
		}
		if i1 != i + 1 {
			return nil, Pop3Error{"LIST Response", str}
		}
		ret = append(ret, i2)
	}
	return ret, nil
}

func (c *Client) Retrieve(index int) ([]byte, error) {
	_, bts, err := c.command(RETRIEVE, strconv.Itoa(index), true)
	if err != nil {
		return nil, Pop3Error{"RETR", err.Error()}
	}

	return bts, nil
}

func (c *Client) command(name, args string, multi bool) (string, []byte, error) {
	var err error
	if len(args) > 0 {
		err = c.printfLine("%s %s", name, args)
	} else {
		err = c.printfLine("%s", name)
	}
		
	if err != nil {
		return "", nil, fmt.Errorf("PrintfLine: %v", err)
	}

	line, err := c.readLine()
	if err != nil {
		return line, nil, err
	}

	if multi == false {
		return line, nil, nil
	}

	bts, err := c.readDotBytes()
	if err != nil {
		return line, nil, err
	}

	return line, bts, nil
}

func (c *Client) readLine() (string, error) {
	line, err := c.rd.ReadLine()
	if err != nil {
		return "", fmt.Errorf("ReadLine: %v", err)
	}

	if strings.HasPrefix(line, "+OK") {
		return line[4:], nil
	} else if strings.HasPrefix(line, "-ERR") {
		return line[5:], fmt.Errorf("Server Response: %s", line)
	}

	return "", fmt.Errorf("Unkown Response: %s", line)
}

func (c *Client) readDotBytes() ([]byte, error) {
	bts, err := c.rd.ReadDotBytes()
	if err != nil {
		return nil, fmt.Errorf("ReadDotBytes: %v", err)
	}

	return bts, nil
}

func (c *Client) printfLine(format string, args ...interface{}) error {
	return c.wt.PrintfLine(format, args...)
}


