package research

import (
	"context"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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

func (cs *CloudFileSynchronizer) Sync(ctx context.Context, c *CloudFile) (*NotionPage, error) {
	key := c.GetKey()
	if err := cs.acquireProc(key); err != nil {
		return nil, errors.Wrap(err, "cloudfile Sync failed")
	}
	defer cs.releaseProc(key)

	storedPageID, err := cs.rdb.Get(ctx, key).Result()
	if err != redis.Nil && err != nil {
		return nil, errors.Wrap(err, "cloudfile Sync failed")
	}
	if err == nil {
		cs.log.WithFields(logrus.Fields{
			"FileID":    c.FileID,
			"FileTitle": c.Title,
			"PageID":    storedPageID,
		}).Info("Notion page found.")

		page, err := cs.nh.UpdatePage(ctx, c, storedPageID)
		return page, errors.Wrap(err, "cloudfile Sync failed")
	}

	c.Tags = append(c.Tags, TagNeedsEdit)
	cs.log.Debugln(c.FileID, c.Title, c.Tags)
	page, err := cs.nh.CreatePage(ctx, c)
	if err != nil {
		return nil, errors.Wrap(err, "cloudfile Sync failed")
	}

	err = cs.rdb.Set(ctx, key, page.ID, 0).Err()
	cs.log.WithFields(logrus.Fields{
		"FileID":    c.FileID,
		"FileTitle": c.Title,
		"PageID":    page.ID,
	}).Info("Notion page created.")
	return page, err
}

type NotionSynchronizer struct {
	cloudFolderPath string
	nh              *NotionHandler
	cu              CloudUploader
	rdb             *redis.Client
	log             *logrus.Logger
	client          *http.Client
}

func NewNotionSynchronizer(path string, nh *NotionHandler, cu CloudUploader, rdb *redis.Client, log *logrus.Logger) *NotionSynchronizer {
	return &NotionSynchronizer{
		cloudFolderPath: path,
		nh:              nh,
		cu:              cu,
		rdb:             rdb,
		log:             log,
		client:          &http.Client{},
	}
}

func (ns *NotionSynchronizer) SyncDatabase(ctx context.Context) ([]*CloudFile, error) {
	pages, err := ns.nh.ListPages(ctx)
	if err != nil {
		return nil, err
	}
	var cloudFiles []*CloudFile
	for _, page := range pages {
		ns.log.Debugln(page.Name, page.Type, page.URL)

		c, err := ns.syncPage(ctx, page)
		if err != nil {
			return nil, err
		}
		if c != nil {
			cloudFiles = append(cloudFiles, c)
		}
	}
	return cloudFiles, nil
}

func (ns *NotionSynchronizer) shouldSync(page *NotionPage) bool {
	if page.Type != "paper" {
		return false
	}
	// TODO
	if strings.HasPrefix(page.URL, "https://www.dropbox.com/") {
		return false
	}
	return true
}

func (ns *NotionSynchronizer) syncPage(ctx context.Context, page *NotionPage) (*CloudFile, error) {
	if !ns.shouldSync(page) {
		return nil, nil
	}

	resp, err := ns.client.Get(page.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cloudFilePath := path.Join(ns.cloudFolderPath, path.Base(page.URL))
	cloudFile, err := ns.cu.Upload(cloudFilePath, resp.Body)
	if err != nil {
		return nil, err
	}
	_, err = ns.nh.UpdatePage(ctx, cloudFile, page.ID)
	if err != nil {
		return nil, err
	}
	err = ns.rdb.Set(ctx, cloudFile.GetKey(), page.ID, 0).Err()
	if err != nil {
		return nil, err
	}
	ns.log.WithFields(logrus.Fields{
		"PageID":    page.ID,
		"PageName":  page.Name,
		"FileID":    cloudFile.FileID,
		"FileTitle": cloudFile.Title,
	}).Info("Cloud file created.")
	return cloudFile, nil
}
