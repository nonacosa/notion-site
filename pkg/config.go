package pkg

import (
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

type Notion struct {
	DatabaseID     string   `yaml:"databaseId"`
	FilterProp     string   `yaml:"filterProp"`
	FilterValue    []string `yaml:"filterValue"`
	PublishedValue string   `yaml:"publishedValue"`
}

type Markdown struct {
	HomePath        string `yaml:"homePath"`
	ImagePublicLink string `yaml:"imagePublicLink"`

	// Optional:
	GroupByMonth bool   `yaml:"groupByMonth,omitempty"`
	Template     string `yaml:"template,omitempty"`
}

// 动态属性配置结构
type PropDef struct {
	Name         string      `yaml:"name"`
	Type         string      `yaml:"type"`
	OutputType   string      `yaml:"outputType"`
	DefaultValue interface{} `yaml:"defaultValue"`
}

type Config struct {
	Notion       `yaml:"notion"`
	Markdown     `yaml:"markdown"`
	DynamicProps []PropDef `yaml:"dynamicProps,omitempty"`
}

func DefaultConfigInit() error {
	defaultCfg := &Config{
		Notion: Notion{
			DatabaseID:     "YOUR-NOTION-DATABASE-ID",
			FilterProp:     "Status",
			FilterValue:    []string{"Finished", "Published"},
			PublishedValue: "Published",
		},
		Markdown: Markdown{
			HomePath: "",
		},
	}
	out, err := yaml.Marshal(defaultCfg)
	if err != nil {
		return err
	}

	defer func() {
		_ = os.WriteFile(".env", []byte("NOTION_SECRET=xxxx"), 0644)
		fmt.Println("Config file notion-site.yaml and .env created, please edit them for yourself.")
	}()

	return os.WriteFile("notion-site.yaml", out, fs.FileMode(0644))
}
