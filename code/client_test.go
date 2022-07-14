package code

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey" //前面加点号"."，以减少冗余的代码
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"
)

func TestClient_Subscribe(t *testing.T) {
	/**
	模拟上下文环境
	*/
	//设置测试的主节点为本机，使用端口为12345
	//SetNewBroker(":12345")

	//开启主节点端口监听（所有测试结束前端口会一直监听）
	go func() {
		listen, err := net.Listen("tcp",":12345")
		if err != nil{
			t.Errorf("端口监听失败 %s\n",err.Error())
			return
		}
		for{
			_, err1 := listen.Accept()
			if err1 != nil{
				t.Errorf("端口监听失败 %s",err1.Error())
				return
			}
		}
	}()
	time.Sleep(time.Millisecond)

	Convey("TestSubscribe",t, func() {
		//mock出net.dial，测试sub函数
		Convey("01 case: Should be return error when net.Dial error", func() {
			patches := ApplyFunc(net.Dial, func(_ string,_ string) (net.Conn,error) {
				return nil, errors.New("dial is failed")
			})
			defer patches.Reset()

			c := NewClient()
			err := c.Subscribe("topic1")
			So(err,ShouldResemble,errors.New("Subscribe Error:dial is failed "))
		})

		//mock出json.Marshal，测试sub函数
		Convey("02 case: Should be return error when json.Marshal error", func() {
			patches := ApplyFunc(json.Marshal, func(_ interface{}) ([]byte,error){
				fmt.Print("test....")
				return []byte(""),errors.New("marshal is failed")
			})
			defer patches.Reset()

			c := NewClient()
			err := c.Subscribe("topic2")
			So(err,ShouldResemble,errors.New("marshal is failed"))
		})

		//测试空串
		Convey("03 case: Should be return error when topic=\"\"", func() {
			c := NewClient()
			err := c.Subscribe("")
			So(err,ShouldNotEqual,nil)
		})

		//测试特殊符号串
		Convey("04 case: Should be return success when topic=special characters", func() {
			c := NewClient()
			c.leader = ":12345"
			err := c.Subscribe("!@#$%^&*()\n_)(_ =++")
			So(err,ShouldEqual,nil)
		})

		//测试字母和数字
		Convey("05 case: Should be return nil when all are successful", func() {
			c := NewClient()
			c.leader = ":12345"
			err := c.Subscribe("topic5")
			So(err,ShouldEqual,nil)
		})

	})
}

//与测试Subscribe函数相一致
func TestClient_UnSubscribe(t *testing.T) {
	Convey("TestUnSubscribe",t, func() {

		Convey("01 case: Should be return error when net.Dial error", func() {
			patches := ApplyFunc(net.Dial, func(_ string,_ string) (net.Conn,error) {
				return nil, errors.New("dial is failed")
			})
			defer patches.Reset()

			c := NewClient()
			err := c.UnSubscribe("topic1")
			So(err,ShouldResemble,errors.New("UnSubscribe Error:dial is failed "))
		})

		Convey("02 case: Should be return error when json.Marshal error", func() {
			patches := ApplyFunc(json.Marshal, func(_ interface{}) ([]byte,error){
				fmt.Print("test....")
				return []byte(""),errors.New("marshal is failed")
			})
			defer patches.Reset()

			c := NewClient()
			err := c.UnSubscribe("topic2")
			So(err,ShouldResemble,errors.New("marshal is failed"))
		})

		Convey("03 case: Should be return error when topic=\"\"", func() {
			c := NewClient()
			err := c.UnSubscribe("")
			So(err,ShouldNotEqual,nil)
		})

		Convey("04 case: Should be return success when topic=special characters", func() {
			c := NewClient()
			c.leader = ":12345"
			err := c.UnSubscribe("!@#$%^&*()\n_)(_ =++")
			So(err,ShouldEqual,nil)
		})


		Convey("05 case: Should be return nil when all are successful", func() {
			c := NewClient()
			c.leader = ":12345"
			err := c.UnSubscribe("topic5")
			So(err,ShouldEqual,nil)
		})

	})
}


func TestClient_Publish(t *testing.T) {
	Convey("TestPublish",t, func() {

		//mock出net.dial，测试pub函数
		Convey("01 case: Should be return error when net.Dial error", func() {
			patches := ApplyFunc(net.Dial, func(_ string,_ string) (net.Conn,error) {
				return nil, errors.New("dial is failed")
			})
			defer patches.Reset()

			msg := Msg{}
			msg.MsgContent=[]byte("hello")
			c := NewClient()
			err := c.Publish("topic1",msg.MsgContent)
			So(err,ShouldResemble,errors.New("Publish Error:dial is failed "))
		})

		//mock出json.Marshal，测试pub函数
		Convey("02 case: Should be return error when json.Marshal error", func() {
			patches := ApplyFunc(json.Marshal, func(_ interface{}) ([]byte,error){
				fmt.Print("test....")
				return []byte(""),errors.New("marshal is failed")
			})
			defer patches.Reset()

			c := NewClient()
			msg := Msg{}
			msg.MsgContent=[]byte("hello")
			err := c.Publish("topic2",msg.MsgContent)
			So(err,ShouldResemble,errors.New("marshal is failed"))
		})

		//测试空消息
		Convey("03 case: Should be return error when topic=\"\"", func() {
			c := NewClient()
			msg := Msg{}
			msg.MsgContent=[]byte("")
			err := c.Publish("",msg.MsgContent)
			So(err,ShouldNotEqual,nil)
		})

		//测试特殊符号的消息
		Convey("04 case: Should be return success when topic and msg are special characters", func() {
			c := NewClient()
			c.leader = ":12345"
			msg := Msg{}
			msg.MsgContent=[]byte("!@#$%$^|r\r%&*()(*&^_YTR")
			err := c.Publish("!@#$%    ^&*()\n_)(_ =++",msg.MsgContent)
			So(err,ShouldEqual,nil)
		})

		//测试为字母和数字的消息
		Convey("05 case: Should be return nil when all are successful", func() {
			c := NewClient()
			c.leader = ":12345"
			msg := Msg{}
			msg.MsgContent=[]byte("hello233")
			err := c.Publish("topic5",msg.MsgContent)
			So(err,ShouldEqual,nil)
		})
	})
}

func TestClient_showReceiveMsg(t *testing.T) {
	Convey("TestShowReceiveMsg",t, func() {

		Convey("01 case: Should be able to read the received data correctly", func() {
			/**
				模拟网络通信环境，测试接收数据是否与发送数据相同
			 */
			listen, err0 := net.Listen("tcp",":12346")
			So(err0,ShouldEqual,nil)
			//开启协程进行网络访问
			go func() {
				conn,err := net.Dial("tcp",":12346")
				if err != nil{
					t.Error(err)
				}
				//So(err,ShouldEqual,nil)
				defer conn.Close()
				_,err = conn.Write([]byte("hello"))
				if err != nil{
					t.Error(err)
				}
			}()

			//接受传来数据，进行数据对比
			conn, err1 := listen.Accept()
			So(err1,ShouldEqual,nil)
			reader := bufio.NewReader(conn)
			buf := &bytes.Buffer{}
			_,err2 := buf.ReadFrom(reader)
			So(err2,ShouldNotEqual,io.EOF)
			data := buf.Bytes()
			So(data,ShouldResemble,[]byte("hello"))
			listen.Close()

		})

		//测试json.Marshal和json.UnMarshal的正常数据转换
		Convey("02 case:  Data should be converted correctly through json.UnMarshal", func() {
			msg :=Msg{}
			msg.Topic = "hello"
			msg.MsgType = UNSUBSCRIBE
			msg.MsgContent = []byte{}
			jsonMsg, err := json.Marshal(msg)
			So(err,ShouldEqual,nil)

			msg1 := Msg{}
			err = json.Unmarshal(jsonMsg,&msg1)
			So(err,ShouldEqual,nil)
			So(msg1,ShouldResemble,msg)
		})

		//测试json.Marshal和json.UnMarshal的空数据转换
		Convey("03 case:  Empty data should be converted correctly through json.UnMarshal", func() {
			msg :=Msg{}
			jsonMsg, err := json.Marshal(msg)
			So(err,ShouldEqual,nil)

			msg1 := Msg{}
			err = json.Unmarshal(jsonMsg,&msg1)
			So(err,ShouldEqual,nil)
			So(msg1,ShouldResemble,msg)


			var buf []byte
			var buff []byte
			jsonBuf, err1 := json.Marshal(buf)
			So(err1,ShouldEqual,nil)
			err = json.Unmarshal(jsonBuf,&buff)
			So(buff,ShouldEqual,buf)


		})

		//测试文件的存储和读取
		Convey("04 case: Proper file writing and reading should be completed", func() {
			//测试存储
			err := ioutil.WriteFile("nodeState.txt", []byte("Message..."), 0666)
			So(err,ShouldEqual,nil)

			//测试读取
			f,err1 := os.Open("nodeState.txt")
			So(err1,ShouldEqual,nil)
			defer f.Close()
			msgBytes,err2 := ioutil.ReadAll(f)
			So(err2,ShouldEqual,nil)
			So(msgBytes,ShouldResemble,[]byte("Message..."))

			err3 := os.Remove("nodeState.txt")
			So(err3,ShouldEqual,nil)
		})
	})
}