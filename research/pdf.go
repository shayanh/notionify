package research

import (
	"errors"
	"io"
	"strings"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
)

var ErrTitleNotFound = errors.New("title not found")

func getTitle(info []string) (string, error) {
	titlePrefix := "Title: "
	for _, line := range info {
		cleaned := strings.TrimSpace(line)
		if strings.HasPrefix(cleaned, titlePrefix) {
			return strings.TrimPrefix(cleaned, titlePrefix), nil
		}
	}
	return "", ErrTitleNotFound
}

func GetPDFTitleFromFile(inFile string) (string, error) {
	info, err := pdfcpu.InfoFile(inFile, []string{}, nil)
	if err != nil {
		return "", err
	}
	return getTitle(info)
}

func GetPDFTitleFromReadSeeker(rs io.ReadSeeker) (string, error) {
	info, err := pdfcpu.Info(rs, []string{}, nil)
	if err != nil {
		return "", err
	}
	return getTitle(info)
}

func IsPDF(filePath string) bool {
	filePath = strings.ToLower(filePath)
	return strings.HasSuffix(filePath, ".pdf")
}
