package main

import (
	"fmt"
	kclient "github.com/sniperHW/flyfish/client"
	"github.com/sniperHW/flyfish/errcode"
	"github.com/sniperHW/kendynet/golog"
	"os"
)

func main() {

	kclient.InitLogger(golog.New("flyfish client", golog.NewOutputLogger("log", "flyfish client", 1024*1024*50)))

	c := kclient.OpenClient(os.Args[1], false)

	//c.GetAll("users1", "sniperHW").Exec()

	//c.Kick("users1", "sniperHW").AsyncExec(func(r *kclient.StatusResult) {
	//	fmt.Println("kick", errcode.GetErrorStr(r.ErrCode))
	//})

	//c.GetAll("users1", "sniperHW").AsyncExec(func(r *kclient.SliceResult) {
	//	fmt.Println(errcode.GetErrorStr(r.ErrCode))
	//})

	r := c.GetAll("users1", "sniperHW").Exec()
	fmt.Println("get", errcode.GetErrorStr(r.ErrCode))

	cc := make(chan struct{})

	_ = <-cc

	/*r1 := c.Kick("users1", "sniperHW").Exec()
	fmt.Println(errcode.GetErrorStr(r1.ErrCode))

	r2 := c.Kick("users1", "sniperHW").Exec()
	fmt.Println(errcode.GetErrorStr(r2.ErrCode))

	r3 := c.Get("users1", "sniperHW", "name", "age", "phone").Exec()
	if r3.ErrCode != errcode.ERR_OK {
		fmt.Println("Get Error:", errcode.GetErrorStr(r3.ErrCode), r3)
		return
	}

	fmt.Println(r3.Fields["name"].GetValue(), r3.Fields["age"].GetValue(), r3.Fields["phone"].GetValue())
	*/

}
