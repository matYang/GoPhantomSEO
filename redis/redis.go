package redis

import (
	"errors"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/matYang/goPhantom/quickHash"
	"github.com/matYang/goPhantom/util"
	"strings"
	"time"
)

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
	SEPERATOR       = "-"
	DEFAULTFRONTURL = "ishDefaultFrontPage"

	EXPIRE_DAY  = 60 //expire in 2 months
	EXPIRE_SEC  = 60 * 60 * 24 * EXPIRE_DAY
	EXPIRE_MILI = 1000 * EXPIRE_SEC
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

//package initialization
func init() {
	//获取可能有的flag
	//parse the possible command line flags
	flag.Parse()
	//新建连接池
	//initialize the global conecction pool
	pool = newPool(*redisServer, *redisPassword)

}

func GetConn() redis.Conn {
	return pool.Get()
}

func set(key string, exp int, value string) (reply string, err error) {
	conn := pool.Get()
	defer conn.Close()

	//set and exp, everything expires in 2 months after each set
	conn.Send("SETEX", key, exp, value)
	conn.Flush()

	data, err := conn.Receive()
	if err != nil {
		return
	}

	ok := false
	if reply, ok = data.(string); !ok {
		err = errors.New("[ERROR] Set Return Value not String")
	}

	return
}

func get(key string) (reply string, err error) {
	conn := pool.Get()
	defer conn.Close()

	conn.Send("GET", key)
	conn.Flush()

	data, err := conn.Receive()
	if err != nil {
		return
	}

	//apparently the raw return type is int slice
	replyArr := []uint8{}
	ok := false
	if replyArr, ok = data.([]uint8); !ok {
		err = errors.New("[ERROR] Get Return Value not int slice")
	} else {
		reply = string(replyArr[:])
	}

	return
}

func Set(key, value string) (reply string, err error) {
	return set(key, EXPIRE_SEC, value)
}

func Get(key string) (reply string, err error) {
	return get(key)
}

func SetByUrl(url string, arg ...string) (hashedUrl, reply string, err error) {
	now := util.GetMili()
	nowStr := util.I64ToStr(now)

	if len(arg) == 1 {
		hashedUrl = arg[0]
	} else if len(arg) == 2 {
		hashedUrl = arg[0]
		nowStr = arg[1]
	} else {
		hashedUrl = quickHash.Hash(url)
	}

	fmt.Println("hashed url is: " + hashedUrl)
	reply, err = Set(url, hashedUrl+SEPERATOR+nowStr)
	return
}

func GetByUrl(url string) (hashedUrl string, mili int64, err error) {
	value, err := Get(url)

	if err != nil {
		return
	}

	valArr := strings.Split(value, SEPERATOR)
	hashedUrl = valArr[0]
	mili, err = util.StrToI64(valArr[1])
	return
}

//this function will block until a lock has been obtained
func LockTempFile() {
	locked := true
	for locked {
		value, err := get("LOCKTEMPFILE")
		locked = (err == nil && value == "LOCK")
		time.Sleep(time.Second) //sleep for 1 second
	}

	set("LOCKTEMPFILE", 10, "LOCK")
	return
}

func ReleaseTempFile() {
	set("LOCKTEMPFILE", 10, "UNLOCK")
	return
}
