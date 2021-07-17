package notionify

import (
	"bytes"
	"context"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/sharing"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
)

// DropboxSynchronizer ensures that all files inside a Dropbox folder are
// synchronized with Notion.
type DropboxSynchronizer struct {
	dh   *DropboxHandler
	cs   *CloudFileSynchronizer
	rdb  *redis.Client
	log  *logrus.Logger
	lock sync.Mutex
}

func NewDropboxSynchronizer(dh *DropboxHandler, ch *CloudFileSynchronizer, rdb *redis.Client, log *logrus.Logger) *DropboxSynchronizer {
	return &DropboxSynchronizer{
		dh:  dh,
		cs:  ch,
		rdb: rdb,
		log: log,
	}
}

func (ds *DropboxSynchronizer) SyncFolder(path string) ([]*NotionPage, error) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	var cursor string
	ctx := context.TODO()
	key := ds.getCursorKey(path)
	if val, err := ds.rdb.Get(ctx, key).Result(); err != redis.Nil {
		if err != nil {
			return nil, err
		}
		cursor = val
		ds.log.WithFields(logrus.Fields{
			"path":   path,
			"cursor": cursor,
		}).Info("Cursor retrieved from redis.")
	}

	entries, newCursor, err := ds.dh.GetAllEntries(path, cursor)
	if err != nil {
		ds.log.Error(err)
	}

	var errs error
	var pages []*NotionPage
	haveErr := false
	for _, entry := range entries {
		switch v := entry.(type) {
		case *files.FileMetadata:
			cloudFile, err := ds.getCloudFile(v)
			if err != nil {
				ds.log.Error(err)
				haveErr = true
				continue
			}
			page, err := ds.cs.Sync(cloudFile)
			if err != nil {
				haveErr = true
				errs = multierr.Append(errs, err)
			} else {
				pages = append(pages, page)
			}
		case *files.FolderMetadata:
			ds.log.Info("Folder:", v.PathDisplay, v.Id)
		case *files.DeletedMetadata:
			ds.log.Info("Deleted:", v.PathDisplay)
		}
	}

	if !haveErr && newCursor != cursor {
		err := ds.rdb.Set(ctx, key, newCursor, 0).Err()
		if err != nil {
			errs = multierr.Append(errs, err)
			return pages, errs
		}
		ds.log.WithFields(logrus.Fields{
			"path":   path,
			"cursor": newCursor,
		}).Info("New cursor saved.")
	}
	return pages, errs
}

func (ds *DropboxSynchronizer) getCloudFile(fileMetadata *files.FileMetadata) (*CloudFile, error) {
	link, err := ds.dh.getFileLink(fileMetadata)
	if err != nil {
		return nil, err
	}
	title := ds.dh.getFileTitle(fileMetadata)
	cloudFile := &CloudFile{
		FileID:   fileMetadata.Id,
		Title:    title,
		URL:      link,
		Provider: "dropbox",
	}
	return cloudFile, nil
}

func (ds *DropboxSynchronizer) getCursorKey(path string) string {
	return "cursor-dropbox-" + path
}

// DropboxHandler handles Dropbox API.
type DropboxHandler struct {
	config dropbox.Config
	fc     files.Client
	sc     sharing.Client
}

func NewDropboxHandler(token string) *DropboxHandler {
	config := dropbox.Config{
		Token:    token,
		LogLevel: dropbox.LogInfo,
	}
	filesClient := files.New(config)
	sharingClient := sharing.New(config)

	return &DropboxHandler{
		config: config,
		fc:     filesClient,
		sc:     sharingClient,
	}
}

func (dh *DropboxHandler) GetAllEntries(path string, cursor string) ([]files.IsMetadata, string, error) {
	entires := []files.IsMetadata{}
	for hasMore := true; hasMore; {
		var err error
		var resp *files.ListFolderResult
		if cursor == "" {
			arg := files.NewListFolderArg(path)
			resp, err = dh.fc.ListFolder(arg)
		} else {
			arg := files.NewListFolderContinueArg(cursor)
			resp, err = dh.fc.ListFolderContinue(arg)
		}
		if err != nil {
			return entires, cursor, err
		}
		entires = append(entires, resp.Entries...)
		cursor = resp.Cursor
		hasMore = resp.HasMore
	}
	return entires, cursor, nil
}

func (dh *DropboxHandler) getFileTitle(fileMetadata *files.FileMetadata) string {
	getBasicTitle := func() string {
		basename := path.Base(fileMetadata.PathDisplay)
		ext := filepath.Ext(basename)
		return strings.TrimSuffix(basename, ext)
	}
	getPDFTitle := func() (string, error) {
		downloadFileArg := files.NewDownloadArg(fileMetadata.PathLower)
		_, reader, err := dh.fc.Download(downloadFileArg)
		if err != nil {
			return "", err
		}
		defer reader.Close()

		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return "", err
		}
		rs := bytes.NewReader(body)
		return GetPDFTitleFromReadSeeker(rs)
	}

	title := getBasicTitle()
	if IsPDF(fileMetadata.PathLower) {
		pdfTitle, err := getPDFTitle()
		if err == nil {
			title = pdfTitle
		}
	}
	return title
}

func (dh *DropboxHandler) getFileLink(fileMetadata *files.FileMetadata) (string, error) {
	// TODO: Use batch API
	arg := sharing.NewGetFileMetadataArg(fileMetadata.PathLower)
	sharedFileMetadata, err := dh.sc.GetFileMetadata(arg)
	if err != nil {
		return "", err
	}
	link := strings.TrimSuffix(sharedFileMetadata.PreviewUrl, "?dl=0")
	return link, nil
}
