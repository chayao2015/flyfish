package main 

import(
	"fmt"
	kclient "flyfish/client"
	"flyfish/errcode"
	"github.com/sniperHW/kendynet/golog"
	"github.com/sniperHW/kendynet"
	"strings"
)

func Set(c *kclient.Client,i int) {
	fields := map[string]interface{}{}
	fields["age"] = 37
	fields["phone"] = strings.Repeat("a",1024)
	fields["name"] = "sniperHW"
	key := fmt.Sprintf("%s:%d","huangwei",i)

	set := c.Set("users1",key,fields)
	set.Exec(func(ret *kclient.Result) {

		if ret.ErrCode != errcode.ERR_OK {
			fmt.Println(errcode.GetErrorStr(ret.ErrCode))
		} else {
			fmt.Println("set ok")
		}
	})
}


func main() {

	golog.DisableStdOut()
	outLogger := golog.NewOutputLogger("log", "flyfish get", 1024*1024*50)
	kendynet.InitLogger(outLogger,"flyfish get")

	c := kclient.OpenClient("localhost:10012")//eventQueue)

	Set(c,1)
	Set(c,2)
	Set(c,3)
	Set(c,4)

	//eventQueue.Run()

	sigStop := make(chan bool)
	_, _ = <-sigStop
}