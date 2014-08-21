package refreshStore

import (
	"fmt"
	"github.com/matYang/goPhantom/redis"
	"github.com/matYang/goPhantom/util"
	"io/ioutil"
	"os"
)

func Clean() {
	//this method is concurrency safe as any records 2 month order will be considered not exist
	now := util.GetMili()
	paths, _ := ioutil.ReadDir("./")
	//remove all expired directories
	for _, path := range paths {
		if path.Name() == util.TEMPFILE || path.Name() == util.PRODUCEFILE {
			continue
		}
		//convert the directory name back to mili first
		pathMili, err := util.StrToI64(path.Name())
		if err != nil {
			fmt.Println("[Clean] Error at directory name to long conversion error")
			fmt.Println(err)
			return
		}
		//if expired, remove that directory
		if (now - pathMili) >= redis.EXPIRE_SEC {
			err = util.RemoveDirectory(path.Name() + string(os.PathSeparator))
			if err != nil {
				fmt.Println("[Clean] Error when removing directory: " + path.Name())
				fmt.Println(err)
			}
		}
	}
}
