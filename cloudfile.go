package notionify

import (
	"context"
	"errors"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type CloudFile struct {
	FileID   string
	Title    string
	URL      string
	Tags     []string
	Provider string
}

type CloudFileHandler struct {
	nh  *NotionHandler
	rdb *redis.Client
	log *logrus.Logger

	lock   sync.Mutex
	inProc map[string]bool
}

func NewCloudFileHandler(nh *NotionHandler, rdb *redis.Client, log *logrus.Logger) *CloudFileHandler {
	return &CloudFileHandler{
		nh:     nh,
		rdb:    rdb,
		log:    log,
		inProc: make(map[string]bool),
	}
}

func (ch *CloudFileHandler) getCloudFileKey(c CloudFile) string {
	return "cloudfile-" + c.Provider + "-" + c.FileID
}

func (ch *CloudFileHandler) acquireProc(key string) error {
	ch.lock.Lock()
	defer ch.lock.Unlock()
	if _, ok := ch.inProc[key]; ok {
		return errors.New("key is already in processing")
	}
	ch.inProc[key] = true
	return nil
}

func (ch *CloudFileHandler) releaseProc(key string) {
	ch.lock.Lock()
	defer ch.lock.Unlock()
	delete(ch.inProc, key)
}

var TagNeedsEdit = "needs edit"

func (ch *CloudFileHandler) HandleCloudFile(c CloudFile) error {
	key := ch.getCloudFileKey(c)
	if err := ch.acquireProc(key); err != nil {
		return err
	}
	defer ch.releaseProc(key)

	ctx := context.TODO()
	pageID, err := ch.rdb.Get(ctx, key).Result()
	if err != redis.Nil && err != nil {
		return err
	}

	if err == nil {
		ch.log.WithFields(logrus.Fields{
			"FileID": c.FileID,
			"Title":  c.Title,
		}).Info("Notion page found.")

		_, err := ch.nh.UpdatePage(c, pageID, []string{"URL"})
		return err
	}

	c.Tags = append(c.Tags, TagNeedsEdit)
	pageID, err = ch.nh.CreatePage(c)
	if err != nil {
		return err
	}

	err = ch.rdb.Set(ctx, key, pageID, 0).Err()
	ch.log.WithFields(logrus.Fields{
		"FileID": c.FileID,
		"Title":  c.Title,
	}).Info("Notion page created.")
	return err
}
