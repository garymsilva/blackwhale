package mongo

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"

	"fmt"

	"github.com/joaopandolfi/blackwhale/configurations"
	"github.com/joaopandolfi/blackwhale/utils"
	"gopkg.in/mgo.v2"
)

// Session exported struct
type Session struct {
	session *mgo.Session
}

var session Session
var pool []Session
var looper int
var mpos sync.RWMutex
var mrec sync.RWMutex

var maxPool int = configurations.Configuration.MongoPool

// NewSessionSsl -
// Create session with ssl and ignore the validation cert (more common)
func NewSessionSsl(mongoURL string) (s *mgo.Session, err error) {
	if session.session == nil {
		url := strings.Replace(mongoURL, "ssl=true", "", -1)
		url = strings.Replace(url, "readPreference=secondaryPreferred", "", -1)
		dialInfo, err := mgo.ParseURL(url)
		if err != nil {
			utils.CriticalError("[Mongo SSL] ERROR Url parsing", err)
		}
		//utils.Debug("[Mongo] - Before connection")
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			tlsConfig := &tls.Config{}
			tlsConfig.InsecureSkipVerify = true
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			if err != nil {
				utils.CriticalError("[Mongo SSL] ERROR SSL Connection ", err.Error(), addr.String())
				log.Println(err)
				panic(err)
			}
			return conn, err
		}
		s, err = mgo.DialWithInfo(dialInfo)

		if err != nil {
			return nil, err
		}

		//session.session.SetMode(mgo.SecondaryPreferred,true) // Unecessary
	}
	return s, err
}

// NewSessionSSLMETHOD2 -
// Create session with ssl and use sign cert
func NewSessionSSLMETHOD2(mongoURL string) (s *Session, err error) {
	// --sslCAFile
	rootCerts := x509.NewCertPool()
	if ca, err := ioutil.ReadFile("ca.crt"); err == nil {
		rootCerts.AppendCertsFromPEM(ca)
	}

	// --sslPEMKeyFile
	clientCerts := []tls.Certificate{}
	if cert, err := tls.LoadX509KeyPair("client.crt", "client.key"); err == nil {
		clientCerts = append(clientCerts, cert)
	}

	// Dial with TLS
	sess, err := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs: []string{mongoURL},
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{
				RootCAs:      rootCerts,
				Certificates: clientCerts,
			})
		},
	})
	session.session = sess

	return &session, err
}

// Create session without ssl
func newSession(mongoURL string) (s *mgo.Session, err error) {
	se, err := mgo.Dial(mongoURL)
	if err != nil {
		return nil, err
	}
	return se, err
}

// GetPoolSession - return session fom a pool or create a new if do not exists
func GetPoolSession() (*Session, error) {
	return GetCustomPoolSession(configurations.Configuration.MongoUrl)
}

// GetCustomPoolSession - return session fom a pool or create a new if do not exists based on mongoURL
func GetCustomPoolSession(mongoURL string) (*Session, error) {
	var err error
	lenPool := len(pool)
	pos := 0

	if lenPool <= maxPool {
		mpos.Lock()
		looper = lenPool
		pos = looper
		mpos.Unlock()

		s, err := createMgoSession(mongoURL)

		if err != nil {
			return nil, err
		}

		pool = append(pool, Session{session: s})
		return &pool[pos], nil
	}

	mpos.Lock()
	if looper >= maxPool {
		looper = 0
	} else {
		looper++
	}
	pos = looper
	mpos.Unlock()

	if pool[pos].Health() != nil {
		mrec.Lock()
		pool[pos].Close()
		s, errr := createMgoSession(mongoURL)
		err = errr
		pool[pos] = Session{session: s}
		mrec.Unlock()
	}

	return &pool[pos], err
}

// FlushPull - clear the session pool
func FlushPull() {
	for _, p := range pool {
		go p.Close()
	}
	pool = nil
	looper = 0
}

func createMgoSession(mongoURL string) (*mgo.Session, error) {
	if strings.Contains(mongoURL, "ssl=") {
		return NewSessionSsl(mongoURL)
	}
	return newSession(mongoURL)
}

func NewSessionManual(url string) (s *Session, err error) {
	se, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Session{session: se}, err
}

// NewSession - create a new session with ssl or not based on config mongourl
// https://godoc.org/gopkg.in/mgo.v2#Dial
func NewSession() (s *Session, err error) {
	return NewCustomSession(configurations.Configuration.MongoUrl)
}

func NewCustomSessionFresh(mongoURL string) (s *Session, err error) {
	sess := Session{}
	if strings.Contains(mongoURL, "ssl=") {
		sess.session, err = NewSessionSsl(mongoURL)
	} else {
		sess.session, err = newSession(mongoURL)
	}

	return &sess, err
}

// NewCustoSession - create a new session wuth ssl or not based on setted mongo url
func NewCustomSession(mongoURL string) (s *Session, err error) {
	if session.session == nil {
		if strings.Contains(mongoURL, "ssl=") {
			session.session, err = NewSessionSsl(mongoURL)
		} else {
			session.session, err = newSession(mongoURL)
		}
	}
	return &session, err
}

// Copy session
func (s *Session) Copy() *Session {
	return &Session{s.session.Copy()}
}

// GetCollection return a specific collection
// Get mongo collection
func (s *Session) GetCollection(col string) *mgo.Collection {
	return s.session.DB(configurations.Configuration.MongoDb).C(col)
}

// GetCollectionOnDB return a specific collection on database
// Get mongo collection
func (s *Session) GetCollectionOnDB(db, col string) *mgo.Collection {
	return s.session.DB(db).C(col)
}

// Run arbitrary commando direct on mongo
func (s *Session) Run(cmd interface{}) {
	var result interface{}
	s.session.DB(configurations.Configuration.MongoDb).Run(cmd, result) //.C(col)
	fmt.Println(result)
}

// Close session
func (s *Session) Close() {
	if s.session != nil {
		s.session.Close()
	}
}

// Health check sanity of connection
func (s *Session) Health() error {
	if s.session != nil {
		return s.session.Ping()
	}
	return fmt.Errorf("checking health: session is nill")
}
