package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/matYang/goPhantom/quickHash"
	"net/http"
	"strings"
	"time"
)

//创建一个新的Redis连接池
//create a new redis connection pool
func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,                //最多闲置连接: 10
		IdleTimeout: 300 * time.Second, //闲置自动断链时间: 5min
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", server)
			if err != nil {
				panic(err)
			}
			if password != "" {
				if _, err := conn.Do("AUTH", password); err != nil {
					conn.Close()
					panic(err)
				}
			}
			return conn, err
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			_, err := conn.Do("PING")
			return err
		},
	}
}

var (
	//global的连接池
	//global connection pool
	pool *redis.Pool
	//标帜command line中可以对redis使用的 -<flag> 以及对应的默认值
	//flags to be used in command line
	redisServer   = flag.String("redisServer", ":6379", "")
	redisPassword = flag.String("redisPassword", "", "")
)

const (
	MODE   = "http"
	DOMAIN = "www.ishangke.cn"

	DEFAULT = "/"
	FRONT   = "/front"
	SEARCH  = "/search"
	COURSE  = "/course"

	REFRESH_MAX    = 1000 * 60 * 60 * 25      //fresh the urls tracked in previous 25 hours
	EXPIRETIME_MIN = 1000 * 60 * 60 * 24 * 60 //expire the urls tracked 2 months ago
)

func getUrl(sufix string) (url string) {
	url = MODE + "://" + DOMAIN + sufix
	return
}

func getMili() int64 {
	return int64(time.Now().Unix())
}

func snapshotHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RequestURI)
	fmt.Println(r.URL.Path)
	fmt.Println(r.RemoteAddr)

	reqPath := r.RequestURI
	//front page
	if reqPath == DEFAULT || strings.HasPrefix(reqPath, FRONT) {
		fmt.Println("[MATCH][Default/Front]")

		qualifiedUrl := getUrl(reqPath)
		realUrl := getUrl("/#" + reqPath[1:])
		hashedUrl := "ishDefaultFrontPage"
	} else if strings.HasPrefix(reqPath, SEARCH) {
		fmt.Println("[MATCH][Search]")

		qualifiedUrl := getUrl(reqPath)
		realUrl := getUrl("/#" + reqPath[1:])
		hashedUrl := quickHash.Hash(realUrl)
	} else if strings.HasPrefix(reqPath, COURSE) {
		fmt.Println("[MATCH][Course]")

		qualifiedUrl := getUrl(reqPath)
		realUrl := getUrl("/#" + reqPath[1:])
		hashedUrl := quickHash.Hash(realUrl)
	} else {
		fmt.Println("[MISMATCH][path=" + reqPath + "]")

		http.Redirect(w, r, getUrl(DEFAULT), http.StatusFound)
		return
	}

	fmt.Println("qualified url is: " + qualifiedUrl)
	fmt.Println("transalated url is: " + realUrl)
	fmt.Println("hashed url is: " + hashedUrl)
}

func main() {
	//获取可能有的flag
	//parse the possible command line flags
	flag.Parse()
	//新建连接池
	//initialize the global conecction pool
	pool = newPool(*redisServer, *redisPassword)
	//create a test connection, note that redigo does not return error immediately, if an error occues it will be returned the first time the connection is used
	//获得测试连接，注意如果连接不上，redigo不会在新建连接池或者连接时出err，而是会推迟在该连接第一次被使用的时候
	testConn := pool.Get()
	//发送测试请求
	//launch the test request, this is where errors might occur
	testConn.Send("SET", "TESTGOPHANTOM", getMili())
	//redigo使用pipeline并未抽象，方便使用pipeline对redis实现事务，因此需要手动清空pipeline
	//redigo uses pipeline for the benifits of transaction, it must be manually flushed
	testConn.Flush()
	//receive获得（Set）请求结果
	//receive gets the operation result
	testConn.Receive()
	//因为是连接池，因此每次都需要关闭连接
	//must close connection everytime one is fetched from connection pool
	testConn.Close()

	//decalre the http handler
	http.HandleFunc("/", snapshotHandler)
	http.ListenAndServe(":8081", nil)

}
