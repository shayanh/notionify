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
)

type DropboxHandler struct {
	log    *logrus.Logger
	ch     *CloudFileHandler
	rdb    *redis.Client
	config dropbox.Config
	fc     files.Client
	sc     sharing.Client
	lock   sync.Mutex
}

func NewDropboxHandler(token string, cloudFileHandler *CloudFileHandler, rdb *redis.Client, log *logrus.Logger) *DropboxHandler {
	config := dropbox.Config{
		Token:    token,
		LogLevel: dropbox.LogInfo,
	}
	filesClient := files.New(config)
	sharingClient := sharing.New(config)

	return &DropboxHandler{
		log:    log,
		ch:     cloudFileHandler,
		rdb:    rdb,
		config: config,
		fc:     filesClient,
		sc:     sharingClient,
	}
}

func (dh *DropboxHandler) HandleFolder(path string) {
	dh.lock.Lock()
	defer dh.lock.Unlock()

	var cursor string
	ctx := context.TODO()
	key := dh.getCursorKey(path)
	if val, err := dh.rdb.Get(ctx, key).Result(); err != redis.Nil {
		if err != nil {
			dh.log.Error(err)
			return
		}
		cursor = val
		dh.log.WithFields(logrus.Fields{
			"path":   path,
			"cursor": cursor,
		}).Info("Cursor retrieved from redis.")
	}

	entries, newCursor, err := dh.getAllEntries(path, cursor)
	if err != nil {
		dh.log.Error(err)
	}

	haveErr := false
	for _, entry := range entries {
		switch v := entry.(type) {
		case *files.FileMetadata:
			cloudFile, err := dh.getCloudFile(v)
			if err != nil {
				dh.log.Error(err)
				haveErr = true
				continue
			}
			err = dh.ch.HandleCloudFile(cloudFile)
			if err != nil {
				dh.log.Error(err)
				haveErr = true
			}
		case *files.FolderMetadata:
			dh.log.Info("Folder:", v.PathDisplay, v.Id)
		case *files.DeletedMetadata:
			dh.log.Info("Deleted:", v.PathDisplay)
		}
	}

	if !haveErr && newCursor != cursor {
		err := dh.rdb.Set(ctx, key, newCursor, 0).Err()
		if err != nil {
			dh.log.Error(err)
			return
		}
		dh.log.WithFields(logrus.Fields{
			"path":   path,
			"cursor": newCursor,
		}).Info("New cursor saved.")
	}
}

func (dh *DropboxHandler) getCursorKey(path string) string {
	return "cursor-dropbox-" + path
}

func (dh *DropboxHandler) getAllEntries(path string, cursor string) ([]files.IsMetadata, string, error) {
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

func (dh *DropboxHandler) getCloudFile(fileMetadata *files.FileMetadata) (CloudFile, error) {
	link, err := dh.getFileLink(fileMetadata)
	if err != nil {
		return CloudFile{}, err
	}
	title := dh.getFileTitle(fileMetadata)
	cloudFile := CloudFile{
		FileID:   fileMetadata.Id,
		Title:    title,
		URL:      link,
		Provider: "dropbox",
	}
	return cloudFile, nil
}
