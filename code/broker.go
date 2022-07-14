package code

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Broker struct {
	stop chan bool
	topics map[string]*Channel //消息队列
	clusterMaxMin ClusterMaxMinIndicators
	sync.RWMutex
}

// ClusterMaxMinIndicators 集群性能指标最大最小值
type ClusterMaxMinIndicators struct {
	MaxIndicators [3]float64
	MinIndicators [3]float64
}

// Channel 频道类型
type Channel struct {
	subscribers []ClientID
	sync.RWMutex
}

// NodeState 记录结点状态
type NodeState struct {
	PingFlag bool
	PingTime float64
	PingIP string
}


func NewClusterMaxMinIndicators() ClusterMaxMinIndicators {
	cmmi := ClusterMaxMinIndicators{}
	cmmi.MinIndicators[0] = GetLocalCpuFrequency()
	cmmi.MinIndicators[1],_ = GetLocalMemoryInfo()
	cmmi.MinIndicators[2],_ = GetLocalDiskInfo()
	cmmi.MaxIndicators[0] = cmmi.MinIndicators[0]
	cmmi.MaxIndicators[1] = cmmi.MinIndicators[1]
	cmmi.MaxIndicators[2] = cmmi.MinIndicators[2]
	return cmmi
}


func NewBroker() *Broker{
	return &Broker{
		topics: make(map[string]*Channel),
		stop: make(chan bool),
		clusterMaxMin: NewClusterMaxMinIndicators(),
	}
}


// NewChannel 新建频道
func NewChannel() *Channel  {
	return &Channel{
		subscribers: make([]ClientID,0),

	}
}

//// ListenPort 监听指定端口，获取连接
//func (b *Broker) ListenPort() {
//	//代理端监听端口统一为本机的12345端口
//	listen, err := net.Listen("tcp",":12345")
//	if err != nil{
//		fmt.Println("ListenPort: Broker Listen is failed!")
//		return
//	}
//	log.Printf("Broker（%s:12345）: start port listening",GetLocalIP())
//
//	for{
//		conn, err1 := listen.Accept()
//		if err1 != nil{
//			fmt.Println("ListenPort: Broker Accept is failed!")
//			continue
//		}
//		go b.handleLeaderMsg(conn)
//	}
//}

//记录集群中各个指标的最大最小值
func (b *Broker) updateClusterMaxMin(msg Msg)  {
	log.Println("update the maximum and minimum index values in the cluster")
	pi := PerformanceIndicators{}
	err := json.Unmarshal(msg.MsgContent,&pi)
	if err != nil {
		log.Printf("updateClusterMaxMin: %v",err)
		return
	}

	if pi.CpuFrequency > b.clusterMaxMin.MaxIndicators[0]{
		b.clusterMaxMin.MaxIndicators[0] = pi.CpuFrequency
	}else if pi.CpuFrequency<b.clusterMaxMin.MinIndicators[0]{
		b.clusterMaxMin.MinIndicators[0] = pi.CpuFrequency
	}
	if pi.MemCapacity > b.clusterMaxMin.MaxIndicators[1]{
		b.clusterMaxMin.MaxIndicators[1] = pi.MemCapacity
	}else if pi.MemCapacity < b.clusterMaxMin.MinIndicators[1]{
		b.clusterMaxMin.MinIndicators[1] = pi.MemCapacity
	}
	if pi.DiskCapacity > b.clusterMaxMin.MaxIndicators[2]{
		b.clusterMaxMin.MaxIndicators[2] = pi.DiskCapacity
	}else if pi.DiskCapacity < b.clusterMaxMin.MinIndicators[2]{
		b.clusterMaxMin.MinIndicators[2] = pi.DiskCapacity
	}


	jsonBytes,_ :=json.Marshal(b.clusterMaxMin)
	msg.MsgContent = jsonBytes
	b.BroadCast(msg)

}

//获取所有从节点
func (b *Broker) searchFollowersInfo(cid ClientID,msg Msg)  {
	b.RLock()
	channel,found := b.topics["nodeState"]
	b.RUnlock()
	if found{
		channel.RLock()

		jsonFollowers,err := json.Marshal(channel.subscribers)
		if err != nil{
			fmt.Printf("searchFollowersInfo: json.Marshal is failed!\n")
			return
		}

		msg.MsgContent = jsonFollowers
		TransmitToNode(cid.Ip+cid.Port,msg)
		channel.RUnlock()
	}
}

// 处理接收到的订阅类型消息
func (b *Broker) handleSubscribe(cid ClientID,msg Msg)  {
	b.RLock()
	channel,found := b.topics[msg.Topic]
	b.RUnlock()
	if found{
		//log.Printf("Broker: 在消息队列中找到%s频道，将客户端添加到该频道消费组中",cid.msg.Topic)
		flag := true
		channel.Lock()
		for _,subscriber := range channel.subscribers{
			if subscriber.Ip == cid.Ip && subscriber.Port == cid.Port{
				flag = false
				break
			}
		}
		if flag == true{
			channel.subscribers = append(channel.subscribers,cid)
		}
		channel.Unlock()
	}else{
		//log.Printf("Broker: 未能在消息队列找到%s频道，创建该频道，将客户端添加到该频道消费组中",cid.msg.Topic)
		ch := NewChannel()
		ch.subscribers = append(ch.subscribers,cid)
		b.Lock()
		b.topics[msg.Topic] = ch
		b.Unlock()

	}

	b.RLock()
	count := len(b.topics[msg.Topic].subscribers)
	b.RUnlock()
	result := fmt.Sprintf("成功订阅%s频道，目前订阅者数：%d",msg.Topic,count)
	msg.MsgContent = []byte(result)
	TransmitToNode(cid.Ip+cid.Port,msg)
}

//处理接收到的解订阅类型消息
func (b *Broker) handleUnsubscribe(cid ClientID,msg Msg)  {
	b.RLock()
	channel,found := b.topics[msg.Topic]
	b.RUnlock()
	if found {
		b.Lock()

		ch := NewChannel()
		for _,subscriber := range channel.subscribers{
			if subscriber.Ip == cid.Ip && subscriber.Port == cid.Port{
				continue
			}
			ch.subscribers = append(ch.subscribers,subscriber)
		}
		b.topics[msg.Topic] = ch
		count := len(b.topics[msg.Topic].subscribers)

		b.Unlock()

		result := fmt.Sprintf("成功解订阅%s频道，目前订阅者数：%d",msg.Topic,count)
		msg.MsgContent = []byte(result)
		TransmitToNode(cid.Ip+cid.Port,msg)
	}else{
		result := fmt.Sprintf("消息队列中无%s频道",msg.Topic)
		msg.MsgContent = []byte(result)
		TransmitToNode(cid.Ip+cid.Port,msg)
	}

}

// BroadCast 将消息广播给指定频道的消费组
func (b *Broker) BroadCast(msg Msg)  {
	b.RLock()
	channel, found := b.topics[msg.Topic]
	b.RUnlock()

	if found {
		//log.Printf("Broker: 在消息队列中找到%s频道，在该频道广播消息",c.msg.Topic)
		channel.RLock()
		defer channel.RUnlock()

		//返回publish操作结果
		//backResult := fmt.Sprintf("发布的%s频道中订阅者数: %d",msg.Topic,len(channel.subscribers))
		//TransmitToNode(c,backResult)

		//msg.MsgType = MESSAGE
		//将publish消息广播指定消费组
		for _,subscriber := range channel.subscribers{
				go TransmitToNode(subscriber.Ip+subscriber.Port,msg)
		}
	}
	//else{
	//	//backResult := fmt.Sprintf("未能在消息队列中找到%s频道",c.msg.Topic)
	//	backResult := fmt.Sprintf("发布的%s频道中订阅者数: 0",msg.Topic)
	//	msg.MsgContent = []byte(backResult)
	//	TransmitToNode(c.ip+c.port,msg)
	//}
}


// TimedPubNetMsg 设置定时任务，每5秒将集群中网络信息发布一次
func (b *Broker) TimedPubNetMsg()  {
	//timedTask := cron.New()
	timedTask := cron.New(cron.WithSeconds())
	spec := "*/5 * * * * ?"
	_, err1 := timedTask.AddFunc(spec,b.PubNetMsg)
	if err1 != nil {
		fmt.Printf("TimedPubNetMsg: %v\n",err1)
		return
	}
	timedTask.Start()
	defer timedTask.Stop()
	select {
	case <- b.stop:
		return
	}//查询语句，保持程序运行，在这里等同于for{}
}

// PubNetMsg 发布网络信息,发布频道默认为nodeState
func (b *Broker) PubNetMsg()  {
	//log.Println("-------------------周期性发布结点信息------------------------")
	var nodes []NodeState       //存放每个结点的网络信息
	b.RLock()
	channel, found := b.topics["nodeState"]
	b.RUnlock()
	if found {
		nodes = pingNetNodes(channel)
	}

	log.Printf("cluster network condition messsage")
	log.Println(nodes)

	jsonNodes,err := json.Marshal(nodes)
	if err != nil{
		fmt.Printf("PubNetMsg: 结点网络信息转成[]byte失败: %s\n",err.Error())
		return
	}

	msg := Msg{}
	msg.MsgType = PUBLISH
	msg.Topic = "nodeState"
	msg.MsgContent = jsonNodes
	b.BroadCast(msg)
}

//去ping集群中的所有结点，返回网络信息
func pingNetNodes(channel *Channel) []NodeState {
	var ips []string
	var nodes []NodeState

	channel.RLock()

	for _, client := range channel.subscribers{
		ips = append(ips,client.Ip)
	}
	channel.RUnlock()

	//自定义的ip地址
	//ips=append(ips,"192.168.112.130")
	//ips=append(ips,"www.baidu.com")

	var rw sync.RWMutex
	wg := sync.WaitGroup{}     //同步协程
	wg.Add(len(ips))
	for i := 0; i < len(ips); i++ {
		go pingNode(ips[i], &wg,&rw, &nodes)
	}
	wg.Wait()
	return nodes
}

//ping结点
func pingNode(addr string,wg *sync.WaitGroup,rw *sync.RWMutex,nodes *[]NodeState) {

	ctx,cancel := context.WithTimeout(context.Background(),time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,"ping","-c","1",addr)
	var out bytes.Buffer
	cmd.Stdout = &out //把执行命令的标准输出定向到out
	cmd.Stderr = &out //把命令的错误输出定向到out

	//启动一个子进程执行命令,阻塞到子进程结束退出
	err := cmd.Run()
	if err != nil {
		//未能ping通，标志置为false
		node := NewNodeState(false,-1,addr)

		rw.Lock()
		*nodes = append(*nodes,node)
		rw.Unlock()
		//log.Printf("IP: %s ping失败  错误原因: %s", addr,err.Error())
	}else{
		//截取ping返回结果的有用信息
		index1 := strings.Index(out.String(),"\n")
		str1 := out.String()[index1+1:]
		index2 := strings.Index(str1,"\n")
		str2 := str1[:index2]

		//利用正则表达获取信息
		rex := regexp.MustCompile("[0-9|.]+")
		pingMsg := rex.FindAllString(str2,-1)
		//fmt.Println(pingMsg[len(pingMsg)-1])

		//获取ping时间
		pingTime,err1 :=strconv.ParseFloat(pingMsg[len(pingMsg)-1],64)
		if err1 != nil{
			fmt.Println("pingNode: ParseFloat is failed!")
			return
		}


		//ping通，标志置为true
		node := NewNodeState(true,pingTime,addr)
		rw.Lock()
		*nodes = append(*nodes,node)
		rw.Unlock()
	}

	wg.Done()

}
