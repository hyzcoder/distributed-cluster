package main
//
//import (
//	"context"
//	"errors"
//	"fmt"
//	clientv3 "go.etcd.io/etcd/client/v3"
//	"go.etcd.io/etcd/client/v3/concurrency"
//	"log"
//	"net"
//	"sync"
//	"time"
//)
//
//const CampaignPrefix = "/election-test-demo" // 这是选举的prefix
//
//func Campaign(c *clientv3.Client, parentCtx context.Context, wg *sync.WaitGroup) (success <-chan struct{}) {
//	// 我们设置etcd的value为当前机器的ip，这个不是关键
//	ip, _ := getLocalIP()
//	// 当外层的context关闭时，我们也会优雅的退出。
//	ctx, _ := context.WithCancel(parentCtx)
//	// ctx的作用是让外面通知我们要退出，wg的作用是我们通知外面已经完全退出了。当然外面要wg.Wait等待我们。
//	if wg != nil {
//		wg.Add(1)
//	}
//	// 创建一个信号channel，并返回，所有worker可以监听这个channel，这种实现可以让worker阻塞等待节点成为leader，而不是轮询是否是leader节点。
//	// 返回只读channel，所有worker可以阻塞在这。
//	notify := make(chan struct{}, 100)
//
//	go func() {
//		defer func() {
//			if wg != nil {
//				wg.Done()
//			}
//		}()
//		for {
//			select {
//			case <-ctx.Done(): // 如果是非leader节点，会阻塞在Campaign方法，context被cancel后，Campaign报错，最终会从这里退出。
//				return
//			default:
//			}
//
//			// 创建session，session参与选主，etcd的client需要自己传入。
//			// session中keepAlive机制会一直续租，如果keepAlive断掉，session.Done会收到退出信号。
//			s, err := concurrency.NewSession(c, concurrency.WithTTL(1))
//			if err != nil {
//				fmt.Println("NewSession", "error", "err", err)
//				//time.Sleep(time.Second * 2)
//				continue
//			}
//			log.Println("participate election")
//			// 创建一个新的etcd选举election
//			e := concurrency.NewElection(s, CampaignPrefix)
//			//调用Campaign方法，成为leader的节点会运行出来，非leader节点会阻塞在里面。
//			if err = e.Campaign(ctx, ip); err != nil {
//				fmt.Println("Campaign", "error", "err", err)
//				s.Close()
//				//time.Sleep(1 * time.Second) //不致于重试的频率太高
//				continue
//			}
//			// 运行到这的协程，成为leader，分布式下只有一个。
//			log.Println("elect", "success", "ip：", ip)
//			fmt.Printf("start become leader time:%v\n",time.Now())
//			time.Sleep(10*time.Second)
//			fmt.Printf("end leader time:%v\n",time.Now())
//			break
//			shouldBreak := false
//			for !shouldBreak {
//				select {
//				case notify <- struct{}{}: // 不断向所有worker协程发信号
//				case <-s.Done():  // 如果因为网络因素导致与etcd断开了keepAlive，这里break，重新创建session，重新选举
//					fmt.Println("campaign", "session has done")
//					shouldBreak = true
//					break
//				case <-ctx.Done():
//					ctxTmp, _ := context.WithTimeout(context.Background(), time.Second*1)
//					e.Resign(ctxTmp)
//					s.Close()
//					return
//				}
//			}
//		}
//	}()
//	return notify
//}
//
//// 获取本机网卡IP
//func getLocalIP() (ipv4 string, err error) {
//	var (
//		addrs   []net.Addr
//		addr    net.Addr
//		ipNet   *net.IPNet // IP地址
//		isIpNet bool
//	)
//	// 获取所有网卡
//	if addrs, err = net.InterfaceAddrs(); err != nil {
//		return
//	}
//	// 取第一个非lo的网卡IP
//	for _, addr = range addrs {
//		//fmt.Println(addr)
//		// 这个网络地址是IP地址: ipv4, ipv6
//		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
//			// 跳过IPV6
//			if ipNet.IP.To4() != nil {
//				ipv4 = ipNet.IP.String() // 192.168.1.1
//				return
//			}
//		}
//	}
//
//	err = errors.New("no local ip")
//	return
//}
//
//func main()  {
//	var wg sync.WaitGroup
//	config := clientv3.Config{
//		Endpoints: []string{"localhost:2379"},
//		DialTimeout: 5*time.Second,
//	}
//	client,err := clientv3.New(config)
//	if err != nil{
//		log.Println("clientv3.New is error!")
//		return
//	}
//	defer client.Close()
//
//	Campaign(client,context.TODO(),&wg)
//
//	wg.Wait()
//}









//package main
//
//import (
//	"context"
//	"flag"
//	"log"
//	"time"
//)
//
//func main() {
//	id := flag.Uint64("id", 1, "node id")
//	flag.Parse()
//	log.Printf("I'am node %v\n", *id)
//
//	cluster := map[uint64]string{
//		1: "http://127.0.0.1:22210",
//		2: "http://127.0.0.1:22220",
//		3: "http://127.0.0.1:22230",
//	}
//	n := newRaftNode(*id, cluster)
//
//	if *id == 1 {
//		time.Sleep(5 * time.Second)
//		for {
//			log.Printf("Propose on node %v\n", *id)
//			n.node.Propose(context.TODO(), []byte("hello"))
//			time.Sleep(time.Second)
//		}
//
//	}
//
//	select {}
//
//}