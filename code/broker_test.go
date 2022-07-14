package code

import (
	"bufio"
	"bytes"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestBroker_BroadCast(t *testing.T) {

	Convey("TestBroadCast",t, func() {

		//测试消息队列有topic频道和无topic频道广播情况
		Convey("01 case: Test the message queue has topic channel or no topic channel", func() {
				//在消息队列上新建一个hello频道，挂上一个消费者
				b := NewBroker()
				channel := NewChannel()
				c := ClientID{}
				channel.subscribers = append(channel.subscribers,c)
				b.topics["hello"] = channel

				//测试查询不存在的频道
				_,found0 := b.topics["new hello"]
				So(found0,ShouldBeFalse)

				//测试查询存在频道
				_,found := b.topics["hello"]
				So(found,ShouldBeTrue)
		})

		//测试向频道的客户组中发送消息后客户能否接到正确消息
		Convey("02 case: Test received data by broadcast", func() {
			channel := NewChannel()
			c := ClientID{}
			c.Port = ":12346"
			//c.msg.Topic = "hello"
			channel.subscribers = append(channel.subscribers,c)

			//模拟好网络通信环境，测试广播
			listen, err0 := net.Listen("tcp",c.Port)
			So(err0,ShouldEqual,nil)
			for _,clientId := range channel.subscribers{
				//开启协程访问具体客户
				go func(client ClientID) {
					conn,err := net.Dial("tcp",client.Port)
					if err != nil{
						t.Error(err)
					}
					defer conn.Close()
					_,err = conn.Write([]byte("hello test!"))
					if err != nil{
						t.Error(err)
					}
				}(clientId)
			}
			
			//接受广播过来的消息
			conn, err1 := listen.Accept()
			So(err1,ShouldEqual,nil)
			reader := bufio.NewReader(conn)
			buf := &bytes.Buffer{}
			_,err2 := buf.ReadFrom(reader)
			So(err2,ShouldNotEqual,io.EOF)
			data := buf.Bytes()

			//测试接受数据与广播出的是否相等
			So(data,ShouldResemble,[]byte("hello test!"))
			listen.Close()

		})
	})
}

func TestBroker_handleMsg(t *testing.T) {
	Convey("TestHandleMsg",t, func() {

		//模拟网络通信环境，测试接收数据是否与发送数据相同
		Convey("01 case: Should be able to read the received data correctly", func() {
			//测试net.Listen
			listen, err0 := net.Listen("tcp",":12345")
			So(err0,ShouldEqual,nil)
			go func() {
				conn,err := net.Dial("tcp",":12345")
				if err != nil{
					t.Error(err)
				}
				defer conn.Close()

				msg := Msg{}
				msg.MsgType = SUBSCRIBE
				jsonMsg,_ := json.Marshal(msg)

				_,err = conn.Write(jsonMsg)
				if err != nil{
					t.Error(err)
				}
			}()

			//测试listen.Aceept
			conn, err1 := listen.Accept()
			So(err1,ShouldEqual,nil)

			//测试ReadFrom
			reader := bufio.NewReader(conn)
			buf := &bytes.Buffer{}
			_,err2 := buf.ReadFrom(reader)
			So(err2,ShouldNotEqual,io.EOF)

			//测试接收数据消息类型是否与发送数据消息类型相同
			data := buf.Bytes()
			msg0 := Msg{}
			err3 := json.Unmarshal(data,&msg0)
			So(err3,ShouldEqual,nil)
			So(msg0.MsgType,ShouldEqual,SUBSCRIBE)

			//测试接受的ip和发送请求的远程节点ip地址是否相同
			remoteAddr := conn.RemoteAddr().String()
			index := strings.Index(remoteAddr,":")
			remoteAddr = remoteAddr[:index]
			addr := conn.LocalAddr().String()
			index = strings.Index(addr,":")
			addr = addr[:index]
			So(addr,ShouldEqual,remoteAddr)

			listen.Close()
		})
	})
}

func TestBroker_PutNetMsg(t *testing.T) {
	Convey("TestPutNetMsg",t, func() {

		//测试节点结构体数组的json.Marshal和json.UnMarshal转化是否正确
		Convey("01 case: Nodes Data should be converted correctly through json.UnMarshal", func() {
			 var nodes []NodeState
			 node := NewNodeState(true,2.23,"www.baidu.com")
			 nodes = append(nodes,node)
			jsonNodes,err := json.Marshal(nodes)
			So(err,ShouldEqual,nil)

			var nodes0 []NodeState
			err1 := json.Unmarshal(jsonNodes,&nodes0)
			So(err1,ShouldEqual,nil)
			So(nodes0,ShouldResemble,nodes)

		})
	})
}

func TestBroker_PingNodes(t *testing.T) {
	Convey("TestPingNodes",t, func() {

		//测试ping节点后返回网络信息是否正确
		Convey("01 case: Test function interface when net normal", func() {
			ch := NewChannel()
			ch.subscribers = append(ch.subscribers,ClientID{Ip: "192.168.112.160"})
			ch.subscribers = append(ch.subscribers,ClientID{Ip:	"www.baidu.com"})
			nodes := pingNetNodes(ch)
			So(nodes[0].PingFlag,ShouldBeTrue)
			So(nodes[1].PingFlag,ShouldBeFalse)

		})

		//测试多个协程对切片类型操作时，最终切片中的结果是否正确
		Convey("02 case: Test Slice data when many goroutine operate it", func() {
			var nodes2 []NodeState
			var rw sync.RWMutex
			wg := sync.WaitGroup{}
			wg.Add(100)
			for i := 0; i<100 ; i++	{
				t := i
				go func() {
					rw.Lock()
					nodes2 = append(nodes2,NewNodeState(true,float64(t),""))
					rw.Unlock()
					wg.Done()
				}()
			}
			wg.Wait()

			cnt := len(nodes2)
			So(cnt,ShouldEqual,100)

		})

		//测试正则表达提取信息是否正确
		Convey("03 case: Test the information extracted by regular expression", func() {
			//测试含有字母和特殊符号字符串
			rex := regexp.MustCompile("[0-9|.]+")
			m := rex.FindAllString("ad'!'sdg;@#%192.168.112.160,time:2.13",-1)
			So(m[0],ShouldEqual,"192.168.112.160")
			So(m[1],ShouldEqual,"2.13")
			//测试空串
			m = rex.FindAllString("",-1)
			So(m,ShouldEqual,nil)

			//测试含有空格字符串
			m = rex.FindAllString("ip  192.168.112.160  time 2.13",-1)
			So(m[0],ShouldEqual,"192.168.112.160")
			So(m[1],ShouldEqual,"2.13")

		})
	})
}