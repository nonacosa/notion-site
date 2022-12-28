package generator

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/dstotijn/go-notion"
	"github.com/mitchellh/mapstructure"
	utils "github.com/pkwenda/notion-site/pkg"
	"gopkg.in/yaml.v3"
	"io"
	"reflect"
	"strings"
	"text/template"
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
	children []notion.Block
	Depth    int
	Extra    map[string]interface{}
}

type MediaBlock struct {
	notion.ImageBlock
	notion.VideoBlock
	notion.PDFBlock
	notion.AudioBlock
	notion.FileBlock
}

type ToMarkdown struct {
	NotionProps       *NotionProp
	Files             *Files
	FrontMatter       map[string]interface{}
	ContentBuffer     *bytes.Buffer
	ImgSavePath       string
	GallerySavePath   string
	ImgVisitPath      string
	ArticleFolderPath string
	ContentTemplate   string
	extra             map[string]interface{}
}

type FrontMatter struct {
	Title        interface{}   `yaml:",flow"`
	Status       interface{}   `yaml:",flow"`
	Position     interface{}   `yaml:",flow"`
	Categories   []interface{} `yaml:",flow"`
	Tags         []interface{} `yaml:",flow"`
	Keywords     []interface{} `yaml:",flow"`
	CreateAt     interface{}   `yaml:",flow"`
	Author       interface{}   `yaml:",flow"`
	IsTranslated interface{}   `yaml:",flow"`
	Lastmod      interface{}   `yaml:",flow"`
	Description  interface{}   `yaml:",flow"`
	Draft        interface{}   `yaml:",flow"`
	ExpiryDate   interface{}   `yaml:",flow"`
	//PublishDate   interface{}   `yaml:",flow"`
	Show_comments interface{} `yaml:",flow"`
	//Calculate Chinese word count accurately. Default is true
	//IsCJKLanguage interface{} `yaml:",flow"`
	Slug  interface{} `yaml:",flow"`
	Image interface{} `yaml:",flow"`
}

func New() *ToMarkdown {
	return &ToMarkdown{
		FrontMatter:   make(map[string]interface{}),
		ContentBuffer: new(bytes.Buffer),
		extra:         make(map[string]interface{}),
	}
}

func (tm *ToMarkdown) WithFrontMatter(page notion.Page) {
	tm.injectFrontMatterCover(page.Cover)
	pageProps := page.Properties.(notion.DatabasePageProperties)
	for fmKey, property := range pageProps {
		tm.injectFrontMatter(fmKey, property)
	}
	tm.FrontMatter["Title"] = tm.NotionProps.GetTitle()
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

func (tm *ToMarkdown) GenerateTo(ns *NotionSite) error {
	if tm.NotionProps.IsSettingFile != true {
		if err := tm.GenFrontMatter(ns.files.currentWriter); err != nil {
			return err
		}
	}
	if err := tm.GenContentBlocks(ns.currentBlocks, 0); err != nil {
		return err
	}

	if tm.ContentTemplate != "" {
		t, err := template.ParseFiles(tm.ContentTemplate)
		if err != nil {
			return err
		}
		return t.Execute(ns.files.currentWriter, tm)
	}
	_, err := io.Copy(ns.files.currentWriter, tm.ContentBuffer)
	return err
}

func (tm *ToMarkdown) GenFrontMatter(writer io.Writer) error {
	fm := &FrontMatter{}
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
	// hugo open translate https://gohugo.io/variables/page/
	fm.IsTranslated = true
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
	var currentBlockType string

	hasMoreTag := false
	for index, block := range blocks {
		var addMoreTag = false
		currentBlockType = utils.GetBlockType(block)

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

		var generate = func(more bool) error {
			if err := tm.GenBlock(currentBlockType, mdb, addMoreTag, false); err != nil {
				return err
			}
			lastBlockType = reflect.TypeOf(block)
			fmt.Println(fmt.Sprintf("Processing the %d th %s tpye block  -> %s ", index, reflect.TypeOf(block), block.ID()))
			return nil
		}

		if tm.NotionProps.IsSettingFile == true {
			if reflect.TypeOf(block) == reflect.TypeOf(&notion.CodeBlock{}) {
				generate(false)
				continue
			}
		}

		err := tm.inject(&mdb, blocks, index)

		if err != nil {
			return err
		}

		// todo configurable
		if tm.ContentBuffer.Len() > 60 && !hasMoreTag {
			addMoreTag = tm.ContentBuffer.Len() > 60
			hasMoreTag = true
		}
		act := tm.GalleryAction(blocks, index)
		if act == "skip" {
			continue
		}

		if act == "write" {
			tm.Files.NeedSaveGallery = true
			tm.Files.CurrentNTPL = "gallery"
			currentBlockType = "gallery"
		}

		generate(addMoreTag)
	}
	return nil
}

func (tm *ToMarkdown) GalleryAction(blocks []notion.Block, i int) string {
	imageType := reflect.TypeOf(&notion.ImageBlock{})
	if tm.FrontMatter["Type"] != "gallery" {
		return "nothing"
	}
	if reflect.TypeOf(blocks[i]) != imageType {
		return "noting"
	}
	if len(blocks) == 1 {
		return "nothing"
	}
	if i == 0 && imageType == reflect.TypeOf(blocks[i+1]) {
		return "skip"
	}
	if i == len(blocks)-1 && imageType == reflect.TypeOf(blocks[i-1]) {
		return "write"
	}

	if imageType != reflect.TypeOf(blocks[i-1]) && imageType == reflect.TypeOf(blocks[i]) && imageType == reflect.TypeOf(blocks[i+1]) {
		return "skip"
	}

	if imageType == reflect.TypeOf(blocks[i-1]) && imageType == reflect.TypeOf(blocks[i+1]) {
		return "skip"
	}
	if imageType == reflect.TypeOf(blocks[i-1]) && imageType != reflect.TypeOf(blocks[i+1]) {
		return "write"
	}

	return "nothing"
}

// GenBlock notion to hugo shortcodes template
func (tm *ToMarkdown) GenBlock(bType string, block MdBlock, addMoreTag bool, skip bool) error {
	if tm.NotionProps.IsSettingFile == true {
		bType = "noop"
	}
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
			tm.NotionProps.getChildrenBlocks(&block)
			return tm.GenContentBlocks(block.children, block.Depth)
		}
	}

	return nil
}

func (tm *ToMarkdown) downloadFrontMatterImage(url string) string {

	image := &notion.FileBlock{
		Type: "external",
		File: nil,
		External: &notion.FileExternal{
			URL: url,
		},
	}
	if err := tm.Files.DownloadMedia(image); err != nil {
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
		if strings.TrimSpace(t.Text.Content) == "" {
			return ""
		}
		return fmt.Sprintf(emphFormat(t.Annotations), strings.TrimSpace(t.Text.Content))
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
		s = " ***%s***"
	case a.Bold:
		s = " **%s**"
	case a.Italic:
		s = " *%s*"
	}
	if a.Underline {
		s = "__" + s + "__"
	} else if a.Strikethrough {
		s = "~~" + s + "~~"
	}
	s = textColor(a, s)
	return s
}

func textColor(a *notion.Annotations, text string) (s string) {
	s = text
	if a.Color == "default" {
		return
	}

	var cssKey = "color"
	if strings.Contains(string(a.Color), "_background") {
		cssKey = "background-color"
	}
	s = fmt.Sprintf(`<span style="%s: %s;">%s</span>`, cssKey, ColorMap[string(a.Color)], text)
	return
}
