package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/dstotijn/go-notion"
	"github.com/gohugoio/hugo/common/paths"
	"log"
	"net/url"
	"os"
	"strings"
)

type NotionSite struct {
	api             *NotionAPI
	tm              *ToMarkdown
	files           *Files
	config          Config
	currentPage     notion.Page
	currentPageProp *NotionProp
	currentBlocks   []notion.Block
	caches          []*NotionCache
}

func NewNotionSite(api *NotionAPI, tm *ToMarkdown, files *Files, config Config, caches []*NotionCache) *NotionSite {
	return &NotionSite{api: api, tm: tm, files: files, config: config, caches: caches}
}

func Run(ns *NotionSite) error {
	fmt.Printf("init save path %s", ns.files.HomePath)
	if err := ns.files.mkdirHomePath(); err != nil {
		return fmt.Errorf("couldn't create content folder: %s", err)
	}
	var fms []*FrontMatter
	// find and process database page
	fms, err := processDatabase(ns, ns.config.DatabaseID)
	if err != nil {
		return err
	}
	for _, cache := range ns.caches {
		//ns.files.MediaPath = cache.ParentFilesInfo.MediaPath
		tmps, err := processDatabase(ns, cache.ChildDatabaseId)
		if err != nil {
			fmt.Errorf("process child database erro but continu %s", err)
		}
		fms = append(fms, tmps...)
	}
	// Set GITHUB_ACTIONS info variables : https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		str := os.Getenv("GITHUB_OUTPUT")
		err := os.Setenv("GITHUB_OUTPUT", str+"name=articles_published::")
		if err != nil {
			return err
		}
	}
	// fms, err = convertFolderPath(fms)
	// if err != nil {
	// 	return err
	// }
	fmsBytes, err := json.Marshal(fms)
	if err != nil {
		return err
	}
	return os.WriteFile(ns.files.HomePath+"/content/blogs.json", fmsBytes, 0644)
}

func convertFolderPath(fms []*FrontMatter) ([]*FrontMatter, error) {
	for _, fm := range fms {
		path := fm.Title.(string)
		if fm.Slug.(string) != "" {
			path = fm.Slug.(string)
		}

		// https://github.com/gohugoio/hugo/blob/master/helpers/url.go#L41
		path = strings.ToLower(strings.ReplaceAll(path, " ", "-"))
		parsedURI, err := url.Parse(path)
		if err != nil {
			return nil, err
		}

		// https://github.com/gohugoio/hugo/blob/master/helpers/path.go#L59
		fm.FolderPath = paths.Sanitize(parsedURI.String())
	}
	return fms, nil
}

func generate(ns *NotionSite, page notion.Page, blocks []notion.Block) (*FrontMatter, error) {
	// Generate markdown content to the file
	initNotionSite(ns, page, blocks)

	if ns.api.CheckHasChildDataBase(blocks, func(b bool, id string) {
		// cache child database block id
		if b {
			ns.caches = append(ns.caches, &NotionCache{
				ParentFilesInfo: ns.files,
				ParentPropInfo:  ns.currentPageProp,
				ChildDatabaseId: id,
			})
		}
	}) {
		return nil, nil
	}

	ns.files.mkdirPath(ns.files.FileFolderPath)

	if !ns.currentPageProp.IsSetting() {
		ns.tm.ContentTemplate = ns.config.Template
		ns.tm.WithFrontMatter(ns.currentPage)
	}
	var err error
	// save current io
	if !ns.currentPageProp.IsFolder() {
		ns.files.currentWriter, err = os.Create(ns.files.FilePath)
	}

	if err != nil {
		return nil, fmt.Errorf("error create file: %s", err)
	}

	// todo edit frontMatter
	//if ns.config.Markdown.ShortcodeSyntax != "" {
	//	ns.tm.EnableExtendedSyntax(ns.config.Markdown.ShortcodeSyntax)
	//}

	//// todo how to support mention feature ???

	fm, err := ns.tm.GenerateTo(ns)
	if fm != nil {
		fm.FolderPath = ns.files.FileFolderPath
	}
	return fm, err
}

func initNotionSite(ns *NotionSite, page notion.Page, blocks []notion.Block) {
	// set current origin page
	ns.currentPage = page
	// set current notion page prop
	ns.currentPageProp = NewNotionProp(ns.currentPage)
	ns.SetFileInfo(ns.currentPageProp.Position)
	// set notion site files info
	ns.tm.NotionProps = ns.currentPageProp
	ns.tm.Files = ns.files
	ns.currentBlocks = blocks
}

func processDatabase(ns *NotionSite, id string) ([]*FrontMatter, error) {
	var fms []*FrontMatter
	q, err := ns.api.queryDatabase(ns.api.Client, ns.config.Notion, id)
	if err != nil {
		return fms, fmt.Errorf("❌ Querying Notion database: %s", err)
	}
	fmt.Println("✔ Querying Notion database: Completed")
	// fetch page children
	//changed = 0 // number of article status changed
	for i, page := range q.Results {
		fmt.Printf("-- Article [%d/%d] -- %s \n", i+1, len(q.Results), page.URL)
		// Get page blocks tree
		blocks, err := ns.api.queryBlockChildren(ns.api.Client, page.ID)
		if err != nil {
			log.Println("❌ Getting blocks tree:", err)
			continue
		}
		fmt.Println("✔ Getting blocks tree: Completed")

		// Generate content to file
		fm, err := generate(ns, page, blocks)
		if err != nil {
			fmt.Println("❌ Generating blog post:", err)
			continue
		}
		if fm != nil {
			fms = append(fms, fm)
		}
		fmt.Println("✔ Generating blog post: Completed")
		// Change status of blog post if desired
		if ns.api.changeStatus(ns.api.Client, page, ns.config.Notion) {
			//changed++
		}
	}
	return fms, nil
}
