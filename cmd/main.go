package main

import (
	"github.com/go-redis/redis/v8"
	"github.com/shayanh/notionify"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()

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

	pages, err := ds.SyncFolder(config.Dropbox.RootFolder)
	if err != nil {
		log.Fatal(err)
	}
	for _, page := range pages {
		log.Info(page.ID, page.Name)
	}
}
