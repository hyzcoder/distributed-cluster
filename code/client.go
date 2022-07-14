package code

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"io"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ClientID 节点ID信息
type ClientID struct {
	Ip string
	Port string
	//ID string
}

//节点身份信息
const (
	Leader string = "leader"
	Follower string = "follower"
)

// Client 节点类型
type Client struct {
	clientId ClientID
	broker *Broker
	leader string
	identity string
	indicatorsUtilization PerformanceIndicatorsUtilization
	pastDownCounts int64
	sync.RWMutex
	testApply chan int
}

// CandidateInfo 候选节点信息
type CandidateInfo struct{
	CandidateID string
	Score float64
	DownCounts int64
}

// PerformanceIndicatorsUtilization 机器性能指标使用率
type PerformanceIndicatorsUtilization struct {
	CpuUtilization float64
	MemUtilization float64
	DiskUtilization float64
}



// PerformanceIndicators 机器性能指标
type PerformanceIndicators struct {
	CpuFrequency float64
	MemCapacity float64
	DiskCapacity float64
}

// NewClient 新建客户端
func NewClient() *Client{
	client := &Client{
		identity: Follower,
		testApply: make(chan int),
	}
	return client
}

// NewNodeState 新建结点状态结构体
func NewNodeState(flag bool,time float64,ip string) NodeState  {
	return NodeState{
		PingFlag: flag,
		PingTime: time,
		PingIP: ip,
	}
}

// SetMasterAddr 设置新的主库地址
func (c *Client) SetMasterAddr(masterIpPort string)  {
	 c.leader = masterIpPort
}

// SetMasterIdentity 设置节点为主节点还是从节点
func (c *Client) SetMasterIdentity(isMaster bool)  {
	c.Lock()
	defer c.Unlock()
	if isMaster{
		c.identity = Leader
	}else{
		c.identity = Follower
	}
}

// SetListenPort 设置监听端口
func (c *Client) SetListenPort(port string)  {
	c.clientId.Port = port
}

func (c *Client) Identity() string {
	c.RLock()
	defer c.RUnlock()
	return c.identity
}

// Run 运行节点的任务
func (c *Client) Run()  {

	c.clientId.Ip = GetLocalIP()

	go c.ListenPort()
	time.Sleep(10*time.Millisecond)

	if c.Identity() == Leader{
		log.Printf("Master（%v）: start port listening",c.clientId.Ip+c.clientId.Port)
		log.Printf("Run master node task")
		c.broker = NewBroker()
		go c.broker.TimedPubNetMsg()

	}else if c.Identity() == Follower{
		log.Printf("Follower（%v）: start port listening",c.clientId.Ip+c.clientId.Port)
		//获取过去宕机次数
		c.pastDownCounts = readPastDownCount()
		savePastDownCount(1+c.pastDownCounts)

		err :=c.Subscribe("nodeState")
		if err != nil{
			log.Println(err)
			return
		}
	}
}

// Subscribe 结点订阅
func (c *Client) Subscribe(topic string) error {
	if topic == ""{
		return fmt.Errorf("Subscribe: topic can not be empty ")
	}

	//log.Printf("Client: 正在订阅 %s 的 %s频道",Host,topic)
	conn,err := net.Dial("tcp",c.leader)
	if err!=nil{
		return fmt.Errorf("Subscribe Error:%s ",err.Error())
	}
	defer conn.Close()

	msg :=Msg{}
	msg.Topic = topic
	msg.MsgType = SUBSCRIBE
	msg.MsgContent = []byte{}
	msg.Port = c.clientId.Port
	//addr := conn.LocalAddr().String()
	//msg.Addr = strings.Split(addr,":")[0] + c.addr

	//将Msg类型转为[]byte
	jsonMsg, err1 := json.Marshal(msg)
	if err1 != nil {
		return fmt.Errorf("Subscribe Error:%s ",err1.Error())
	}

	_ , err1 = conn.Write(jsonMsg)
	if err1 != nil {
		return fmt.Errorf("Subscribe Error:%s ",err1.Error())
	}

	return nil
}

// UnSubscribe 结点取消订阅
func (c *Client) UnSubscribe(topic string) error {
	if topic == ""{
		return fmt.Errorf("UnSubscribe: topic can not be empty ")
	}

	//log.Printf("Client: 正在解订阅 %s 的 %s频道",Host,topic)
	conn,err := net.Dial("tcp",c.leader)
	if err!=nil{
		return fmt.Errorf("UnSubscribe Error:%s ",err.Error())
	}
	defer conn.Close()

	msg :=Msg{}
	msg.Topic = topic
	msg.MsgType = UNSUBSCRIBE
	msg.MsgContent = []byte{}
	msg.Port = c.clientId.Port
	//addr := conn.LocalAddr().String()
	//msg.Addr = strings.Split(addr,":")[0] + c.addr

	jsonMsg, err1 := json.Marshal(msg)
	if err1 != nil {
		return fmt.Errorf("UnSubscribe Error:%s ",err1.Error())
	}
	_ , err1 = conn.Write(jsonMsg)
	if err1 != nil {
		return fmt.Errorf("UnSubscribe Error:%s ",err1.Error())
	}

	return nil
}

// Publish 结点发布消息
func (c *Client) Publish(topic string,msgContent []byte) error {
	if topic == ""{
		return fmt.Errorf("Publish: topic can not be empty ")
	}
	//log.Printf("Client: 正在发布消息到%s频道",topic)
	conn,err := net.Dial("tcp",c.leader)
	if err!=nil{
		return fmt.Errorf("Publish Error:%s ",err.Error())
	}
	defer conn.Close()

	msg :=Msg{}
	msg.Topic=topic
	msg.MsgType = PUBLISH
	msg.MsgContent=msgContent
	//msg.Port = f.client.Port


	jsonMsg,err1 := json.Marshal(msg)
	if err1 != nil{
		return fmt.Errorf("Publish Error:%s ",err1.Error())
	}

	_ , err2 := conn.Write(jsonMsg)
	if err2 != nil {
		return fmt.Errorf("Publish Error:%s ",err2.Error())
	}

	return nil
}

// ListenPort 客户端来监听指定端口，接收消息
func (c *Client) ListenPort() {
	//客户端监听端口
	listen, err := net.Listen("tcp",c.clientId.Port)
	if err != nil{
		log.Printf("ListenPort: %v!",err)
		return
	}

	for{
		conn, err2 := listen.Accept()
		if err2 != nil{
			log.Println("ListenPort: Accept is failed!")
			continue
		}
		if c.Identity() == Leader{
			go c.handleLeaderMsg(conn)
		}else if c.Identity() == Follower {
			go c.handleFollowerMsg(conn)
		}
	}

}

//处理连接后接收的数据
func (c *Client) handleLeaderMsg(conn net.Conn)  {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	//node1 := make([]code.NodeState,0)
	//err = json.Unmarshal(json1,&node1)
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(reader)
	if err == io.EOF {
		fmt.Printf("handleLeaderMsg: %v\n",err)
		return
	}

	data := buf.Bytes()
	if string(data) == ""{
		return
	}

	msg := Msg{}
	err = json.Unmarshal(data,&msg)
	if err != nil{
		fmt.Printf("handleLeaderMsg: %v\n",err)
		return
	}

	//存上发来请求的客户端IP+Port
	clientId := ClientID{}
	addr := conn.RemoteAddr().String()
	clientId.Ip = strings.Split(addr,":")[0]
	clientId.Port = msg.Port
	//client.addr = msg.Addr
	//addr := conn.RemoteAddr().String()
	//index := strings.Index(addr,":")
	//client.addr = addr[:index]
	//client.msg = msg


	switch msg.MsgType {
	case SUBSCRIBE:
		c.broker.handleSubscribe(clientId,msg)
	case UNSUBSCRIBE:
		c.broker.handleUnsubscribe(clientId,msg)
	case PUBLISH:
		c.broker.BroadCast(msg)
	case INFO:
		c.broker.searchFollowersInfo(clientId,msg)
	case PERFORMANCE_INDICATORS:
		c.broker.updateClusterMaxMin(msg)
	case MASTER_GENERATION:
		c.updateMasterConfiguration(msg.MsgContent)
	case TEST:
		msg.MsgContent=[]byte("response")
		TransmitToNode(clientId.Ip+clientId.Port,msg)
	}
}


//客户端接收到消息后根据类型分类处理
func (c *Client) handleFollowerMsg(conn net.Conn)  {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(reader)
	if err == io.EOF {
		fmt.Println("handleFollowerMsg: Client read data is failed!")
		return
	}

	data := buf.Bytes()
	if string(data) == ""{
		return
	}

	msg := Msg{}
	err = json.Unmarshal(data,&msg)
	if err != nil{
		fmt.Printf("handleFollowerMsg: %v\n",err)
		return
	}

	switch msg.MsgType {
	case SUBSCRIBE:
		log.Printf("Follower: %s",string(msg.MsgContent))
		//订阅成功后 发送自身的静态性能指标给主节点
		SendLocalPerformanceIndicators(c.leader)
	case UNSUBSCRIBE:
		log.Printf("Follower: %s",string(msg.MsgContent))
	case PUBLISH:
		saveNetMsg(msg.MsgContent)
	case PERFORMANCE_INDICATORS:
		saveClusterPerformanceMaxMinIndicators(msg.MsgContent)
	case CALCULATE_SCORE:
		c.calculateSelfScore(msg)
	case NEW_MASTER_READY:
		c.readyBecomeNewMaster(msg.MsgContent)
	case MASTER_GENERATION:
		c.updateMasterConfiguration(msg.MsgContent)
	case TEST:
		seq, _ := strconv.Atoi(msg.Topic)
		c.testApply <- seq
	}

}

// TimedObtainLocalUtilization 定期获得动态指标
func (c *Client) TimedObtainLocalUtilization()  {
	timeTask := cron.New(cron.WithSeconds())
	spec := "*/2 * * * * ?"
	_,err := timeTask.AddFunc(spec,c.obtainLocalUtilization)
	if err != nil{
		log.Printf("TimedObtainLocalUtilization: %v",err)
		return
	}
	timeTask.Start()
	defer timeTask.Stop()
	select {

	}
}

func (c *Client) obtainLocalUtilization()  {
	c.indicatorsUtilization.CpuUtilization = GetLocalCpuUtilization()
	_,c.indicatorsUtilization.MemUtilization = GetLocalMemoryInfo()
	_,c.indicatorsUtilization.DiskUtilization = GetLocalDiskInfo()
}



//将本从节点升级为主节点
func (c *Client) readyBecomeNewMaster(msgContent []byte)  {
	//本节点成为领导者
	log.Println("I become the master")

	c.Lock()
	c.leader = c.clientId.Ip + c.clientId.Port
	c.broker = NewBroker()
	c.identity = Leader
	c.Unlock()
	go c.broker.TimedPubNetMsg()
	FinishNewMasterReady(string(msgContent))
}

//更新节点中的主节点配置
func (c *Client) updateMasterConfiguration(masterAddr []byte)  {
	c.leader = string(masterAddr)
	log.Printf("New Master(%v) is elected; Client update Master Configuration",c.leader)
	if c.leader != c.clientId.Ip + c.clientId.Port {
		if c.identity == Leader{
			c.broker.stop <- true
			c.Lock()
			c.identity = Follower
			c.Unlock()
		}

		err := c.Subscribe("nodeState")
		if err != nil{
			log.Printf("Follower: updateMasterConfiguration is failed: Subscribe is failed")
		}
	}

}

//计算自身的综合指标得分
func (c *Client) calculateSelfScore(msg Msg)  {
	var cpuNormal,memNormal,diskNormal,netNormal,performanceScore,totalScore float64
	//获取动态和静态性能指标
	cpuFrequency := GetLocalCpuFrequency()
	cpuUsed := GetLocalCpuUtilization()
	mem,memUsed:= GetLocalMemoryInfo()
	disk,diskUsed := GetLocalDiskInfo()

	cpuUtilizationChange := 0.5+math.Abs(c.indicatorsUtilization.CpuUtilization - cpuUsed)
	memUtilizationChange := 0.3+math.Abs(c.indicatorsUtilization.MemUtilization - memUsed)
	diskUtilizationChange := 0.2+math.Abs(c.indicatorsUtilization.DiskUtilization - diskUsed)


	total := cpuUtilizationChange + memUtilizationChange + diskUtilizationChange
	w1 := cpuUtilizationChange/total
	w2 := memUtilizationChange/total
	w3 := diskUtilizationChange/total
	log.Printf("cpuChangeWeight=%f,memChangeWeight=%f,diskChangeWeight=%f",w1,w2,w3)

	var wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		//标准化处理
		cpuNormal,memNormal,diskNormal = performanceDataNormalization(cpuFrequency,mem,disk)
		log.Printf("calculateSelfScore: cpuNormal=%f,memNormal=%f,diskNormal=%f",cpuNormal,memNormal,diskNormal)

		log.Printf("cpuUsed=%f,memUsed=%f,diskUsed=%f",cpuUsed,memUsed,diskUsed)
		performanceScore = w1*cpuNormal*(1-cpuUsed) + w2*memNormal*(1-memUsed) + w3*diskNormal*(1-diskUsed)
		log.Printf("performanceScore=%f",performanceScore)
		//log.Printf("calculateSelfScore: performanceScore = %f",performanceScore)
		wg.Done()
	}()
	go func() {
		//标准化处理
		netNormal = netDataNormalization(c.clientId.Ip)
		log.Printf("calculateSelfScore: netScore=%f",netNormal)
		wg.Done()
	}()
	wg.Wait()

	if cpuNormal<0  || memNormal<0 || diskNormal<0 || netNormal<0{
		totalScore = 0
	}else{
		totalScore = 0.9*performanceScore + 0.1*netNormal
	}

	log.Printf("pastDownCounts= %d",c.pastDownCounts)
	log.Printf("totalScore= %f",totalScore)

	CalculateCandidateScoreResponse(string(msg.MsgContent),c.clientId.Ip+c.clientId.Port,c.pastDownCounts,totalScore)

}
