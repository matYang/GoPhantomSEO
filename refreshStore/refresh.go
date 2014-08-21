package refreshStore

import (
	"fmt"
	"github.com/matYang/goPhantom/redis"
	"github.com/matYang/goPhantom/util"
	"io/ioutil"
	"os"
)

const (
	PRODUCEFILE = "HOTSPOTURLS.txt"
)

func Clean() {
	//this method is concurrency safe as any records 2 month order will be considered not exist
	now := util.GetMili()
	paths, _ := ioutil.ReadDir("./")
	//remove all expired directories
	for _, path := range paths {
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

func Generate() {
	if !util.FileNotExist(PRODUCEFILE) {
		util.RemoveFile(PRODUCEFILE)
	}

	f, err := os.OpenFile(PRODUCEFILE, os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	conn := redis.GetConn()
	defer conn.Close()
	conn.Send("Keys", "*ishangke*")
	conn.Flush()
	data, err := conn.Receive()

	keys := []string{}
	ok := false
	if keys, ok = data.([]string); !ok {
		fmt.Println("[ERROR] Keys Returned Value not String Slice")
		panic("[ERROR] Keys Returned Value not String Slice")
	} else {
		for _, key := range keys {
			hash, mili, err := redis.GetByUrl(key)
			if err != nil {
				fmt.Println("[Generate] Error when getting from Redis")
			}
			_ = util.AssembleFilename(hash, mili)
		}
	}

	//write to that file, with text being a line
	//_, err := f.WriteString(text)
}
