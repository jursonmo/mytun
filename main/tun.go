package main

import (
	"fmt"
	"forfstack/lab11/go-tuntap/tuntap"
	"sync"

	//"forfstack/logs"
	logs "forfstack/mylog"
	"os/exec"
)

type TunConfig struct {
	Br      string `toml:"br"`
	TunName string `toml:"tunName"`
	TunType int    `toml:"tunType"`
	Ipstr   string `toml:"ipstr"`
	Mtu     int    `toml:"mtu"`
	DevUpSh string `toml:"devUpSh"`
}

type mytun struct {
	sync.Mutex
	TunConfig
	Index    int
	tund     *tuntap.Interface
	isClosed bool
}

func NewMytun(tc TunConfig) *mytun {
	return &mytun{
		TunConfig: tc,
	}
}

func (tun *mytun) isTap() bool {
	return tun.TunType == int(tuntap.DevTap)
}

func (tun *mytun) String() string {
	return fmt.Sprintf("name=%s,type=%d", tun.TunName, tun.TunType)
}

func (tun *mytun) IsClose() bool {
	return tun.isClosed
}

func (tun *mytun) Close() error {
	logs.Notice("close dev = %s ", tun.Name())
	tun.isClosed = true

	out, err := RunCmd("ip", "link", "delete", tun.Name())
	if err != nil {
		logs.Error("cmd run err: %s, out=%s, cmd＝ip link delete %s", err.Error(), out, tun.Name())
	}
	logs.Info("ip link delete over: %s", tun.Name()) //route will delelte
	return tun.tund.Close()
}

func (tun *mytun) Open() error {
	var err error
	logs.Info("ready create tunName :%s , *tunType=%d", tun.TunName, tun.TunType)

	tun.tund, err = tuntap.Open(tun.TunName, tuntap.DevKind(tun.TunType), false)
	if err != nil {
		logs.Error("tun/tap open err:%s, tunname = %s", err.Error(), tun.TunName)
		return err
	}
	return nil
}
func (tun *mytun) Config() error {
	tunname := tun.TunName
	ipstr := tun.Ipstr
	br := tun.Br
	mtu := tun.Mtu
	if mtu == 0 {
		mtu = 1500
	}
	logs.Notice("==========tun config: tunname=%s, tun.TunType=%d, br=%s, ipstr=%s==============", tunname, tun.TunType, br, ipstr)
	confs := fmt.Sprintf("ifconfig %s up\n", tunname)
	confs += fmt.Sprintf("ifconfig %s txqueuelen 10000\n", tunname)
	if br != "" { //must be tap
		if !tun.isTap() {
			logs.Error("br=%s can't not addif tuntype=%d", tun.String(), tun.TunType)
			panic("")
		}
		confs += fmt.Sprintf("brctl addbr %s\n", br)
		confs += fmt.Sprintf("brctl addif %s %s\n", br, tunname)
		if ipstr != "" {
			// confs += fmt.Sprintf("ifconfig %s %s\n", br, ipstr) // openwrt exec failure, in addition to, we need't config this in D100, because it config default by system or dhcp.
			confs += fmt.Sprintf("ip addr add %s dev %s\n", ipstr, br)
		}
	} else { // maybe tun or tap
		if ipstr != "" {
			//confs += fmt.Sprintf("ip addr add %s dev %s\n", ipstr, tunname) // this cmd can assign E Class IP Address
			confs += fmt.Sprintf("ifconfig %s %s\n", tunname, ipstr)
		}
	}
	confs += fmt.Sprintf("ifconfig %s mtu %d\n", tunname, mtu)
	out, err := exec.Command("sh", "-c", confs).CombinedOutput()
	if err != nil {
		logs.Error("open err:%s,out=%s", err.Error(), string(out))
		return err
	}
	logs.Info("tun dev:%s open successfully", tun.tund.Name())
	b, _ := exec.Command("ls", "/sys/class/net/mytun/queues/").CombinedOutput()
	fmt.Print("----------- queue info: %s\n", string(b))
	return nil
}

func (tun *mytun) Name() string {
	return tun.tund.Name()
}

func (tun *mytun) Read(buf []byte) (n int, err error) {
	// var inpkt *tuntap.Packet
	// inpkt, err = tun.tund.ReadPacket2(buf[0:])
	// if err != nil {
	// 	logs.Error("%s ReadPacket error:%s", tun.Name(), err.Error())
	// 	return
	// }
	// n = len(inpkt.Packet)
	//return

	return tun.tund.ReadPacket3(buf[0:])
}
