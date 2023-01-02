package pkg

import (
	"fmt"
	"github.com/dstotijn/go-notion"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// all user wr | group wr | other user wr
const defaultPermission = 0755
const mediaRelativePath = "media"
const defaultMarkdownName = "index.md"

type Files struct {
	Permission               uint32
	MediaPath                string
	Position                 string
	FileName                 string
	FileFolderPath           string
	FilePath                 string
	HomePath                 string
	DefaultMarkdownName      string
	DefaultMediaFolderName   string
	DefaultgalleryFolderName string
	currentWriter            io.Writer
	CurrentNTPL              string
}

func NewFiles(config Config) (files *Files) {
	files = &Files{
		Permission: defaultPermission,
		HomePath:   config.HomePath,
		//Position:               position,
		DefaultMarkdownName:    defaultMarkdownName,
		DefaultMediaFolderName: mediaRelativePath,
	}
	files.MediaPath = filepath.Join(config.HomePath, files.Position, mediaRelativePath)
	return
}

func (files *Files) mkdirHomePath() error {
	return os.MkdirAll(files.HomePath, os.FileMode(files.Permission))
}

func (files *Files) mkdirPositionPath(position string) error {
	err := os.MkdirAll(filepath.Join(files.HomePath, position), os.FileMode(files.Permission))
	if err != nil {
		fmt.Errorf("couldn't create content folder: %s", err)
	}
	return err
}

func (files *Files) mkdirPath(path string) error {
	err := os.MkdirAll(path, os.FileMode(files.Permission))
	if err != nil {
		fmt.Errorf("couldn't create content folder: %s", err)
	}
	return err
}

func (ns *NotionSite) getArticleFolderPath() string {
	escapedTitle := strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(strings.TrimSpace(ns.currentPageProp.Name)),
			"",
		),
		" ", "-",
	)
	if ns.config.GroupByMonth {
		return filepath.Join(ns.currentPageProp.CreateAt.Format("2006-01-02"), escapedTitle)
	}

	return escapedTitle
}

func (ns *NotionSite) getFilename() string {
	filename := ns.currentPageProp.GetFileName()
	name := strings.ReplaceAll(
		strings.ToValidUTF8(
			strings.ToLower(strings.TrimSpace(filename)),
			"",
		),
		" ", "-",
	)
	if !ns.currentPageProp.IsSettingFile && !strings.Contains(filename, ".md") {
		name += ".md"
	}
	return name
}

func (ns *NotionSite) SetFileInfo(position string) {
	ns.files.Position = position
	if ns.currentPageProp.IsSettingFile {
		ns.files.FileName = ns.getFilename()
		ns.files.FileFolderPath = filepath.Join(ns.config.HomePath, ns.files.Position)
		ns.files.FilePath = filepath.Join(ns.files.FileFolderPath, ns.files.FileName)
	} else if ns.currentPageProp.IsCustomNameFile {
		ns.files.FileName = ns.getFilename()
		ns.files.MediaPath = filepath.Join(ns.config.HomePath, ns.files.Position, mediaRelativePath)
		ns.files.FileFolderPath = filepath.Join(ns.config.HomePath, ns.files.Position)
		ns.files.FilePath = filepath.Join(ns.config.HomePath, ns.files.Position, ns.files.FileName)
	} else {
		ns.files.FileName = filepath.Join(ns.getArticleFolderPath(), defaultMarkdownName)
		ns.files.MediaPath = filepath.Join(ns.config.HomePath, ns.files.Position, ns.getArticleFolderPath(), mediaRelativePath)
		ns.files.FileFolderPath = filepath.Join(ns.config.HomePath, ns.files.Position, ns.getArticleFolderPath())
		ns.files.FilePath = filepath.Join(ns.files.FileFolderPath, defaultMarkdownName)
	}
}

func (files *Files) DownloadMedia(dynamicMedia any) error {

	download := func(imgURL string) (string, error) {
		var savePath string
		savePath = files.MediaPath
		resp, err := http.Get(imgURL)
		if err != nil {
			return "", err
		}

		imgFilename, err := files.saveTo(resp.Body, imgURL, savePath)
		if err != nil {
			return "", err
		}
		var convertWinPath = strings.ReplaceAll(filepath.Join(files.DefaultMediaFolderName, imgFilename), "\\", "/")

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

func (files *Files) saveTo(reader io.Reader, rawURL, distDir string) (string, error) {
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

func (files *Files) copyDir(src, dst string) error {
	_, err := os.Stat(src)
	if err != nil {
		return err
	}
	err = files.mkdirPath(dst)
	if err != nil {
		return err
	}
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		dstPath := filepath.Join(dst, path[len(src):])
		return files.copyFile(path, dstPath)
	})
	return err
}

func (files *Files) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}
