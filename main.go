package main

import (
//	log "github.com/apsdehal/go-logger"
	log "github.com/inconshreveable/log15"
	_"github.com/spf13/viper"
	"flag"
	_"reflect"
	"time"
	_"log"
	_"fmt"
	"./ccfg"
	"./csender"
	"./cdb"
	"./worker"
	"sync"
)

var udpMutex *sync.Mutex

func main() {
	udpMutex = &sync.Mutex{}
	
	configPath := flag.String("c", "./hpoller.toml", "Config file location")
	logPath := flag.String("l", "/var/log/hpoller.log", "Log file")
	flag.Parse()
	
	cfg := ccfg.New(configPath)
	
	// Logging
	Log := log.New()
	Log.SetHandler( log.Must.FileHandler(*logPath, log.TerminalFormat() ) )
	
	// Load & parse config
	cdb := cdb.New(cfg.Dbuser, cfg.Dbpassword, cfg.Dbhost, cfg.Dbsid, Log)
	
	// Channel for messaging
	chanUdp := make(chan string)
	
	// make udp sender
	sender := csender.New( cfg.Hosts, Log )
	go sender.Listen( chanUdp )
	
	// make & start snmp-workers
	var workers = make([]*worker.Worker, cfg.WorkersCount)
	for i := 0; i < cfg.WorkersCount; i++ {
		workers[i] = worker.New( Log )
		go workers[i].Start( cfg.Models, chanUdp )
	}
	
	
	
	var lastLoopEnd int64 = 0
	MainLoop:
	for {
//		fmt.Printf("Checking todo size for all workers...\n")
		Log.Info("Checking todo size for all workers...")
		for i := 0; i < len(workers); i++ {
			len := workers[i].GetTodoLen()
			if len > 0 {
//				fmt.Printf("Not all jobs done, sleeping 10s\n")
				Log.Info("Not all jobs done, sleeping 10s...")
				time.Sleep(10 * time.Second)
				continue MainLoop
			}
		}
//		fmt.Printf("Checking last loop time...\n")
		Log.Info("Checking last loop time...")
		var now int64 = time.Now().Unix()
		if( (now - cfg.Interval) < lastLoopEnd ) {
			toSleep := lastLoopEnd - (now - cfg.Interval)
			if( toSleep > 0 ) {
//				fmt.Printf("Time interval between iterations not reached yet. Sleeping %d s...\n", toSleep)
				Log.Info("Time interval between iterations not reached yet.", "Sleeping", toSleep)
				time.Sleep( time.Duration(toSleep) * time.Second )
			}
		}
		
//		fmt.Printf("No unfinished jobs.\n")
		Log.Info("No unfinished jobs.")
		switches := cdb.GetSwitches( cfg.Alive_param_id, cfg.Sw_root_id )
		
		j := 0
		for _, sw := range switches {
			// send switch to one of workers
			
			workers[j].AddToQueue( sw )
			
			j++
			if( j == cfg.WorkersCount ) {
				j = 0
			}
		}
		lastLoopEnd = time.Now().Unix()
	}
}
