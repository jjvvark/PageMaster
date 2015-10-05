package main

import (
	"errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
)

type host struct {
	Id bson.ObjectId	`bson:"_id,omitempty" json:"-"`
	Scheme string		`bson:"scheme" json:"scheme"`
	Domain string		`bson:"domain" json:"domain"`
	Code string			`bson:"code" json:"code"`
	Subs []sub			`bson:"subs" json:"subs"`
	Redirect int		`bson:"redirect" json:"redirect"`
}

type sub struct {
	Sub string			`bson:"sub" json:"sub"`
	Redirect int		`bson:"redirect" json:"redirect"`
}

type mongo struct {
	databaseUrl string
	databaseName string
	session *mgo.Session
}

const (
	COLL_HOSTS = "hosts"
)

var (
	ErrorHostAlreadyExists = errors.New( "Host already exists." )
	ErrorHostNotFound = errors.New( "Host not found." )
	ErrorSubAlreadyExists = errors.New( "Sub already exists." )
	ErrorSubNotFound = errors.New( "Sub not found." )
)

func newMongo( databaseUrl, databaseName string ) ( *mongo, error ) {

	m := &mongo{ databaseUrl: databaseUrl, databaseName: databaseName }

	session, err := m.initDb()
	if err != nil {

		return nil, err

	}

	m.session = session

	return m, nil

}

func (m *mongo) initDb() ( *mgo.Session, error ) {

	session, err := mgo.Dial( m.databaseUrl )
	if err != nil {
		return nil, err
	}

	session.SetMode( mgo.Monotonic, true )

	// set indexes

	return session, nil

}

func ( m *mongo ) withCollection( collectionName string, fn func( collection *mgo.Collection ) ( error ) ) ( error ) {

	sess := m.session.Copy()

	// extra check
	if sess == nil {

		session, err := m.initDb()
		if err != nil {

			return err

		}

		sess = session.Copy()

	}

	defer sess.Close()

	collection := sess.DB( m.databaseName ).C( collectionName )

	err := fn( collection )
	if err != nil {

		return err

	}

	return nil

}


func ( m *mongo ) getHost( scheme, domain, code string ) ( []host, error ) {

	var results []host

	err := m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.Find( bson.M{ "scheme": scheme, "domain" : domain, "code" : code } ).All( &results )

		} )

	if err != nil {

		return nil, err

	}

	return results, nil

}

func ( m *mongo ) AddHost( scheme, domain, code string, redirect int ) ( error ) {

	// check if host exists
	results, err := m.getHost( scheme, domain, code )
	if err != nil {

		return err

	}

	if len( results ) != 0 {

		return ErrorHostAlreadyExists

	}

	// add host
	err = m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.Insert( host{ Scheme: scheme, Domain: domain, Code: code, Subs: []sub{}, Redirect: redirect } )

		} )

	return nil

}

func ( m *mongo ) UpdateHost( newScheme, newDomain, newCode, scheme, domain, code string, redirect int ) ( error ) {

	// check if new already exist
	results, err := m.getHost( newScheme, newDomain, newCode )
	if err != nil {

		return err

	}

	if len( results ) == 1 {

		return ErrorHostAlreadyExists

	}

	// remove old
	results, err = m.getHost( scheme, domain, code )
	if err != nil {

		return err

	}

	if len( results ) != 1 {

		return ErrorHostNotFound

	}

	// add new
	results[0].Scheme = newScheme
	results[0].Domain = newDomain
	results[0].Code = newCode
	results[0].Redirect = redirect

	err = m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.UpdateId( results[0].Id, results[0] )

		} )

	if err != nil {

		return err

	}

	return nil

}

func ( m *mongo ) AddSubHost( subname, scheme, domain, code string, redirect int ) ( error ) {

	// check host
	results, err := m.getHost( scheme, domain, code )
	if err != nil {

		return err

	}

	if len( results ) != 1 {

		return ErrorHostNotFound

	}

	if m.checkSub( results[0].Subs, subname ) {

		return ErrorSubAlreadyExists

	}

	results[0].Subs = append( results[0].Subs, sub{ subname, redirect } )

	err = m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.UpdateId( results[0].Id, results[0] )

		} )

	if err != nil {

		return err

	}

	return nil

}

func ( m *mongo ) checkSub( data []sub, name string ) ( bool ) {

	for _, s := range data {

		if s.Sub == name {

			return true

		}

	}

	return false

}

func ( m *mongo ) RemoveHost( scheme, domain, code string ) ( error ) {

	results, err := m.getHost( scheme, domain, code )
	if err != nil {

		return err

	}

	if len( results ) == 0 {

		return ErrorHostNotFound

	}

	err = m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.RemoveId( results[0].Id )

		} )

	if err != nil {

		return err

	}

	return nil

}

func ( m *mongo ) RemoveSub( subname, scheme, domain, code string ) ( error ) {

	results, err := m.getHost( scheme, domain, code )
	if err != nil {

		return err

	}

	if len( results ) != 1 {

		return ErrorHostNotFound

	}

	newSubs := make( []sub, 0 )
	flag := false

	for _, s := range results[0].Subs {

		if s.Sub != subname {

			newSubs = append( newSubs, s )

		} else {

			flag = true

		}

	}

	if !flag {

		return ErrorSubNotFound

	}

	results[0].Subs = newSubs

	err = m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.UpdateId( results[0].Id, results[0] )

		} )

	if err != nil {

		return err

	}

	return nil

}

func ( m *mongo ) GetDataJson() ( []byte, error ) {

	var results []host

	m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.Find( bson.M{} ).All( &results )

		} )

	data, err := json.Marshal( results )
	if err != nil {

		return nil, err

	}

	return data, err

}

func( m *mongo ) GetData( result interface{} ) ( error ) {

	value, ok := result.(*[]host)
	if !ok {

		return errors.New( "Wrong type, should be *[]host" )

	}

	var data []host

	err := m.withCollection( COLL_HOSTS, func( collection *mgo.Collection ) ( error ) {

			return collection.Find( bson.M{} ).All( &data )

		} )

	if err != nil {

		return err

	}

	*value = data

	return nil

}



























