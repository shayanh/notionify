package notionify

import (
	"context"

	"github.com/jomei/notionapi"
)

type NotionHandler struct {
	databaseID notionapi.DatabaseID
	nc         *notionapi.Client
}

func NewNotionHandler(token string, databaseID string) *NotionHandler {
	return &NotionHandler{
		nc:         notionapi.NewClient(notionapi.Token(token)),
		databaseID: notionapi.DatabaseID(databaseID),
	}
}

func (nh *NotionHandler) getProperties(c CloudFile) notionapi.Properties {
	return notionapi.Properties{
		"Name": notionapi.PageTitleProperty{
			Title: notionapi.Paragraph{
				notionapi.RichText{
					Text: notionapi.Text{
						Content: c.Title,
					},
				},
			},
		},
		"Tags": notionapi.MultiSelectOptionsProperty{
			Type: "multi_select",
			MultiSelect: func() []notionapi.Option {
				res := []notionapi.Option{}
				for _, tag := range c.Tags {
					res = append(res, notionapi.Option{Name: tag})
				}
				return res
			}(),
		},
		"URL": notionapi.URLProperty{
			Type: "url",
			URL:  c.URL,
		},
	}
}

func (nh *NotionHandler) CreatePage(c CloudFile) (string, error) {
	req := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			DatabaseID: nh.databaseID,
		},
		Properties: nh.getProperties(c),
	}
	ctx := context.TODO()
	page, err := nh.nc.Page.Create(ctx, req)
	if err != nil {
		return "", err
	}
	return page.ID.String(), nil
}

func (nh *NotionHandler) UpdatePage(c CloudFile, pageID string, selectedProps []string) (string, error) {
	req := &notionapi.PageUpdateRequest{
		Properties: nh.getProperties(c),
	}
	for prop := range req.Properties {
		found := false
		for _, p := range selectedProps {
			if prop == p {
				found = true
				break
			}
		}
		if !found {
			delete(req.Properties, prop)
		}
	}

	ctx := context.TODO()
	page, err := nh.nc.Page.Update(ctx, notionapi.PageID(pageID), req)
	if err != nil {
		return "", err
	}
	return page.ID.String(), nil
}
