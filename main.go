package main

import (
	"fmt"
	"os"
	"os/exec"
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

func (p projectItem) Title() string {
	return p.project.Name
}

func (p projectItem) Description() string {
	return fmt.Sprintf("%s (%s)", p.project.Path, p.project.Language)
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
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
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
		case "e", "enter":
			// Edit selected project
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				if projItem, ok := selectedItem.(projectItem); ok {
					return initialEditModel(projItem.project, m.list.Index()), nil
				}
			}
		case "d":
			// Delete selected project
			selectedIdx := m.list.Index()
			if selectedIdx >= 0 && selectedIdx < len(m.projects) {
				// Load config
				config, err := project.LoadConfig()
				if err != nil {
					return m, nil
				}

				// Remove project
				config.Projects = append(config.Projects[:selectedIdx], config.Projects[selectedIdx+1:]...)

				// Save config
				if err := project.SaveConfig(config); err != nil {
					return m, nil
				}

				// Reload the list
				return initialManageModel(config.Projects), nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m manageProjectsModel) View() string {
	return m.list.View() + "\n\nPress 'a' to add, 'e/enter' to edit, 'd' to delete, 'q' to quit"
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

type editProjectModel struct {
	inputs       []textinput.Model
	cursor       int
	submitted    bool
	cancelled    bool
	languages    []string
	selectedLang int
	customLang   textinput.Model
	originalIdx  int
	useNeovim    bool
}

func initialEditModel(proj project.Project, idx int) editProjectModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Project name"
	nameInput.SetValue(proj.Name)
	nameInput.Focus()

	pathInput := textinput.New()
	pathInput.Placeholder = "Project path"
	pathInput.SetValue(proj.Path)

	customLangInput := textinput.New()
	customLangInput.Placeholder = "Enter custom language"

	inputs := []textinput.Model{nameInput, pathInput}

	languages := project.GuessLanguage(proj.Path)
	if len(languages) == 0 {
		languages = []string{"other"}
	} else {
		languages = append(languages, "other")
	}

	// Find the selected language
	selectedLang := len(languages) - 1 // Default to "Other"
	for i, lang := range languages {
		if lang == proj.Language {
			selectedLang = i
			break
		}
	}

	// If language not in list, it's custom
	if selectedLang == len(languages)-1 && proj.Language != "other" {
		customLangInput.SetValue(proj.Language)
	}

	m := editProjectModel{
		inputs:       inputs,
		cursor:       0,
		languages:    languages,
		selectedLang: selectedLang,
		customLang:   customLangInput,
		originalIdx:  idx,
		useNeovim:    false,
	}

	return m
}

func (m editProjectModel) Init() tea.Cmd {
	return nil
}

func (m editProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+n":
			m.useNeovim = true
			m.submitted = true
			return m, tea.Quit
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		case "tab", "shift+tab":
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

			// Update focus
			for i := 0; i < len(m.inputs); i++ {
				if i == m.cursor {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			if m.cursor == 2 {
				// Language selection, blur text inputs
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
				m.customLang.Blur()
			} else if m.cursor == 3 {
				m.customLang.Focus()
			} else {
				m.customLang.Blur()
			}

		case "enter":
			if m.cursor == 2 {
				// In language selection, enter moves to next field
				m.cursor++
				if m.selectedLang == len(m.languages)-1 {
					m.customLang.Focus()
				}
			} else {
				m.submitted = true
				return m, tea.Quit
			}

		case "up", "down":
			if m.cursor == 2 {
				// Navigate language options
				if msg.String() == "down" {
					m.selectedLang = (m.selectedLang + 1) % len(m.languages)
				} else {
					m.selectedLang = (m.selectedLang - 1 + len(m.languages)) % len(m.languages)
				}

				// Focus custom input if "other" selected
				if m.selectedLang == len(m.languages)-1 {
					m.customLang.Focus()
				} else {
					m.customLang.Blur()
				}
			}

		default:
			var cmd tea.Cmd
			if m.cursor < 2 {
				m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
			} else if m.cursor == 3 {
				m.customLang, cmd = m.customLang.Update(msg)
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m editProjectModel) View() string {
	s := "Edit Project\n\n"

	for i, input := range m.inputs {
		s += input.View()
		if i == m.cursor {
			s += " <--"
		}
		s += "\n"
	}

	s += "\nLanguage:\n"
	for i, lang := range m.languages {
		cursor := " "
		if m.cursor == 2 && i == m.selectedLang {
			cursor = ">"
		}
		if lang == "other" {
			customView := m.customLang.View()
			if m.cursor == 3 {
				customView += " <--"
			}
			s += fmt.Sprintf("%s %s: %s\n", cursor, strings.Title(lang), customView)
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, strings.Title(lang))
		}
	}

	s += "\nTab/Shift+Tab to navigate, Up/Down in language selection"
	s += "\nEnter to save, Ctrl+N to edit in Neovim, Esc to cancel"
	return s
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

func openInNeovim(proj project.Project) (project.Project, error) {
	// Create a temporary file with project data
	tmpFile, err := os.CreateTemp("", "project-*.txt")
	if err != nil {
		return proj, err
	}
	defer os.Remove(tmpFile.Name())

	// Write project data in a simple format
	content := fmt.Sprintf("name: %s\npath: %s\nlanguage: %s\n", proj.Name, proj.Path, proj.Language)
	if _, err := tmpFile.WriteString(content); err != nil {
		return proj, err
	}
	tmpFile.Close()

	// Open in neovim
	cmd := exec.Command("nvim", tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return proj, err
	}

	// Read back the edited content
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return proj, err
	}

	// Parse the edited content
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			switch key {
			case "name":
				proj.Name = value
			case "path":
				proj.Path = value
			case "language":
				proj.Language = value
			}
		}
	}

	return proj, nil
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

		} else if editModel, ok := m.(editProjectModel); ok && editModel.submitted {
			// Load config
			config, err := project.LoadConfig()
			if err != nil {
				fmt.Println("Error loading config:", err)
				os.Exit(1)
			}

			var updatedProject project.Project

			if editModel.useNeovim {
				// Get the original project
				originalProject := config.Projects[editModel.originalIdx]

				// Open in neovim
				updatedProject, err = openInNeovim(originalProject)
				if err != nil {
					fmt.Println("Error opening neovim:", err)
					os.Exit(1)
				}
			} else {
				// Get values from the TUI
				name := editModel.inputs[0].Value()
				path := editModel.inputs[1].Value()
				var language string
				if editModel.selectedLang == len(editModel.languages)-1 {
					language = editModel.customLang.Value()
				} else {
					language = editModel.languages[editModel.selectedLang]
				}

				if language == "" {
					fmt.Println("Please specify a language")
					os.Exit(1)
				}

				// Resolve path
				home, _ := os.UserHomeDir()
				if !filepath.IsAbs(path) {
					path = filepath.Join(home, path)
				}

				updatedProject = project.Project{
					Name:     name,
					Path:     path,
					Language: language,
				}
			}

			// Update the project in config
			config.Projects[editModel.originalIdx] = updatedProject

			if err := project.SaveConfig(config); err != nil {
				fmt.Println("Error saving config:", err)
				os.Exit(1)
			}

			fmt.Println("Project updated successfully!")

		} else if editModel, ok := m.(editProjectModel); ok && editModel.cancelled {
			// User cancelled, return to manage view
			config, err := project.LoadConfig()
			if err != nil {
				fmt.Println("Error loading config:", err)
				os.Exit(1)
			}

			p := tea.NewProgram(initialManageModel(config.Projects))
			_, err = p.Run()
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
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
