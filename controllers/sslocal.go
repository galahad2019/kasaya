package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/galahad2019/kasaya/providers"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	configPath = "/tmp/ss.json"
)

type SSLocalProxyController struct {
	ch           chan int
	ss           []*providers.Server
	lockObj      *sync.RWMutex
	provider     providers.ServerProvider
	fileLocation string
}

func NewSSLocalProxyController(fileLocation string) *SSLocalProxyController {
	return &SSLocalProxyController{fileLocation: fileLocation}
}

func (c *SSLocalProxyController) Initialize(provider providers.ServerProvider) {
	c.provider = provider
	c.ch = make(chan int)
	c.lockObj = &sync.RWMutex{}
	sl, err := c.provider.GetServerList()
	if err != nil {
		logrus.Errorf("Failed to list available server, error: %s", err.Error())
	}
	c.ss = sl
	go c.periodicallyUpdate()
}

func (c *SSLocalProxyController) Run() {
	var err error
	for {
		server := c.selectServer()
		if server == nil {
			logrus.Warn("No any available servers can be select...")
			time.Sleep(time.Second * 10)
			continue
		}
		err = c.setupLocalProxy(server)
		if err != nil {
			logrus.Errorf("Failed to setup server %s, error: %s", server.Server, err.Error())
			c.lockObj.Lock()
			c.ss = c.ss[1:]
			c.lockObj.Unlock()
			continue
		}
		<-c.ch
	}
}

func (c *SSLocalProxyController) selectServer() *providers.Server {
	c.lockObj.RLock()
	defer c.lockObj.RUnlock()
	if c.ss == nil || len(c.ss) == 0 {
		return nil
	}
	logrus.Warnf("Selected server: %s", c.ss[0].Server)
	return c.ss[0]
}

func (c *SSLocalProxyController) setupLocalProxy(s *providers.Server) error {
	_ = os.RemoveAll(configPath)
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configPath, data, 0644)
	if err != nil {
		logrus.Errorf("Failed to write ss configuration file, error: %s", err.Error())
		return err
	}
	cmd := exec.Command(c.fileLocation, "-c", configPath, "-u")
	err = cmd.Start()
	if err != nil {
		logrus.Errorf("Failed to run proxy, error: %s", err.Error())
		return err
	}
	logrus.Infof("Successfully run proxy up with server: %s", s.Server)
	tempCh := make(chan int)
	go c.checkGoogleAccessible(tempCh, s, cmd)
	err = cmd.Wait()
	close(tempCh)
	return err
}

func (c *SSLocalProxyController) periodicallyUpdate() {
	for {
		time.Sleep(time.Minute * 1)
		sl, err := c.provider.GetServerList()
		if err != nil {
			logrus.Errorf("Failed to list available server, error: %s", err.Error())
			continue
		}
		c.lockObj.Lock()
		c.ss = sl
		c.lockObj.Unlock()
		logrus.Warn("Server list has been updated successfully!")
	}
}

func (c *SSLocalProxyController) checkGoogleAccessible(ch chan int, s *providers.Server, cmd *exec.Cmd) {
	time.Sleep(time.Second * 5)
	for {
		select {
		case <-ch:
			return
		default:
			proxyAddr := fmt.Sprintf("%s:%d", s.LocalAddress, s.LocalPort)
			url := "https://www.google.com"
			// create a socks5 dialer
			dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
			if err != nil {
				logrus.Errorf("Failed to connect to local proxy, err: %s", err.Error())
				_ = cmd.Process.Kill()
				return
			}
			// setup a http client
			httpTransport := &http.Transport{}
			httpClient := &http.Client{
				Transport: httpTransport,
				Timeout:   time.Second * 10,
			}
			// set our socks5 as the dialer
			httpTransport.Dial = dialer.Dial
			// create a request
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				logrus.Errorf("Failed to create SOCK5 web request, err: %s", err.Error())
				_ = cmd.Process.Kill()
				return
			}
			// use the http client to fetch the page
			resp, err := httpClient.Do(req)
			if err != nil {
				logrus.Errorf("Failed to invite destination website, err: %s", err.Error())
				_ = cmd.Process.Kill()
				return
			}
			defer resp.Body.Close()
			_, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("Failed to read destination website stream, err: %s", err.Error())
				_ = cmd.Process.Kill()
				return
			}
			logrus.Infof("server %s check OK!", s.Server)
		}
		time.Sleep(time.Minute * 1)
	}
}
