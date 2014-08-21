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

	EXPIRE_SEC = 60 * 60 * 24 * 60 //expire 2 months
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

func Set(key, value string) (reply string, err error) {
	conn := pool.Get()
	defer conn.Close()

	//set and exp, everything expires in 2 months after each set
	data, err := conn.Do("SETEX", key, EXPIRE_SEC, value)

	ok := false
	if reply, ok = data.(string); !ok {
		fmt.Println("[ERROR] Set Value not String")
		err = errors.New("Value not String")
	}

	return
}

func Get(key string) (reply string, err error) {
	conn := pool.Get()
	defer conn.Close()

	data, err := conn.Do("GET", key)

	//apparently the raw return type is byte slice
	replyArr := []byte{}
	ok := false
	if replyArr, ok = data.([]byte); !ok {
		fmt.Println("[ERROR] Get Value not String")
		err = errors.New("[ERROR] Get Value not String")
	} else {
		reply = string(replyArr[:])
	}

	return
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
