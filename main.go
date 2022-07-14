/**
	The Program was written by hyz
 */
package main

import (
	"GraduationProject/code"
)

func main() {

	//创建Broker，开启端口监听
	b := code.NewClient()
	b.SetListenPort(":12345")
	b.SetMasterIdentity(true)
	b.Run()

	//定义客户端，开启端口监听
	//c := code.NewClient()
	//c.SetMasterAddr("192.168.112.215:12345")
	//c.SetListenPort(":10002")
	//c.Run()

	
	//code.ContinuousTestTask()

	//定义哨兵端
	//sentinel := code.NewSentinel()
	//sentinel.SetMasterAddr("192.168.112.215:12345")
	//sentinel.SetListenPort(":11002")
	//sentinel.Run()



	select {

	}



}
