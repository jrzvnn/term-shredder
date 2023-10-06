package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"gopkg.in/yaml.v2"
)

type config struct {
	DownloadsDir string `yaml:"downloads_dir"`
}

var (
	files         []string
	currentFile   = 0
	fileList      *widgets.List
	fileListTitle *widgets.Paragraph
	done          chan struct{}
)

func main() {
	if err := ui.Init(); err != nil {
		fmt.Printf("Failed to initialize termui: %v\n", err)
		return
	}
	defer ui.Close()

	cfg, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	err = filepath.Walk(cfg.DownloadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Failed to access path: %v\n", err)
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Failed to walk directory: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No files found in the directory.")
		return
	}

	initUI()
	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				confirm := confirmExit()
				if confirm {
					return
				}
			case "<Enter>":
				deleteCurrentFile()
			case "<Space>":
				moveToNextFile()
			case "j":
				moveToPreviousFile()
			case "k":
				moveToNextFile()
			case "<Up>":
				moveToPreviousFile()
			case "<Down>":
				moveToNextFile()
			}
		case <-done:
			return
		}
	}
}

func loadConfig(filename string) (*config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	cfg := &config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return cfg, nil
}

func initUI() {
	termWidth, termHeight := ui.TerminalDimensions()
	fileListTitle = widgets.NewParagraph()
	fileListTitle.Text = "Files:"
	fileListTitle.SetRect(0, 0, termWidth, 3)

	fileList = widgets.NewList()
	fileList.Title = "Files"
	fileList.Rows = files
	fileList.TextStyle = ui.NewStyle(ui.ColorYellow)
	fileList.WrapText = false
	fileList.SetRect(0, 4, termWidth, termHeight-1)

	ui.Render(fileListTitle, fileList)

	done = make(chan struct{})
}

func deleteCurrentFile() {
	if currentFile < len(files) {
		err := os.Remove(files[currentFile])
		if err != nil {
			fmt.Printf("Failed to delete file: %v\n", err)
		}
		files = append(files[:currentFile], files[currentFile+1:]...)
	}

	fileList.Rows = files
	ui.Render(fileList)
}

func moveToPreviousFile() {
	if currentFile > 0 {
		currentFile--
		fileList.ScrollUp()
		ui.Render(fileListTitle, fileList)
	}
}

func moveToNextFile() {
	if currentFile < len(files)-1 {
		currentFile++
		fileList.ScrollDown()
		ui.Render(fileListTitle, fileList)
	} else {
		ui.Render(fileListTitle)
		close(done)
	}
}

func confirmExit() bool {
	termWidth, _ := ui.TerminalDimensions()

	confirmText := widgets.NewParagraph()
	confirmText.Text = " Are you sure you want to exit? (y/n)"
	confirmText.SetRect(0, 0, termWidth, 3)

	ui.Render(confirmText)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "y":
			return true
		case "n":
			ui.Render(fileListTitle, fileList)
			return false
		}
	}
}
