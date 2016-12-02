package ccfg

import (
	"github.com/spf13/viper"
	"fmt"
	"io"
	"encoding/json"
	"html"
)

type Model struct {
	Name		string
	Min		int64
	Max		int64
	Community	string
	SnmpOffset	int64
	Oids		map[string]string
}

type Def struct {
	Name		string
	Minport		int64
	Maxport		int64
	Community	string
	SnmpOffset	int64
	Oids		map[string]string
}

type Cfg struct {
	Dbhost		string
	Dbuser		string
	Dbpassword	string
	Dbsid		string
	Sw_root_id	int
	Alive_param_id	int
	WorkersCount	int
	Interval	int64
	Models		map[string]Model
	Hosts		[]string
	
	DefModel	Def
}

func New(path *string) *Cfg {
	
	viper.SetConfigFile(*path)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err.Error())
	}
	
	c := new(Cfg)
	c.Dbhost	= viper.GetString("db.host")
	c.Dbuser	= viper.GetString("db.user")
	c.Dbpassword	= viper.GetString("db.password")
	c.Dbsid		= viper.GetString("db.sid")
	c.Sw_root_id	= viper.GetInt("db.switch_root_id")
	c.Alive_param_id= viper.GetInt("db.alive_param_id")
	c.WorkersCount	= viper.GetInt("workers")
	c.Interval	= viper.GetInt64("interval")
	
	if( c.Dbhost == "" || c.Dbuser == "" || c.Dbpassword == "" || c.Dbsid == "" || c.Sw_root_id == 0 || c.Alive_param_id == 0 || c.WorkersCount == 0 || c.Interval == 0 ) {
		panic( fmt.Errorf("Fatal: missing mandatory config parameters.") )
	}
	
	hosts := viper.Get("carbon.hosts").([]interface{})
	hostsCount := len(hosts)
	if hostsCount < 1 {
		panic("No carbon hosts defined in config!")
	}
	
	c.Hosts = make([]string, hostsCount)
	for idx := 0; idx < hostsCount; idx++ {
		c.Hosts[idx] = hosts[idx].(string)
	}
	
	c.DefModel.Name = viper.GetString("default.name")
	c.DefModel.Minport = viper.GetInt64("default.minport")
	c.DefModel.Maxport = viper.GetInt64("default.maxport")
	c.DefModel.Community = viper.GetString("default.community")
	c.DefModel.SnmpOffset = viper.GetInt64("default.snmp_offset")
	
	c.DefModel.Oids = make(map[string]string)
	c.DefModel.Oids = viper.GetStringMapString("default.oids")
	
	c.Models = make(map[string]Model)
	
	modelsCfg := viper.Get("models").([]interface{})
	for _, v := range modelsCfg {
		value := v.(map[string]interface{})
		name, ok  := value["name"].(string)
		if( !ok ) {
			continue
		}
		
		var Mdl Model
		Mdl.Name = name
		
		min, ok := value["minport"].(int64)
		if( ok ) {
			Mdl.Min = min
		} else {
			Mdl.Min= c.DefModel.Minport
		}
		
		max, ok := value["maxport"].(int64)
		if( ok ) {
			Mdl.Max = max
		} else {
			Mdl.Max = c.DefModel.Maxport
		}
		
		community, ok := value["community"].(string)
		if( ok ) {
			Mdl.Community = community
		} else {
			Mdl.Community = c.DefModel.Community
		}
		
		so, ok := value["snmp_offset"].(int64)
		if( ok ) {
			Mdl.SnmpOffset = so
		} else {
			Mdl.SnmpOffset = c.DefModel.SnmpOffset
		}
		
		// Hard: oids
		oidsVal, ok := value["oids"]
		if( !ok ) {
			Mdl.Oids = c.DefModel.Oids
		} else {
			oids := oidsVal.(map[string]interface{})
			Mdl.Oids = make(map[string]string)
			for x, y := range oids {
				Mdl.Oids[x] = y.(string)
			}
		}
		
		c.Models[name] = Mdl
	}
	
	/*
	modelsCfg := viper.Get("models").([]interface{})
//	fmt.Printf("%v\n", modelsCfg)
	c.Models = make(map[string]Model)
	
//	for num, v := range modelsCfg.(map[string]interface{}) {
	for num, v := range modelsCfg {
//		fmt.Printf("%d => %v\n", num, v)
		value := v.(map[string]interface{})
		name := value["name"].(string)
		min := value["min"].(int64)
		max := value["max"].(int64)
		community := value["community"].(string)
		if( name == "" || min == 0 || max == 0 || community == "" ) {
			fmt.Printf("Missing required config parameters for model #%d\n", num)
			panic("Exiting")
		}
		
		var Mdl Model
		Mdl.Name = name
		Mdl.Min = min
		Mdl.Max = max
		Mdl.Community = community
		
		//oidsCfg := value["oids"].([]map[string]interface{})[0]
		oidsCfg := value["oids"].([]interface{})
		Mdl.Oids = make(map[string]string)
		
		for oname, oid := range oidsCfg[0].(map[string]interface{}) {
			//fmt.Printf("\t%s => %s\n", oname, oid)
			Mdl.Oids[oname] = oid.(string)
		}
		
		c.Models[name] = Mdl
	}
	*/
//	fmt.Printf("%#v\n", c)
//	printVars(os.Stdout, true, c)
	
	return c
}

func printVars(w io.Writer, writePre bool, vars ...interface{}) {
    if writePre {
	io.WriteString(w, "<pre>\n")
    }
    for i, v := range vars {
	fmt.Fprintf(w, "» item %d type %T:\n", i, v)
	j, err := json.MarshalIndent(v, "", "    ")
	switch {
	case err != nil:
	    fmt.Fprintf(w, "error: %v", err)
	case len(j) < 3: // {}, empty struct maybe or empty string, usually mean unexported struct fields
	    w.Write([]byte(html.EscapeString(fmt.Sprintf("%+v", v))))
	default:
	    w.Write(j)
	}
	w.Write([]byte("\n\n"))
    }
    if writePre {
	io.WriteString(w, "</pre>\n")
    }
}
