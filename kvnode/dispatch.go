package kvnode

import (
	"github.com/sniperHW/flyfish/net"
	"github.com/sniperHW/flyfish/proto"
	"github.com/sniperHW/kendynet"
	"sync"
	"sync/atomic"
	"time"
)

var sessions sync.Map
var clientCount int64

type handler func(*KVNode, *cliConn, *net.Message)

type dispatcher struct {
	handlers map[uint16]handler
}

func (this *dispatcher) Register(cmd uint16, h handler) {
	_, ok := this.handlers[cmd]
	if ok {
		return
	}

	this.handlers[cmd] = h
}

func (this *dispatcher) Dispatch(kvnode *KVNode, session kendynet.StreamSession, cmd uint16, msg *net.Message) {
	if nil != msg {
		if cmd == uint16(proto.CmdType_Ping) {
			resp := net.NewMessage(net.CommonHead{}, &proto.PingResp{
				Timestamp: time.Now().UnixNano(),
			})
			session.Send(resp)
			return
		}

		handler, ok := this.handlers[cmd]
		if ok {
			//投递给线程池处理
			kvnode.pushNetCmd(handler, session.GetUserData().(*cliConn), msg)
		} else {
			logger.Errorln("invaild cmd", cmd)
		}
	} else {
		logger.Errorln("msg is nil")
	}
}

func (this *dispatcher) OnClose(session kendynet.StreamSession, reason string) {
	u := session.GetUserData()
	if nil != u {
		switch u.(type) {
		case *cliConn:
			u.(*cliConn).clear()
		}
	}
	atomic.AddInt64(&clientCount, -1)
	sessions.Delete(session)

	/*u := session.GetUserData()
	if nil != u {
		u.(*scaner).close()
	}
	atomic.AddInt64(&clientCount, -1)
	sessions.Delete(session)*/
}

func (this *dispatcher) OnNewClient(session kendynet.StreamSession) {
	atomic.AddInt64(&clientCount, 1)
	session.SetUserData(
		&cliConn{
			session:  session,
			replyers: map[int64]*replyer{},
		},
	)
	sessions.Store(session, session)
}

func ping(kvnode *KVNode, conn *cliConn, msg *net.Message) {
	head := msg.GetHead()
	req := msg.GetData().(*proto.PingReq)
	resp := net.NewMessage(head, &proto.PingResp{
		Timestamp: req.GetTimestamp(),
	})
	conn.send(resp)
}
