package main

import (
	"github.com/shayanh/notionify/research"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func logDecorator(h http.Handler, log *logrus.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := research.NewResponseWriterWrapper(w)
		h.ServeHTTP(wr, r)
		log.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": wr.Status(),
		}).Info()
	})
}

func newFormatter() *logrus.TextFormatter {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	return customFormatter
}

func newLogger() *logrus.Logger {
	log := logrus.New()
	// log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(newFormatter())
	return log
}

func main() {
	// logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(newFormatter())
	log := newLogger()

	config, err := research.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	nh := research.NewNotionHandler(config.Notion.Token, config.Notion.DatabaseID)
	ch := research.NewCloudSynchronizer(nh, rdb, log)
	dh := research.NewDropboxHandler(config.Dropbox.Token)
	ds := research.NewDropboxSynchronizer(dh, ch, rdb, log)

	router := mux.NewRouter()
	router.StrictSlash(true)
	dropboxWebhookHandler := research.NewDropboxWebhookHandler(config.Dropbox.RootFolder, ds, log)
	dropboxWebhookHandler.HandleFuncs(router.PathPrefix("/dropbox-webhook").Subrouter())

	log.Infof("Listening on %s", config.Web.Addr)
	err = http.ListenAndServe(config.Web.Addr, logDecorator(router, log))
	if err != nil {
		log.Fatal(err)
	}
}
