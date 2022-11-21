package tomarkdown

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/druidcaesa/gotool"
	"github.com/dstotijn/go-notion"
	"github.com/otiai10/opengraph"
	"gopkg.in/yaml.v3"
)

//go:embed templates
var mdTemplatesFS embed.FS

var (
	extendedSyntaxBlocks            = []notion.BlockType{notion.BlockTypeBookmark, notion.BlockTypeCallout}
	blockTypeInExtendedSyntaxBlocks = func(bType notion.BlockType) bool {
		for _, blockType := range extendedSyntaxBlocks {
			if blockType == bType {
				return true
			}
		}

		return false
	}
)

type MdBlock struct {
	notion.Block
	Depth int
	Extra map[string]interface{}
}

type ToMarkdown struct {
	// todo
	FrontMatter     map[string]interface{}
	ContentBuffer   *bytes.Buffer
	ImgSavePath     string
	ImgVisitPath    string
	ContentTemplate string

	extra map[string]interface{}
}

type FrontMatter struct {
	//Image         interface{}   `yaml:",flow"`
	Title         interface{}   `yaml:",flow"`
	Status        interface{}   `yaml:",flow"`
	Position      interface{}   `yaml:",flow"`
	Categories    []interface{} `yaml:",flow"`
	Tags          []interface{} `yaml:",flow"`
	Keywords      []interface{} `yaml:",flow"`
	CreateAt      interface{}   `yaml:",flow"`
	Author        interface{}   `yaml:",flow"`
	Date          interface{}   `yaml:",flow"`
	Lastmod       interface{}   `yaml:",flow"`
	Description   interface{}   `yaml:",flow"`
	Draft         interface{}   `yaml:",flow"`
	ExpiryDate    interface{}   `yaml:",flow"`
	PublishDate   interface{}   `yaml:",flow"`
	Show_comments interface{}   `yaml:",flow"`
	//Calculate Chinese word count accurately. Default is true
	IsCJKLanguage interface{} `yaml:",flow"`
	Slug          interface{} `yaml:",flow"`
}

func New() *ToMarkdown {
	return &ToMarkdown{
		FrontMatter:   make(map[string]interface{}),
		ContentBuffer: new(bytes.Buffer),
		extra:         make(map[string]interface{}),
	}
}

func (tm *ToMarkdown) WithFrontMatter(page notion.Page) {
	// todo image cover to image frontMatter
	tm.injectFrontMatterCover(page.Cover)
	pageProps := page.Properties.(notion.DatabasePageProperties)
	for fmKey, property := range pageProps {
		tm.injectFrontMatter(fmKey, property)
	}
}

func (tm *ToMarkdown) EnableExtendedSyntax(target string) {
	tm.extra["ExtendedSyntaxEnabled"] = true
	tm.extra["ExtendedSyntaxTarget"] = target
}

func (tm *ToMarkdown) ExtendedSyntaxEnabled() bool {
	if v, ok := tm.extra["ExtendedSyntaxEnabled"].(bool); ok {
		return v
	}

	return false
}

func (tm *ToMarkdown) shouldSkipRender(bType notion.BlockType) bool {
	return !tm.ExtendedSyntaxEnabled() && blockTypeInExtendedSyntaxBlocks(bType)
}

func (tm *ToMarkdown) GenerateTo(blocks []notion.Block, writer io.Writer, fm *FrontMatter) error {
	if tm.FrontMatter["IsSetting"] != true {
		if err := tm.GenFrontMatter(writer, fm); err != nil {
			return err
		}
	}

	if err := tm.GenContentBlocks(blocks, 0); err != nil {
		return err
	}

	if tm.ContentTemplate != "" {
		t, err := template.ParseFiles(tm.ContentTemplate)
		if err != nil {
			return err
		}
		return t.Execute(writer, tm)
	}

	_, err := io.Copy(writer, tm.ContentBuffer)
	return err
}

func (tm *ToMarkdown) GenFrontMatter(writer io.Writer, fm *FrontMatter) error {
	if len(tm.FrontMatter) == 0 {
		return nil
	}
	var imageKey string
	var imagePath string

	nfm := make(map[string]interface{})
	for key, value := range tm.FrontMatter {
		nfm[strings.ToLower(key)] = value
		// find image FrontMatter
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "image|") {
				imageKey = key
				imageOriginPath := v[len("image|"):]
				imagePath = tm.downloadFrontMatterImage(imageOriginPath)
				fmt.Println(imagePath)
			}
		default:

		}

	}
	if err := mapstructure.Decode(tm.FrontMatter, &fm); err != nil {
	}

	// chinese character statistics
	//fm.IsCJKLanguage = true
	frontMatters, err := yaml.Marshal(fm)

	if err != nil {
		return nil
	}

	buffer := new(bytes.Buffer)
	buffer.WriteString("---\n")
	buffer.Write(frontMatters)
	// todo write dynamic key image FrontMatter
	if len(imagePath) > 0 {
		buffer.WriteString(fmt.Sprintf("%s: \"%s\"\n", strings.ToLower(imageKey), imagePath))
	}
	buffer.WriteString("---\n")
	_, err = io.Copy(writer, buffer)
	return err
}

func (tm *ToMarkdown) GenContentBlocks(blocks []notion.Block, depth int) error {
	var sameBlockIdx int
	var lastBlockType notion.BlockType

	hasMoreTag := false
	for index, block := range blocks {
		if tm.shouldSkipRender(block.Type) {
			continue
		}

		mdb := MdBlock{
			Block: block,
			Depth: depth,
			Extra: tm.extra,
		}

		sameBlockIdx++
		if block.Type != lastBlockType {
			sameBlockIdx = 0
		}
		mdb.Extra["SameBlockIdx"] = sameBlockIdx

		if tm.FrontMatter["IsSetting"] == true {
			if block.Type == notion.BlockTypeCode {
				if err := tm.GenBlock("setting", mdb, false, true); err != nil {
					return err
				}
				lastBlockType = block.Type
				fmt.Println(fmt.Sprintf("Processing the %b th block ↓ type -> %s \n %s ", index, block.Type, block.ID))
				continue
			}
		}

		var err error
		switch block.Type {
		// todo image
		case notion.BlockTypeImage:
			err = tm.downloadMedia(block.Image)
		//todo hugo
		case notion.BlockTypeBookmark:
			err = tm.injectBookmarkInfo(block.Bookmark, &mdb.Extra)
		case notion.BlockTypeVideo:
			err = tm.injectVideoInfo(block.Video, &mdb.Extra)
		case notion.BlockTypeFile:
			err = tm.downloadMedia(block.File)
			err = tm.injectFileInfo(block.File, &mdb.Extra)
		case notion.BlockTypeLinkPreview:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypeLinkToPage:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypeEmbed:
			err = tm.injectEmbedInfo(block.Embed, &mdb.Extra)
		case notion.BlockTypeCallout:
			err = tm.injectCalloutInfo(block.Callout, &mdb.Extra)
		case notion.BlockTypeBreadCrumb:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypeChildDatabase:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypeChildPage:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypePDF:
			err = tm.downloadMedia(block.PDF)
			err = tm.injectFileInfo(block.PDF, &mdb.Extra)
		case notion.BlockTypeSyncedBlock:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypeTemplate:
			err = tm.todo(block.Video, &mdb.Extra)
		case notion.BlockTypeUnsupported:
			err = tm.todo(block.Video, &mdb.Extra)
		case "audio":
			// todo go-notion not support
		}

		if err != nil {
			return err
		}
		addMoreTag := false
		// todo configurable
		if tm.ContentBuffer.Len() > 60 && !hasMoreTag {
			addMoreTag = tm.ContentBuffer.Len() > 60
			hasMoreTag = true
		}
		if err := tm.GenBlock(block.Type, mdb, addMoreTag, false); err != nil {
			return err
		}
		lastBlockType = block.Type
		fmt.Println(fmt.Sprintf("Processing the %b th block ↓ type -> %s \n %s ", index, block.Type, block.ID))

	}

	return nil
}

// 模板
func (tm *ToMarkdown) GenBlock(bType notion.BlockType, block MdBlock, addMoreTag bool, skip bool) error {
	funcs := sprig.TxtFuncMap()
	funcs["deref"] = func(i *bool) bool { return *i }
	funcs["rich2md"] = ConvertRichText

	t := template.New(fmt.Sprintf("%s.ntpl", bType)).Funcs(funcs)
	tpl, err := t.ParseFS(mdTemplatesFS, fmt.Sprintf("templates/%s.*", bType))
	if err != nil {
		return err
	}
	if bType == "embed" {
		println(bType)
	}
	if err := tpl.Execute(tm.ContentBuffer, block); err != nil {
		return err
	}

	if !skip {
		if addMoreTag {
			tm.ContentBuffer.WriteString("<!--more-->")
		}

		if block.HasChildren {
			block.Depth++
			return tm.GenContentBlocks(getChildrenBlocks(block), block.Depth)
		}
	}

	return nil
}

func (tm *ToMarkdown) downloadMedia(media *notion.FileBlock) error {
	download := func(imgURL string) (string, error) {
		resp, err := http.Get(imgURL)
		if err != nil {
			return "", err
		}

		imgFilename, err := tm.saveTo(resp.Body, imgURL, tm.ImgSavePath)
		if err != nil {
			return "", err
		}
		var convertWinPath = strings.ReplaceAll(filepath.Join(tm.ImgVisitPath, imgFilename), "\\", "/")

		return convertWinPath, nil
	}

	var err error
	//if media.Type == notion.FileTypeFile {
	//	media.External.URL, err = download(media.External.URL)
	//}
	println(media.Type)
	if media.Type == notion.FileTypeExternal {
		media.External.URL, err = download(media.External.URL)
	}
	if media.Type == notion.FileTypeFile {
		media.File.URL, err = download(media.File.URL)
	}

	return err
}

func (tm *ToMarkdown) saveTo(reader io.Reader, rawURL, distDir string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("malformed url: %s", err)
	}

	// gen file name
	splitPaths := strings.Split(u.Path, "/")
	imageFilename := splitPaths[len(splitPaths)-1]
	if strings.HasPrefix(imageFilename, "Untitled.") {
		imageFilename = splitPaths[len(splitPaths)-2] + filepath.Ext(u.Path)
	}

	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", fmt.Errorf("%s: %s", distDir, err)
	}

	filename := fmt.Sprintf("%s_%s", u.Hostname(), imageFilename)
	out, err := os.Create(filepath.Join(distDir, filename))
	if err != nil {
		return "", fmt.Errorf("couldn't create image file: %s", err)
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return filename, err
}

// injectBookmarkInfo set bookmark info into the extra map field
func (tm *ToMarkdown) injectBookmarkInfo(bookmark *notion.Bookmark, extra *map[string]interface{}) error {
	og, err := opengraph.Fetch(bookmark.URL)
	if err != nil {
		return err
	}
	og.ToAbsURL()
	for _, img := range og.Image {
		if img != nil && img.URL != "" {
			(*extra)["Image"] = img.URL
			break
		}
	}
	(*extra)["Title"] = og.Title
	(*extra)["Description"] = og.Description
	return nil
}

func (tm *ToMarkdown) injectVideoInfo(video *notion.FileBlock, extra *map[string]interface{}) error {
	videoUrl := video.External.URL
	var id, plat string
	if strings.Contains(videoUrl, "youtube") {
		plat = "youtube"
		id = videoUrl[strings.Index(videoUrl, "?v=")+3:]
	}
	(*extra)["Plat"] = plat
	(*extra)["Id"] = id
	return nil
}
func (tm *ToMarkdown) injectEmbedInfo(embed *notion.Embed, extra *map[string]interface{}) error {
	var plat = ""
	url := embed.URL
	if len(url) == 0 {
		url = "http://www.baidu.com"
	} else {
		if strings.Contains(url, "bilibili.com") {
			url = url[strings.Index(url, "video/")+6:]
			plat = "bilibili"
		}
		if strings.Contains(url, "twitter.com") {
			url = url[strings.Index(url, "status/")+7:]
			plat = "twitter"
		}
		if strings.Contains(url, "gist.github.com") {
			urls := url[strings.Index(url, ".com/")+5:]
			url = ""
			for i, s := range strings.Split(urls, "/") {
				url += s
				if i < len(urls)-1 {
					url += " "
				}
			}
			plat = "gist"
		}

	}
	(*extra)["Url"] = url
	(*extra)["Plat"] = plat
	return nil
}

// todo real file position
func (tm *ToMarkdown) injectFileInfo(pdf *notion.FileBlock, extra *map[string]interface{}) error {
	url := pdf.File.URL
	(*extra)["Url"] = url
	name, _ := gotool.StrUtils.RemoveSuffix(url)
	(*extra)["FileName"] = name
	return nil
}
func (tm *ToMarkdown) injectCalloutInfo(callout *notion.Callout, extra *map[string]interface{}) error {
	var text = ""
	for _, richText := range callout.RichTextBlock.Text {
		// todo if link ? or change highlight hugo
		text += richText.Text.Content
	}
	(*extra)["Emoji"] = callout.Icon.Emoji
	(*extra)["Text"] = text
	return nil
}

func (tm *ToMarkdown) todo(video *notion.FileBlock, extra *map[string]interface{}) error {

	return nil
}

// injectFrontMatter convert the prop to the front-matter
func (tm *ToMarkdown) injectFrontMatter(key string, property notion.DatabasePageProperty) {
	var fmv interface{}

	switch prop := property.Value().(type) {
	case *notion.SelectOptions:
		if prop != nil {
			fmv = prop.Name
		}
	case []notion.SelectOptions:
		opts := make([]string, 0)
		for _, options := range prop {
			opts = append(opts, options.Name)
		}
		fmv = opts
	case []notion.RichText:
		if prop != nil {
			fmv = ConvertRichText(prop)
		}
	case *time.Time:
		if prop != nil {
			fmv = prop.Format("2006-01-02T15:04:05+07:00")
		}
	case *notion.Date:
		if prop != nil {
			fmv = prop.Start.Format("2006-01-02T15:04:05+07:00")
		}
	case *notion.User:
		fmv = prop.Name
	case *notion.File:
		fmv = prop.File.URL
	case []notion.File:
		// 最后一个图片最为 banner
		fmt.Printf("")
		for i, image := range prop {
			if i == len(prop)-1 {
				// todo notion image download real path
				fmv = fmt.Sprintf("image|%s", image.File.URL)
			}
		}
	case *notion.FileExternal:
		fmv = prop.URL
	case *notion.FileFile:
		fmv = prop.URL
	case *notion.FileBlock:
		fmv = prop.File.URL
	case *string:
		fmv = *prop
	case *float64:
		if prop != nil {
			fmv = *prop
		}
	default:
		if property.Type == "checkbox" {
			fmv = property.Checkbox
		} else {
			fmt.Printf("Unsupport prop: %s - %T\n", prop, prop)
		}
	}

	if fmv == nil {
		return
	}

	// todo support settings mapping relation
	tm.FrontMatter[key] = fmv
}

func (tm *ToMarkdown) injectFrontMatterCover(cover *notion.Cover) {
	if cover == nil {
		return
	}

	image := &notion.FileBlock{
		Type:     cover.Type,
		File:     cover.File,
		External: cover.External,
	}
	if err := tm.downloadMedia(image); err != nil {
		return
	}

	if image.Type == notion.FileTypeExternal {
		tm.FrontMatter["image"] = image.External.URL
	}
	if image.Type == notion.FileTypeFile {
		tm.FrontMatter["image"] = image.File.URL
	}
}

func (tm *ToMarkdown) downloadFrontMatterImage(url string) string {

	image := &notion.FileBlock{
		Type: "external",
		File: nil,
		External: &notion.FileExternal{
			URL: url,
		},
	}
	if err := tm.downloadMedia(image); err != nil {
		return ""
	}

	return image.External.URL
}

func ConvertRichText(t []notion.RichText) string {
	buf := &bytes.Buffer{}
	for _, word := range t {
		buf.WriteString(ConvertRich(word))
	}

	return buf.String()
}

func ConvertRich(t notion.RichText) string {
	switch t.Type {
	case notion.RichTextTypeText:
		if t.Text.Link != nil {
			return fmt.Sprintf(
				emphFormat(t.Annotations),
				fmt.Sprintf("[%s](%s)", t.Text.Content, t.Text.Link.URL),
			)
		}
		return fmt.Sprintf(emphFormat(t.Annotations), t.Text.Content)
	case notion.RichTextTypeEquation:
	case notion.RichTextTypeMention:
	}
	return ""
}

func emphFormat(a *notion.Annotations) (s string) {
	s = "%s"
	if a == nil {
		return
	}

	if a.Code {
		return "`%s`"
	}

	switch {
	case a.Bold && a.Italic:
		s = "***%s***"
	case a.Bold:
		s = "**%s**"
	case a.Italic:
		s = "*%s*"
	}

	if a.Underline {
		s = "__" + s + "__"
	} else if a.Strikethrough {
		s = "~~" + s + "~~"
	}

	// TODO: color

	return s
}

func getChildrenBlocks(block MdBlock) []notion.Block {
	switch block.Type {
	case notion.BlockTypeQuote:
		return block.Quote.Children
	case notion.BlockTypeToggle:
		return block.Toggle.Children
	case notion.BlockTypeParagraph:
		return block.Paragraph.Children
	case notion.BlockTypeCallout:
		return block.Callout.Children
	case notion.BlockTypeBulletedListItem:
		return block.BulletedListItem.Children
	case notion.BlockTypeNumberedListItem:
		return block.NumberedListItem.Children
	case notion.BlockTypeToDo:
		return block.ToDo.Children
	case notion.BlockTypeCode:
		return block.Code.Children
	case notion.BlockTypeColumn:
		return block.Column.Children
	case notion.BlockTypeColumnList:
		return block.ColumnList.Children
	case notion.BlockTypeTable:
		return block.Table.Children
	case notion.BlockTypeSyncedBlock:
		return block.SyncedBlock.Children
	case notion.BlockTypeTemplate:
		return block.Template.Children
	}

	return nil
}
