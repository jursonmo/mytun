# mytun
tun 内核模块的 tun_rx_batched 和 napi 方式，应该可以 解决tun_fd write 占用CPU 太多的问题。

tun 多队列，应用层把某个数据流从应用层注入到内核时所使用的队列，应用层也希望这个数据流从这个队列里读到。但是 经过测试，发现有的内核不同版本没有达到这个效果，比如5.4.x.
1. 4.15.x 内核tun_flow_update没有问题，即从哪个tun_fd 发到内核的流，数据返回时，也从原来的tun_fd read，但是tun_fd write 占用CPU 太多(因为它包含了调用netif_rx_ni 消耗的cpu)，以至于tun write 性能过低。
   默认情况下，向内核注入数据走到流程是 tun_rx_batched 再走 tun_flow_update 逻辑，是没有问题的。
2. 5.4.x 有tun napi, tun_rx_batched 等方式解决tun_fd write 占用CPU 太多的问题，但是 5.4.x 的走 tun_rx_batched再走 tun_flow_update 有问题，即如果先调用了 tun_rx_batched再调用 tun_flow_update，会出现数据流返回tun_select_queue会先于tun_flow_update 执行，所以 往内核注入数据时，tun_flow_update应该要在 tun_rx_batched 之前执行，保证 tun_flow_update 中的学习逻辑优先

bootlin 上看 4.15.x 内核 和 5.4.x 内核在  tun_rx_batched 的区别：
```c
4.15.x内核:
static void tun_rx_batched(struct tun_struct *tun, struct tun_file *tfile,
			   struct sk_buff *skb, int more)
{
	struct sk_buff_head *queue = &tfile->sk.sk_write_queue;
	struct sk_buff_head process_queue;
	u32 rx_batched = tun->rx_batched;
	bool rcv = false;

	if (!rx_batched || (!more && skb_queue_empty(queue))) {
		local_bh_disable();
		netif_receive_skb(skb);
		local_bh_enable();
		return;
	}
....
}
```

```c
5.4.x内核:
static void tun_rx_batched(struct tun_struct *tun, struct tun_file *tfile,
			   struct sk_buff *skb, int more)
{
	struct sk_buff_head *queue = &tfile->sk.sk_write_queue;
	struct sk_buff_head process_queue;
	u32 rx_batched = tun->rx_batched;
	bool rcv = false;

	if (!rx_batched || (!more && skb_queue_empty(queue))) {
		local_bh_disable();
		skb_record_rx_queue(skb, tfile->queue_index);  // --------区别这里???-------------
		netif_receive_skb(skb);
		local_bh_enable();
		return;
	}
....
}

static inline void skb_record_rx_queue(struct sk_buff *skb, u16 rx_queue)
{
	skb->queue_mapping = rx_queue + 1;
}
static inline u16 skb_get_rx_queue(const struct sk_buff *skb)
{
	return skb->queue_mapping - 1;
}

static inline bool skb_rx_queue_recorded(const struct sk_buff *skb)
{
	return skb->queue_mapping != 0;
}
```
区别是 5.4.x内核 多了 skb_record_rx_queue ,  但是 skb_record_rx_queue 只是记录内核接受到数据的队列，按道理不影响内核收发数据的调度顺序。现在暂时也看不出来，我直接把tun_flow_update学习过程放在netif_receive_skb(skb) 之前就肯定没有问题。比如：https://github.com/jursonmo/mytun/blob/5b8113981efb10063e0095d44c6a292898a25be9/mnt/overlay/usr/src/linux-source-5.4.0/drivers/net/mytun/tun.c#L1972
--------------------------------------------------------------
#### 下面看下tun 模块接受应用层write 数据 的一些变化
1. linux 4.10.x 版本后，就增加了netif_receive_skb 处理：
```c
/* Get packet from user space buffer */
static ssize_t tun_get_user(struct tun_struct *tun, struct tun_file *tfile,
			    void *msg_control, struct iov_iter *from,
			    int noblock, bool more)
{
.........
#ifndef CONFIG_4KSTACKS
	local_bh_disable();
	netif_receive_skb(skb);  // 现在的做法
	local_bh_enable();
#else
	netif_rx_ni(skb); // 以前的做法
#endif
	stats = get_cpu_ptr(tun->pcpu_stats);
	u64_stats_update_begin(&stats->syncp);
	stats->rx_packets++;
	stats->rx_bytes += len;
	u64_stats_update_end(&stats->syncp);
	put_cpu_ptr(stats);

	tun_flow_update(tun, rxhash, tfile);
	return total_len;
}
```
2. linux 4.11.x 版本后，就换成了 tun_rx_batched  处理：
```c
/* Get packet from user space buffer */
static ssize_t tun_get_user(struct tun_struct *tun, struct tun_file *tfile,
			    void *msg_control, struct iov_iter *from,
			    int noblock, bool more)
{
..........
	rxhash = skb_get_hash(skb);
#ifndef CONFIG_4KSTACKS
	tun_rx_batched(tun, tfile, skb, more);
#else
	netif_rx_ni(skb);
#endif

	stats = get_cpu_ptr(tun->pcpu_stats);
	u64_stats_update_begin(&stats->syncp);
	stats->rx_packets++;
	stats->rx_bytes += len;
	u64_stats_update_end(&stats->syncp);
	put_cpu_ptr(stats);

	tun_flow_update(tun, rxhash, tfile);
	return total_len;
}
```

3. linux 4.15.x 开始增加了tun napi 功能 ，减少tun_fd write 占用CPU 太多的问题
https://kernelnewbies.org/Linux_4.15#:~:text=TUN%3A%20enable%20NAPI%20for%20TUN/TAP%20driver
```c
/* Get packet from user space buffer */
static ssize_t tun_get_user(struct tun_struct *tun, struct tun_file *tfile,
			    void *msg_control, struct iov_iter *from,
			    int noblock, bool more)
{
.........
 	else if (tfile->napi_enabled) {
		struct sk_buff_head *queue = &tfile->sk.sk_write_queue;
		int queue_len;

		spin_lock_bh(&queue->lock);
		__skb_queue_tail(queue, skb);
		queue_len = skb_queue_len(queue);
		spin_unlock(&queue->lock);

		if (!more || queue_len > NAPI_POLL_WEIGHT)
			napi_schedule(&tfile->napi);

		local_bh_enable();
	} else if (!IS_ENABLED(CONFIG_4KSTACKS)) {
		tun_rx_batched(tun, tfile, skb, more);   //============ 正常情况下走tun_rx_batched 这个逻辑=======
	} else {
		netif_rx_ni(skb);
	}

	stats = get_cpu_ptr(tun->pcpu_stats);
	u64_stats_update_begin(&stats->syncp);
	stats->rx_packets++;
	stats->rx_bytes += len;
	u64_stats_update_end(&stats->syncp);
	put_cpu_ptr(stats);

	tun_flow_update(tun, rxhash, tfile);
	return total_len;
}
```

tun_fd  tun_sendmsg() 方法，按道理，可以在大流量的情况下，减少tun_fd 往内核注入数据的系统调用的次数，同时 让 tun napi 的 一次处理多个数据，把数据在内核协议栈的处理用软中断处理，这样tun_fd  注入内核就没那么消耗CPU， 应用层就有更多cpu资源来处理业务逻辑。

但是golang 是用syscall 来调用senmsg, 不受netpoller 控制， 只有正常tun_fd_conn write 才由netpoller 控制。



#### 后来再测试：
后来测试，发现5.4.34 内核版本跟上面描述的问题一样，从tun queue1 写入数据，同一个连接的数据却从tun 其他queue 读出来
```
# uname -a
Linux 5.4.34.obc-hy #1 SMP Mon Jun 1 18:46:27 CST 2020 x86_64 x86_64 x86_64 GNU/Linux
```
我在虚拟机上测试， 却能保证从tun queue 写入的，就从那个tun queue 读出。即从哪个队列写入，就从哪个队列读出。
查看5.4.0 内核：
```
root@ubuntu:~# uname -a
Linux ubuntu 5.4.0-90-generic #101-Ubuntu SMP Fri Oct 15 20:00:55 UTC 2021 x86_64 x86_64 x86_64 GNU/Linux
```