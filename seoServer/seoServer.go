package main

import (
	"fmt"
	"github.com/matYang/goPhantom/quickHash"
	"github.com/matYang/goPhantom/redis"
	"github.com/matYang/goPhantom/refreshStore"
	"github.com/matYang/goPhantom/util"
	"io/ioutil"
	"net/http"
	"os"
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

	CLEAN_TICKLEPPERIOD = 60 * 60 * 24 //clean tickle every 24 hours
)

//定义一个Record struct，类似于Java里的class
type Record struct {
	url       string
	hashedUrl string
	mili      int64
}

//给Record定义一个方法，该方法用于将一个record变成一条string用于存在临时文件中
func (r Record) toFileString() string {
	return r.url + "@" + util.AssembleFilename(r.hashedUrl, r.mili)
}

//新建url通道，该通道用于让各个处理http请求的goroutine将新的请求记录发送至唯一的一条专门用于存储记录的goroutine中，并发安全
var (
	urlChan chan Record
)

func init() {
	urlChan = make(chan Record)
}

//拼接请求的url
func getUrl(sufix string) (url string) {
	url = MODE + "://" + DOMAIN + sufix
	return
}

//该function用于处理每个请求，将会以goroutine的形式被调用
func snapshotHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := r.RequestURI
	//无视掉静态文件请求
	if strings.Contains(reqPath, ".css") || strings.Contains(reqPath, ".js") || strings.Contains(reqPath, ".ico") || strings.Contains(reqPath, ".png") || strings.Contains(reqPath, ".jpg") {
		//block stupid files
		http.Error(w, http.StatusText(503), 503)
		return
	}
	fmt.Println("ReqURI is: " + r.RequestURI)
	//如果没有/结尾则自动加上，统一请求后缀，这样以后好处理
	if reqPath == "" {
		reqPath = "/"
	}

	//拼接新的url，加上#号符
	realUrl := getUrl("/#" + reqPath[1:])
	fmt.Println("transalated url is: " + realUrl)

	if reqPath == DEFAULT || strings.HasPrefix(reqPath, FRONT) {
		fmt.Println("[MATCH][Default/Front]")
	} else if strings.HasPrefix(reqPath, SEARCH) {
		fmt.Println("[MATCH][Search]")
	} else if strings.HasPrefix(reqPath, COURSE) {
		fmt.Println("[MATCH][Course]")
	} else {
		//如果请求的不是首页，搜索页或者课程详情页，则自动转接到首页去
		fmt.Println("[MISMATCH][path=" + reqPath + "]")
		http.Redirect(w, r, getUrl(DEFAULT), http.StatusFound)
		return
	}

	//如果今天对应的文件夹不存在，则自动创建一个新的文件夹
	//create the new date directory if it does not exist
	now := util.GetMili()
	directory := util.AssembleDirectory(now)
	if util.DirectoryNotExist(directory) {
		util.CreateDirectory(directory)
	}

	//通过url向redis请求是否有之前的记录
	previousHash, previousMili, err := redis.GetByUrl(realUrl)
	if err != nil {
		//如果redis中找不到对应记录，则hash一下url，用以作为对应html文件的名称，这样能去掉特殊字符，不然不方便文件读写
		//if not exist, err will be non-nil
		hashedUrl := quickHash.Hash(realUrl)

		//向redis添加新的记录
		//add new record to redis, make the record consistent with file name parameters
		//ignore redis error here, as there is nothing we can possily do
		_, _, _ = redis.SetByUrl(realUrl, hashedUrl, util.I64ToStr(now))
		filename := util.AssembleFilename(hashedUrl, now)

		//盲目从今天对应文件夹中尝试一下对应的html是否存在，只能尝试今天因为我们不知道这个url对应的请求时间信息
		//just a blind try, since if it does not exist in redis, we do not have time info, so just guess today
		if util.FileNotExist(filename) {
			//this snapshot has not been generated yet
			//return 503 to tell crawler to come back later
			fmt.Printf("[NOT FOUND][Not In Redis] no such file or directory: %s \n", filename)
			http.Error(w, http.StatusText(503), 503)

			//找不到相关记录的html，将新的记录传递到url channel中以供记载
			//this is a fresh hit, not file nor redis record exist, so send to channel to create new pending record
			record := Record{url: realUrl, hashedUrl: hashedUrl, mili: now}
			urlChan <- record
			return
		} else {
			//找到了对应的html，则直接返回该html
			//no need to copy files, as this file must be in today's folder, it is the only guess
			//and there is no way to sync redis' time record with reality as it is lost, good thing is the date is correct
			//and because the file is already today's version, do not update it
			//serve the static file
			fmt.Printf("[FOUND][Not In Redis] found file not in redis: %s \n", filename)
			http.ServeFile(w, r, filename)
			return
		}
	} else {
		//如果redis里存在该记录，说明该url最近被baidu查找过
		//exist in redis, then we can simply use previousHash and previousMili
		//still set, pass in prevousHash just for safety
		//ignore redis error here, as there is nothing we can possily do
		_, _, _ = redis.SetByUrl(realUrl, previousHash, util.I64ToStr(now))
		filename := util.AssembleFilename(previousHash, previousMili)
		//判断记录是否是今天的，如果不是的话，重新将该记录记载到临时文件中之后生成
		//保证旧记录被刷新，而当天已经被新请求过的记录不需要重复刷新
		if util.GetDayFromMili(now) > util.GetDayFromMili(previousMili) {
			//util.FileNotExist(util.AssembleFilename(previousHash, now))
			//if hit before but not today, add it to the generation list
			//if hit and its today, then it is already in the list or refreshed, do not add again
			record := Record{url: realUrl, hashedUrl: previousHash, mili: now}
			urlChan <- record
		}

		if util.FileNotExist(filename) || (now-previousMili) >= redis.EXPIRE_MILI {
			//如果记录在DB但是对应的html没有找到，或者Redis中的记录已经过期，则返回503，此时记录会已经被存储到Redis和临时文件中
			//record is in DB but its corresponding file does not exist or record has expired
			//eg crawler hitting the same url many times a day or cleaned
			//nothing much we can do
			fmt.Printf("[NOT FOUND][In Redis] no such file or directory: %s \n", filename)
			http.Error(w, http.StatusText(503), 503)
			return
		} else {
			//如果记录在DB而且对应的html找到了，则把对应的html移动到今天的目录中并且返回html
			//in redis and file does exist, hopefully best case scenerio
			newFilename := util.AssembleFilename(previousHash, now)
			if newFilename != filename {
				//if not match, given hash is same, the date (folder) must be different
				//copy the file from previous date folder to the new date folder to indicate a crawler hit
				err = util.MoveFile(filename, newFilename)
				if err != nil {
					//copy file failed
					fmt.Printf("[FOUND][In Redis] file copy failed: %s \n", filename)
					fmt.Println(err)
				}
			}
			//serve the static file
			http.ServeFile(w, r, newFilename)
			return
		}
	}
}

func main() {
	//initial basic test
	fmt.Println("Redis Self Testing...")
	fmt.Println(redis.Set("TESTGOPHANTOM_A", "test"))
	if result, _ := redis.Get("TESTGOPHANTOM_A"); result != "test" {
		panic("Initial Redis Test Failed")
	}

	redis.SetByUrl("www.ishangke.cn")
	redis.GetByUrl("www.ishangke.cn")

	//pipeline test
	//create a test connection, note that redigo does not return error immediately, if an error occues it will be returned the first time the connection is used
	//获得测试连接，注意如果连接不上，redigo不会在新建连接池或者连接时出err，而是会推迟在该连接第一次被使用的时候
	testConn := redis.GetConn()
	//发送测试请求，使用pipeline
	//launch the test request using redis pipelie, this is where errors might occur
	testConn.Send("SET", "TESTGOPHANTOM_B", util.I64ToStr(util.GetMili()))
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
	fmt.Println("Self Testing Finished")

	//发起异步的用于清理过期文件goroutine
	//schedule the clean event, which will clear old stuff
	go timerEventDisPatcher()
	//发起异步的store goroutine
	//the go-routine responsible for writing hit urls to files
	go store()

	//配置请求处理function
	//decalre the http handler
	http.HandleFunc("/", snapshotHandler)
	http.ListenAndServe(":8085", nil)
}

//该goroutine用于监听url channel，把新发送过来的record写入临时文件中
func store() {
	//如果临时文件不存在，则创建临时文件
	if util.FileNotExist(util.TEMPFILE) {
		err := ioutil.WriteFile(util.TEMPFILE, []byte{}, 0666)
		if err != nil {
			fmt.Println("[Error][Store] Failed to create temp file")
			panic(err)
		}
	}
	//测试打开临时文件，如果能打开则测试成功，关闭之
	//test open the temp file
	f, err := os.OpenFile(util.TEMPFILE, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("[Error][Store] Failed to open temp file")
		panic(err)
	}
	f.Close()

	for {
		select {
		case record := <-urlChan:
			//向Redis信号将要更改临时存储文件，如果该Redis中有该文件正在被使用的信号，则一直等待（记录10秒钟自动过期，所以最长等待10秒钟，下列代码运行时间最长也是10秒钟）
			//signal lock to redis, this could be blocking, max runtime for the following code must be within 10 seconds
			redis.LockTempFile()

			//如果临时文件不存在，则创世创建一个新的临时文件
			if util.FileNotExist(util.TEMPFILE) {
				err = ioutil.WriteFile(util.TEMPFILE, []byte{}, 0666)
				//如果因为某些原因创建失败，则无视该记录直接进入下一个循环，到时候重新尝试创建
				if err != nil {
					fmt.Println("[Error][Store] Failed to create temp file")
					fmt.Println(err)
					continue
				}
			}
			//打开临时文件
			f, err := os.OpenFile(util.TEMPFILE, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err != nil {
				fmt.Println("[Error][Store] Failed to open temp file")
				fmt.Println(err)
				continue
			}

			fmt.Println("[RECORD][Store] recoding: " + record.toFileString())
			//在临时文件底部添加新的记录
			//write a line
			_, err = f.WriteString(record.toFileString() + "\n")
			if err != nil {
				fmt.Println("[Error][Store] Failed to write line: " + record.toFileString() + " to temp file")
				fmt.Println(err)
				continue
			}
			f.Close()

			//解锁临时文件
			//signal unlock
			redis.ReleaseTempFile()
		}
	}
}

//一条goroutine，专门用于处理定时的清理任务
func timerEventDisPatcher() {
	//创建一个定时器的channel，到时定时器会自动给channel一个信号
	//scheduled events using ticker channels
	timer_cleanChan := time.NewTicker(time.Second * CLEAN_TICKLEPPERIOD).C
	for {
		//无限循环，等待定时器的信号才会继续执行
		select {
		case <-timer_cleanChan:
			fmt.Println("[Dispatcher][Clean]")
			//清理掉过期的html
			refreshStore.Clean()
		}
	}
}
