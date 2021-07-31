package recurring

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jomei/notionapi"
	"github.com/sirupsen/logrus"
)

type NotionTask struct {
	ID      string
	Name    string
	Status  string
	Tags    []string
	DueDate time.Time
}

const tagRecurring = "🔁 recurring"
const statusDone = "Done"

const isoLayout = "2006-01-02"

func debugJSON(obj interface{}) {
	b, err := json.Marshal(obj)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Debug(string(b))
}

func NewNotionTask(page *notionapi.Page) *NotionTask {
	res := new(NotionTask)
	res.ID = string(page.ID)

	nameProp := page.Properties["Name"]
	if nameProp != nil {
		titles := nameProp.(*notionapi.PageTitleProperty).Title
		if len(titles) > 0 {
			res.Name = titles[0].PlainText
		}
	}

	statusProp := page.Properties["Status"]
	if statusProp != nil {
		res.Status = statusProp.(*notionapi.SelectOptionProperty).Select.Name
	}

	tagsProp := page.Properties["Tags"]
	if tagsProp != nil {
		res.Tags = func(options []notionapi.Option) []string {
			var names []string
			for _, option := range options {
				names = append(names, option.Name)
			}
			return names
		}(tagsProp.(*notionapi.MultiSelectOptionsProperty).MultiSelect)
	}

	dueDateProp := page.Properties["Due Date"]
	if dueDateProp != nil {
		dueDateStr := dueDateProp.(*notionapi.DateProperty).Date.Start
		res.DueDate, _ = time.Parse(isoLayout, dueDateStr)
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

// ListTasks lists all recurring tasks
func (nh *NotionHandler) ListTasks(ctx context.Context) ([]*NotionTask, error) {
	var tasks []*NotionTask
	var cursor notionapi.Cursor
	for hasMore := true; hasMore; {
		req := &notionapi.DatabaseQueryRequest{
			Sorts: []notionapi.SortObject{},
			Filter: map[string]interface{}{
				"property": "Tags",
				"multi_select": map[string]interface{}{
					"contains": tagRecurring,
				},
			},
			StartCursor: cursor,
		}
		resp, err := nh.nc.Database.Query(ctx, nh.databaseID, req)
		if err != nil {
			return nil, err
		}
		for _, page := range resp.Results {
			tasks = append(tasks, NewNotionTask(&page))
		}
		hasMore = resp.HasMore
		cursor = resp.NextCursor
	}
	return tasks, nil
}

// UpdateTask updates the given NotionTask's DueDate and Status
func (nh *NotionHandler) UpdateTask(ctx context.Context, t *NotionTask) (*NotionTask, error) {
	// It doesn't work with one request, wtf
	req := &notionapi.PageUpdateRequest{
		Properties: notionapi.Properties{
			"Due Date": notionapi.DateProperty{
				Type: notionapi.PropertyTypeDate,
				Date: notionapi.Date{
					Start: t.DueDate.Format(isoLayout),
				},
			},
		},
	}
	_, err := nh.nc.Page.Update(ctx, notionapi.PageID(t.ID), req)
	if err != nil {
		return nil, err
	}

	req = &notionapi.PageUpdateRequest{
		Properties: notionapi.Properties{
			"Status": func() *notionapi.SelectOptionProperty {
				if t.Status == "" {
					return nil
				}
				return &notionapi.SelectOptionProperty{
					Type:   notionapi.PropertyTypeSelect,
					Select: notionapi.Option{Name: t.Status},
				}
			}(),
		},
	}
	page, err := nh.nc.Page.Update(ctx, notionapi.PageID(t.ID), req)
	if err != nil {
		return nil, err
	}
	return NewNotionTask(page), err
}
