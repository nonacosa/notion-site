package generator

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkwenda/notion-site/pkg/tomarkdown"

	"github.com/dstotijn/go-notion"
)

func Run(config Config) error {
	if err := os.MkdirAll(config.Markdown.PostSavePath, 0755); err != nil {
		return fmt.Errorf("couldn't create content folder: %s", err)
	}

	// find database page
	client := notion.NewClient(os.Getenv("NOTION_SECRET"))
	q, err := queryDatabase(client, config.Notion)
	if err != nil {
		return fmt.Errorf("❌ Querying Notion database: %s", err)
	}
	fmt.Println("✔ Querying Notion database: Completed")

	// fetch page children
	changed := 0 // number of article status changed
	for i, page := range q.Results {
		fmt.Printf("-- Article [%d/%d] -- %s \n", i+1, len(q.Results), page.URL)

		// Get page blocks tree
		blocks, err := queryBlockChildren(client, page.ID)
		if err != nil {
			log.Println("❌ Getting blocks tree:", err)
			continue
		}
		fmt.Println("✔ Getting blocks tree: Completed")

		// Generate content to file
		if err := generate(client, page, blocks, config.Markdown); err != nil {
			fmt.Println("❌ Generating blog post:", err)
			continue
		}
		fmt.Println("✔ Generating blog post: Completed")

		// Change status of blog post if desired
		if changeStatus(client, page, config.Notion) {
			changed++
		}

	}

	// Set GITHUB_ACTIONS info variables
	// https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Printf("::set-output name=articles_published::%d\n", changed)
	}

	return nil
}

func generate(client *notion.Client, page notion.Page, blocks []notion.Block, config Markdown) error {
	// Create file
	pageName := tomarkdown.ConvertRichText(page.Properties.(notion.DatabasePageProperties)["Name"].Title)
	//title := tomarkdown.ConvertRichText(page.Properties.(notion.DatabasePageProperties)["Title"].RichText)
	//status := page.Properties.(notion.DatabasePageProperties)["Status"].Select.Name
	//var date notion.DateTime
	//if page.Properties.(notion.DatabasePageProperties)["Date"].Date != nil {
	//	date = page.Properties.(notion.DatabasePageProperties)["Date"].Date.Start
	//}

	f, err := os.Create(filepath.Join(config.PostSavePath, generateArticleFilename(pageName, page.CreatedTime, config)))
	if err != nil {
		return fmt.Errorf("error create file: %s", err)
	}

	// Generate markdown content to the file
	tm := tomarkdown.New()
	tm.ImgSavePath = filepath.Join(config.ImageSavePath, pageName)
	tm.ImgVisitPath = filepath.Join(config.ImagePublicLink, url.PathEscape(pageName))
	tm.ContentTemplate = config.Template
	// todo edit frontMatter
	tm.WithFrontMatter(page)
	if config.ShortcodeSyntax != "" {
		tm.EnableExtendedSyntax(config.ShortcodeSyntax)
	}

	//parentId := strings.ReplaceAll(page.Parent.DatabaseID, "-", "")

	var fm = &tomarkdown.FrontMatter{}

	blocks, _ = syncMentionBlocks(client, blocks)

	// save last update time
	//websiteItemMeta.LastUpdate = page.LastEditedTime
	//websiteItemJson, _ := json.Marshal(websiteItemMeta)

	//storage.Save(fmt.Sprintf("%s_%s", parentId, page.ID), string(websiteItemJson))

	return tm.GenerateTo(blocks, f, fm)
}

func generateArticleFilename(title string, date time.Time, config Markdown) string {
	escapedTitle := strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(title),
			"",
		),
		" ", "-",
	)
	escapedFilename := escapedTitle + ".md"

	if config.GroupByMonth {
		return filepath.Join(date.Format("2006-01-02"), escapedFilename)
	}

	return escapedFilename
}

// todo pref
func syncMentionBlocks(client *notion.Client, blocks []notion.Block) (retBlocks []notion.Block, err error) {

	for _, block := range blocks {
		switch block.Type {
		// todo image
		case notion.BlockTypeParagraph:
			richTexts := block.Paragraph.Text
			for _, rich := range richTexts {
				// todo mention .type = user
				if rich.Type == "mention" {
					// todo mention has many types !!! how to work ?
					//if rich.Mention.Type == "page" {
					//	pageId := rich.Mention.Page.ID
					//	return queryBlockChildren(client, pageId)
					//}
					//if rich.Mention.Type == "user" {
					//	_ = rich.Mention.User.ID
					//}
				}
			}
		default:
			{
			}
		}
	}
	return blocks, nil
}
