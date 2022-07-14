package code

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// CheckPortFree 检测端口是否空闲可用
func CheckPortFree(port string) error {
	conn,err := net.DialTimeout("tcp",":"+port,time.Second)
	if err != nil{
		return err
	}
	defer conn.Close()
	return nil
}

// GetLocalIP 获取本地IP
func GetLocalIP() string {
	addresses,err := net.InterfaceAddrs()
	if err != nil{
		log.Printf("GetLocalIP: %v",err)
		return ""
	}
	for _,address := range addresses{
		if ipNet,ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback(){
			if ipNet.IP.To4() != nil{
				return ipNet.IP.String()
			}
		}
	}
	log.Println("GetLocalIP: can not find local ip! ")
	return ""
}

// GetLocalCpuUtilization 获取CPU利用率
func GetLocalCpuUtilization() float64 {
	//100ms 总 CPU 使用率
	totalPercent,err := cpu.Percent(time.Millisecond*100,false)
	if err != nil{
		//log.Println("getLocalCpuUtilization is failed")
		return 1
	}
	//fmt.Printf("cpuUtilization: %f\n",totalPercent[0]*0.01)
	return totalPercent[0]*0.01
}

// GetLocalCpuFrequency 获取CPU主频
func GetLocalCpuFrequency() float64 {
	var total float64
	total = 0
	cpuInfos,err := cpu.Info()
	if err != nil{
		//log.Println("getLocalCpuFrequency is failed")
		return 0
	}
	for _,info := range cpuInfos{
		modelName := strings.Split(info.ModelName,"@")[1]
		cores := info.Cores
		length := len(modelName)
		frequencyString := modelName[1:length-3]
		cpuFrequency,err1 := strconv.ParseFloat(frequencyString,64)
		if err1 != nil{
			return 0
		}
		total += cpuFrequency*float64(cores)
		//frequencyMagnitude := modelName[length-3]
		//cpuFrequency := frequencyValue*
	}
	//fmt.Printf("cpuFrequency: %f\n",total)
	return total
}

// GetLocalMemoryInfo 获取本机内存信息
//第一返回值： 内存大小；第二返回值：内存使用率
func GetLocalMemoryInfo() (float64,float64) {
	memoryInfo,err := mem.VirtualMemory()
	if err != nil {
		return 0,1
	}
	//fmt.Printf("memoryTotal: %d, memoryUtilization: %f\n",memoryInfo.Total,memoryInfo.UsedPercent*0.01)
	return float64(memoryInfo.Total),memoryInfo.UsedPercent*0.01
}

// GetLocalDiskInfo 获取本机磁盘信息
//第一返回值： 磁盘大小；第二返回值：磁盘使用率
func GetLocalDiskInfo() (float64,float64) {
	var total,used uint64
	var usedPercent float64

	infos,err := disk.Partitions(false)
	if err != nil {
		return 0,1
	}
	for _,info := range infos{
		diskInfo,err1 := disk.Usage(info.Mountpoint)
		if err1 != nil{
			return 0,1
		}
		total += diskInfo.Total
		used += diskInfo.Used
	}
	usedPercent = float64(used)/float64(total)
	//fmt.Printf("diskTotal: %d, diskUtilization: %f\n",total,usedPercent)
	return float64(total),usedPercent
}