package main

import (
	"bufio"
	"fmt"
	"github.com/matYang/goPhantom/util"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	GENERATE_TICKLEPPERIOD = 5 //generate tickle every 5 hours
)

func main() {
	timer_genChan := time.NewTicker(time.Second * GENERATE_TICKLEPPERIOD).C
	for {
		select {
		case <-timer_genChan:
			fmt.Println("[Gen][Store] genChan received")
			if util.FileNotExist(util.TEMPFILE) {
				fmt.Println("[Error][Store] failed to locate temp file with name: " + util.TEMPFILE)
				continue
			}
			if !util.FileNotExist(util.PRODUCEFILE) {
				util.RemoveFile(util.PRODUCEFILE)
			}
			err := util.MoveFile(util.TEMPFILE, util.PRODUCEFILE)
			if err != nil {
				fmt.Println("[Error][Store] failed to move temp file to produce file")
				panic(err)
			}

			file, err := os.Open(util.PRODUCEFILE)
			if err != nil {
				fmt.Println("[Error][Store] failed to open produce file")
				panic(err)
			}
			defer file.Close()

			fmt.Println("[Gen][Store] file scanning initiated")
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				line = strconv.Quote(line)

				fmt.Println("Feeding: " + line)
				//file ready, execute cmd
				cmd := exec.Command("phantomjs", "phantomjs.js", line)
				err = cmd.Run()
				if err != nil {
					fmt.Println("[Error][Store] failed to execute cmd")
					fmt.Println(err)
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Println("[Error][Store] scanner error")
				panic(err)
			}
		}
	}
}
