package code

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// Msg 网络传递的消息
type Msg struct {
	MsgType int32
	Topic string
	Port string
	MsgContent []byte
}

//消息类型
const(
	SUBSCRIBE int32 = 0
	PUBLISH int32 = 1
	UNSUBSCRIBE int32 = 2
	INFO int32 = 3
	PERFORMANCE_INDICATORS int32 = 4
	MONITOR_REQUEST int32 = 5
	MONITOR_RESULT int32 = 6
	ELECTION_SENTINEL_REQUEST int32 = 7
	ELECTION_SENTINEL_RESPONSE int32 = 8
	ELECTION_LEAD_SENTINEL_SUCCESS int32 = 9
	CALCULATE_SCORE int32 = 10
	MASTER_GENERATION int32 = 11
	NEW_MASTER_READY int32 = 12
	TEST int32 = 13
)

func AdvancedTellNewMasterReady(dest string,src string)  {
	msg := Msg{}
	msg.MsgType = NEW_MASTER_READY
	msg.MsgContent = []byte(src)
	TransmitToNode(dest,msg)
}

func FinishNewMasterReady(dest string)  {
	msg := Msg{}
	msg.MsgType = NEW_MASTER_READY
	TransmitToNode(dest,msg)
}

func SendLocalPerformanceIndicators(dest string)  {
	msg := Msg{}
	msg.MsgType = PERFORMANCE_INDICATORS
	msg.Topic = "nodeState"

	pi := PerformanceIndicators{ }
	pi.CpuFrequency = GetLocalCpuFrequency()
	pi.MemCapacity,_ = GetLocalMemoryInfo()
	pi.DiskCapacity,_ = GetLocalDiskInfo()

	jsonPi,err := json.Marshal(pi)
	if err != nil{
		fmt.Printf("SendLocalPerformanceIndicators: %v",err)
	}
	msg.MsgContent = jsonPi

	TransmitToNode(dest,msg)
}

func VoteElectLeadSentinelResponse(dest string,voteNode string,currentEpoch int)  {
	msg := Msg{}
	msg.MsgType = ELECTION_SENTINEL_RESPONSE

	vi := VoteInfo{VoteFor: voteNode, Epoch:currentEpoch }
	jsonVr,err := json.Marshal(vi)
	if err != nil {
		fmt.Printf("VoteElectionResponse: %v",err)
	}

	msg.MsgContent = jsonVr

	TransmitToNode(dest,msg)
}

func NotifyLeadSentinelElectionSuccess(dest string,voteNode string,currentEpoch int)  {
	msg := Msg{}
	msg.MsgType = ELECTION_LEAD_SENTINEL_SUCCESS

	vi := VoteInfo{VoteFor: voteNode, Epoch:currentEpoch }
	jsonVr,err := json.Marshal(vi)
	if err != nil {
		fmt.Printf("NotifyElectionSucces: %v",err)
	}

	msg.MsgContent = jsonVr

	TransmitToNode(dest,msg)
}

func VoteElectLeadSentinelRequest(dest string,voteNode string,currentEpoch int)  {
	msg := Msg{}
	msg.MsgType = ELECTION_SENTINEL_REQUEST

	vi := VoteInfo{VoteFor: voteNode, Epoch:currentEpoch }
	jsonVr,err := json.Marshal(vi)
	if err != nil {
		fmt.Printf("VoteElectionRequest: %v",err)
	}

	msg.MsgContent = jsonVr

	TransmitToNode(dest,msg)
}

func CalculateCandidateScoreRequest(dest string,srcAddr string)  {
	msg := Msg{}
	msg.MsgType = CALCULATE_SCORE
	msg.MsgContent = []byte(srcAddr)

	TransmitToNode(dest,msg)
}

func CalculateCandidateScoreResponse(dest string,candidateAddr string,downCounts int64,score float64)  {
	msg := Msg{}
	msg.MsgType =CALCULATE_SCORE

	ci := CandidateInfo{}
	ci.CandidateID = candidateAddr
	ci.Score = score
	ci.DownCounts = downCounts
	jsonCi,err := json.Marshal(ci)
	if err != nil{
		log.Printf("CalculateCandidateScoreResponse: %v",err)
	}
	msg.MsgContent = jsonCi

	TransmitToNode(dest,msg)
}

func NotifyNewMasterMsg(dest string,masterAddr string)  {
	msg := Msg{}
	msg.MsgType = MASTER_GENERATION
	msg.MsgContent = []byte(masterAddr)

	TransmitToNode(dest,msg)

}

func TransmitToNode(dest string,msg Msg){
	conn,err := net.Dial("tcp",dest)
	if err != nil{
		//fmt.Printf("TransmitToNode: %v\n",err)
		return
	}
	defer conn.Close()

	jsonMsg,err1 := json.Marshal(msg)
	if err1 != nil{
		fmt.Printf("TransmitToNode: %v\n",err1)
		return
	}

	_,err2 := conn.Write(jsonMsg)
	if err2 != nil{
		fmt.Printf("TransmitToNode: %v\n",err2)
		return
	}

}
