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
	client   = flag.Bool("c", false, "client mode")
	queueNum = flag.Int("n", 4, "queueNum")
	tunip    = flag.String("tunip", "192.168.2.3", "tun dev ip")
	stunip   = flag.String("stunip", "192.168.2.1", "server tun dev ip")
	addr     = flag.String("addr", "192.168.1.2:8080", "connect ip")
)

func main() {
	flag.Parse()
	tc := TunConfig{
		TunName: "mytun",
		TunType: 1,
		Ipstr:   *tunip,
	}

	var conn net.Conn
	if *client {
		fmt.Printf("client mode: queueNum=%d\n", *queueNum)
		for i := 0; i < *queueNum; i++ {
			var err error
			fmt.Printf("dial addr:%s\n", *addr)
			conn, err = net.Dial("tcp", *addr)
			if err != nil {
				fmt.Printf("connect failed, err : %v\n", err.Error())
				time.Sleep(time.Second)
				continue
			}
			tun := NewMytun(tc)
			tun.Index = i
			err = tun.Open()
			if err != nil {
				logs.Error("%v", err)
				return
			}
			err = tun.Config()
			if err != nil {
				logs.Error("%v", err)
				return
			}

			defer conn.Close()
			logs.Info("i:%d, conn:%s-->%s", i, conn.LocalAddr().String(), conn.RemoteAddr().String())
			go forwardToTun(conn, tun)
			go tunToConn(conn, tun)
			time.Sleep(time.Second * 2)
		}
		time.Sleep(time.Hour)
	}
	// server
	index := 0
	tc.Ipstr = *stunip
	tc.TunName = tc.TunName + "_s"
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Panicln(err)
	}
	for {
		logs.Info("server listening %s .....", *addr)
		conn, err = ln.Accept()
		if err != nil {
			log.Panicln(err)
		}
		tun := NewMytun(tc)
		tun.Index = index
		err = tun.Open()
		if err != nil {
			log.Panicln(err)
			return
		}
		err = tun.Config()
		if err != nil {
			logs.Error("%v", err)
			return
		}

		defer conn.Close()
		logs.Info("Accept, index:%d, conn:%s-->%s", index, conn.LocalAddr().String(), conn.RemoteAddr().String())
		go forwardToTun(conn, tun)
		go tunToConn(conn, tun)
		index++
	}
	time.Sleep(time.Hour)
}

func forwardToTun(conn net.Conn, tun *mytun) {
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			os.Exit(-1)
		}
		logs.Info("conn forward to tun:%d, n: %d", tun.Index, n)
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
		logs.Info("tun forward to conn, tun index:%d,  read %d-->conn %d", tun.Index, rn, wn)
	}
}
