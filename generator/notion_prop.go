package generator

import (
	"github.com/dstotijn/go-notion"
	"reflect"
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
	Name          string
	Title         string
	Status        string
	Categories    string
	Tags          []string
	Position      string
	FileName      string
	Description   string
	CreateAt      *time.Time
	Author        string
	LastMod       time.Time
	ExpiryDate    time.Time
	PublishDate   time.Time
	ShowComments  *bool
	Slug          string
	Types         string
	IsSettingFile bool
}

func NewNotionProp(page notion.Page) (np *NotionProp) {
	np = &NotionProp{
		Name:        getTitle(page, nameProp),
		Title:       getRichText(page, titleProp),
		Status:      getSelect(page, statusProp),
		Categories:  getSelect(page, categoriesProp),
		Tags:        getMultiSelect(page, TagsProp),
		Position:    getSelect(page, PositionProp),
		FileName:    getRichText(page, fileNameProp),
		Description: getRichText(page, descriptionProp),
		CreateAt:    getPropValue(page, createAtProp).CreatedTime,
		//Author: author,
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

	if !np.IsSetting() {
		return np.Name
	}
	if np.FileName != "" {
		fileName = np.FileName
	} else if np.Title != "" {
		fileName = np.Title
	} else {
		fileName = np.Name
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

func (np *NotionProp) getChildrenBlocks(block *MdBlock) {
	switch reflect.TypeOf(block.Block) {
	case reflect.TypeOf(&notion.QuoteBlock{}):
		block.children = block.Block.(*notion.QuoteBlock).Children
	case reflect.TypeOf(&notion.ToggleBlock{}):
		block.children = block.Block.(*notion.ParagraphBlock).Children
	case reflect.TypeOf(&notion.ParagraphBlock{}):
		block.children = block.Block.(*notion.CalloutBlock).Children
	case reflect.TypeOf(&notion.CalloutBlock{}):
		block.children = block.Block.(*notion.BulletedListItemBlock).Children
	case reflect.TypeOf(&notion.BulletedListItemBlock{}):
		block.children = block.Block.(*notion.QuoteBlock).Children
	case reflect.TypeOf(&notion.NumberedListItemBlock{}):
		block.children = block.Block.(*notion.NumberedListItemBlock).Children
	case reflect.TypeOf(&notion.ToDoBlock{}):
		block.children = block.Block.(*notion.ToDoBlock).Children
	case reflect.TypeOf(&notion.CodeBlock{}):
		block.children = block.Block.(*notion.CodeBlock).Children
	case reflect.TypeOf(&notion.CodeBlock{}):
		block.children = block.Block.(*notion.ColumnBlock).Children
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
