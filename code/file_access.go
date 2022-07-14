package code

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func saveClusterPerformanceMaxMinIndicators(msgContent []byte)  {
	err1 := ioutil.WriteFile("./log/clusterMaxMinIndicators.txt",msgContent,0666)
	if err1 != nil{
		log.Printf("saveClusterPerformanceMaxMinIndicators: %v",err1)
	}
}

//保存集群中网络信息
func saveNetMsg(msgContent []byte) {
	//var nodes []NodeState
	//err1 := json.Unmarshal(msgContent,&nodes)
	//if err1 != nil{
	//	log.Printf("saveNetMsg: %v",err1)
	//}

	//log.Printf("\nClient: 从%s频道接收到发布来的消息如下：",msg.Topic)
	////log.Println(nodes)
	//for i,node := range nodes{
	//	fmt.Printf("(%d) ",i+1)
	//	fmt.Printf("结点ip：%s  ",node.PingIP)
	//	fmt.Printf("主结点ping其花费时间：%.3fms\n",node.PingTime)
	//}


	//将指定内容写入到文件中
	err := ioutil.WriteFile("./log/nodeState.txt", msgContent, 0666)
	if err != nil {
		log.Printf("saveNetMsg: 网络状态信息写入文件失败: %v",err)

	}
}

func savePastDownCount(i int64)  {
	bytes := Int64ToBytes(i)
	err := ioutil.WriteFile("./log/pastDownCount.txt",bytes,0666)
	if err != nil{
		log.Printf("savePastDownCount: %v",err)
	}
}

func saveTestData(cost time.Duration,seq int)  {
	//bytes := Int64ToBytes(i)
	//err := ioutil.WriteFile("./log/pastDownCount.txt",bytes,0666)
	//if err != nil{
	//	log.Printf("savePastDownCount: %v",err)
	//}
	filePath := ""
	if seq == 1{
		filePath = "./log/testApplyTime.txt"
	}else if seq == 2{
		filePath = "./log/testApplyTime2.txt"
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	//及时关闭file句柄
	defer file.Close()

	str := fmt.Sprintf("%v\n",cost)
	if _, err = file.WriteString(str); err != nil {
		panic(err)
	}

	////写入文件时，使用带缓存的 *Writer
	//write := bufio.NewWriter(file)
	//for i := 0; i < 5; i++ {
	//	write.WriteString("C语言中文网 \r\n")
	//}
	////Flush将缓存的文件真正写入到文件中
	//write.Flush()
}

func readPastDownCount() int64 {
	if !exist("./log/pastDownCount.txt"){
		return 0
	}

	msgByte,err := ioutil.ReadFile("./log/pastDownCount.txt")
	if err != nil{
		return 0
	}

	return BytesToInt64(msgByte)

}

//判断文件是否存在
func exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
//读取文件中信息
func readFromIndicatorsFile(fileName string) ClusterMaxMinIndicators{
	cmmi := ClusterMaxMinIndicators{}

	msgBytes,err := ioutil.ReadFile(fileName)
	if err != nil{
		fmt.Println(err)
	}


	err1 := json.Unmarshal(msgBytes,&cmmi)
	if err1 != nil{
		fmt.Println(err1)
	}

	return  cmmi
}


//读取文件中信息
func readFromNetFile(fileName string) []NodeState{
	var nodes []NodeState

	msgBytes,err := ioutil.ReadFile(fileName)
	if err != nil{
		fmt.Println(err)
	}


	err1 := json.Unmarshal(msgBytes,&nodes)
	if err1 != nil{
		fmt.Println(err1)
	}

	return  nodes
}