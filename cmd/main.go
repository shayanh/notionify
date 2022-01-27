package main

import (
	"context"
	"net/http"
	"time"

	"github.com/shayanh/notionify"
	"github.com/shayanh/notionify/recurring"

	"github.com/shayanh/notionify/research"

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

	rootConfig, err := notionify.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     rootConfig.Redis.Addr,
		Password: rootConfig.Redis.Password,
		DB:       rootConfig.Redis.DB,
	})

	router := mux.NewRouter()
	router.StrictSlash(true)

	func(config notionify.ResearchConfig) {
		nh := research.NewNotionHandler(config.Notion.Token, config.Notion.DatabaseID)
		ch := research.NewCloudFileSyncerImpl(nh, rdb, log)
		dh := research.NewDropboxHandler(config.Dropbox.Token, log)
		ds := research.NewDropboxSynchronizer(dh, ch, rdb, log)
		dwh := research.NewDropboxWebhookHandler(config.Dropbox.RootFolder, ds, log)
		dwh.HandleFuncs(router.PathPrefix("/dropbox-webhook").Subrouter())
	}(rootConfig.Research)

	go func(config notionify.RecurringConfig) {
		nh := recurring.NewNotionHandler(config.Notion.Token, config.Notion.DatabaseID)
		th := recurring.NewTasksHandler(nh, log)
		ctx := context.Background()
		for {
			err := th.Handle(ctx)
			if err != nil {
				log.Error(err)
			}
			time.Sleep(config.Interval)
		}
	}(rootConfig.Recurring)

	go func() {
		log.Infof("Listening on %s", rootConfig.Web.Addr)
		err = http.ListenAndServe(rootConfig.Web.Addr, logDecorator(router, log))
		if err != nil {
			log.Fatal(err)
		}
	}()

	select {}
}
