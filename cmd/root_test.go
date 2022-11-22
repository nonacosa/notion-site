package cmd

import (
	"fmt"
	"github.com/dstotijn/go-notion"
	"log"
	"os"
)

func run() {
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
}
