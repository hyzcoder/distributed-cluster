package code

import (
	. "github.com/smartystreets/goconvey/convey"
	"net"
	"testing"
	"time"
)

func TestSentinel_TimedCheckNode(t *testing.T) {
	Convey("TestCheckNode",t, func() {

		Convey("01 case: test correct back when connect not timed out ", func() {
			_,err := net.DialTimeout("tcp",":12345",time.Second)
			So(err,ShouldEqual,nil)
		})

		Convey("02 case: test error back when connect is timed out", func() {
			_,err := net.DialTimeout("tcp",":12345",time.Microsecond)
			So(err,ShouldNotEqual,nil)
		})

	})
}

func TestSentinel_getSentinelOnlineCounts(t *testing.T) {
	Convey("TestGetOnlineCounts",t, func() {
		s := NewSentinel()
		Convey("01 case: test get online sentinel counts when sentinelList is empty", func() {
			cnt := s.getSentinelOnlineCounts()
			So(cnt,ShouldEqual,0)

		})

		Convey("02 case: test get online sentinel counts when sentinelList is not empty", func() {
			s.sentinelList.sentinels[ClientID{Ip: "192.168.112.201"}] = true
			s.sentinelList.sentinels[ClientID{Ip: "192.168.112.190"}] = false
			cnt := s.getSentinelOnlineCounts()
			So(cnt,ShouldEqual,1)
		})
	})

}

func TestSentinel_ElectMaster(t *testing.T) {
	Convey("TestElectMaster",t, func() {

		Convey("01 case: test timeout when some nodes not back message", func() {
			timeout := afterBetween(time.Millisecond*150,time.Millisecond*200)
			candidateCount := 1
			flag := true
			s := NewSentinel()
			for candidateCount != 0 {
				select {
				case <-s.candidateResponse:
					candidateCount--
				case <-timeout:
					candidateCount =0
					flag = false
				}
			}
			ShouldBeFalse(flag)
		})


	})
}

func TestSentinel_ElectLeadSentinel(t *testing.T) {
	Convey("TestElectLeadSentinel",t, func() {

		Convey("01 case: test elect timeout", func() {
			since := time.Now()
			waitRandomTime := afterBetween(50*time.Millisecond,100*time.Millisecond)
			select {
			case <- waitRandomTime:
				dur := time.Now().Sub(since)
				bo := dur>=50*time.Millisecond && dur<=100*time.Millisecond
				ShouldBeTrue(bo,true)
			}
		})


		Convey("02 case: test it should become lead sentinel when one more than half is agreed", func() {
			quorumSize := 3
			voteGrantedCount := 0
			s := NewSentinel()
			go func() {
				for i:=0;i<1+quorumSize/2;i++{
					s.voteGranted <- true
					time.Sleep(time.Millisecond)
				}
			}()
			for voteGrantedCount <= quorumSize/2{
				select{
				case <- s.voteGranted:
					voteGrantedCount++
				}
			}
			So(voteGrantedCount,ShouldEqual,2)

		})

		Convey("03 case: test it shouldn't become leading sentinel when less than half agreed", func() {
			quorumSize := 3
			voteGrantedCount := 0
			flag := true
			timeoutChan := afterBetween(50*time.Millisecond,100*time.Millisecond)
			s := NewSentinel()
			go func() {
					s.voteGranted <- true
			}()

			for voteGrantedCount <= quorumSize/2{
				select {
				case <- s.voteGranted:
					voteGrantedCount++
				case <- timeoutChan:
					flag = false
					voteGrantedCount = quorumSize+1
				}
			}
			So(flag,ShouldEqual,false)
		})

	})
}





