package notionify

import (
	"context"
	"errors"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// CloudFile represents a file that is stored in the cloud.
type CloudFile struct {
	FileID   string
	Title    string
	URL      string
	Tags     []string
	Provider string
}

func (c CloudFile) GetKey() string {
	return "cloudfile-" + c.Provider + "-" + c.FileID
}

// CloudFileSynchronizer ensures that a page for this CloudFile is exist in the Notion.
type CloudFileSynchronizer struct {
	nh  *NotionHandler
	rdb *redis.Client
	log *logrus.Logger

	lock   sync.Mutex
	inProc map[string]bool
}

func NewCloudSynchronizer(nh *NotionHandler, rdb *redis.Client, log *logrus.Logger) *CloudFileSynchronizer {
	return &CloudFileSynchronizer{
		nh:     nh,
		rdb:    rdb,
		log:    log,
		inProc: make(map[string]bool),
	}
}

func (cs *CloudFileSynchronizer) acquireProc(key string) error {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	if _, ok := cs.inProc[key]; ok {
		return errors.New("key is already in processing")
	}
	cs.inProc[key] = true
	return nil
}

func (cs *CloudFileSynchronizer) releaseProc(key string) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	delete(cs.inProc, key)
}

var TagNeedsEdit = "needs edit"

func (cs *CloudFileSynchronizer) Sync(c *CloudFile) (*NotionPage, error) {
	key := c.GetKey()
	if err := cs.acquireProc(key); err != nil {
		return nil, err
	}
	defer cs.releaseProc(key)

	ctx := context.TODO()
	pageID, err := cs.rdb.Get(ctx, key).Result()
	if err != redis.Nil && err != nil {
		return nil, err
	}

	if err == nil {
		cs.log.WithFields(logrus.Fields{
			"FileID": c.FileID,
			"Title":  c.Title,
		}).Info("Notion page found.")

		_, err := cs.nh.UpdatePage(c, pageID)
		return nil, err
	}

	c.Tags = append(c.Tags, TagNeedsEdit)
	page, err := cs.nh.CreatePage(c)
	if err != nil {
		return nil, err
	}

	err = cs.rdb.Set(ctx, key, pageID, 0).Err()
	cs.log.WithFields(logrus.Fields{
		"FileID": c.FileID,
		"Title":  c.Title,
	}).Info("Notion page created.")
	return page, err
}
