package main

import (
	"bufio"
	"fmt"
	"github.com/matYang/goPhantom/redis"
	"github.com/matYang/goPhantom/util"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	//设置生成html的间隔
	GENERATE_TICKLEPPERIOD = 60 * 60 * 5 //generate tickle every 5 hours
)

//之所以要额外开一个程序，是因为一下两点原因：
//每次生成页面都最好等phantom完成，不然多phantom进程并发难以控制并且消耗CPU，日后可以改成phantom进程池并发
//因为要等待phantom完成，因此必须使用run命令，该命令会一直等到命令彻底执行完成才结束
//而goroutine的scheduler不是preemptive的，在run命令等待时整个程序会卡死，导致新的请求需要等很久才会被接受
//因此该功能不能与主服务器放在一起，独立开来之后相当于利用了操作系统自己的preemptive sheculder来安排
//该程序在等待命令执行时的卡死不会影响到其他程序(主SEO服务器)
func main() {
	//创建一个定时器的channel，到时定时器会自动给channel一个信号
	timer_genChan := time.NewTicker(time.Second * GENERATE_TICKLEPPERIOD).C
	for {
		//无限循环，等待定时器的信号才会继续执行
		select {
		case <-timer_genChan:
			fmt.Println("[Gen][Store] genChan received")

			//向Redis信号将要更改临时存储文件，如果该Redis中有该文件正在被使用的信号，则一直等待（记录10秒钟自动过期，所以最长等待10秒钟，下列代码运行时间最长也是10秒钟）
			//signal lock to redis, this could be blocking, max runtime for the following code must be within 10 seconds
			redis.LockTempFile()

			//如果临时文件不存在，则证明没有新的请求，则直接进入下一次循环等待定时器信号
			if util.FileNotExist(util.TEMPFILE) {
				fmt.Println("[Error][Store] failed to locate temp file with name: " + util.TEMPFILE)
				continue
			}
			//如果之前的生成文件存在，删除它
			if !util.FileNotExist(util.PRODUCEFILE) {
				util.RemoveFile(util.PRODUCEFILE)
			}
			//将临时文件移动到生成文件去
			err := util.MoveFile(util.TEMPFILE, util.PRODUCEFILE)
			if err != nil {
				fmt.Println("[Error][Store] failed to move temp file to produce file")
				panic(err)
			}

			//向Redis信号解锁临时文件
			//signal unlock
			redis.ReleaseTempFile()

			//打开生成文件
			file, err := os.Open(util.PRODUCEFILE)
			if err != nil {
				fmt.Println("[Error][Store] failed to open produce file")
				panic(err)
			}
			defer file.Close()

			fmt.Println("[Gen][Store] file scanning initiated")
			//逐行读取
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				//这一行很重要，因为读取的记录中有许多特殊字符，为了能将其pass给命令行，必须包含在引号中
				line = strconv.Quote(line)

				fmt.Println("Feeding: " + line)
				//创建启动Phantom的命令
				//file ready, execute cmd
				cmd := exec.Command("phantomjs", "phantomjs.js", line)
				//执行该命令，run方法会一直等到命令执行完成才继续，我们需要这一点
				err = cmd.Run()
				if err != nil {
					fmt.Println("[Error][Store] failed to execute cmd")
					fmt.Println(err)
				}
			}
			//如果读取文件错误，则遇到了不可修复的问题，调用panic，crash掉程序
			if err := scanner.Err(); err != nil {
				fmt.Println("[Error][Store] scanner error")
				panic(err)
			}
		}
	}
}
