ifeq ($(KERNELRELEASE),)

KDIR := /usr/src/linux-source-4.15.0
PWD := $(shell pwd)

modules:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

modules_install:
	$(MAKE) -C $(KDIR) M=$(PWD) modules_install

clean:
	rm -rf *.o *.ko .depend *.mod.o *.mod.c Module.* modules.* 

.PHONY:modules modules_install clean

else

obj-m :=tun.o

endif
