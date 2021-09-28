/*
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	//"forfstack/logs"
	"forfstack/lab11/go-tuntap/tuntap"
	logs "forfstack/mylog"
	"net"
	"time"
)

var (
	client = flag.Bool("c", true, "client mode")
	tunip  = flag.String("tunip", "192.168.2.3", "tun dev ip")
	addr   = flag.String("addr", "192.168.1.2:80", "connect ip")
)

func main() {
	tc := TunConfig{
		TunName: "mytun",
		TunType: 1,
		Ipstr:   *tunip,
	}
	tun := NewMytun(tc)
	err := tun.Open()
	if err != nil {
		logs.Error("%v", err)
		return
	}
	err = tun.Config()
	if err != nil {
		logs.Error("%v", err)
		return
	}
	var conn net.Conn
	if *client {
		for {
			conn, err = net.Dial("tcp", *addr)
			if err != nil {
				fmt.Printf("connect failed, err : %v\n", err.Error())
				time.Sleep(time.Second)
				continue
			}
			defer conn.Close()
			logs.Info("conn:%s-->%s", conn.LocalAddr().String(), conn.RemoteAddr().String())
			go forwardToTun(conn, tun)
			go tunToConn(conn, tun)
			break
		}
		time.Sleep(time.Hour)
	}
	// server
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Panicln(err)
	}
	conn, err = ln.Accept()
	if err != nil {
		log.Panicln(err)
	}
	defer conn.Close()
	logs.Info("Accept, conn:%s-->%s", conn.LocalAddr().String(), conn.RemoteAddr().String())
	go forwardToTun(conn, tun)
	go tunToConn(conn, tun)
	time.Sleep(time.Hour)
}

func forwardToTun(conn net.Conn, tun *mytun) {
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			os.Exit(-1)
		}
		logs.Info("conn forward to tun, n: %d", n)
		inpkt := &tuntap.Packet{Packet: buf[:n]}
		err = tun.tund.WritePacket(inpkt)
		if err != nil {
			log.Panicln(err)
		}
	}
}

func tunToConn(conn net.Conn, tun *mytun) {
	var err error
	buf := make([]byte, 2048)
	rn, wn := 0, 0
	for {
		rn, err = tun.Read(buf)
		if err != nil {
			logs.Error("%v", err)
			return
		}
		//send by tcp
		if rn <= 0 {
			continue
		}
		wn, err = conn.Write(buf[:rn])
		if err != nil {
			logs.Error("%v", err)
			log.Panicln(err)
			return
		}
		logs.Info("tun %d-->conn %d", rn, wn)
	}
}
