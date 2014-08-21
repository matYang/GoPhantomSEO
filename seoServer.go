package main

import (
	"fmt"
	"github.com/matYang/goPhantom/quickHash"
	"github.com/matYang/goPhantom/redis"
	"github.com/matYang/goPhantom/refreshStore"
	"github.com/matYang/goPhantom/util"
	"net/http"
	"strings"
	"time"
)

const (
	MODE   = "http"
	DOMAIN = "www.ishangke.cn"

	DEFAULT = "/"
	FRONT   = "/front"
	SEARCH  = "/search"
	COURSE  = "/course"

	GENERATE_TICKLEPPERIOD = 1000 * 60 * 60 * 6  //generate tickle every 6 hours
	CLEAN_TICKLEPPERIOD    = 1000 * 60 * 60 * 24 //clean tickle every 24 hours
)

func getUrl(sufix string) (url string) {
	url = MODE + "://" + DOMAIN + sufix
	return
}

func snapshotHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RequestURI)
	fmt.Println(r.URL.Path)
	fmt.Println(r.RemoteAddr)

	reqPath := r.RequestURI
	if reqPath == "" {
		reqPath = "/"
	}

	realUrl := getUrl("/#" + reqPath[1:])
	fmt.Println("transalated url is: " + realUrl)

	if reqPath == DEFAULT || strings.HasPrefix(reqPath, FRONT) {
		fmt.Println("[MATCH][Default/Front]")
	} else if strings.HasPrefix(reqPath, SEARCH) {
		fmt.Println("[MATCH][Search]")
	} else if strings.HasPrefix(reqPath, COURSE) {
		fmt.Println("[MATCH][Course]")
	} else {
		fmt.Println("[MISMATCH][path=" + reqPath + "]")
		http.Redirect(w, r, getUrl(DEFAULT), http.StatusFound)
		return
	}

	//create the new date directory if it does not exist
	now := util.GetMili()
	directory := util.AssembleDirectory(now)
	if util.DirectoryNotExist(directory) {
		util.CreateDirectory(directory)
	}

	previousHash, previousMili, err := redis.GetByUrl(realUrl)
	if err != nil {
		//if not exist, err will be non-nil

		hashedUrl := quickHash.Hash(realUrl)

		//add new record to redis, make the record consistent with file name parameters
		//ignore redis error here, as there is nothing we can possily do
		_, _, _ = redis.SetByUrl(realUrl, hashedUrl, util.I64ToStr(now))
		filename := util.AssembleFilename(hashedUrl, now)

		//just a blind try, since if it does not exist in redis, we do not have time info, so just guess today
		if util.FileNotExist(filename) {
			//this snapshot has not been generated yet
			//return 503 to tell crawler to come back later
			fmt.Printf("[NOT FOUND][Not In Redis] no such file or directory: %s", filename)
			http.Error(w, http.StatusText(503), 503)
			return
		} else {
			//no need to copy files, as this file must be in today's folder, it is the only guess
			//and there is no way to sync redis' time record with reality as it is lost, good thing is the date is correct
			//serve the static file
			http.ServeFile(w, r, filename)
		}
	} else {
		//exist in redis, then we can simply use previousHash and previousMili
		//still set, pass in prevousHash just for safety
		//ignore redis error here, as there is nothing we can possily do
		_, _, _ = redis.SetByUrl(realUrl, previousHash, util.I64ToStr(now))
		filename := util.AssembleFilename(previousHash, previousMili)

		if util.FileNotExist(filename) || (now-previousMili) >= redis.EXPIRE_SEC {
			//record is in DB but its corresponding file does not exist or record has expired
			//eg crawler hitting the same url many times a day or cleaned
			//nothing much we can do
			fmt.Printf("[NOT FOUND][In Redis] no such file or directory: %s", filename)
			http.Error(w, http.StatusText(503), 503)
			return
		} else {
			//in redis and file does exist, hopefully best case scenerio
			newFilename := util.AssembleFilename(previousHash, now)
			if newFilename != filename {
				//if not match, given hash is same, the date (folder) must be different
				//copy the file from previous date folder to the new date folder to indicate a crawler hit
				err = util.MoveFile(filename, newFilename)
				if err != nil {
					//copy file failed
					fmt.Printf("[FOUND][In Redis] file copy failed: %s", filename)
					fmt.Println(err)
				}
			}
			//serve the static file
			http.ServeFile(w, r, newFilename)
		}
	}
}

func scheduledEvent() {
	tickChan := time.NewTicker(CLEAN_TICKLEPPERIOD).C
	for {
		select {
		case <-tickChan:
			refreshStore.Clean()
		}
	}
}

func main() {
	//initial basic test
	fmt.Println(redis.Get("TESTGOPHANTOM_A"))
	fmt.Println(redis.Set("TESTGOPHANTOM_A", "test"))
	if result, _ := redis.Get("TESTGOPHANTOM"); result != "test" {
		panic("Initial Redis Test Failed")
	}

	//pipeline test
	//create a test connection, note that redigo does not return error immediately, if an error occues it will be returned the first time the connection is used
	//获得测试连接，注意如果连接不上，redigo不会在新建连接池或者连接时出err，而是会推迟在该连接第一次被使用的时候
	testConn := redis.GetConn()
	//发送测试请求，使用pipeline
	//launch the test request using redis pipelie, this is where errors might occur
	testConn.Send("SET", "TESTGOPHANTOM_B", util.GetMili())
	testConn.Send("GET", "TESTGOPHANTOM_B")
	//redigo使用pipeline并未抽象，方便使用pipeline对redis实现事务，因此需要手动清空pipeline
	//when using pipeline for the benifits of transaction, it must be manually flushed
	testConn.Flush()
	//receive获得（Set, Get）请求结果
	//receive gets the operation result
	fmt.Println(testConn.Receive())
	fmt.Println(testConn.Receive())
	//因为是连接池，因此每次都需要关闭连接
	//must close connection everytime one is fetched from connection pool
	testConn.Close()

	//schedule the clean event, which will clear old stuff
	go scheduledEvent()

	//decalre the http handler
	http.HandleFunc("/", snapshotHandler)
	http.ListenAndServe(":8081", nil)
}
