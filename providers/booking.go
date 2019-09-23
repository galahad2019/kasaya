package providers

import (
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BookingServerProvider struct {
	addr string
}

func NewBookingServerProvider(addr string) *BookingServerProvider {
	return &BookingServerProvider{addr: addr}
}

func (sp *BookingServerProvider) GetServerList() ([]*Server, error) {
	sl := []*Server{}
	rsl := []*Server{}
	req, err := http.NewRequest("GET", sp.addr, nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Transport: http.DefaultTransport,
		Timeout:   time.Second * 10,
	}
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	strData := string(data)
	strData = strings.Replace(strData, "_", "/", -1)
	strData = strings.Replace(strData, "-", "+", -1)
	data, err = base64.RawStdEncoding.DecodeString(strData)
	if err != nil {
		return nil, err
	}
	rss := strings.Split(string(data), "\n")
	if len(rss) == 0 {
		return sl, nil
	}
	logrus.Infof("Retrieved raw server count: %d", len(rss))
	for i := 0; i < len(rss); i++ {
		if rss[i] == "" {
			continue
		}
		s, err := sp.processServer(rss[i])
		if err != nil {
			return nil, err
		}
		if s == nil {
			continue
		}
		sl = append(sl, s)
	}
	logrus.Infof("After ETL, remainning: %d", len(sl))
	wg := sync.WaitGroup{}
	wg.Add(len(sl))
	for i := 0; i < len(sl); i++ {
		go func(temp *Server) {
			if sp.checkServerPing(temp) {
				rsl = append(rsl, temp)
			}
			wg.Done()
		}(sl[i])
	}
	wg.Wait()
	logrus.Infof("After ping check, remainning: %d", len(rsl))
	return rsl, nil
}

func (sp *BookingServerProvider) processServer(strData string) (*Server, error) {
	strData = strings.Replace(strData, "ssr://", "", -1)
	strData = strings.Replace(strData, "_", "/", -1)
	strData = strings.Replace(strData, "-", "+", -1)
	data, err := base64.RawStdEncoding.DecodeString(strData)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(data), ":")
	port, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, nil
	}
	pw := fields[5][0:strings.Index(fields[5], "?")]
	data, _ = base64.StdEncoding.DecodeString(pw)
	return &Server{
		Server:       fields[0],
		ServerPort:   port,
		LocalAddress: "127.0.0.1",
		LocalPort:    1080,
		Timeout:      5000, //ms
		Workers:      5,
		Method:       fields[3],
		Plugin:       "",
		Password:     string(data),
	}, nil
}

func (sp *BookingServerProvider) checkServerPing(s *Server) bool {
	timeout := 1500 * time.Millisecond
	t1 := time.Now()
	_, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Server, s.ServerPort), timeout)
	if err != nil {
		return false
	}
	s.PingSpeed = time.Now().Sub(t1).Seconds() * 1000
	logrus.Infof("Server %s ping succeed = %s", s.Server, time.Now().Sub(t1))
	return true
}

func (sp *BookingServerProvider) checkServerGoogleWebsiteAccessible(s Server) bool {
	timeout := 1500 * time.Millisecond
	t1 := time.Now()
	_, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", s.Server, s.ServerPort), timeout)
	if err != nil {
		return false
	}
	logrus.Infof("Server %s ping succeed = %s", s.Server, time.Now().Sub(t1))
	return true
}
