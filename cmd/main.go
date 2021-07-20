package main

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/shayanh/notionify"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.Level = logrus.DebugLevel

	config, err := notionify.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	ctx := context.Background()

	nh := notionify.NewNotionHandler(config.Notion.Token, config.Notion.DatabaseID)
	ch := notionify.NewCloudSynchronizer(nh, rdb, log)
	dh := notionify.NewDropboxHandler(config.Dropbox.Token)
	ds := notionify.NewDropboxSynchronizer(dh, ch, rdb, log)

	syncDropbox := func() {
		pages, err := ds.SyncFolder(ctx, config.Dropbox.RootFolder)
		if err != nil {
			log.Fatal(err)
		}
		for _, page := range pages {
			log.Info(page.ID, " ", page.Name, " ", page.Type, " ", page.URL)
		}
	}

	listNotion := func() {
		pages, err := nh.ListPages(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Info(len(pages))
		for _, page := range pages {
			log.Info(page.ID, " ", page.Name, " ", page.Type, " ", page.URL)
		}
	}

	syncNotion := func() {
		ns := notionify.NewNotionSynchronizer(config.Dropbox.RootFolder, nh, dh, rdb, log)
		cloudFiles, err := ns.SyncDatabase(ctx)
		if err != nil {
			log.Fatal(err)
		}
		for _, c := range cloudFiles {
			log.Infoln(c.FileID, c.URL)
		}
	}

	tasks := []func(){syncDropbox, listNotion, syncNotion}
	activeTask := tasks[0]
	activeTask()
}
