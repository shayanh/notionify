package research

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type DropboxWebhookHandler struct {
	rootPath string
	ds       *DropboxSynchronizer
	log      *logrus.Logger
}

func NewDropboxWebhookHandler(path string, ds *DropboxSynchronizer, log *logrus.Logger) *DropboxWebhookHandler {
	return &DropboxWebhookHandler{
		rootPath: path,
		ds:       ds,
		log:      log,
	}
}

func (dwh *DropboxWebhookHandler) handleChallenge(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("challenge")
	w.Header().Add("Content-Type", "text-plain")
	w.Header().Add("X-Content-Type-Options", "nosniff")
	_, err := w.Write([]byte(challenge))
	if err != nil {
		dwh.log.Errorf("Error while writing handleChallenge response: %v", err)
	}
}

func (dwh *DropboxWebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// TODO: authentication
	// Assuming that we have only one user
	go func() {
		defer func() {
			if r := recover(); r != nil {
				dwh.log.Errorf("Recovered from panic: %s", r)
			}
		}()

		pages, err := dwh.ds.SyncFolder(context.Background(), dwh.rootPath)
		if err != nil {
			dwh.log.Error(err)
			return
		}
		for _, page := range pages {
			dwh.log.WithFields(logrus.Fields{
				"Name": page.Name,
				"ID":   page.ID,
			}).Info("Synced page")
		}
	}()
}

func (dwh *DropboxWebhookHandler) HandleFuncs(router *mux.Router) {
	router.HandleFunc("", dwh.handleChallenge).Methods("GET")
	router.HandleFunc("", dwh.handleWebhook).Methods("POST")
}
