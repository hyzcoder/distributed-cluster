package code

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Sentinel struct {
	client *Client
	epoch int
	agreedDownCount int
	masterState string
	voteFor string
	doVote bool
	voteGranted chan bool
	stopVote chan bool
	ready chan bool
	candidateResponse chan CandidateInfo
	electionTimeout time.Duration
	downtimeLimit time.Duration
	sentinelList SentinelList
	followerList FollowerList
	sync.RWMutex
}

//var now time.Time

// SentinelList 哨兵列表
type SentinelList struct{
	sentinels map[ClientID]bool
	sync.RWMutex
}

// FollowerList 从节点列表
type FollowerList struct {
	followers map[ClientID]bool
	sync.RWMutex
}

// VoteInfo 投票信息
type VoteInfo struct {
	Epoch int
	VoteFor string
}


//主机状态
const (
	RUNNING string = "Running"
	SDOWN string = "SubjectDown"
	ODOWN string = "ObjectDown"
)

func NewSentinel () *Sentinel {
	return &Sentinel{
		client:        NewClient(),
		downtimeLimit: 100*time.Millisecond,
		masterState:   RUNNING,
		electionTimeout: 100*time.Millisecond,
		agreedDownCount: 0,
		voteFor: "",
		doVote: true,
		voteGranted: make(chan bool),
		stopVote: make(chan bool),
		ready: make(chan bool),
		candidateResponse: make(chan CandidateInfo),
		epoch: 0,
		sentinelList: SentinelList{
			sentinels: make(map[ClientID]bool),
		},
		followerList: FollowerList{
			followers: make(map[ClientID]bool),
		},
	}
}

func (s *Sentinel) SetMasterAddr(masterIpPort string) {
	s.client.leader = masterIpPort
}

func (s *Sentinel) SetListenPort(port string)  {
	s.client.clientId.Port = port
}

// Run 运行哨兵的任务
func (s *Sentinel) Run()  {
	go s.ListenPort()
	time.Sleep(10*time.Millisecond)

	err1 := s.Subscribe("sentinel")
	if err1 != nil{
		log.Println(err1)
		return
	}

	go s.TimedSendSelfInfo()
	go s.TimedGetFollowers()
	go s.TimedCheckNode()
	go s.TimedOrganizeMonitorList()
}

func (s *Sentinel) Leader() string {
	s.client.RLock()
	defer  s.client.RUnlock()
	return s.client.leader
}

func (s *Sentinel) DoVote() bool {
	s.RLock()
	defer s.RUnlock()
	return s.doVote
}

func (s *Sentinel) VoteFor() string {
	s.RLock()
	defer s.RUnlock()
	return s.voteFor
}

func (s *Sentinel) Epoch() int {
	s.RLock()
	defer s.RUnlock()
	return s.epoch
}

func (s *Sentinel) MasterState() string {
	s.RLock()
	defer s.RUnlock()
	return s.masterState
}

// Subscribe 哨兵订阅功能
func (s *Sentinel) Subscribe(topic string) error  {
	err := s.client.Subscribe(topic)
	return err
}

// Publish 哨兵发布功能
func (s *Sentinel) Publish(topic string,msgContent []byte) error {
	err := s.client.Publish(topic,msgContent)
	return err
}

// ListenPort 监听方法重写
func (s *Sentinel) ListenPort()  {
	s.client.clientId.Ip = GetLocalIP()

	//监听端口
	listen, err := net.Listen("tcp",s.client.clientId.Port)
	if err != nil{
		fmt.Println("ListenPort: Sentinel Listen is failed!")
		return
	}
	log.Printf("Sentinel（%s）: start port listening",s.client.clientId.Ip+s.client.clientId.Port)

	for{
		conn, err1 := listen.Accept()
		if err1 != nil{
			fmt.Println("ListenPort: Sentinel Accept is failed!")
			continue
		}

		go s.handleSentinelMsg(conn)
	}
}

// 处理哨兵接收到的消息
func (s *Sentinel) handleSentinelMsg(conn net.Conn){
	defer conn.Close()
	reader := bufio.NewReader(conn)
	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(reader)
	if err == io.EOF {
		fmt.Println("handleSentinelMsg: Sentinel read data is failed!")
		return
	}

	data := buf.Bytes()
	if string(data) == ""{
		return
	}

	msg := Msg{}
	err = json.Unmarshal(data,&msg)
	if err != nil{
		fmt.Printf("handleSentinelMsg: %v\n",err)
		return
	}

	switch msg.MsgType {
	case SUBSCRIBE:
		log.Printf("Sentinel: %s",string(msg.MsgContent))
	case UNSUBSCRIBE:
		log.Printf("Sentinel: %s",string(msg.MsgContent))
	case PUBLISH:
		s.updateSentinelList(msg.MsgContent)
	case INFO:
		s.updateFollowers(msg.MsgContent)
	case MONITOR_REQUEST:
		s.ackMonitorResult(msg,conn.RemoteAddr().String())
	case MONITOR_RESULT:
		s.ackObjectDown(msg.MsgContent)
	case ELECTION_SENTINEL_REQUEST:
		s.responseElectionSentinel(msg.MsgContent)
	case ELECTION_SENTINEL_RESPONSE:
		s.countElectionVotes(msg.MsgContent)
	case ELECTION_LEAD_SENTINEL_SUCCESS:
		s.updateSentinelConfig(msg.MsgContent)
	case CALCULATE_SCORE:
		s.receiveCandidateCalculateMsg(msg.MsgContent)
	case MASTER_GENERATION:
		s.updateMasterConfiguration(msg.MsgContent)
	case NEW_MASTER_READY:
		s.ready <- true
	}
}

//接收sentinel频道的信息，以更新哨兵队列
func (s *Sentinel) updateSentinelList(msgContent []byte)  {
	addr := string(msgContent)
	clientId := ClientID{}
	clientId.Ip = strings.Split(addr,":")[0]
	clientId.Port = ":"+strings.Split(addr,":")[1]

	s.sentinelList.Lock()
	s.sentinelList.sentinels[clientId] = true
	s.sentinelList.Unlock()
}

//主节点选举完毕，更新哨兵配置
func (s *Sentinel) updateMasterConfiguration(masterAddr []byte)  {
	log.Printf("New Master(%v) is elected; Sentinel update Master Configuration",string(masterAddr))

	//old master become follower
	oldMasterAddr := strings.Split(s.Leader(),":")
	oldMaster := ClientID{Ip: oldMasterAddr[0],Port:":"+oldMasterAddr[1] }
	s.followerList.Lock()
	s.followerList.followers[oldMaster] = false
	s.followerList.Unlock()


	masterStr := string(masterAddr)

	//new master delete from follower
	newMasterAddr := strings.Split(masterStr,":")
	newMaster := ClientID{Ip: newMasterAddr[0],Port: ":"+newMasterAddr[1]}
	s.followerList.Lock()
	delete(s.followerList.followers,newMaster)
	s.followerList.Unlock()



	s.client.Lock()
	s.client.leader = masterStr
	s.client.Unlock()

	s.Lock()
	//log.Printf("sentinelUpdateMasterConfig: test")
	s.doVote = true
	s.voteFor = ""
	s.masterState = RUNNING
	s.Unlock()
	err := s.Subscribe("sentinel")
	if err != nil{
		log.Printf("Sentinel: updateMasterConfiguration is failed: Subscribe is failed")
	}

}

//接受候选节点传来的竞选消息
func (s *Sentinel) receiveCandidateCalculateMsg(msgContent []byte)  {
	ci := CandidateInfo{}
	err := json.Unmarshal(msgContent,&ci)
	if err != nil {
		log.Printf("receiveCandidateCalculateMsg: %v",err)
	}
	s.candidateResponse <- ci
}


//接收到领导哨兵选举成功消息，更新自身配置
func (s *Sentinel) updateSentinelConfig(revMsgContent []byte)  {
	log.Println("Leading sentinel selection is successful; update sentinel configuration")
	vi := VoteInfo{}
	if err := json.Unmarshal(revMsgContent,&vi); err!= nil{
		log.Printf("updateSentinelConfig: %v",err)
	}

	s.Lock()
	s.epoch = vi.Epoch
	s.doVote = false
	s.Unlock()

	s.stopVote <- true
}

//统计收到的选票
func (s *Sentinel) countElectionVotes(revMsgContent []byte)  {
	vi := VoteInfo{}
	err := json.Unmarshal(revMsgContent,&vi)
	if err != nil{
		log.Printf("countElectionVotes: %v",err)
		return
	}

	if vi.VoteFor == s.VoteFor() {
		s.voteGranted <- true
	}else if vi.Epoch>s.Epoch(){
		s.Lock()
		s.epoch = vi.Epoch
		s.Unlock()
	}

}

//回复其他哨兵传来的请求投票消息
func (s *Sentinel) responseElectionSentinel(revMsgContent []byte)  {

	vi := VoteInfo{}
	err := json.Unmarshal(revMsgContent,&vi)
	if err != nil {
		log.Printf("responseElectionSentinel: %v",err)
	}

	s.Lock()
	if s.voteFor != ""{
		log.Printf("responseElectionSentinel: already voteFor=%v",s.voteFor)
		VoteElectLeadSentinelResponse(vi.VoteFor,s.voteFor,s.epoch)
	}else{
		if s.epoch > vi.Epoch{
			log.Printf("responseElectionSentinel: %v epoch is too early",vi.VoteFor)
			VoteElectLeadSentinelResponse(vi.VoteFor,"",s.epoch)
		}else{
			log.Printf("responseElectionSentinel: voteForSender=%v",vi.VoteFor)
			VoteElectLeadSentinelResponse(vi.VoteFor,vi.VoteFor,vi.Epoch)
			s.voteFor = vi.VoteFor
			s.epoch = vi.Epoch
		}

	}
	s.Unlock()
}

//获得在线的哨兵数目
func (s *Sentinel) getSentinelOnlineCounts() int {
	length := 0
	s.sentinelList.RLock()
	defer s.sentinelList.RUnlock()
	for _,state := range s.sentinelList.sentinels{
		if state{
			length++
		}
	}
	return length
}

//确认主节点是否客观下线，确认主节点客观下线后开启领导哨兵选举
func (s *Sentinel) ackObjectDown(sentinelOpinion []byte)  {
	result := string(sentinelOpinion)
	if result == "true"{

		//get sentinel online counts
		length := s.getSentinelOnlineCounts()

		s.Lock()
		defer s.Unlock()

		log.Println("objective offline consent number plus one")
		s.agreedDownCount++
		//大多数哨兵同意则进行领头哨兵选举
		if s.agreedDownCount> (length/2){
			if s.masterState == SDOWN{
				log.Println("masterState: ObjectDown")
				s.masterState = ODOWN
				/**
				start lead sentinel election
				*/
				go s.electionLeadSentinel()
				//now = time.Now()
			}
		}
	}
}

//选举领导哨兵
func (s *Sentinel) electionLeadSentinel()  {
	quorumSize := 0
	voteGrantedCount := 0

	 //等待一个随机时间后将会发起投票
	waitRandomTime := afterBetween(50*time.Millisecond,200*time.Millisecond)
	select {
		case <- waitRandomTime:
	}


	//一轮选举超时时间
	timeoutChan := afterBetween(s.electionTimeout,2*s.electionTimeout)
	if s.DoVote() == true{
		log.Println("start electing leading sentinel!")
	}

	 for s.DoVote() == true {
		 if s.VoteFor() == ""{
			 s.Lock()
			 s.epoch++
			 s.voteFor = s.client.clientId.Ip + s.client.clientId.Port
			 s.Unlock()

			 s.sentinelList.RLock()
			 //log.Printf("vote for myself:%s",s.voteFor)
			 for sentinel := range s.sentinelList.sentinels{
				 if sentinel != s.client.clientId{
					 go VoteElectLeadSentinelRequest(sentinel.Ip+sentinel.Port,s.voteFor,s.epoch)
				 }
			 }
			 s.sentinelList.RUnlock()

			 quorumSize = s.getSentinelOnlineCounts()/2
			 //quorumSize = (len(s.sentinelList.sentinels))/2
			 voteGrantedCount = 1
			 timeoutChan = afterBetween(s.electionTimeout,2*s.electionTimeout)
			 log.Printf("electionLeadSentinel: vote myself")
		 }

		 if voteGrantedCount > quorumSize{
		 	go func() {
		 		s.stopVote <- true
			}()
		 	log.Printf("I(%v) successed in becoming a leading sentinel",s.client.clientId.Ip+s.client.clientId.Port)
			 /**
			    Notify election lead sentinel success!
			 */
			 //log.Printf("%s is successful become lead sentinel",s.client.Ip+s.client.Port)
			 s.Lock()
			 s.doVote = false
			 s.Unlock()

			 s.sentinelList.RLock()
			 for sentinel := range s.sentinelList.sentinels{
				 if sentinel != s.client.clientId{
					 go NotifyLeadSentinelElectionSuccess(sentinel.Ip+sentinel.Port,s.voteFor,s.epoch)
				 }
			 }
			 s.sentinelList.RUnlock()
			 /**
			 	start elect master
			 */
			 go s.electMasterNode()

		 }

		 select {
		 case <- s.voteGranted:
		 	voteGrantedCount++
			 log.Printf("the sentinel won %v votes",voteGrantedCount)
		 case <- timeoutChan:
		 	log.Println("elect leading sentinel timeout")
			 s.Lock()
			 s.voteFor = ""
			 s.Unlock()
		 	//等待一个随机时间后发起投票
			 waitRandomTime = afterBetween(50*time.Millisecond,200*time.Millisecond)
			 select {
			 	case <- waitRandomTime:
			 		}

		 case <- s.stopVote:
			log.Printf("sentinel stop voting")
			return
		 }
	 }
	 //避免上一次领头哨兵选举留下的chan信号影响下次选举
	select {
	case <- s.stopVote:
		log.Printf("sentinel stop voting")
	 }
}

//选举集群主节点
func (s *Sentinel) electMasterNode()  {
	candidateCount := 0
	best := CandidateInfo{"",-1,0}

	log.Printf("start to elect Master")

	s.followerList.RLock()
	for candidate,state := range s.followerList.followers{
		if state == true{
			candidateCount++
			go CalculateCandidateScoreRequest(candidate.Ip+candidate.Port,
				s.client.clientId.Ip+s.client.clientId.Port)
		}
	}
	s.followerList.RUnlock()

	timeout := afterBetween(time.Second,time.Second)//避免无限制等待候选者返回消息
	for candidateCount != 0 {
		select {
			case ci := <-s.candidateResponse:
				candidateCount--
				if ci.Score > best.Score{
					best = ci
				}else if ci.Score == best.Score{
					if ci.DownCounts < best.DownCounts{
						best = ci
					}else if ci.DownCounts == best.DownCounts && ci.CandidateID>best.CandidateID{
						best = ci
					}
				}
			case <-timeout:
				//log.Println("electMasterNode: timeout")
				candidateCount = 0
		}
	}
	log.Printf("new Master(%v) is elected,broadcast the election results to the cluster nodes",best.CandidateID)
	//dur := time.Now().Sub(now)
	//log.Println(dur)
	AdvancedTellNewMasterReady(best.CandidateID,s.client.clientId.Ip+s.client.clientId.Port)
	s.broadClusterNewMasterNode(best.CandidateID)

}

//将新主节点在集群中广播告知
func (s *Sentinel) broadClusterNewMasterNode(masterAddr string)  {
	<- s.ready

	s.followerList.RLock()
	for candidate := range s.followerList.followers{
		go NotifyNewMasterMsg(candidate.Ip+candidate.Port,masterAddr)
	}
	s.followerList.RUnlock()

	s.sentinelList.RLock()
	for sentinel := range s.sentinelList.sentinels{
		go NotifyNewMasterMsg(sentinel.Ip+sentinel.Port,masterAddr)
	}
	s.sentinelList.RUnlock()

	go NotifyNewMasterMsg(s.Leader(),masterAddr)

}

//接收到其他哨兵传来的主节点下线消息，检测主节点状态
func (s *Sentinel) ackMonitorResult(recMsg Msg,addr string)  {
	timeLimit := s.downtimeLimit
	dest := string(recMsg.MsgContent)
	conn,err := net.DialTimeout("tcp",dest,timeLimit)

	//log.Printf("ackMonitorResult: dest=%s,srcSentinel=%s",dest,strings.Split(addr,":")[0]+recMsg.Port)
	msg := Msg{}
	msg.MsgType = MONITOR_RESULT
	if err != nil{
		msg.MsgContent = []byte("true")
	}else {
		//fmt.Println("ackMonitorResult: false")
		msg.MsgContent = []byte("false")
		conn.Close()
	}

	srcIP := strings.Split(addr,":")[0]
	TransmitToNode(srcIP+recMsg.Port,msg)
}

//更新哨兵维护的跟随者列表
func (s *Sentinel) updateFollowers(msgContent []byte) {
	clients := make([]ClientID,0)
	err := json.Unmarshal(msgContent,&clients)
	if err != nil {
		log.Printf("updateFollowers: %v",err)
	}

	s.followerList.Lock()
	for _,c := range clients {
		if _,found := s.followerList.followers[c]; !found {
			s.followerList.followers[c] = true
		}
	}
	s.followerList.Unlock()

}

// TimedGetFollowers 定时任务：获取跟随者信息
func (s *Sentinel) TimedGetFollowers(){
	timeTask := cron.New(cron.WithSeconds())
	spec := "*/5 * * * * ?"
	_,err := timeTask.AddFunc(spec,s.getFollowers)
	if err != nil {
		fmt.Printf("TimedGetFollowers: %v\n",err)
		return
	}
	timeTask.Start()
	defer timeTask.Stop()

	select {
	}
}

//获取跟随者信息
func (s *Sentinel) getFollowers()  {
	msg := Msg{}
	msg.MsgType = INFO
	msg.Port = s.client.clientId.Port
	dest := s.Leader()

	TransmitToNode(dest,msg)
}

// TimedOrganizeMonitorList 定期整理监测队列，将下线节点从列表移除
func (s *Sentinel) TimedOrganizeMonitorList()  {
	timeTask := cron.New(cron.WithSeconds())
	spec := "*/30 * * * * ?"
	_,err := timeTask.AddFunc(spec,s.organizeMonitorList)
	if err != nil{
		log.Printf("TimedOrganizeSentinelLis: %v",err)
		return
	}
	timeTask.Start()
	defer timeTask.Stop()
	select {

	}
}

//整理监测队列，将下线节点从列表移除
func (s *Sentinel) organizeMonitorList()  {
	s.sentinelList.Lock()
	for sentinel,state := range s.sentinelList.sentinels{
		if state == false{
			delete(s.sentinelList.sentinels,sentinel)
		}
	}
	s.sentinelList.Unlock()

	s.followerList.Lock()
	for follower,state := range s.followerList.followers{
		if state == false{
			delete(s.followerList.followers,follower)
		}
	}
	s.followerList.Unlock()

}

// TimedSendSelfInfo 周期性（每两秒）的发送自身的信息
func (s *Sentinel) TimedSendSelfInfo() {
	timeTask := cron.New(cron.WithSeconds())
	spec := "*/2 * * * * ?"
	_,err := timeTask.AddFunc(spec,s.sendSelfInfo)
	if err != nil {
		fmt.Printf("TimedSendSelfInfo: %v\n",err)
		return
	}
	timeTask.Start()
	defer timeTask.Stop()

	select {
	}
}


// SendSelfInfo 发布自身哨兵信息
func (s *Sentinel) sendSelfInfo()  {
	if err := s.Publish("sentinel",[]byte(s.client.clientId.Ip+s.client.clientId.Port)); err != nil{
		//fmt.Println(err)
		return
	}
}

// TimedCheckNode 定期检测集群节点是否在线
func (s *Sentinel) TimedCheckNode(){
	timeTask := cron.New(cron.WithSeconds())
	spec := "* * * * * ?"
	_,err := timeTask.AddFunc(spec,s.checkNodes)
	if err != nil{
		fmt.Printf("TimedCheckNode: %v\n",err)
	}
	timeTask.Start()
	defer timeTask.Stop()

	select {
	}

}

//检测节点是否在线
func (s *Sentinel) checkNodes()  {
	var wg sync.WaitGroup
	timeoutLimit := s.downtimeLimit

	s.followerList.RLock()
	wg.Add(len(s.followerList.followers))
	//检测从节点在线情况
	for follower := range s.followerList.followers {
		dest := follower
		go func() {
			conn,err := net.DialTimeout("tcp",dest.Ip+dest.Port,timeoutLimit)
			s.followerList.Lock()
			if err != nil{
				//log.Printf("checkNode: Follower%s=False,Error=%s,context:%v",dest.Ip+dest.Port,err.Error(),ctxt.Err())
				s.followerList.followers[dest] = false
			}else{
				s.followerList.followers[dest] = true
				conn.Close()
			}
			s.followerList.Unlock()
			wg.Done()
		}()
	}
	log.Printf("followerList: %v",s.followerList.followers)
	s.followerList.RUnlock()

	s.sentinelList.RLock()
	wg.Add(len(s.sentinelList.sentinels))
	//检测哨兵节点在线情况
	for sentinel := range s.sentinelList.sentinels{
		dest := sentinel
		go func() {
			conn,err := net.DialTimeout("tcp",dest.Ip+dest.Port,timeoutLimit)
			s.sentinelList.Lock()
			if err != nil{
				//log.Printf("checkNode: Sentinel%s=False,Error=%s,context:%v",dest.Ip+dest.Port,err.Error(),ctxt.Err())
				s.sentinelList.sentinels[dest] = false
			}else{
				s.sentinelList.sentinels[dest] = true
				conn.Close()
			}
			s.sentinelList.Unlock()
			wg.Done()
		}()
	}
	log.Printf("sentinelList: %v",s.sentinelList.sentinels)
	s.sentinelList.RUnlock()


	//检测主节点
	//检测时间超过最大限制时间或出错，则master被标记为主观下线
	conn,err := net.DialTimeout("tcp",s.Leader(),timeoutLimit*2)
	//conn,err := net.DialTimeout("tcp",s.Leader(),timeoutLimit)
	log.Printf("master: %v",s.Leader())
	log.Printf("materState: %v",s.MasterState())
	s.Lock()
	if err != nil {
		//log.Printf("checkNode: Master%s=False,Error=%s,context:%v",Host,err.Error(),ctxt.Err())
		//之前未发现主节点下线，询问其他哨兵意见
		if s.masterState == RUNNING{
			s.masterState = SDOWN
			s.agreedDownCount = 1
			log.Printf("materState: %v",s.masterState)
			//go s.askOtherSentinelMasterState()
		}
	}else{
		if s.masterState == SDOWN{
			s.masterState = RUNNING
		}
		conn.Close()
	}
	s.Unlock()

	wg.Wait()

	if s.MasterState() == SDOWN{
		//通知监测此节点的所有哨兵
		go s.askOtherSentinelMasterState()
	}
}

//询问其他哨兵对主节点的宕机的意见
func (s *Sentinel) askOtherSentinelMasterState()  {
	if s.getSentinelOnlineCounts() == 1{
		s.Lock()
		if s.masterState == SDOWN{
			log.Println("masterState: ObjectDown")
			s.masterState = ODOWN
			/**
			start lead sentinel election
			*/
			go s.electionLeadSentinel()
			//now = time.Now()
		}
		s.Unlock()
	}else{
		msg := Msg{}
		msg.MsgType = MONITOR_REQUEST
		msg.Port = s.client.clientId.Port
		msg.MsgContent = []byte(s.Leader())

		s.sentinelList.RLock()
		for dest := range s.sentinelList.sentinels{
			if dest != s.client.clientId{
				go TransmitToNode(dest.Ip+dest.Port,msg)
			}
		}
		s.sentinelList.RUnlock()
	}

}

