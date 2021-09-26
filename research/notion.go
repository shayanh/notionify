package research

import (
	"context"
	"encoding/json"

	"github.com/jomei/notionapi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	nameProp := page.Properties["Name"]
	if nameProp != nil {
		titles := nameProp.(*notionapi.PageTitleProperty).Title
		if len(titles) > 0 {
			res.Name = titles[0].PlainText
		}
	}

	typeProp := page.Properties["Type"]
	if typeProp != nil {
		res.Type = typeProp.(*notionapi.SelectOptionProperty).Select.Name
	}

	urlProp := page.Properties["URL"]
	if urlProp != nil {
		res.URL = page.Properties["URL"].(*notionapi.URLProperty).URL.(string)
	}
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
				var res []notionapi.Option
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

func debugJSON(obj interface{}) {
	b, err := json.Marshal(obj)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Debug(string(b))
}

func (nh *NotionHandler) CreatePage(ctx context.Context, c *CloudFile) (*NotionPage, error) {
	req := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			DatabaseID: nh.databaseID,
		},
		Properties: nh.getProperties(c),
	}
	// debugJSON(req)
	page, err := nh.nc.Page.Create(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "notion handler CreatePage failed")
	}
	return NewNotionPage(page), nil
}

func (nh *NotionHandler) UpdatePage(ctx context.Context, c *CloudFile, pageID string) (*NotionPage, error) {
	req := &notionapi.PageUpdateRequest{
		Properties: nh.getProperties(c),
	}
	// We only update URL property
	for prop := range req.Properties {
		if prop != "URL" {
			delete(req.Properties, prop)
		}
	}

	page, err := nh.nc.Page.Update(ctx, notionapi.PageID(pageID), req)
	if err != nil {
		return nil, errors.Wrap(err, "notion handler UpdatePage failed")
	}
	return NewNotionPage(page), nil
}

func (nh *NotionHandler) ListPages(ctx context.Context) ([]*NotionPage, error) {
	var pages []*NotionPage
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
