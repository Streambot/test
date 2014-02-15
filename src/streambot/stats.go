package streambot

import(
	"net"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Statter struct {
	StatConn net.Conn
	Prefix 	 string
}

func NewLocalStatsDStatter() (s *Statter, err error) {
	conn, err := net.Dial("udp", ":8125")
	if err != nil {
		err = errors.New(fmt.Sprintf("Statter Error when instantiate UDP statting connection: %v", err))
		return
	}
	host, err := os.Hostname()
	if err != nil {
		err = errors.New(fmt.Sprintf("Statter Error when retrieving host name: %v", err))
		return
	}
	host = strings.Replace(host, ".", "-", -1)
	s = &Statter{conn, host + "."}
	return
}

func(s *Statter) Count(key string) {
	if key == "" {
		return
	}
	fmt.Fprintln(s.StatConn, fmt.Sprintf("%s%s:1|c", s.Prefix, key))
	return
}

func(s *Statter) Time(key string, val int) {
	if key == "" {
		return
	}
	fmt.Fprintln(s.StatConn, fmt.Sprintf("%s%s:%d|ms", s.Prefix, key, val))
	return
}