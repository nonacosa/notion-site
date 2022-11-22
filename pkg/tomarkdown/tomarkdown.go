package tomarkdown

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	utils "github.com/pkwenda/notion-site/pkg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
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
	extendedSyntaxBlocks            = []any{reflect.TypeOf(&notion.BookmarkBlock{}), reflect.TypeOf(&notion.CalloutBlock{})}
	blockTypeInExtendedSyntaxBlocks = func(bType any) bool {
		for _, blockType := range extendedSyntaxBlocks {
			if blockType == bType {
				return true
			}
		}

		return false
	}
	mediaBlocks          = []any{reflect.TypeOf(&notion.VideoBlock{}), reflect.TypeOf(&notion.ImageBlock{}), reflect.TypeOf(&notion.FileBlock{}), reflect.TypeOf(&notion.PDFBlock{}), reflect.TypeOf(&notion.AudioBlock{})}
	blockTypeMediaBlocks = func(bType any) bool {
		for _, blockType := range mediaBlocks {
			if blockType == reflect.TypeOf(bType) {
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

type MediaBlock struct {
	notion.ImageBlock
	notion.VideoBlock
	notion.PDFBlock
	notion.AudioBlock
	notion.FileBlock
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

func (tm *ToMarkdown) shouldSkipRender(bType any) bool {
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
	var lastBlockType any

	hasMoreTag := false
	for index, block := range blocks {
		if tm.shouldSkipRender(reflect.TypeOf(block)) {
			continue
		}

		mdb := MdBlock{
			Block: block,
			Depth: depth,
			Extra: tm.extra,
		}

		sameBlockIdx++
		if reflect.TypeOf(block) != lastBlockType {
			sameBlockIdx = 0
		}
		mdb.Extra["SameBlockIdx"] = sameBlockIdx

		if tm.FrontMatter["IsSetting"] == true {
			if reflect.TypeOf(block) == reflect.TypeOf(&notion.CodeBlock{}) {
				if err := tm.GenBlock("setting", mdb, false, true); err != nil {
					return err
				}
				lastBlockType = reflect.TypeOf(block)
				fmt.Println(fmt.Sprintf("Processing the %b th block ↓ type -> %s \n %s ", index, reflect.TypeOf(block), block.ID()))
				continue
			}
		}

		var err error
		switch reflect.TypeOf(block) {
		// todo image
		case reflect.TypeOf(&notion.ImageBlock{}):
			err = tm.downloadMedia(block.(*notion.ImageBlock))
		//todo hugo
		case reflect.TypeOf(&notion.BookmarkBlock{}):
			err = tm.injectBookmarkInfo(block.(*notion.BookmarkBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.VideoBlock{}):
			err = tm.injectVideoInfo(block.(*notion.VideoBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.FileBlock{}):
			err = tm.downloadMedia(block.(*notion.FileBlock))
			err = tm.injectFileInfo(block.(*notion.FileBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.LinkPreviewBlock{}):
			err = tm.todo(block.(*notion.LinkPreviewBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.LinkToPageBlock{}):
			err = tm.todo(block.(*notion.LinkToPageBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.EmbedBlock{}):
			err = tm.injectEmbedInfo(block.(*notion.EmbedBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.CalloutBlock{}):
			err = tm.injectCalloutInfo(block.(*notion.CalloutBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.BreadcrumbBlock{}):
			err = tm.todo(block.(*notion.BreadcrumbBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.ChildDatabaseBlock{}):
			err = tm.todo(block.(*notion.ChildDatabaseBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.ChildPageBlock{}):
			err = tm.todo(block.(*notion.ChildPageBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.PDFBlock{}):
			err = tm.downloadMedia(block.(*notion.PDFBlock))
			err = tm.injectFileInfo(block.(*notion.PDFBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.SyncedBlock{}):
			err = tm.todo(block.(*notion.SyncedBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.TemplateBlock{}):
			err = tm.todo(block.(*notion.TemplateBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.AudioBlock{}):
			err = tm.injectFileInfo(block.(*notion.AudioBlock), &mdb.Extra)
		case reflect.TypeOf(&notion.ToDoBlock{}):
			mdb.Block = block.(*notion.ToDoBlock)
		case reflect.TypeOf(&notion.TableBlock{}):
			mdb.Block = block.(*notion.TableBlock)
		default:
			print(1)
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
		if err := tm.GenBlock(utils.GetBlockType(block), mdb, addMoreTag, false); err != nil {
			return err
		}
		lastBlockType = reflect.TypeOf(block)
		fmt.Println(fmt.Sprintf("Processing the %b th block ↓ type -> %s \n %s ", index, reflect.TypeOf(block), block.ID()))

	}

	return nil
}

// 模板
func (tm *ToMarkdown) GenBlock(bType string, block MdBlock, addMoreTag bool, skip bool) error {
	funcs := sprig.TxtFuncMap()
	funcs["deref"] = func(i *bool) bool { return *i }
	funcs["rich2md"] = ConvertRichText
	funcs["table2md"] = ConvertTable
	funcs["log"] = func(p any) string {
		s, _ := json.Marshal(p)
		return string(s)
	}

	t := template.New(fmt.Sprintf("%s.ntpl", bType)).Funcs(funcs)
	tpl, err := t.ParseFS(mdTemplatesFS, fmt.Sprintf("templates/%s.*", bType))
	//t := template.New(fmt.Sprintf("%s.ntpl", "log")).Funcs(funcs)
	//tpl, err := t.ParseFS(mdTemplatesFS, fmt.Sprintf("templates/%s.*", "log"))
	if err != nil {
		return err
	}
	if bType == "code" {
		println(bType)
	}
	if err := tpl.Execute(tm.ContentBuffer, block); err != nil {
		return err
	}

	if !skip {
		if addMoreTag {
			tm.ContentBuffer.WriteString("<!--more-->")
		}

		if block.HasChildren() {
			block.Depth++
			return tm.GenContentBlocks(getChildrenBlocks(block), block.Depth)
		}
	}

	return nil
}

func (tm *ToMarkdown) downloadImage(media notion.ImageBlock) error {
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

func (tm *ToMarkdown) downloadMedia(dynamicMedia any) error {

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

	if blockTypeMediaBlocks(dynamicMedia) {
		if reflect.TypeOf(dynamicMedia) == reflect.TypeOf(&notion.ImageBlock{}) {
			media := dynamicMedia.(*notion.ImageBlock)
			if media.Type == notion.FileTypeExternal {
				media.External.URL, err = download(media.External.URL)
			}
			if media.Type == notion.FileTypeFile {
				media.File.URL, err = download(media.File.URL)
			}
		}
		if reflect.TypeOf(dynamicMedia) == reflect.TypeOf(&notion.FileBlock{}) {
			media := dynamicMedia.(*notion.FileBlock)
			if media.Type == notion.FileTypeExternal {
				media.External.URL, err = download(media.External.URL)
			}
			if media.Type == notion.FileTypeFile {
				media.File.URL, err = download(media.File.URL)
			}
		}
		if reflect.TypeOf(dynamicMedia) == reflect.TypeOf(&notion.VideoBlock{}) {
			media := dynamicMedia.(*notion.VideoBlock)
			if media.Type == notion.FileTypeExternal {
				media.External.URL, err = download(media.External.URL)
			}
			if media.Type == notion.FileTypeFile {
				media.File.URL, err = download(media.File.URL)
			}
		}
		if reflect.TypeOf(dynamicMedia) == reflect.TypeOf(&notion.PDFBlock{}) {
			media := dynamicMedia.(*notion.PDFBlock)
			if media.Type == notion.FileTypeExternal {
				media.External.URL, err = download(media.External.URL)
			}
			if media.Type == notion.FileTypeFile {
				media.File.URL, err = download(media.File.URL)
			}
		}
		if reflect.TypeOf(dynamicMedia) == reflect.TypeOf(&notion.AudioBlock{}) {
			media := dynamicMedia.(*notion.AudioBlock)
			if media.Type == notion.FileTypeExternal {
				media.External.URL, err = download(media.External.URL)
			}
			if media.Type == notion.FileTypeFile {
				media.File.URL, err = download(media.File.URL)
			}
		}
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
func (tm *ToMarkdown) injectBookmarkInfo(bookmark *notion.BookmarkBlock, extra *map[string]interface{}) error {
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

func (tm *ToMarkdown) injectVideoInfo(video *notion.VideoBlock, extra *map[string]interface{}) error {
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
func (tm *ToMarkdown) injectEmbedInfo(embed *notion.EmbedBlock, extra *map[string]interface{}) error {
	var plat = ""
	url := embed.URL
	if len(url) == 0 {
		url = "http://www.baidu.com"
	} else {
		if strings.Contains(url, utils.Bilibili) {
			url = utils.FindTextP(url, "video/")
			plat = "bilibili"
		}
		if strings.Contains(url, utils.Twitter) {
			url = url[strings.Index(url, "status/")+7:]
			url = utils.FindTextP(url, "status/")
			plat = "twitter"
		}
		if strings.Contains(url, utils.Gist) {
			url = strings.Join(strings.Split(utils.FindTextP(url, utils.Gist), "/"), " ")
			plat = "gist"
		}

	}
	(*extra)["Url"] = url
	(*extra)["Plat"] = plat
	return nil
}

// todo real file position
func (tm *ToMarkdown) injectFileInfo(file any, extra *map[string]interface{}) error {
	var url string
	if reflect.TypeOf(file) == reflect.TypeOf(&notion.FileBlock{}) {
		f := file.(*notion.FileBlock)
		if f.Type == notion.FileTypeExternal {
			url = f.External.URL
		}
		if f.Type == notion.FileTypeFile {
			url = f.File.URL
		}
	}
	if reflect.TypeOf(file) == reflect.TypeOf(&notion.PDFBlock{}) {
		f := file.(*notion.PDFBlock)
		if f.Type == notion.FileTypeExternal {
			url = f.External.URL
		}
		if f.Type == notion.FileTypeFile {
			url = f.File.URL
		}
	}
	if reflect.TypeOf(file) == reflect.TypeOf(&notion.AudioBlock{}) {
		f := file.(*notion.AudioBlock)
		if f.Type == notion.FileTypeExternal {
			url = f.External.URL
		}
		if f.Type == notion.FileTypeFile {
			url = f.File.URL
		}
	}
	(*extra)["Url"] = url
	name, _ := gotool.StrUtils.RemoveSuffix(url)
	(*extra)["FileName"] = name
	return nil
}
func (tm *ToMarkdown) injectCalloutInfo(callout *notion.CalloutBlock, extra *map[string]interface{}) error {
	var text = ""
	for _, richText := range callout.RichText {
		// todo if link ? or change highlight hugo
		text += richText.Text.Content
	}
	(*extra)["Emoji"] = callout.Icon.Emoji
	(*extra)["Text"] = text
	return nil
}

func (tm *ToMarkdown) todo(video any, extra *map[string]interface{}) error {

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

func ConvertTable(rows []notion.Block) string {
	buf := &bytes.Buffer{}

	if len(rows) == 0 {
		return ""
	}
	var head = ""
	l := len((rows[0]).(*notion.TableRowBlock).Cells)
	for i := 0; i < l; i++ {
		head += "| "
		if i == l-1 {
			head += "|\n"
		}
	}
	for i := 0; i < l; i++ {
		head += "| - "
		if i == l-1 {
			head += "|\n"
		}
	}
	buf.WriteString(head)
	for _, row := range rows {
		rowBlock := row.(*notion.TableRowBlock)
		buf.WriteString(ConvertRow(rowBlock))
	}

	return buf.String()
}

func ConvertRow(r *notion.TableRowBlock) string {
	var rowMd = ""
	for i, cell := range r.Cells {
		if i == 0 {
			rowMd += "|"
		}
		for _, rich := range cell {
			a := ConvertRich(rich)
			print(a)
			rowMd += " " + a + " |"

		}
		if i == len(r.Cells)-1 {
			rowMd += "\n"
		}
	}
	return rowMd
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
	switch reflect.TypeOf(block.Block) {
	case reflect.TypeOf(&notion.QuoteBlock{}):
		return block.Block.(*notion.QuoteBlock).Children
	case reflect.TypeOf(&notion.ToggleBlock{}):
		return block.Block.(*notion.ParagraphBlock).Children
	case reflect.TypeOf(&notion.ParagraphBlock{}):
		return block.Block.(*notion.CalloutBlock).Children
	case reflect.TypeOf(&notion.CalloutBlock{}):
		return block.Block.(*notion.BulletedListItemBlock).Children
	case reflect.TypeOf(&notion.BulletedListItemBlock{}):
		return block.Block.(*notion.QuoteBlock).Children
	case reflect.TypeOf(&notion.NumberedListItemBlock{}):
		return block.Block.(*notion.NumberedListItemBlock).Children
	case reflect.TypeOf(&notion.ToDoBlock{}):
		return block.Block.(*notion.ToDoBlock).Children
	case reflect.TypeOf(&notion.CodeBlock{}):
		return block.Block.(*notion.CodeBlock).Children
	case reflect.TypeOf(&notion.CodeBlock{}):
		return block.Block.(*notion.ColumnBlock).Children
	case reflect.TypeOf(&notion.ColumnListBlock{}):
		//todo return block.block.(*notion.ColumnListBlock).Children
	case reflect.TypeOf(&notion.TableBlock{}):
		return block.Block.(*notion.TableBlock).Children
	case reflect.TypeOf(&notion.SyncedBlock{}):
		return block.Block.(*notion.SyncedBlock).Children
	case reflect.TypeOf(&notion.TemplateBlock{}):
		return block.Block.(*notion.TemplateBlock).Children
	}

	return nil
}
