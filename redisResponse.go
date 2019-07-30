package flyfish

import (
	"github.com/sniperHW/flyfish/errcode"
	"sync/atomic"
)

func onRedisResp(ctx *processContext) {

	newVersion := int64(0)
	//先发出响应

	if ctx.errno != errcode.ERR_STALE_CACHE && !ctx.replyOnDbOk {
		newVersion = ctx.fields["__version__"].GetInt()
		ctx.reply(ctx.errno, ctx.fields, newVersion)
	}

	Debugln("onRedisResp key:", ctx.getCmdType(), ctx.getUniKey(), newVersion, ctx.errno)

	ckey := ctx.getCacheKey()
	if ctx.errno == errcode.ERR_STALE_CACHE {
		/*  redis中的数据与flyfish key不一致
		 *  将ckey重置为cache_new，强制从数据库取值刷新redis
		 */
		ckey.unit.onRedisStale(ckey, ctx)

		/*ckey.reset()
		ctx.writeBackFlag = write_back_none //数据存在执行update
		ctx.redisFlag = redis_none
		//到数据库加载
		if !ckey.unit.pushSqlLoadReqOnRedisReply(ctx) {
			atomic.AddInt32(&cmdCount, -1)
			//ctx.reply(errcode.ERR_BUSY, nil, -1)
		}*/

	} else {
		if ctx.errno == errcode.ERR_OK {
			if ctx.redisFlag == redis_get || ctx.redisFlag == redis_set_only {
				ckey.setOK(newVersion)
			} else if ctx.redisFlag == redis_del {
				ckey.setMissing()
				//投递sql删除请求
				ckey.unit.pushSqlWriteBackReq(ctx)
			} else {
				ckey.setOK(newVersion)
				ckey.unit.pushSqlWriteBackReq(ctx)
			}
		}

		if !ctx.replyOnDbOk {
			ckey.processQueueCmd()
		}
	}

	atomic.AddInt32(&redisReqCount, -1)
}
