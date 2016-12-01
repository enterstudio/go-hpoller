package cdb

import (
	log "github.com/inconshreveable/log15"
	"database/sql"
	_"github.com/mattn/go-oci8"
	"os"
	"fmt"
	"time"
	"strings"
)

type Cdb struct {
	user		string
	host		string
	password	string
	sid		string
	lastDbTime	int64
	Log		log.Logger
	
	switches	map[int]Switch
}

type Switch struct {
	Ip	string
	Model	string
}

func New(dbuser string, dbpassword string, dbhost string, dbsid string, logger log.Logger) *Cdb {
	os.Setenv("NLS_LANG", "AMERICAN_AMERICA.AL32UTF8")
	
	c := new(Cdb)
	c.user = dbuser
	c.host = dbhost
	c.password = dbpassword
	c.sid = dbsid
	c.lastDbTime = 0
	c.switches = make(map[int]Switch)
	c.Log = logger
	
	return c
}

func (c *Cdb) GetSwitches( alive_param_id int, switch_root_id int) map[int]Switch {
	// if switches is empty, OR if more then 1hour past from prev DB query, refresh map
	nowSwitches := len( c.switches )
	if( nowSwitches <= 0 || ( time.Now().Unix() - 3600 ) > c.lastDbTime ) {
//		fmt.Printf("Updating switches from DB\n");
		c.Log.Info("Updating switches from DB")
		
		db, err := sql.Open("oci8", fmt.Sprintf("%s/%s@%s", c.user, c.password, c.sid) )
		if err != nil {
//			fmt.Printf("Error connecting to DB: %s", err)
			c.Log.Error("Error connecting to DB: %s", err)
			time.Sleep( 300 * time.Second )
			return c.GetSwitches( alive_param_id, switch_root_id )
		}
		defer db.Close()
		
		// get switches
		
		// clear old map
		c.switches = make(map[int]Switch)
		
		// query DB
		switches, err := db.Query(`
			SELECT O.N_OBJECT_ID, IPADR.VC_VISUAL_CODE, G.VC_GOOD_NAME AS MODEL
			FROM SI_V_OBJECTS_SIMPLE O
			INNER JOIN SR_V_GOODS_SIMPLE G
				ON G.N_GOOD_TYPE_ID = 1 AND G.N_GOOD_ID=O.N_GOOD_ID
			INNER JOIN SR_V_GOODS_SIMPLE G2
				ON G2.N_GOOD_ID=G.N_PARENT_GOOD_ID
			INNER JOIN SI_V_OBJECTS_SPEC_SIMPLE OSPEC
				ON OSPEC.N_MAIN_OBJECT_ID=O.N_OBJECT_ID AND OSPEC.VC_NAME LIKE 'CPU %'
			INNER JOIN SI_V_OBJ_ADDRESSES_SIMPLE_CUR IPADR
				ON IPADR.N_ADDR_TYPE_ID=SYS_CONTEXT('CONST', 'ADDR_TYPE_IP') AND IPADR.N_OBJECT_ID=OSPEC.N_OBJECT_ID
			LEFT JOIN SI_V_OBJ_VALUES V1
				ON V1.N_OBJECT_ID=O.N_OBJECT_ID AND V1.N_GOOD_VALUE_TYPE_ID=:1
			WHERE G2.N_PARENT_GOOD_ID=:2
			AND V1.VC_VISUAL_VALUE='Y'
		`, alive_param_id, switch_root_id)
		
		if err != nil {
//			fmt.Printf("Error fetching switches from DB: %s", err)
			c.Log.Error("Error fetching switches from DB: %s", err)
			time.Sleep( 300 * time.Second )
			return c.GetSwitches( alive_param_id, switch_root_id )
		}
		
		num := 1
		for switches.Next() {
			var id		int
			var ip		string
			var model	string
			switches.Scan( &id, &ip, &model )
			
			// Format model name
			model = strings.Replace(model, " ", "_", -1)
			model = strings.Replace(model, ".", "_", -1)
			
			var sw Switch
			sw.Ip		= ip
			sw.Model	= model
			c.switches[num]	= sw
			num++
//			fmt.Printf("%s : %s\n", ip, model)
		}
//		panic("s")
		c.lastDbTime = time.Now().Unix()
		switches.Close()
	}
	
	return c.switches
}
