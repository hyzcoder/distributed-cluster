# 项目简介  
搭建多台虚拟机实现分布式环境，设立消息队列完成节点通信，并实现pub/sub机制；  
建立哨兵集群并实现哨兵监测功能，哨兵发现节点宕机后发起投票进行领头哨兵选举；  
选举完领头哨兵后，根据节点性能选举新任领导者；  

# 实现环境  
虚拟机软件：VMware  
虚拟机系统：Centos7.9  
IDE：GoLand  
语言：Go语言  

# 代码使用说明

```

	//主节点创建
	b := code.NewClient()
	b.SetListenPort(":12345")//设置节点监听端口
	b.SetMasterIdentity(true)//设置是否为主节点
	b.Run()//运行主节点任务

	////从节点创建
	//c := code.NewClient()
	//c.SetMasterAddr("192.168.112.211:12345")//必须配置好主节点地址
	//c.SetListenPort(":10001")//必须配置好自身的监听端口
	//c.Run()
	

	////哨兵进程创建
	//sentinel := code.NewSentinel()
	//sentinel.SetMasterAddr("192.168.112.211:12345")//必须配置好主节点地址
	//sentinel.SetListenPort(":11001")////必须配置好自身的监听端口
	//sentinel.Run()


	//使主程序堵塞，一直运行程序
	select {

	}

```

