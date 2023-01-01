package pkg

import (
	"fmt"
	"github.com/dstotijn/go-notion"
	"log"
	"os"
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
	// find and process database page
	processDatabase(ns, ns.config.DatabaseID)
	for _, cache := range ns.caches {
		//ns.files.MediaPath = cache.ParentFilesInfo.MediaPath
		if err := processDatabase(ns, cache.ChildDatabaseId); err != nil {
			fmt.Errorf("process child database erro but continu %s", err)
		}
		// del
		ns.caches = append(ns.caches[:0], ns.caches[1:]...)
	}
	// Set GITHUB_ACTIONS info variables : https://docs.github.com/en/actions/learn-github-actions/workflow-commands-for-github-actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Printf("::set-output name=articles_published::\n")
	}
	return nil
}

func generate(ns *NotionSite, page notion.Page, blocks []notion.Block) error {
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
		return nil
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
		return fmt.Errorf("error create file: %s", err)
	}

	// todo edit frontMatter
	//if ns.config.Markdown.ShortcodeSyntax != "" {
	//	ns.tm.EnableExtendedSyntax(ns.config.Markdown.ShortcodeSyntax)
	//}

	//// todo how to support mention feature ???

	return ns.tm.GenerateTo(ns)
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

func processDatabase(ns *NotionSite, id string) error {
	q, err := ns.api.queryDatabase(ns.api.Client, ns.config.Notion, id)
	if err != nil {
		return fmt.Errorf("❌ Querying Notion database: %s", err)
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
		if err := generate(ns, page, blocks); err != nil {
			fmt.Println("❌ Generating blog post:", err)
			continue
		}
		fmt.Println("✔ Generating blog post: Completed")
		// Change status of blog post if desired
		if ns.api.changeStatus(ns.api.Client, page, ns.config.Notion) {
			//changed++
		}
	}
	return nil
}
