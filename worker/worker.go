package worker

import (
	log "github.com/inconshreveable/log15"
	"time"
	"sync"
	"strconv"
	"fmt"
	"../ccfg"
	"../cdb"
	"strings"
	"github.com/cdevr/WapSNMP"
)

type Worker struct {
	Todo		map[int]cdb.Switch
	LastKey		int
	KeyLock		sync.Mutex
	MapLock		sync.Mutex
	Log		log.Logger
}

func New( logger log.Logger ) *Worker {
	w := new(Worker)
	w.Todo = make(map[int]cdb.Switch)
	w.LastKey = 0
	w.Log = logger
	
	return w
}

func (w *Worker) NewKey() int {
	w.KeyLock.Lock()
	defer w.KeyLock.Unlock()
	w.LastKey++
	return w.LastKey
}

func (w *Worker) GetTodoLen() int {
	size := len(w.Todo)
	return size
}

func (w *Worker) AddToQueue(s cdb.Switch) {
	NewKey := w.NewKey()
	w.KeyLock.Lock()
	defer w.KeyLock.Unlock()
	w.Todo[NewKey] = s
	
	return
}

func (w *Worker) RemoveFromQueue(DeletKey int) {
	w.MapLock.Lock()
	defer w.MapLock.Unlock()
	delete(w.Todo, DeletKey)
	return
}

func (w *Worker) GetNextKey() int {
	var ReturnKey int
	for key, _ := range w.Todo {
		ReturnKey = key
		break
	}
	return ReturnKey
}

func (w *Worker) Start( models map[string]ccfg.Model, chanUdp chan<- string ) {
	for {
		if len(w.Todo) == 0 {
			time.Sleep( 3 * time.Second )
			continue
		}
		
		JobKey := w.GetNextKey()
		sw := w.Todo[JobKey]
		
		model, ok := models[sw.Model]
		if( !ok ) {
//			fmt.Printf("Warning! Skipping unknown model %s [%s]\n", sw.Model, sw.Ip)
			w.Log.Warn("Skipping unknown model:", "model", sw.Model, "ip", sw.Ip)
			w.RemoveFromQueue(JobKey)
			continue
		}
		
//		fmt.Printf("contacting %v %v %v\n", sw.Ip, model.Community, wapsnmp.SNMPv2c)
		snmp, err := wapsnmp.NewWapSNMP(sw.Ip, model.Community, wapsnmp.SNMPv2c, 1 * time.Second, 1)
		if err != nil {
//			fmt.Printf("Error creating wsnmp: %v\n", snmp)
			w.Log.Error("Error creating wsnmp \n", "data", snmp)
			continue
		}
		
		// DO SNMP STUFF
		for key, oid := range model.Oids {
			o := wapsnmp.MustParseOid(oid)
			table, err := snmp.GetTable(o)
			if err != nil {
//				fmt.Printf("Error getting table => %v\n", snmp)
//				w.Log.Debug("Error getting table", "table", snmp)
				continue
			}
			
			Table:
			for k,v := range table {
				spl := strings.Split(k, ".")
				i := spl[len(spl)-1]
				index, err := strconv.ParseInt(i, 10, 64)
				if err != nil {
//					fmt.Printf("Error parsing index->int64\n")
					w.Log.Debug("Error parsing index->int64")
					continue
				}
				port := index - model.SnmpOffset
				
				if( port < model.Min || port > model.Max ) {
					continue Table
				}
				
				var toSend string
				ipcode := strings.Replace( sw.Ip, ".", "_", -1)
				switch t := v.(type) {
					default:
//						fmt.Printf("[WARN] Unknown value type: %s (%v)", t, v)
						w.Log.Debug("[WARN] Unknown value type: %s (%v)", t, v)
						continue Table
					case wapsnmp.Counter:
						var value uint32
						value = uint32(v.(wapsnmp.Counter))
//						fmt.Printf("%s/%s/%d (%d): %s => %d\n", sw.Ip, key, index, port, k, value)
						strVal := fmt.Sprintf("%d", value)
						strPort := fmt.Sprintf("%d", port)
						tm := time.Now().Unix()
						toSend = "switch." + ipcode + "." + key + "." + strPort + " " + strVal + " " + fmt.Sprintf("%d", tm)
//					fmt.Printf("%s\n", toSend)
						chanUdp <- toSend
					case wapsnmp.Counter64:
						var value uint64
						value = uint64(v.(wapsnmp.Counter64))
//						fmt.Printf("%s/%s/%d (%d): %s => %d\n", sw.Ip, key, index, port, k, value)
						strVal := fmt.Sprintf("%d", value)
						strPort := fmt.Sprintf("%d", port)
						tm := time.Now().Unix()
						toSend = "switch." + ipcode + "." + key + "." + strPort + " " + strVal + " " + fmt.Sprintf("%d", tm)
						chanUdp <- toSend
//					fmt.Printf("%s\n", toSend)
				}
				
			}
		}
		
		w.RemoveFromQueue(JobKey)
	}
}

