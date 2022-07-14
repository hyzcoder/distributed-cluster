package code

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Test struct {
	cost time.Duration
	sync.RWMutex
}
var test Test

func (c *Client) TestApplyTime(connectCnt int,wg *sync.WaitGroup)  {
	record := make([]time.Time,connectCnt)
	cnt := 0
	var cost time.Duration

	msg := Msg{}
	msg.MsgType = TEST
	msg.Port = c.clientId.Port
	msg.MsgContent = []byte("request")
	for j := 0 ; j<connectCnt;j++{
		msg.Topic = strconv.Itoa(j)
		record[j]= time.Now()
		go TransmitToNode(c.leader,msg)
	}
	for cnt < connectCnt{
		select {
		case k := <- c.testApply:
			cost += time.Now().Sub(record[k])
			cnt++
		}
	}

	test.Lock()
	test.cost += cost
	test.Unlock()

	wg.Done()
}

func (c *Client) TestRandomApplyTime(connectCnt int,wg *sync.WaitGroup)  {
	record := make([]time.Time,connectCnt)
	cnt := 0
	var cost time.Duration

	msg := Msg{}
	msg.MsgType = TEST
	msg.Port = c.clientId.Port
	msg.MsgContent = []byte("request")
	for i := 0 ; i<connectCnt;i++{
		msg.Topic = strconv.Itoa(i)
		j := i
		go func(msg Msg) {
			waitTime := afterBetween(100*time.Millisecond,time.Millisecond*200)

			select {
			case <- waitTime:
			}
			record[j]= time.Now()
			TransmitToNode(c.leader,msg)
		}(msg)
	}

	for cnt < connectCnt{
		select {
		case k := <- c.testApply:
			cost += time.Now().Sub(record[k])
			//fmt.Printf("cost:%v",cost)
			cnt++
		}
	}

	test.Lock()
	test.cost += cost
	test.Unlock()

	wg.Done()
}

func RandomTestTask()  {
	wg := sync.WaitGroup{}
	clients := make([]*Client,100)
	for i:=0 ; i<100 ;i++{
		wg.Add(1)
		clients[i] = NewClient()
		clients[i].SetMasterAddr("192.168.112.147:12345")
		clients[i].SetListenPort(":200"+strconv.Itoa(i))
		clients[i].Run()
		go clients[i].TestRandomApplyTime(5,&wg)

	}
	wg.Wait()
	fmt.Println(test.cost)
	saveTestData(test.cost,2)
}

func ContinuousTestTask()  {
	wg := sync.WaitGroup{}
	clients := make([]*Client,100)
	for i:=0 ; i<45 ;i++{
		wg.Add(1)
		clients[i] = NewClient()
		clients[i].SetMasterAddr("192.168.112.147:12345")
		clients[i].SetListenPort(":200"+strconv.Itoa(i))
		clients[i].Run()
		go clients[i].TestApplyTime(20,&wg)
	}
	wg.Wait()
	fmt.Println(test.cost)
	saveTestData(test.cost,2)
	//write file

	//test = Test{}
	//for j:=0;j<1;j++{
	//	time.Sleep(time.Second*60)
	//	for i:=0 ; i<40 ;i++{
	//		wg.Add(1)
	//		go clients[i].TestApplyTime(5,&wg)
	//	}
	//	wg.Wait()
	//	fmt.Println(test.cost)
	//	//write file
	//	test = Test{}
	//}
}
