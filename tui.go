package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

const (
	normalMode = iota
	doneTaskListMode
	additionalMode
	editMode
	helpMode

	usage = `

--Syntax--
{Task text}
	or
{Task text} t=({year}-{month}-{day})

--Normal Mode--

j - move cursor one line down
k - move cursor one line up
a - add a new task(move to additional mode)
d - remove a task
e - edit the task name(mode to edit mode)
h - help(switch to help mode)
x, enter - mark as done
t - switch to done tasks list mode
q - save tasks and close this app

--Done Tasks List Mode--

j - move cursor one line down
k - move cursor one line up
d - remove a task
t - switch to normal mode
x, enter - mark as not done
q - save tasks and close this app

--Additional Mode--

ctrl+q - switch to normal mode
enter - submit

--Edit Mode--

ctrl+q - switch to normal mode
enter - submit

--Help Mode--

q - switch to normal mode
`
)

type model struct {
	cursor       int
	mode         int
	latestTaskID int
	tasks        []*Task
	doneTasks    []*Task

	newTaskNameInput  input.Model
	editTaskNameInput input.Model
}

func initializeModel() tea.Model {
	tasks, doneTasks, ltID := loadTasksFromRepositoryFile()

	cursor := 0
	if len(tasks) != 0 {
		cursor = 1
	}

	newTaskNameModel := input.NewModel()
	newTaskNameModel.Placeholder = "New task name..."
	newTaskNameModel.Focus()
	editTaskNameModel := input.NewModel()
	editTaskNameModel.Focus()

	return model{
		cursor:            cursor,
		mode:              normalMode,
		latestTaskID:      ltID,
		tasks:             tasks,
		doneTasks:         doneTasks,
		newTaskNameInput:  newTaskNameModel,
		editTaskNameInput: editTaskNameModel,
	}
}

func (model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case normalMode:
		return m.normalUpdate(msg)
	case doneTaskListMode:
		return m.doneTaskListUpdate(msg)
	case additionalMode:
		return m.addingTaskUpdate(msg)
	case editMode:
		return m.editTaskUpdate(msg)
	case helpMode:
		return m.helpUpdate(msg)
	}

	return m, nil
}

func (m model) normalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			if m.cursor < len(m.tasks) {
				m.cursor++
			}
		case "k":
			if m.cursor > 1 {
				m.cursor--
			}
		case "a":
			m.mode = additionalMode
			return m, nil //input.Blink()
		case "d":
			if m.cursor == 0 {
				break
			}
			m.tasks = append(m.tasks[:m.cursor-1], m.tasks[m.cursor:]...)
			if len(m.tasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "e":
			if m.cursor == 0 {
				break
			}
			m.mode = editMode
			m.editTaskNameInput.Placeholder = m.tasks[m.cursor-1].Name
			return m, nil //input.Blink()
		case "h":
			m.mode = helpMode
		case "x", "enter":
			if m.cursor == 0 {
				break
			}

			t := m.tasks[m.cursor-1]
			t.IsDone = true
			m.doneTasks = append(m.doneTasks, t)
			m.tasks = append(m.tasks[:m.cursor-1], m.tasks[m.cursor:]...)

			if len(m.tasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "t":
			if m.mode == doneTaskListMode {
				if len(m.tasks) == 0 {
					m.cursor = 0
				} else {
					m.cursor = 1
				}
				m.mode = normalMode
				return m, nil
			}
			if len(m.doneTasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
			m.mode = doneTaskListMode
		case "q", "ctrl+c":
			m.saveTasks()
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) doneTaskListUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			if m.cursor < len(m.doneTasks) {
				m.cursor++
			}
		case "k":
			if m.cursor > 1 {
				m.cursor--
			}
		case "d":
			if m.cursor == 0 {
				break
			}
			m.doneTasks = append(m.doneTasks[:m.cursor-1], m.doneTasks[m.cursor:]...)
			if len(m.doneTasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "t":
			if len(m.tasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
			m.mode = normalMode
		case "x", "enter":
			if m.cursor == 0 {
				break
			}
			t := m.doneTasks[m.cursor-1]
			t.IsDone = false
			m.tasks = append(m.tasks, t)
			m.doneTasks = append(m.doneTasks[:m.cursor-1], m.doneTasks[m.cursor:]...)
			if len(m.doneTasks) == 0 {
				m.cursor = 0
			} else {
				m.cursor = 1
			}
		case "q", "ctrl+c":
			m.saveTasks()
			return m, tea.Quit
		}
	}

	return m, nil
}

func readInputTask(value string) (string, bool, time.Time) {
	values := strings.Split(value, "t=")
	var t time.Time
	var err error
	deadlined := false
	if len(values) == 2 {
		lay := "2006-01-02"
		t, err = time.Parse(lay, values[1])
		if err != nil {
			t = time.Now()
		}
		deadlined = true
	}
	return values[0], deadlined, t

}

func (m model) addingTaskUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.saveTasks()
			return m, tea.Quit
		case "ctrl+q":
			m.mode = normalMode
			m.newTaskNameInput.Reset()
			return m, nil
		case "enter":
			if m.newTaskNameInput.Value() == "" {
				return m, nil
			}
			m.latestTaskID++
			new_name, deadlined, t := readInputTask(m.newTaskNameInput.Value())
			m.tasks = append(m.tasks, &Task{
				ID:          m.latestTaskID,
				Name:        new_name,
				CreatedAt:   time.Now(),
				HasDeadline: deadlined,
				Deadline:    t,
			})

			m.cursor++
			m.mode = normalMode
			m.newTaskNameInput.Reset()
			return m, nil
		}
	}

	m.newTaskNameInput, cmd = m.newTaskNameInput.Update(msg)

	return m, cmd
}

func (m model) editTaskUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.saveTasks()
			return m, tea.Quit
		case "ctrl+q":
			m.mode = normalMode
			m.editTaskNameInput.Reset()
			return m, nil
		case "enter":
			if m.editTaskNameInput.Value() == "" {
				return m, nil
			}
			new_name, deadlined, t := readInputTask(m.editTaskNameInput.Value())
			m.tasks[m.cursor-1] = &Task{
				ID:          m.tasks[m.cursor-1].ID,
				Name:        new_name,
				CreatedAt:   time.Now(),
				HasDeadline: deadlined,
				Deadline:    t,
			}

			m.mode = normalMode
			m.editTaskNameInput.Reset()
			return m, nil
		}
	}

	m.editTaskNameInput, cmd = m.editTaskNameInput.Update(msg)

	return m, cmd
}

func (m model) helpUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.saveTasks()
			return m, tea.Quit
		case "q":
			m.mode = normalMode
		}
	}

	return m, nil
}

func (m model) View() string {
	switch m.mode {
	case normalMode, doneTaskListMode:
		return m.normalView()
	case additionalMode:
		return m.addingTaskView()
	case editMode:
		return m.editTaskView()
	case helpMode:
		return m.helpView()
	}

	return ""
}

func (m model) normalView() string {
	var s string
	var title termenv.Style
	var tasksToDisplay []*Task
	switch m.mode {
	case normalMode:
		if len(m.tasks) == 0 {
			return "You have no tasks. Press 'a' to add your task!\n"
		}
		title = termenv.String("TASKS")
		tasksToDisplay = m.tasks
	case doneTaskListMode:
		if len(m.doneTasks) == 0 {
			return "You have no done tasks.\n"
		}
		title = termenv.String("DONE TASKS")
		tasksToDisplay = m.doneTasks
	}
	title = title.Bold().Underline()
	s = fmt.Sprintf("%v\n\n", title)

	for i, v := range tasksToDisplay {
		cursor := termenv.String(" ")
		if m.cursor == i+1 {
			cursor = termenv.String(">").Foreground(termenv.ANSIYellow)
		}
		taskName := termenv.String(v.Name).Foreground(termenv.ANSIBrightGreen)
		taskName = taskName.Bold()
		timeLayout := "2006-01-02"
		start := termenv.String(v.CreatedAt.Format(timeLayout)).Foreground(termenv.ANSIBrightBlue)
		end := termenv.String(v.Deadline.Format(timeLayout)).Foreground(termenv.ANSIBrightCyan)

		if v.HasDeadline {
			s += fmt.Sprintf("%v #%d: %s (%s) -> ", cursor, v.ID, taskName, start)
			s += fmt.Sprintf("(%s) \n", end)
		} else {
			s += fmt.Sprintf("%v #%d: %s (%s)\n", cursor, v.ID, taskName, v.CreatedAt.Format(timeLayout))
		}
	}

	return s
}

func (m model) addingTaskView() string {
	title := termenv.String("Additional Mode").Bold().Underline()
	return fmt.Sprintf("%v\n\nInput the new task name\n\n%s\n", title, m.newTaskNameInput.View())
}

func (m model) editTaskView() string {
	title := termenv.String("Edit Mode").Bold().Underline()
	return fmt.Sprintf("%v\n\nInput the new task name\n\n%s\n", title, m.editTaskNameInput.View())
}

func (m model) helpView() string {
	title := termenv.String("USAGE").Bold().Underline()
	return fmt.Sprintf("%v"+usage, title)
}

func report(err error) {
	fmt.Printf("todo-cli: %s\n", err.Error())
	os.Exit(1)
}
