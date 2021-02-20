package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/beeep"
	"github.com/spf13/pflag"
)

func emitSignal(title string, body string, icon string) error {
	err := beeep.Notify(title, body, icon)
	return err
}

func main() {
	var (
		showHelp      bool
		checkDeadline bool
	)
	pflag.BoolVarP(&showHelp, "help", "h", false, "show help")
	pflag.BoolVarP(&checkDeadline, "deadline", "d", false, "check deadline")
	pflag.Parse()
	if showHelp {
		pflag.Usage()
		fmt.Println(usage)
		return
	}
	if checkDeadline {
		todos, _, _ := loadTasksFromRepositoryFile()
		for _, t := range todos {
			now := time.Now()
			if t.HasDeadline {
				if now.Year() == t.Deadline.Year() && now.YearDay() == t.Deadline.YearDay() {
					if err := emitSignal("TODO", t.Name, ""); err != nil {
						panic(err)
					}
				}
			}
		}
	} else {
		p := tea.NewProgram(initializeModel())
		if err := p.Start(); err != nil {
			report(err)
		}
	}
}
