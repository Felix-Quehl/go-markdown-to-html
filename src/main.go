package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
)

const ENVIRONMENTAL_VARIABLE = "CONFIG"
const META_BLOCK_START = "```json\n"
const META_BLOCK_END = "```\n"
const MARKDOWN_FILE_ENDING = ".md"

type Configuration struct {
	Input         string
	Output        string
	TemplatePage  string
	TemplateIndex string
}
type Author struct {
	Name         string
	Mail         string
	Organization string
	ORCID        string
}
type MetaBlock struct {
	Title   string
	Date    time.Time
	Authors []Author
}
type Page struct {
	Title   string
	Date    string
	Authors []Author
	Content string
}

type Link struct {
	Title string
	Date  string
	Url   string
}

type Index struct {
	Links []Link
}

func loadConfig() (Configuration, error) {
	var configuration Configuration
	var err error
	path := os.Getenv(ENVIRONMENTAL_VARIABLE)
	if len(path) > 0 {
		var data []byte
		data, err = ioutil.ReadFile(path)
		if err == nil {
			err = json.Unmarshal([]byte(data), &configuration)
		}
	} else {
		err_msg := fmt.Sprintf("missing environmental variable '%s'", ENVIRONMENTAL_VARIABLE)
		err = errors.New(err_msg)
	}
	return configuration, err
}

func checkPathError(path string) error {
	_, err := os.Stat(path)
	return err
}

func getMetaBlock(text string) (MetaBlock, int, error) {
	var metaBlock MetaBlock
	var contentStart int
	var err error
	if strings.HasPrefix(text, META_BLOCK_START) {
		index := strings.Index(text, META_BLOCK_END)
		if index != -1 {
			metaBlockText := text[len(META_BLOCK_START):index]
			contentStart = index + len(META_BLOCK_END)
			err = json.Unmarshal([]byte(metaBlockText), &metaBlock)
		} else {
			err = errors.New("missing meta code block end")
		}
	} else {
		err = errors.New("missing meta code block start")
	}
	return metaBlock, contentStart, err
}

func renderMarkdown(text string) string {
	md := []byte(text)
	data := markdown.ToHTML(md, nil, nil)
	html := string(data)
	return html
}

func renderFile(path string) (Page, error) {
	var page Page
	data, err := ioutil.ReadFile(path)
	if err == nil {
		text := string(data)
		if len(text) > 0 {
			var contentStart int
			var metaBlock MetaBlock
			metaBlock, contentStart, err = getMetaBlock(text)
			if err == nil {
				text = text[contentStart:]
				text = renderMarkdown(text)
				page = Page{
					metaBlock.Title,
					metaBlock.Date.Format("2006-01-02"),
					metaBlock.Authors,
					text,
				}
			} else {
				msg := fmt.Sprintf("meta block error: %s", err)
				err = errors.New(msg)
			}
		} else {
			err = errors.New("file is empty")
		}
	}
	return page, err
}

func doTemplating(outputPath string, templatePath string, page Page) error {
	var file *os.File
	var templateObj *template.Template
	var err error

	file, err = os.Create(outputPath)
	if err == nil {
		defer file.Close()
		templateObj, err = template.ParseFiles(templatePath)
		if err == nil {
			err = templateObj.Execute(file, page)
		}
	}
	return err
}

func doIndex(outputPath string, templatePath string, index Index) error {

	var file *os.File
	var templateObj *template.Template
	var err error

	file, err = os.Create(outputPath)
	if err == nil {
		defer file.Close()
		templateObj, err = template.ParseFiles(templatePath)
		if err == nil {
			err = templateObj.Execute(file, index)
		}
	}
	return err
}

func renderFiles(inputPath string, outputPath string, templatePath string, templateIndex string) error {
	var content Index
	inputFiles, err := ioutil.ReadDir(inputPath)
	count := len(inputFiles)
	for index := 0; index < count; index++ {
		inputFile := inputFiles[index]
		fileName := inputFile.Name()
		if !inputFile.IsDir() && strings.HasSuffix(fileName, MARKDOWN_FILE_ENDING) {
			inputFilePath := fmt.Sprintf("%s/%s", inputPath, fileName)
			log.Print("processing: ", inputFilePath)
			var page Page
			page, err = renderFile(inputFilePath)
			if err == nil {
				htmlFileName := strings.ReplaceAll(fileName, MARKDOWN_FILE_ENDING, ".html")
				outputFilePath := fmt.Sprintf("%s/%s", outputPath, htmlFileName)
				err = doTemplating(outputFilePath, templatePath, page)
				if err == nil {
					link := Link{
						Title: page.Title,
						Date:  page.Date,
						Url:   fmt.Sprintf("/%s", htmlFileName),
					}
					content.Links = append(content.Links, link)
				}
			}
			if err != nil {
				log.Fatal("page render error: ", err)
			}
		}
	}
	indexHtmlPath := fmt.Sprintf("%s/index.html", outputPath)
	err2 := doIndex(
		indexHtmlPath,
		templateIndex,
		content,
	)
	if err2 != nil {
		log.Fatal("index render error: ", err2)
	}
	return err
}

func main() {
	configuration, err := loadConfig()
	if err != nil {
		log.Fatal("configuration file path: ", err)
		os.Exit(1)
	} else {
		log.Print("configuration was loaded")
	}
	if checkPathError(configuration.Input) != nil {
		log.Fatal("input directory error: ", err)
		os.Exit(2)
	} else {
		log.Print("input directory found")
	}
	if checkPathError(configuration.Output) != nil {
		log.Fatal("output directory error: ", err)
		os.Exit(3)
	} else {
		log.Print("output directory found")
	}

	err = renderFiles(
		configuration.Input,
		configuration.Output,
		configuration.TemplatePage,
		configuration.TemplateIndex,
	)
	if err != nil {
		log.Fatal("render error: ", err)
	}
}
