package main

import (
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/shayanh/notionify"
	"github.com/sirupsen/logrus"
)

func logDecorator(h http.Handler, log *logrus.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := notionify.NewResponseWriterWrapper(w)
		h.ServeHTTP(wr, r)
		log.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": wr.Status(),
		}).Info()
	})
}

func newLogger() *logrus.Logger {
	log := logrus.New()
	log.Level = logrus.DebugLevel
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	return log
}

func main() {
	log := newLogger()

	config, err := notionify.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	nh := notionify.NewNotionHandler(config.Notion.Token, config.Notion.DatabaseID)
	ch := notionify.NewCloudSynchronizer(nh, rdb, log)
	dh := notionify.NewDropboxHandler(config.Dropbox.Token)
	ds := notionify.NewDropboxSynchronizer(dh, ch, rdb, log)

	router := mux.NewRouter()
	router.StrictSlash(true)
	dropboxWebhookHandler := notionify.NewDropboxWebhookHandler(config.Dropbox.RootFolder, ds, log)
	dropboxWebhookHandler.HandleFuncs(router.PathPrefix("/dropbox-webhook").Subrouter())

	log.Infof("Listening on %s", config.Web.Addr)
	err = http.ListenAndServe(config.Web.Addr, logDecorator(router, log))
	if err != nil {
		log.Fatal(err)
	}
}
