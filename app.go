package main

import (
	// "net/url"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"mauscode/configurationfile"
	"flag"
)

const (
	DB_URL  string = "127.0.0.1:27017"
	DB_NAME string = "PageMasterV2"
)

var (
	conf                string
	configurationValues map[string]string = map[string]string{
		"manager": "~/www/pagemaster-www/htdocs",
		"managerPort":":2020",
		"router":  ":8080",
		"ssl":	   ":9090",
		"certFile":"pagemaster.crt",
		"keyFile": "pagemaster.key",
	}
)

type DataManager interface {
	AddHost( scheme, domain, code string, redirect int ) ( error ) //domain and code splitted ex: studiomaus + nl or google + com
	UpdateHost( newScheme, newDomain, newCode, scheme, domain, code string, redirect int ) ( error )
	RemoveHost( scheme, domain, code string ) ( error )
	AddSubHost( sub, scheme, domain, code string, redirect int ) ( error ) //domain and code should exists
	RemoveSub( sub, scheme, domain, code string ) ( error )
	GetDataJson() ( []byte, error )
	GetData( result interface{} ) ( error )
}

func init() {

	flag.StringVar(&conf, "conf", "settings.conf", "Configurationfile location.")
	flag.Parse()

}

func main() {

	

	// init configuration file
	var err error
	configurationValues, err = configurationfile.Parse(configurationValues, conf)
	if err != nil {

		log.Fatal(err)

	}

	// init data manager
	var dm DataManager

	dm, err = newMongo( DB_URL, DB_NAME )
	if err != nil {
		log.Fatal( err )
	}

	// err = dm.AddHost( "http://", "prudonforensics", "com", 5050 )
	// if err != nil {
	// 	log.Fatal( err )
	// }

	// err = dm.AddSubHost( "test", "http://", "prudonforensics", "com", 5051 )
	// if err != nil {
	// 	log.Fatal( err )
	// }

	// err = dm.AddSubHost( "hoi.kim", "http://", "prudonforensics", "com", 5052 )
	// if err != nil {
	// 	log.Fatal( err )
	// }

	// err = dm.AddSubHost( "liefde", "http://", "prudonforensics", "com", 5053 )
	// if err != nil {
	// 	log.Fatal( err )
	// }

	// dm.AddHost( "https://", "studiomaus", "nl", 5151 )

	log.Println( dm )

	var result []host
	err = dm.GetData( &result )
	if err != nil {

		log.Fatal( err )

	}

	log.Println( result )

	// main redirect router
	r, err := initRouter( dm )
	if err != nil {

		log.Fatal( err )

	}

	go func(){
			log.Fatal( http.ListenAndServe( configurationValues["router"], r ) )
		}()

	go func(){
			log.Fatal( http.ListenAndServeTLS(configurationValues["ssl"], configurationValues["certFile"], configurationValues["keyFile"], r) )
		}()


	// port manager site
	w := mux.NewRouter()

	w.PathPrefix( "/" ).Handler( http.FileServer( http.Dir( configurationValues["manager"] ) ) )

	go func(){
			log.Fatal( http.ListenAndServe( configurationValues["managerPort"], w ) )
		}()

	select{}


}