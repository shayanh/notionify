package notionify

import "io"

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

type CloudUploader interface {
	Upload(cloudFilePath string, content io.Reader) (*CloudFile, error)
}
