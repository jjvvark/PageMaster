package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httputil"
	"net/url"
	"log"
	"strings"
	"fmt"
)

type SuperRouter struct {
	dm DataManager
	data rData
}

type rData struct {
	http []rDomain
	https []rDomain
}

type rDomain struct {
	name string
	code string
	redirect int
	subs []rSub
}

type rSub struct {
	name string
	redirect int
}

func (sr *SuperRouter) RefreshData() ( error ) {

	var result []host

	err := sr.dm.GetData( &result )
	if err != nil {

		return err

	}

	htp := make( []rDomain, 0 )
	hts := make( []rDomain, 0 )

	for _, h := range result {

		subs := make( []rSub, len( h.Subs ) )

		for i, s := range h.Subs {

			subs[i] = rSub{ s.Sub, s.Redirect }

		}

		d := rDomain{ h.Domain, h.Code, h.Redirect, subs }

		if h.Scheme == "http://" {

			htp = append( htp, d )

		} else {

			hts = append( hts, d )

		}

	}

	sr.data = rData{ htp, hts }

	return nil

}

func (sr *SuperRouter) ServeHTTP( rw http.ResponseWriter, req *http.Request ) {

	h := strings.Split( req.Host, "." )
	hSize := len( h )
	if hSize < 2 {

		rw.Write( []byte( "string split error" ) )
		return

	}

	var redirect int

	if req.TLS == nil {

		redirect = sr.checkRoute( sr.data.http, h, hSize )

	} else {

		redirect = sr.checkRoute( sr.data.https, h, hSize )

	}

	if redirect != 0 {

		url, err := url.Parse( fmt.Sprintf( "http://localhost:%d", redirect ) )
		if err != nil {

			log.Println( err )
			http.Error( rw, "Internal error.", http.StatusInternalServerError )
			return

		}

		proxy := httputil.NewSingleHostReverseProxy( url )
		proxy.ServeHTTP( rw, req )
		return

	}

	rw.Write( []byte( "Hello from studio maus." ) )

}

func (sr *SuperRouter) checkRoute( data []rDomain, split []string, length int ) ( int ) {

	for _, d := range data {

		if d.name == split[ length - 2 ] && d.code == split[ length - 1 ] {

			if length > 2 {

				var sub string

				for i := 0; i < ( length - 2 ); i++ {

					sub += split[i]

					if i != ( length - 3 ) {

						sub += "."

					}

				}

				for _, s := range d.subs {

					if s.name == sub {

						return s.redirect

					}

				}

			}

			return d.redirect

		}

	}

	return 0

}

func initRouter( dm DataManager ) ( *mux.Router, error ) {

	sr := &SuperRouter{ dm, rData{} }
	sr.RefreshData()

	r := mux.NewRouter()
	r.PathPrefix( "/" ).Handler( sr )

	log.Println( sr.data )

	return r, nil

}