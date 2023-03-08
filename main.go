package main

import (
	"crypto/tls"
	"encoding/json"
	"expvar"
	"flag"
	"net/http"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/acme/autocert"

	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

var (
	// Tag is set by Gitlab's CI build process
	Tag string
	// Build is set by Gitlab's CI build process
	Build string

	// WebhookURL is the url for the webhook
	WebhookURL string
	// WebHookHeaders is an env var to transmit HTTP headers in JSON
	WebHookHeaders map[string]string
	// WebhookBearerToken is the bearer token passed to the webhook
	WebhookBearerToken string
	// SecretKey is used to check JWT tokens signatures
	SecretKey string
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 4096,
}

var log = logrus.New()

func main() {

	var goroutines = expvar.NewInt("num_goroutine")
	var interval = time.Duration(5) * time.Second
	go func() {
		for {
			<-time.After(interval)
			// The next line goes after the runtime.NumGoroutine() call
			goroutines.Set(int64(runtime.NumGoroutine()))
		}
	}()

	log.Formatter = &prefixed.TextFormatter{
		DisableTimestamp: true,
		ForceFormatting:  true,
	}

	loglevel := os.Getenv("LOGLEVEL")

	switch loglevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
	log.SetOutput(os.Stdout)

	if Build != "" {
		if Tag == "" {
			log.Infof("GeeoServer - build %s", Build)
		} else {
			log.Infof("GeeoServer %s - build %s", Tag, Build)
		}
	} else {
		log.Infof("GeeoServer development version")
	}

	var hostPort = flag.String("host", "localhost:8000", "host and port for http server")
	var dbfile = flag.String("db", "bolt.db", "database file name")
	var secret = flag.String("secret", "developmentKey", "secret for JWT signatures")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	var memprofile = flag.String("memprofile", "", "write memory profile to this file")

	var ssl = flag.Bool("ssl", false, "Enable SSL support")
	var sslhost = flag.String("sslhost", "", "FQDN for the SSL certificate")
	var dev = flag.Bool("dev", false, "allow development routes")
	flag.Parse()

	var webhookwriter *WebhookWriter

	SecretKey = *secret

	if wh := os.Getenv("WEBHOOK_URL"); wh != "" {
		WebhookURL = wh

		if whbt := os.Getenv("WEBHOOK_BEARER"); whbt != "" {
			WebhookBearerToken = whbt
		}

		if whh := os.Getenv("WEBHOOK_HEADERS"); whh != "" {
			err := json.Unmarshal([]byte(whh), &WebHookHeaders)
			if err != nil {
				log.Error(err)
				log.Fatal("Can't JSON parse WEBHOOK_HEADERS")
			}
			webhookwriter = NewWebhookWriter(wh, WebHookHeaders, WebhookBearerToken)
		} else {
			webhookwriter = NewWebhookWriter(wh, nil, WebhookBearerToken)
		}
	}

	if s := os.Getenv("SECRET"); s != "" {
		SecretKey = s
	}

	if hp := os.Getenv("HOST_PORT"); hp != "" {
		hostPort = &hp
	}

	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		dbfile = &dbname
	}

	if envSSL := os.Getenv("SSL"); envSSL != "" {
		*ssl = true
	}
	if envSSLhost := os.Getenv("SSL_HOST"); envSSLhost != "" {
		sslhost = &envSSLhost
	}
	if envDev := os.Getenv("DEV"); envDev != "" {
		*dev = true
	}

	if *cpuprofile != "" {
		after2min := time.After(time.Minute * 2)

		log.Debug("Setting up CPU Prof")
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		go func() {
			<-after2min
			log.Debug("Starting CPU prof output")
			pprof.StopCPUProfile()
			log.Debug("Done CPU prof output")
		}()

	}
	if *memprofile != "" {
		after2min := time.After(time.Minute * 1)
		log.Debug("Setting up Mem Prof")

		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			<-after2min
			log.Debug("Starting Mem prof output")
			pprof.WriteHeapProfile(f)
			log.Debug("Done Mem prof output")
			f.Close()
		}()
	}

	//persister := newNullPersister()
	persister := newBoltDBPersister(*dbfile)
	defer persister.close()

	geeodb := NewGeeoDB(persister, 5)

	wshandler := NewWSRouter(geeodb, webhookwriter)

	r := mux.NewRouter()

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	r.HandleFunc("/ws", wshandler.handle(upgrader))

	subrouter := r.PathPrefix("/api").Subrouter()
	NewHTTPRouter(subrouter, geeodb, wshandler)

	// TODO make it really private
	r.HandleFunc("/api/private/backup", persister.BackupHandleFunc)
	r.HandleFunc("/api/private/jsondump", persister.JSONDumpHandleFunc)

	if *dev {
		r.HandleFunc("/api/dev/token", DevHelperGetToken)
	}

	corsOptions := cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE"},
		AllowedHeaders: []string{"X-GEEO-TOKEN"},
	}
	if o := os.Getenv("ORIGIN"); o != "" {
		corsOptions.AllowedOrigins = strings.Split(o, ",")
	}
	c := cors.New(corsOptions)

	withCors := c.Handler(r) // TODO LATER finer handling of allowed origins
	http.Handle("/", withCors)

	if *ssl {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*sslhost),
			Cache:      autocert.DirCache("certs"), // TODO LATER store in bolt db or distributed db ?
			Email:      "devteam@xtralife.cloud",   // TODO env var
			ForceRSA:   true,
		}
		srv := &http.Server{
			Addr: ":https",
			TLSConfig: &tls.Config{
				GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					if hello.ServerName == "" {
						hello.ServerName = *sslhost
					}
					return certManager.GetCertificate(hello)
				},
				//NextProtos:               []string{"h2", "http/1.1", acme.ALPNProto},
				MinVersion:               tls.VersionTLS10,
				PreferServerCipherSuites: true,
			},
		}
		log.Info("TLS Server starting")

		s := &http.Server{
			Handler: certManager.HTTPHandler(nil),
			Addr:    ":80",
		}
		go s.ListenAndServe()
		log.Fatal(srv.ListenAndServeTLS("", ""))

		return
	}
	log.Info("Server starting at ", *hostPort)

	srv := &http.Server{
		Addr: *hostPort,
	}
	log.Fatal(srv.ListenAndServe())
}
