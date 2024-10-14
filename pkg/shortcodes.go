package pkg

import (
	"fmt"
	"github.com/dstotijn/go-notion"
	"github.com/otiai10/opengraph"
	"reflect"
	"strings"
	"time"
)

// injectBookmarkInfo set bookmark info into the extra map field
func (tm *ToMarkdown) injectBookmarkInfo(bookmark *notion.BookmarkBlock, extra *map[string]any) error {
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
	(*extra)["Url"] = og.URL
	(*extra)["Title"] = og.Title
	(*extra)["Description"] = og.Description
	(*extra)["Icon"] = og.Favicon
	return nil
}

func (tm *ToMarkdown) injectVideoInfo(video *notion.VideoBlock, extra *map[string]any) error {
	videoUrl := video.External.URL
	var id, plat string
	if strings.Contains(videoUrl, "youtube") {
		plat = "youtube"
		id = FindUrlContext(RegexYoutube, videoUrl)
	}
	(*extra)["Plat"] = plat
	(*extra)["Id"] = id
	return nil
}

func (tm *ToMarkdown) injectEmbedInfo(embed *notion.EmbedBlock, extra *map[string]any) error {
	var plat = ""
	url := embed.URL
	if len(url) == 0 {
		return nil
	} else {
		if strings.Contains(url, Bilibili) {
			url = FindUrlContext(RegexBili, url)
			plat = "bilibili"
		}
		if strings.Contains(url, Jsfiddle) {
			url = FindUrlContext(RegexJsfiddle, url)
			plat = "Jsfiddle"
		}
		if strings.Contains(url, Twitter) {
			user := FindUrlContext(RegexTwitterUser, url)
			url = FindUrlContext(RegexTwitterId, url)
			plat = "twitter"
			(*extra)["User"] = user
		}
		if strings.Contains(url, Gist) {
			url = strings.Join(strings.Split(FindTextP(url, Gist), "/"), " ")
			plat = "gist"
		}

	}
	(*extra)["Url"] = url
	(*extra)["Plat"] = plat
	return nil
}

// todo real file position
func (tm *ToMarkdown) injectFileInfo(file any, extra *map[string]any) error {
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
	name, _ := RemoveSuffix(url)
	(*extra)["FileName"] = name
	return nil
}

func (tm *ToMarkdown) injectCalloutInfo(callout *notion.CalloutBlock, extra *map[string]any) error {
	var text = ""
	for _, richText := range callout.RichText {
		// todo if link ? or change highlight hugo
		text += richText.Text.Content
	}
	(*extra)["Emoji"] = callout.Icon.Emoji
	(*extra)["Text"] = text
	return nil
}

// injectFrontMatter convert the prop to the front-matter
func (tm *ToMarkdown) injectFrontMatter(key string, property notion.DatabasePageProperty) {
	var fmv any

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
			fmv = prop.Format(time.RFC3339)
		}
	case *notion.Date:
		if prop != nil {
			fmv = prop.Start.Format(time.RFC3339)
		}
	case *notion.User:
		fmv = prop.Name
		tm.injectAuthorAvatar(prop.AvatarURL)
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

func (tm *ToMarkdown) injectAuthorAvatar(avatar string) {
	tm.FrontMatter["avatar"] = tm.downloadFrontMatterImage(avatar)
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

	if err := tm.Files.DownloadMedia(image); err != nil {
		return
	}
	if image.Type == notion.FileTypeExternal {
		tm.FrontMatter["image"] = image.External.URL
	}
	if image.Type == notion.FileTypeFile {
		tm.FrontMatter["image"] = image.File.URL
	}
}

func (tm *ToMarkdown) todo(video any, extra *map[string]any) error {
	return nil
}

func (tm *ToMarkdown) inject(mdb *MdBlock, blocks []notion.Block, index int) error {
	var err error
	block := mdb.Block
	switch reflect.TypeOf(block) {
	case reflect.TypeOf(&notion.ImageBlock{}):
		err = tm.Files.DownloadMedia(block.(*notion.ImageBlock))
	//todo hugo
	case reflect.TypeOf(&notion.BookmarkBlock{}):
		err = tm.injectBookmarkInfo(block.(*notion.BookmarkBlock), &mdb.Extra)
	case reflect.TypeOf(&notion.VideoBlock{}):
		err = tm.injectVideoInfo(block.(*notion.VideoBlock), &mdb.Extra)
	case reflect.TypeOf(&notion.FileBlock{}):
		err = tm.Files.DownloadMedia(block.(*notion.FileBlock))
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
		err = tm.Files.DownloadMedia(block.(*notion.PDFBlock))
		err = tm.injectFileInfo(block.(*notion.PDFBlock), &mdb.Extra)
	case reflect.TypeOf(&notion.SyncedBlock{}):
		err = tm.todo(block.(*notion.SyncedBlock), &mdb.Extra)
	case reflect.TypeOf(&notion.TemplateBlock{}):
		err = tm.todo(block.(*notion.TemplateBlock), &mdb.Extra)
	case reflect.TypeOf(&notion.AudioBlock{}):
		err = tm.Files.DownloadMedia(block.(*notion.AudioBlock))
		err = tm.injectFileInfo(block.(*notion.AudioBlock), &mdb.Extra)
	case reflect.TypeOf(&notion.ToDoBlock{}):
		mdb.Block = block.(*notion.ToDoBlock)
	case reflect.TypeOf(&notion.TableBlock{}):
		mdb.Block = block.(*notion.TableBlock)
	}
	return err
}
