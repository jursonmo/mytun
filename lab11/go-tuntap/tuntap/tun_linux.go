package tuntap

/*
#include <stdio.h>
#include <linux/bpf.h>
#include <bpf/libbpf.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <linux/if.h>
#include <linux/if_tun.h>

#include <errno.h>

int loadprog(int tun_fd) {
	 int err, prog_fd ;

    struct bpf_object *obj;
	memset(&obj, 0, sizeof(obj));

	err = bpf_prog_load("tunsteering.o", BPF_PROG_TYPE_SOCKET_FILTER, &obj, &prog_fd);
	if (err) {
		fprintf(stderr, "bpf_prog_load() failed\n");
		return err;
	}
	err = ioctl(tun_fd, TUNSETSTEERINGEBPF, (void *)&prog_fd);
    if (err) {
		fprintf(stderr, "ioctl(...TUNSETSTEERINGEBPF...) failed: %s\n", strerror(errno));
		return err;
	}
	return 0;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

/*
func openDevice(ifPattern string) (*os.File, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	return file, err
}


func createInterface(file *os.File, ifPattern string, kind DevKind, meta bool) (string, error) {
	var req ifReq
	//req.Flags = iffOneQueue
	req.Flags = 0
	if len(ifPattern) > 15 {
		return "", errors.New("tun/tap name too long")
	}
	copy(req.Name[:15], ifPattern)
	switch kind {
	case DevTun:
		req.Flags |= iffTun
	case DevTap:
		req.Flags |= iffTap
	default:
		panic("Unknown interface type")
	}
	if !meta {
		req.Flags |= iffnopi
	}
	//file.Fd() remove fd from netpoll
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if err != 0 {
		return "", err
	}
	return string(req.Name[:len(ifPattern)]), nil
}
*/
//for err:tun not pollable
func openDevice(ifPattern string) (*os.File, error) {
	return nil, nil
}

func createInterface(ifPattern string, kind DevKind, meta bool) (*os.File, string, error) {
	var req ifReq
	//req.Flags = iffOneQueue
	//req.Flags = 0
	/*about  IFF_MULTI_QUEUE,
	// 	#define IFF_ONE_QUEUE	0x2000
	// #define IFF_MULTI_QUEUE 0x0100
	tun_set_iff():
	int queues = ifr->ifr_flags & IFF_MULTI_QUEUE ?
				 MAX_TAP_QUEUES : 1;

	dev = alloc_netdev_mqs(sizeof(struct tun_struct), name,
		NET_NAME_UNKNOWN, tun_setup, queues,
		queues);
	*/
	req.Flags = ifMulti

	if len(ifPattern) > 15 {
		return nil, "", errors.New("tun/tap name too long")
	}
	copy(req.Name[:15], ifPattern)
	tunfd, err := unix.Open("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		log.Fatal(err)
	}
	C.loadprog(C.int(tunfd))
	/*
		var ifr [unix.IFNAMSIZ + 64]byte
		copy(ifr[:], req.Name[:])
		switch kind {
		case DevTun:
			*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = unix.IFF_TUN
		case DevTap:
			*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = unix.IFF_TAP
		default:
			panic("Unknown interface type")
		}
		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(tunfd),
			uintptr(unix.TUNSETIFF),
			uintptr(unsafe.Pointer(&ifr[0])),
		)

		if errno != 0 {
			log.Fatal(errno)
		}
	*/
	switch kind {
	case DevTun:
		req.Flags |= iffTun
	case DevTap:
		req.Flags |= iffTap
	default:
		panic("Unknown interface type")
	}
	if !meta {
		req.Flags |= iffnopi
	}
	/*
		如果IFF_NO_PI标志没有被设置，每一帧格式如下：
		Flags [2 bytes]
		Proto [2 bytes]
		Raw protocol(IP, IPv6, etc) frame.

		// Protocol info prepended to the packets (when IFF_NO_PI is not set)
		#define TUN_PKT_STRIP	0x0001  //#include <linux/if_tun.h>// include/uapi/linux/if_tun.h
		struct tun_pi {
			__u16  flags;
			__be16 proto;
		};
	*/
	//file.Fd() remove fd from netpoll
	_, _, syserr := syscall.Syscall(syscall.SYS_IOCTL, uintptr(tunfd), uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if syserr != 0 {
		return nil, "", errors.New("TUNSETIFF error")
	}
	b, _ := exec.Command("ls", "/sys/class/net/mytun/queues/").CombinedOutput()
	fmt.Print("----------- queue info: %s\n", string(b))
	unix.SetNonblock(tunfd, true)

	file := os.NewFile(uintptr(tunfd), "/dev/net/tun")
	return file, string(req.Name[:len(ifPattern)]), nil
}
