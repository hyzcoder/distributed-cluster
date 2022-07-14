package code

import (
	"math/rand"
	"time"
)

//返回min与max之间的随机时间
func afterBetween(min time.Duration, max time.Duration) <-chan time.Time {
	rands := rand.New(rand.NewSource(time.Now().UnixNano()))
	d, delta := min, max - min
	if delta > 0 {
		d += time.Duration(rands.Int63n(int64(delta)))
	}
	return time.After(d)
}

//将cpu主频、内存容量、磁盘容量进行归一化
func performanceDataNormalization(cpuData float64,memData float64,diskData float64) (float64,float64,float64) {
	cmmi := readFromIndicatorsFile("./log/clusterMaxMinIndicators.txt")
	cm := ClusterMaxMinIndicators{}
	if cmmi == cm {
		return -1,-1,-1
	}
	if cmmi.MinIndicators[0] == cmmi.MaxIndicators[0]{
		cpuData = 1
	}else{
		cpuData = (cpuData - cmmi.MinIndicators[0])/(cmmi.MaxIndicators[0] - cmmi.MinIndicators[0])
	}
	if cmmi.MinIndicators[1] == cmmi.MaxIndicators[1]{
		memData = 1
	}else {
		memData = (memData - cmmi.MinIndicators[1])/(cmmi.MaxIndicators[1] - cmmi.MinIndicators[1])
	}
	if cmmi.MinIndicators[2] == cmmi.MaxIndicators[2]{
		diskData = 1
	}else {
		diskData = (diskData - cmmi.MinIndicators[2])/(cmmi.MaxIndicators[2] - cmmi.MinIndicators[2])
	}

	return cpuData,memData,diskData
}

//将网络信息数据归一化
func netDataNormalization(currentIp string) float64 {
	nodes := readFromNetFile("./log/nodeState.txt")
	if nodes == nil{
		return -1
	}
	var pingData float64
	max := nodes[0].PingTime
	min := max
	for _,node := range nodes{
		if node.PingIP == currentIp{
			pingData = node.PingTime
		}
		if node.PingTime != -1{
			if node.PingTime > max{
				max = node.PingTime
			}else if node.PingTime < min{
				min = node.PingTime
			}
		}
	}
	if max == min {
		return 1
	}

	pingData = (max - pingData)/(max - min)
	return pingData
}
