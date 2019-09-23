package main

import (
	"flag"
	"github.com/galahad2019/kasaya/controllers"
	"github.com/galahad2019/kasaya/providers"
	"github.com/sirupsen/logrus"
)

func main() {
	var p string
	var ba string
	flag.StringVar(&ba, "ba", "", "HTTP booking address given by your SS SP.")
	flag.StringVar(&p, "p", "/usr/local/opt/shadowsocks-libev/bin/ss-local", "The file location of ss-local")
	flag.Parse()
	if ba == "" {
		logrus.Fatalf("\"ba\" argument is needed for initializing project Kasaya!")
		return
	}
	c := controllers.NewSSLocalProxyController(p)
	c.Initialize(providers.NewBookingServerProvider(ba))
	c.Run()
	select {}
}
