package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"sapelkin.av/asap_project_manager/project"
)

type projectItem struct {
	project project.Project
}

func (p projectItem) FilterValue() string {
	return fmt.Sprintf("%s - %s (%s)", p.project.Name, p.project.Path, p.project.Language)
}

type manageProjectsModel struct {
	list     list.Model
	projects []project.Project
}

func (m manageProjectsModel) Init() tea.Cmd {
	return nil
}

func (m manageProjectsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "a":
			// Load current config to get updated projects
			config, err := project.LoadConfig()
			if err != nil {
				// handle error, but for now, use current
			} else {
				m.projects = config.Projects
			}
			return initialAddModel(), nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m manageProjectsModel) View() string {
	return m.list.View() + "\n\nPress 'a' to add new project, 'q' to quit"
}

func initialManageModel(projects []project.Project) manageProjectsModel {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p}
	}

	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "Manage Projects"

	return manageProjectsModel{
		list:     l,
		projects: projects,
	}
}

type addProjectModel struct {
	inputs       []textinput.Model
	cursor       int
	submitted    bool
	languages    []string
	selectedLang int
	customLang   textinput.Model
	editMode     bool
}

func initialAddModel() addProjectModel {

	cwd, _ := os.Getwd()
	dirName := filepath.Base(cwd)

	nameInput := textinput.New()
	nameInput.Placeholder = "Project name"
	nameInput.SetValue(dirName)
	nameInput.Focus()

	pathInput := textinput.New()
	pathInput.Placeholder = "Project path"
	pathInput.SetValue(cwd)

	customLangInput := textinput.New()
	customLangInput.Placeholder = "Enter custom language"

	inputs := []textinput.Model{nameInput, pathInput}

	languages := project.GuessLanguage(cwd)
	if len(languages) == 0 {
		languages = []string{"other"}
	} else {
		languages = append(languages, "other")
	}

	selectedLang := len(languages) - 1 // Default to "Other"

	m := addProjectModel{
		inputs:       inputs,
		cursor:       0,
		languages:    languages,
		selectedLang: selectedLang,
		customLang:   customLangInput,
		editMode:     false,
	}

	// Auto-focus custom language if "other" is selected
	if m.selectedLang == len(m.languages)-1 {
		m.customLang.Focus()
	}

	return m
}

func (m addProjectModel) Init() tea.Cmd {
	return m.customLang.Focus()
}

func (m addProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+e":
			m.editMode = !m.editMode
			if !m.editMode {
				m.cursor = 0 // reset to language
			}

		case "esc":
			if m.editMode {
				m.editMode = false
				m.cursor = 0
				return m, nil
			}
			return m, tea.Quit

		case "tab", "shift+tab", "enter":

			if msg.String() == "enter" {
				m.submitted = true
				return m, tea.Quit
			}

			if !m.editMode {
				// In simple mode, tab cycles through language options
				if msg.String() == "tab" {
					m.selectedLang = (m.selectedLang + 1) % len(m.languages)
				} else if msg.String() == "shift+tab" {
					m.selectedLang = (m.selectedLang - 1 + len(m.languages)) % len(m.languages)
				}
			} else {
				// In edit mode, tab cycles through fields
				s := msg.String()
				if s == "shift+tab" {
					m.cursor--
				} else {
					m.cursor++
				}

				maxCursor := 3 // name, path, lang, custom
				if m.cursor > maxCursor {
					m.cursor = 0
				}
				if m.cursor < 0 {
					m.cursor = maxCursor
				}
			}

			// Update focus
			if m.editMode {
				for i := 0; i < len(m.inputs); i++ {
					if i == m.cursor {
						m.inputs[i].Focus()
					} else {
						m.inputs[i].Blur()
					}
				}
				if m.cursor == 3 {
					m.customLang.Focus()
				} else {
					m.customLang.Blur()
				}
			} else {
				// In simple mode, focus custom if Other selected
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				if m.selectedLang == len(m.languages)-1 {
					m.customLang.Focus()
				} else {
					m.customLang.Blur()
				}
			}

		default:

			var cmd tea.Cmd

			if m.editMode {
				if m.cursor < 2 {
					m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
				} else if m.cursor == 3 {
					m.customLang, cmd = m.customLang.Update(msg)
				}
				// cursor == 2 is language selection, ignore text input
			} else {
				if m.selectedLang == len(m.languages)-1 {
					m.customLang, cmd = m.customLang.Update(msg)
				}
				// else ignore text input in simple mode
			}

			return m, cmd

		}

	}

	return m, nil

}

func (m addProjectModel) View() string {

	s := "Add a new project\n\n"

	if m.editMode {
		for i, input := range m.inputs {
			s += input.View()
			if i == m.cursor {
				s += " <--"
			}
			s += "\n"
		}
	} else {
		s += fmt.Sprintf("Name: %s\n", m.inputs[0].Value())
		s += fmt.Sprintf("Path: %s\n", m.inputs[1].Value())
	}

	s += "\nLanguage:\n"
	for i, lang := range m.languages {
		cursor := " "
		if (m.editMode && m.cursor == 2) || (!m.editMode) {
			if i == m.selectedLang {
				cursor = ">"
			}
		}
		if lang == "other" {
			s += fmt.Sprintf("%s %s: %s\n", cursor, strings.Title(lang), m.customLang.View())
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, strings.Title(lang))
		}
	}

	if !m.editMode {
		s += "\nPress 'Ctrl+E' to edit name/path, "
	}
	s += "Enter to submit, Esc to quit"
	return s
}

func main() {

	if len(os.Args) == 1 {

		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			os.Exit(1)
		}

		// Load config
		config, err := project.LoadConfig()
		if err != nil {
			fmt.Println("Error loading config:", err)
			os.Exit(1)
		}

		// Check if current directory is already a project
		isProject := false
		for _, p := range config.Projects {
			if p.Path == cwd {
				isProject = true
				break
			}
		}

		var initialModel tea.Model
		if isProject {
			// Launch manage projects TUI
			initialModel = initialManageModel(config.Projects)
		} else {
			// Launch add project TUI
			initialModel = initialAddModel()
		}

		p := tea.NewProgram(initialModel)
		m, err := p.Run()

		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		if addModel, ok := m.(addProjectModel); ok && addModel.submitted {
			name := addModel.inputs[0].Value()
			path := addModel.inputs[1].Value()
			var language string
			if addModel.selectedLang == len(addModel.languages)-1 {
				language = addModel.customLang.Value()
			} else {
				language = addModel.languages[addModel.selectedLang]
			}

			if language == "" {
				fmt.Println("Please specify a language")
				os.Exit(1)
			}

			// Load config
			config, err := project.LoadConfig()
			if err != nil {
				fmt.Println("Error loading config:", err)
				os.Exit(1)
			}

			// Resolve path
			home, _ := os.UserHomeDir()
			if !filepath.IsAbs(path) {
				path = filepath.Join(home, path)
			}

			newProject := project.Project{
				Name:     name,
				Path:     path,
				Language: language,
			}

			config.Projects = append(config.Projects, newProject)
			if err := project.SaveConfig(config); err != nil {
				fmt.Println("Error saving config:", err)
				os.Exit(1)
			}

			fmt.Println("Project added successfully!")

		}

	} else {

		// CLI add
		if len(os.Args) < 3 {
			fmt.Println("Usage: asap-pm <name> <path> [language]")
			os.Exit(1)
		}

		name := os.Args[1]
		path := os.Args[2]

		language := ""

		if len(os.Args) > 3 {
			language = os.Args[3]
		}

		// Load config
		config, err := project.LoadConfig()
		if err != nil {
			fmt.Println("Error loading config:", err)
			os.Exit(1)
		}

		// Resolve path
		home, _ := os.UserHomeDir()
		if !filepath.IsAbs(path) {
			path = filepath.Join(home, path)
		}

		// Guess language if not provided
		if language == "" {
			languages := project.GuessLanguage(path)
			if len(languages) > 0 {
				language = languages[0]
			}
			if language == "" {
				fmt.Println("Could not guess language, please specify")
				os.Exit(1)
			}
		}

		newProject := project.Project{
			Name:     name,
			Path:     path,
			Language: language,
		}

		config.Projects = append(config.Projects, newProject)

		if err := project.SaveConfig(config); err != nil {
			fmt.Println("Error saving config:", err)
			os.Exit(1)
		}

		fmt.Println("Project added successfully!")

	}

}
