package notionify

import (
	"context"

	"github.com/jomei/notionapi"
)

type NotionPage struct {
	ID   string
	Name string
	Type string
	URL  string
}

func NewNotionPage(page *notionapi.Page) *NotionPage {
	res := &NotionPage{}
	res.ID = string(page.ID)
	// TODO: type assertions
	res.Name = page.Properties["Name"].(notionapi.PageTitleProperty).Title[0].PlainText
	res.Type = page.Properties["Type"].(notionapi.SelectProperty).Select.Options[0].Name
	res.URL = page.Properties["URL"].(notionapi.URLProperty).URL.(string)
	return res
}

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

func (nh *NotionHandler) getProperties(c *CloudFile) notionapi.Properties {
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

func (nh *NotionHandler) CreatePage(c *CloudFile) (*NotionPage, error) {
	req := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			DatabaseID: nh.databaseID,
		},
		Properties: nh.getProperties(c),
	}
	ctx := context.TODO()
	page, err := nh.nc.Page.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return NewNotionPage(page), nil
}

func (nh *NotionHandler) UpdatePage(c *CloudFile, pageID string) (*NotionPage, error) {
	req := &notionapi.PageUpdateRequest{
		Properties: nh.getProperties(c),
	}
	// We only update URL property
	for prop := range req.Properties {
		if prop != "URL" {
			delete(req.Properties, prop)
		}
	}

	ctx := context.TODO()
	page, err := nh.nc.Page.Update(ctx, notionapi.PageID(pageID), req)
	if err != nil {
		return nil, err
	}
	return NewNotionPage(page), nil
}

func (nh *NotionHandler) ListPages() ([]*NotionPage, error) {
	ctx := context.TODO()
	pages := []*NotionPage{}
	var cursor notionapi.Cursor
	for hasMore := true; hasMore; {
		req := &notionapi.DatabaseQueryRequest{
			Sorts: []notionapi.SortObject{
				{
					Property:  "Created",
					Direction: "ascending",
				},
			},
			StartCursor: cursor,
		}
		resp, err := nh.nc.Database.Query(ctx, nh.databaseID, req)
		if err != nil {
			return nil, err
		}
		for _, page := range resp.Results {
			pages = append(pages, NewNotionPage(&page))
		}
		hasMore = resp.HasMore
		cursor = resp.NextCursor
	}
	return pages, nil
}
