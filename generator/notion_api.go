package generator

import (
	"context"
	"github.com/briandowns/spinner"
	"github.com/dstotijn/go-notion"
	"log"
	"os"
	"reflect"
	"time"
)

var spin = spinner.New(spinner.CharSets[14], time.Millisecond*100)

type NotionAPI struct {
	Client *notion.Client
}

func NewAPI() *NotionAPI {
	return &NotionAPI{
		Client: notion.NewClient(os.Getenv("NOTION_SECRET")),
	}
}

func (api *NotionAPI) filterFromConfig(config Notion) *notion.DatabaseQueryFilter {
	if config.FilterProp == "" || len(config.FilterValue) == 0 {
		return nil
	}
	properties := make([]notion.DatabaseQueryFilter, len(config.FilterValue))
	for i, v := range config.FilterValue {
		properties[i] = notion.DatabaseQueryFilter{
			Property: config.FilterProp,
			DatabaseQueryPropertyFilter: notion.DatabaseQueryPropertyFilter{
				Select: &notion.SelectDatabaseQueryFilter{
					Equals: v,
				},
			},
		}
	}
	return &notion.DatabaseQueryFilter{
		Or: properties,
	}
}

func (api *NotionAPI) FindBlockChildrenCommentLoop(client *notion.Client, blockArr []notion.Block, cursor string) (blocks []notion.Comment, err error) {
	for i := 0; i < len(blockArr); i++ {
		query := notion.FindCommentsByBlockIDQuery{
			BlockID:     blockArr[i].ID(),
			StartCursor: cursor,
			PageSize:    100,
		}
		res, err := client.FindCommentsByBlockID(context.Background(), query)
		if err != nil {
			return nil, err
		}
		if len(res.Results) == 0 {
			continue
		}
		blocks = append(blocks, res.Results...)
		if !res.HasMore {
			return blocks, nil
		}
		cursor = *res.NextCursor
	}
	return blocks, nil
}

func (api *NotionAPI) queryDatabase(client *notion.Client, config Notion) (notion.DatabaseQueryResponse, error) {
	spin.Suffix = " Querying Notion database..."
	spin.Start()
	defer spin.Stop()
	query := &notion.DatabaseQuery{
		Filter:   api.filterFromConfig(config),
		PageSize: 100,
	}
	return client.QueryDatabase(context.Background(), config.DatabaseID, query)
}

func (api *NotionAPI) queryBlockChildren(client *notion.Client, blockID string) (blocks []notion.Block, err error) {
	spin.Suffix = " Fetching blocks tree..."
	spin.Start()
	defer spin.Stop()
	return api.retrieveBlockChildren(client, blockID)
}

func (api *NotionAPI) retrieveBlockChildrenLoop(client *notion.Client, blockID, cursor string) (blocks []notion.Block, err error) {
	for {
		query := &notion.PaginationQuery{
			StartCursor: cursor,
			PageSize:    100,
		}
		res, err := client.FindBlockChildrenByID(context.Background(), blockID, query)

		if err != nil {
			return nil, err
		}

		if len(res.Results) == 0 {
			return blocks, nil
		}

		blocks = append(blocks, res.Results...)
		if !res.HasMore {
			return blocks, nil
		}
		cursor = *res.NextCursor
	}
}

func (api *NotionAPI) retrieveBlockChildren(client *notion.Client, blockID string) (blocks []notion.Block, err error) {
	blocks, err = api.retrieveBlockChildrenLoop(client, blockID, "")
	if err != nil {
		return
	}

	for _, block := range blocks {
		blockType := reflect.TypeOf(block)
		if !block.HasChildren() {
			continue
		}
		switch blockType {
		case reflect.TypeOf(&notion.ParagraphBlock{}):
			block.(*notion.ParagraphBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		case reflect.TypeOf(&notion.CalloutBlock{}):
			block.(*notion.CalloutBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		case reflect.TypeOf(&notion.QuoteBlock{}):
			block.(*notion.QuoteBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		case reflect.TypeOf(&notion.BulletedListItemBlock{}):
			block.(*notion.BulletedListItemBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		case reflect.TypeOf(&notion.NumberedListItemBlock{}):
			block.(*notion.NumberedListItemBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		case reflect.TypeOf(&notion.ToDoBlock{}):
			block.(*notion.ToDoBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		case reflect.TypeOf(&notion.TableBlock{}):
			block.(*notion.TableBlock).Children, err = api.retrieveBlockChildren(client, block.ID())
		}

		if err != nil {
			return
		}
	}

	return blocks, nil
}

// changeStatus changes the Notion article status to the published value if set.
// It returns true if status changed.
func (api *NotionAPI) changeStatus(client *notion.Client, p notion.Page, config Notion) bool {
	// No published value or filter prop to change
	if config.FilterProp == "" || config.PublishedValue == "" {
		return false
	}

	if v, ok := p.Properties.(notion.DatabasePageProperties)[config.FilterProp]; ok {
		if v.Select.Name == config.PublishedValue {
			return false
		}
	} else { // No filter prop in page, can't change it
		return false
	}

	updatedProps := make(notion.DatabasePageProperties)
	updatedProps[config.FilterProp] = notion.DatabasePageProperty{
		Select: &notion.SelectOptions{
			Name: config.PublishedValue,
		},
	}

	// update current update time
	currentTime := api.mustParseDateTime(time.Now().Format("2006-01-02T15:04:05.999Z0"))
	updatedProps["PublishDate"] = notion.DatabasePageProperty{
		Date: &notion.Date{
			Start: currentTime,
		},
	}

	_, err := client.UpdatePage(context.Background(), p.ID,
		notion.UpdatePageParams{
			DatabasePageProperties: updatedProps,
		},
	)
	if err != nil {
		log.Println("error changing status:", err)
	}

	return err == nil
}

func (api *NotionAPI) mustParseDateTime(value string) notion.DateTime {
	dt, err := notion.ParseDateTime(value)
	if err != nil {
		panic(err)
	}
	return dt
}
