package generator

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/pkwenda/notion-site/pkg/tomarkdown"

	"github.com/dstotijn/go-notion"
)

func Run(config Config) error {
	fmt.Printf("init save path %s", config.Markdown.PostSavePath)
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
	// Generate markdown content to the file
	tm := tomarkdown.New()
	var f *os.File
	var err error
	f, err = preCheck(page, config, tm)
	if f == nil {
		path := filepath.Join(config.PostSavePath, generateArticleFolderName(pageName, page.CreatedTime, config))
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Errorf("couldn't create content folder: %s", err)
		}
		tm.ArticleFolderPath = path
		f, err = os.Create(filepath.Join(path, "index.md"))
	}
	if err != nil {
		return fmt.Errorf("error create file: %s", err)
	}
	pageName = strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(strings.TrimSpace(pageName)),
			"",
		),
		" ", "-",
	)
	tm.ImgSavePath = filepath.Join(tm.ArticleFolderPath, "media")
	tm.ImgVisitPath = filepath.Join("p", url.PathEscape(pageName), "media")
	tm.ContentTemplate = config.Template
	// todo edit frontMatter
	tm.WithFrontMatter(page)
	if config.ShortcodeSyntax != "" {
		tm.EnableExtendedSyntax(config.ShortcodeSyntax)
	}

	blocks, _ = syncMentionBlocks(client, blocks)

	return tm.GenerateTo(blocks, f)
}

func preCheck(page notion.Page, config Markdown, tm *tomarkdown.ToMarkdown) (*os.File, error) {
	var savePath = config.PostSavePath
	var pageType = page.Properties.(notion.DatabasePageProperties)["Type"].Select
	var position = page.Properties.(notion.DatabasePageProperties)["Position"].Select
	var pageName = page.Properties.(notion.DatabasePageProperties)["Name"].Title
	if pageType != nil {
		if pageType.Name == "setting" {
			tm.FrontMatter["IsSetting"] = true
		}
	}
	if position != nil {
		tm.FrontMatter["Position"] = position.Name
		savePath = position.Name
		if err := os.MkdirAll(savePath, 0755); err != nil {
			fmt.Errorf("couldn't create content folder: %s", err)
		}
		// todo file name prop should have default value !!
		return os.Create(filepath.Join(savePath, generateSettingFilename(tomarkdown.ConvertRichText(pageName), page.CreatedTime, config)))
	}

	return nil, nil
}

func generateArticleFilename(title string, date time.Time, config Markdown) string {
	escapedTitle := strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(strings.TrimSpace(title)),
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

func generateArticleFolderName(title string, date time.Time, config Markdown) string {
	escapedTitle := strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(strings.TrimSpace(title)),
			"",
		),
		" ", "-",
	)

	if config.GroupByMonth {
		return filepath.Join(date.Format("2006-01-02"), escapedTitle)
	}

	return escapedTitle
}

func generateSettingFilename(title string, date time.Time, config Markdown) string {
	name := strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(strings.TrimSpace(title)),
			"",
		),
		" ", "-",
	)

	if config.GroupByMonth {
		return filepath.Join(date.Format("2006-01-02"), name)
	}
	return name
}

// todo pref
func syncMentionBlocks(client *notion.Client, blocks []notion.Block) (retBlocks []notion.Block, err error) {

	for _, block := range blocks {
		switch reflect.TypeOf(block) {
		// todo image
		case reflect.TypeOf(&notion.ParagraphBlock{}):
			richTexts := block.(*notion.ParagraphBlock).RichText
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
