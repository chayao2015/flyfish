package client

import (
	"github.com/golang/protobuf/proto"
	"github.com/sniperHW/flyfish/codec"
	"github.com/sniperHW/flyfish/errcode"
	protocol "github.com/sniperHW/flyfish/proto"
	"github.com/sniperHW/kendynet/util"
	"sync/atomic"
	"time"
)

type Field protocol.Field

func (this *Field) IsNil() bool {
	return (*protocol.Field)(this).IsNil()
}

func (this *Field) GetString() string {
	return (*protocol.Field)(this).GetString()
}

func (this *Field) GetUint() uint64 {
	return (*protocol.Field)(this).GetUint()
}

func (this *Field) GetInt() int64 {
	return (*protocol.Field)(this).GetInt()
}

func (this *Field) GetFloat() float64 {
	return (*protocol.Field)(this).GetFloat()
}

func (this *Field) GetBlob() []byte {
	return (*protocol.Field)(this).GetBlob()
}

func (this *Field) GetValue() interface{} {
	return (*protocol.Field)(this).GetValue()
}

const (
	wait_none   = 0
	wait_send   = 1
	wait_resp   = 2
	wait_remove = 3
)

type cmdContext struct {
	seqno     int64
	deadline  time.Time
	timestamp int64
	status    int
	cb        callback
	req       proto.Message
	heapIdx   uint32
	key       string
}

func (this *cmdContext) Less(o util.HeapElement) bool {
	return o.(*cmdContext).deadline.After(this.deadline)
}

func (this *cmdContext) GetIndex() uint32 {
	return this.heapIdx
}

func (this *cmdContext) SetIndex(idx uint32) {
	this.heapIdx = idx
}

func (this *cmdContext) onError(errCode int32) {
	this.cb.onError(errCode)
}

func (this *cmdContext) onResult(r interface{}) {
	this.cb.onResult(r)
}

type StatusCmd struct {
	conn  *Conn
	req   proto.Message
	seqno int64
}

func (this *StatusCmd) AsyncExec(cb func(*StatusResult)) {
	context := &cmdContext{
		seqno: this.seqno,
		cb: callback{
			tt: cb_status,
			cb: cb,
		},
		req: this.req,
	}
	this.conn.exec(context)
}

func (this *StatusCmd) Exec() *StatusResult {
	respChan := make(chan *StatusResult)
	this.AsyncExec(func(r *StatusResult) {
		respChan <- r
	})
	return <-respChan
}

type SliceCmd struct {
	conn  *Conn
	req   proto.Message
	seqno int64
}

func (this *SliceCmd) AsyncExec(cb func(*SliceResult)) {
	context := &cmdContext{
		seqno: this.seqno,
		cb: callback{
			tt: cb_slice,
			cb: cb,
		},
		req: this.req,
	}
	this.conn.exec(context)
}

func (this *SliceCmd) Exec() *SliceResult {
	respChan := make(chan *SliceResult)
	this.AsyncExec(func(r *SliceResult) {
		respChan <- r
	})
	return <-respChan
}

func makeReqCommon(table string, key string, seqno int64, timeout int64, respTimeout int64) *protocol.ReqCommon {
	return &protocol.ReqCommon{
		Seqno:       seqno,       //proto.Int64(atomic.AddInt64(&this.seqno, 1)),
		Table:       table,       //proto.String(table),
		Key:         key,         // proto.String(key),
		Timeout:     timeout,     //proto.Int64(int64(ServerTimeout)),
		RespTimeout: respTimeout, //proto.Int64(int64(ClientTimeout)),
	}
}

func (this *Conn) Get(table, key string, fields ...string) *SliceCmd {

	if len(fields) == 0 {
		return nil
	}

	req := &protocol.GetReq{
		Head:   makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
		Fields: fields,
		All:    false, //proto.Bool(false),
	}

	return &SliceCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

func (this *Conn) GetAll(table, key string) *SliceCmd {
	req := &protocol.GetReq{
		Head: makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
		All:  true, //proto.Bool(true),
	}

	return &SliceCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

func (this *Conn) Set(table, key string, fields map[string]interface{}, version ...int64) *StatusCmd {

	if len(fields) == 0 {
		return nil
	}

	req := &protocol.SetReq{
		Head: makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
	}

	if len(version) > 0 {
		req.Head.Version = proto.Int64(version[0])
	}

	for k, v := range fields {
		req.Fields = append(req.Fields, protocol.PackField(k, v))
	}

	return &StatusCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

func (this *Conn) SetNx(table, key string, fields map[string]interface{}) *StatusCmd {
	if len(fields) == 0 {
		return nil
	}

	req := &protocol.SetNxReq{
		Head: makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
	}

	for k, v := range fields {
		req.Fields = append(req.Fields, protocol.PackField(k, v))
	}

	return &StatusCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

//当记录的field == old时，将其设置为new,并返回field的实际值(如果filed != old,将返回filed的原值)
func (this *Conn) CompareAndSet(table, key, field string, oldV, newV interface{}, version ...int64) *SliceCmd {

	if oldV == nil || newV == nil {
		return nil
	}

	req := &protocol.CompareAndSetReq{
		Head: makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
		New:  protocol.PackField(field, newV),
		Old:  protocol.PackField(field, oldV),
	}

	if len(version) > 0 {
		req.Head.Version = proto.Int64(version[0])
	}

	return &SliceCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}

}

//当记录不存在或记录的field == old时，将其设置为new.并返回field的实际值(如果记录存在且filed != old,将返回filed的原值)
func (this *Conn) CompareAndSetNx(table, key, field string, oldV, newV interface{}, version ...int64) *SliceCmd {
	if oldV == nil || newV == nil {
		return nil
	}

	req := &protocol.CompareAndSetNxReq{
		Head: makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
		New:  protocol.PackField(field, newV),
		Old:  protocol.PackField(field, oldV),
	}

	if len(version) > 0 {
		req.Head.Version = proto.Int64(version[0])
	}

	return &SliceCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

func (this *Conn) Del(table, key string, version ...int64) *StatusCmd {

	req := &protocol.DelReq{
		Head: makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
	}

	if len(version) > 0 {
		req.Head.Version = proto.Int64(version[0])
	}

	return &StatusCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}

}

func (this *Conn) IncrBy(table, key, field string, value int64, version ...int64) *SliceCmd {
	req := &protocol.IncrByReq{
		Head:  makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
		Field: protocol.PackField(field, value),
	}

	if len(version) > 0 {
		req.Head.Version = proto.Int64(version[0])
	}

	return &SliceCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

func (this *Conn) DecrBy(table, key, field string, value int64, version ...int64) *SliceCmd {
	req := &protocol.DecrByReq{
		Head:  makeReqCommon(table, key, atomic.AddInt64(&this.seqno, 1), int64(ServerTimeout), int64(ClientTimeout)),
		Field: protocol.PackField(field, value),
	}

	if len(version) > 0 {
		req.Head.Version = proto.Int64(version[0])
	}

	return &SliceCmd{
		conn:  this,
		req:   req,
		seqno: req.Head.GetSeqno(),
	}
}

func (this *Conn) onGetResp(resp *protocol.GetResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {
		ret := SliceResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		if ret.ErrCode == errcode.ERR_OK {
			ret.Fields = map[string]*Field{}
			for _, v := range resp.Fields {
				ret.Fields[v.GetName()] = (*Field)(v)
			}
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onSetResp(resp *protocol.SetResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := StatusResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onSetNxResp(resp *protocol.SetNxResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := StatusResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onCompareAndSetResp(resp *protocol.CompareAndSetResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := SliceResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		if ret.ErrCode == errcode.ERR_OK || ret.ErrCode == errcode.ERR_CAS_NOT_EQUAL {
			ret.Fields = map[string]*Field{}
			ret.Fields[resp.GetValue().GetName()] = (*Field)(resp.GetValue())
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onCompareAndSetNxResp(resp *protocol.CompareAndSetNxResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := SliceResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		if ret.ErrCode == errcode.ERR_OK || ret.ErrCode == errcode.ERR_CAS_NOT_EQUAL {
			ret.Fields = map[string]*Field{}
			ret.Fields[resp.GetValue().GetName()] = (*Field)(resp.GetValue())
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onDelResp(resp *protocol.DelResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := StatusResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onIncrByResp(resp *protocol.IncrByResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := SliceResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		if errcode.ERR_OK == ret.ErrCode {
			ret.Fields = map[string]*Field{}
			ret.Fields[resp.NewValue.GetName()] = (*Field)(resp.NewValue)
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onDecrByResp(resp *protocol.DecrByResp) {
	c := this.removeContext(resp.Head.GetSeqno())
	if nil != c {

		ret := SliceResult{
			Key:     resp.Head.GetKey(),
			ErrCode: resp.Head.GetErrCode(),
			Version: resp.Head.GetVersion(),
		}

		if errcode.ERR_OK == ret.ErrCode {
			ret.Fields = map[string]*Field{}
			ret.Fields[resp.NewValue.GetName()] = (*Field)(resp.NewValue)
		}

		this.c.doCallBack(c.cb, &ret)
	}
}

func (this *Conn) onMessage(msg *codec.Message) {
	this.eventQueue.Post(func() {
		name := msg.GetName()
		switch name {
		//case "*proto.PingResp":
		case "*proto.GetResp":
			this.onGetResp(msg.GetData().(*protocol.GetResp))
		case "*proto.SetResp":
			this.onSetResp(msg.GetData().(*protocol.SetResp))
		case "*proto.SetNxResp":
			this.onSetNxResp(msg.GetData().(*protocol.SetNxResp))
		case "*proto.CompareAndSetResp":
			this.onCompareAndSetResp(msg.GetData().(*protocol.CompareAndSetResp))
		case "*proto.CompareAndSetNxResp":
			this.onCompareAndSetNxResp(msg.GetData().(*protocol.CompareAndSetNxResp))
		case "*proto.DelResp":
			this.onDelResp(msg.GetData().(*protocol.DelResp))
		case "*proto.IncrByResp":
			this.onIncrByResp(msg.GetData().(*protocol.IncrByResp))
		case "*proto.DecrByResp":
			this.onDecrByResp(msg.GetData().(*protocol.DecrByResp))
		case "*proto.ScanResp":
			this.onScanResp(msg.GetData().(*protocol.ScanResp))
		default:
		}
	})

}
