package refreshStore

import (
	"fmt"
	"github.com/matYang/goPhantom/redis"
	"github.com/matYang/goPhantom/util"
	"io/ioutil"
	"os"
)

func Clean() {
	//清理掉过期的文件，线程安全因为当查找html时会做过期判断，过期的文件将自动跳过而不会读取
	//this method is concurrency safe as any records 2 month order will be considered not exist
	now := util.GetMili()
	paths, _ := ioutil.ReadDir("./")
	//remove all expired directories
	for _, path := range paths {

		fmt.Println("[Clean] Parsing..." + path.Name())
		if !path.IsDir() {
			//if not a directory, ignore
			continue
		}

		//convert the directory name back to mili first
		pathDay, err := util.StrToI64(path.Name())
		if err != nil {
			fmt.Println("[Clean] Error at directory name to long conversion error")
			fmt.Println(err)
			continue
		}
		//if expired, remove that directory
		if (util.GetDayFromMili(now) - pathDay) >= redis.EXPIRE_DAY {
			err = util.RemoveDirectory(path.Name() + string(os.PathSeparator))
			if err != nil {
				fmt.Println("[Clean] Error when removing directory: " + path.Name())
				fmt.Println(err)
			}
		}
	}
}
