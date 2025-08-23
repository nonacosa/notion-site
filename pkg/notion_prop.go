package pkg

import (
	"github.com/dstotijn/go-notion"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"reflect"
	"strings"
	"time"
)

const (
	nameProp         = "Name"
	titleProp        = "Title"
	statusProp       = "Status"
	categoriesProp   = "Categories"
	TagsProp         = "Tags"
	PositionProp     = "Position"
	fileNameProp     = "FileName"
	descriptionProp  = "Description"
	createAtProp     = "CreateAt"
	authorProp       = "Author"
	avatarProp       = "Avatar"
	lastModProp      = "Lastmod"
	expiryDateProp   = "ExpiryDate"
	publishDateProp  = "PublishDate"
	showCommentsProp = "ShowComments"
	slugProp         = "Slug"
	typeProp         = "Type"
	grayColor        = "rgba(120, 119, 116, 1)"
	brownColor       = "rgba(159, 107, 83, 1)"
	orangeColor      = "rgba(217, 115, 13, 1)"
	yellowColor      = "rgba(203, 145, 47, 1)"
	greenColor       = "rgba(68, 131, 97, 1)"
	blueColor        = "rgba(51, 126, 169, 1)"
	purpleColor      = "rgba(144, 101, 176, 1)"
	pinkColor        = "rgba(193, 76, 138, 1)"
	redColor         = "rgba(212, 76, 71, 1)"
	grayBackground   = "rgba(241, 241, 239, 1)"
	brownBackground  = "rgba(244, 238, 238, 1)"
	orangeBackground = "rgba(251, 236, 221, 1)"
	yellowBackground = "rgba(251, 243, 219, 1)"
	greenBackground  = "rgba(237, 243, 236, 1)"
	blueBackground   = "rgba(231, 243, 248, 1)"
	purpleBackground = "rgba(244, 240, 247, 0.8)"
	pinkBackground   = "rgba(249, 238, 243, 0.8)"
	redBackground    = "rgba(253, 235, 236, 1)"
)

var ColorMap = map[string]string{
	"gray":              grayColor,
	"brown":             brownColor,
	"orange":            orangeColor,
	"yellow":            yellowColor,
	"green":             greenColor,
	"blue":              blueColor,
	"purple":            purpleColor,
	"pink":              pinkColor,
	"red":               redColor,
	"gray_background":   grayBackground,
	"brown_background":  brownBackground,
	"orange_background": orangeBackground,
	"yellow_background": yellowBackground,
	"green_background":  greenBackground,
	"blue_background":   blueBackground,
	"purple_background": purpleBackground,
	"pink_background":   pinkBackground,
	"red_background":    redBackground,
}

type NotionProp struct {
	Name             string
	Title            string
	Status           string
	Categories       []string
	Tags             []string
	Position         string
	FileName         string
	Description      string
	CreateAt         *time.Time
	Author           string
	Avatar           string
	LastMod          time.Time
	ExpiryDate       time.Time
	PublishDate      time.Time
	ShowComments     *bool
	Slug             string
	Types            string
	IsSettingFile    bool
	IsCustomNameFile bool
	DynamicProps     map[string]interface{} `json:"dynamicProps,omitempty"`
}

// 全局配置缓存（懒加载）
var globalConfig *Config

func NewNotionProp(page notion.Page) (np *NotionProp) {
	np = &NotionProp{
		Name:        getTitle(page, nameProp),
		Title:       getRichText(page, titleProp),
		Status:      getSelect(page, statusProp),
		Categories:  getMultiSelect(page, categoriesProp),
		Tags:        getMultiSelect(page, TagsProp),
		Position:    getSelect(page, PositionProp),
		FileName:    getRichText(page, fileNameProp),
		Description: getRichText(page, descriptionProp),
		CreateAt:    getPropValue(page, createAtProp).CreatedTime,
		//Author: author,
		//Avatar: avatar,
		LastMod:      getDate(page, lastModProp),
		ExpiryDate:   getDate(page, expiryDateProp),
		PublishDate:  getDate(page, publishDateProp),
		ShowComments: getCheckbox(page, showCommentsProp),
		Slug:         getRichText(page, slugProp),
		Types:        getSelect(page, typeProp),
	}
	// default blog position from hugo home path
	if np.Position == "" {
		np.Position = "content/post"
	}
	np.IsSettingFile = np.IsSetting()
	np.IsCustomNameFile = np.IsCustomNameMdFile()

	// 处理动态属性（鲁棒性：配置不存在时不影响程序运行）
	np.processDynamicProps(page)

	return
}

func getPropValue(page notion.Page, key string) notion.DatabasePageProperty {
	properties := page.Properties.(notion.DatabasePageProperties)
	property := properties[key]
	return property
}

func getTitle(page notion.Page, key string) (rst string) {
	prop := getPropValue(page, key).Title
	if prop != nil {
		rst = ConvertRichText(prop)
	}
	return
}

func getRichText(page notion.Page, key string) (rst string) {
	prop := getPropValue(page, key).RichText
	if prop != nil {
		rst = ConvertRichText(prop)
	}
	return
}

func getSelect(page notion.Page, key string) (rst string) {
	prop := getPropValue(page, key).Select
	if prop != nil {
		rst = prop.Name
	}
	return
}

func getMultiSelect(page notion.Page, key string) (rst []string) {
	selects := getPropValue(page, key).MultiSelect
	rst = make([]string, len(selects))
	for i, options := range selects {
		rst[i] = options.Name
	}
	return
}

func getCheckbox(page notion.Page, key string) (rst *bool) {
	prop := getPropValue(page, key).Checkbox
	if prop != nil {
		rst = prop
	}
	return
}

func getDate(page notion.Page, key string) (rst time.Time) {
	prop := getPropValue(page, key).Date
	if prop != nil {
		rst = prop.Start.Time
		if prop.End != nil {
			rst = prop.End.Time
		}
	}
	return
}

func (np *NotionProp) GetFileName() string {
	var fileName string
	// file name  rule : filename > name > title
	if np.FileName != "" {
		fileName = np.FileName
	} else if np.Name != "" {
		fileName = np.Name
	} else {
		fileName = np.Title
	}
	if !np.IsSettingFile {

	}
	return fileName
}

func (np *NotionProp) GetTitle() (title string) {
	title = np.Name
	if np.Title != "" {
		title = np.Title
	}
	if np.IsSetting() {
		if np.FileName != "" {
			title = np.FileName
		}
	}
	return
}

func (np *NotionProp) IsSetting() bool {
	return np.Types == "setting"
}

// IsCustomNameMdFile no setting type but has custom file name
func (np *NotionProp) IsCustomNameMdFile() bool {
	return np.Types != "setting" && np.FileName != ""
}

func (np *NotionProp) IsFolder() bool {
	return np.Types == "folder"
}

func (np *NotionProp) getChildrenBlocks(block *MdBlock) {
	switch reflect.TypeOf(block.Block) {
	case reflect.TypeOf(&notion.QuoteBlock{}):
		block.children = block.Block.(*notion.QuoteBlock).Children
	case reflect.TypeOf(&notion.ToggleBlock{}):
		block.children = block.Block.(*notion.ToggleBlock).Children
	case reflect.TypeOf(&notion.ParagraphBlock{}):
		block.children = block.Block.(*notion.ParagraphBlock).Children
	case reflect.TypeOf(&notion.CalloutBlock{}):
		block.children = block.Block.(*notion.CalloutBlock).Children
	case reflect.TypeOf(&notion.BulletedListItemBlock{}):
		block.children = block.Block.(*notion.BulletedListItemBlock).Children
	case reflect.TypeOf(&notion.NumberedListItemBlock{}):
		block.children = block.Block.(*notion.NumberedListItemBlock).Children
	case reflect.TypeOf(&notion.ToDoBlock{}):
		block.children = block.Block.(*notion.ToDoBlock).Children
	case reflect.TypeOf(&notion.CodeBlock{}):
		block.children = block.Block.(*notion.CodeBlock).Children
	//case reflect.TypeOf(&notion.ColumnListBlock{}):
	//	return block.Block.(*notion.ColumnListBlock).Children
	case reflect.TypeOf(&notion.TableBlock{}):
		block.children = block.Block.(*notion.TableBlock).Children
	case reflect.TypeOf(&notion.SyncedBlock{}):
		block.children = block.Block.(*notion.SyncedBlock).Children
	case reflect.TypeOf(&notion.TemplateBlock{}):
		block.children = block.Block.(*notion.TemplateBlock).Children
	}
}

// 加载主配置文件（懒加载，鲁棒性处理）
func loadGlobalConfig() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	// 尝试从配置文件加载
	configPath := "notion-site.yaml"
	if data, err := ioutil.ReadFile(configPath); err == nil {
		var config Config
		if err := yaml.Unmarshal(data, &config); err == nil {
			globalConfig = &config
			if len(config.DynamicProps) > 0 {
				log.Printf("已加载动态属性配置: %s (%d个属性)", configPath, len(config.DynamicProps))
			}
			return globalConfig
		} else {
			log.Printf("解析配置文件失败: %s - %v", configPath, err)
		}
	}

	// 配置文件不存在或解析失败时，创建空配置确保程序正常运行
	globalConfig = &Config{DynamicProps: []PropDef{}}
	return globalConfig
}

// 处理动态属性
func (np *NotionProp) processDynamicProps(page notion.Page) {
	config := loadGlobalConfig()
	
	if len(config.DynamicProps) == 0 {
		// 没有动态属性配置，直接返回
		return
	}

	np.DynamicProps = make(map[string]interface{})
	
	for _, propDef := range config.DynamicProps {
		value := getDynamicPropValue(page, propDef)
		if value != nil || (propDef.DefaultValue != nil && propDef.DefaultValue != "") {
			// 使用小写键名以符合yaml约定
			key := strings.ToLower(propDef.Name)
			if value != nil {
				np.DynamicProps[key] = value
			} else {
				np.DynamicProps[key] = propDef.DefaultValue
			}
		}
	}
}

// 获取动态属性值
func getDynamicPropValue(page notion.Page, propDef PropDef) interface{} {
	defer func() {
		// 防止panic，确保程序鲁棒性
		if r := recover(); r != nil {
			log.Printf("获取属性 %s 时出错: %v", propDef.Name, r)
		}
	}()

	prop := getPropValue(page, propDef.Name)
	
	switch strings.ToLower(propDef.Type) {
	case "richtext":
		if prop.RichText != nil && len(prop.RichText) > 0 {
			return ConvertRichText(prop.RichText)
		}
	case "select":
		if prop.Select != nil {
			return prop.Select.Name
		}
	case "multiselect":
		if prop.MultiSelect != nil && len(prop.MultiSelect) > 0 {
			var result []string
			for _, sel := range prop.MultiSelect {
				result = append(result, sel.Name)
			}
			return result
		}
	case "number":
		if prop.Number != nil {
			return *prop.Number
		}
	case "checkbox":
		if prop.Checkbox != nil {
			return *prop.Checkbox
		}
	case "date":
		if prop.Date != nil {
			return prop.Date.Start.Time
		}
	case "title":
		if prop.Title != nil && len(prop.Title) > 0 {
			return ConvertRichText(prop.Title)
		}
	default:
		// 未知类型，尝试转换为字符串
		if prop.RichText != nil && len(prop.RichText) > 0 {
			return ConvertRichText(prop.RichText)
		} else if prop.Title != nil && len(prop.Title) > 0 {
			return ConvertRichText(prop.Title)
		}
	}
	
	return nil
}
