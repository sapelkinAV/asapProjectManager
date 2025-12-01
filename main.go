package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"sapelkin.av/asap_project_manager/project"
)

type addProjectModel struct {
	inputs    []textinput.Model
	cursor    int
	submitted bool
}

func initialModel() addProjectModel {

	nameInput := textinput.New()
	nameInput.Placeholder = "Project name"
	nameInput.Focus()
	pathInput := textinput.New()
	pathInput.Placeholder = "Project path"

	langInput := textinput.New()
	langInput.Placeholder = "Language (leave empty to guess)"
	inputs := []textinput.Model{nameInput, pathInput, langInput}

	return addProjectModel{
		inputs: inputs,
		cursor: 0,
	}

}

func (m addProjectModel) Init() tea.Cmd {
	return nil
}

func (m addProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c", "esc":

			return m, tea.Quit

		case "tab", "shift+tab", "enter":

			if msg.String() == "enter" {
				m.submitted = true
				return m, tea.Quit
			}

			s := msg.String()
			if s == "shift+tab" {
				m.cursor--
			} else {
				m.cursor++
			}

			if m.cursor >= len(m.inputs) {
				m.cursor = 0
			}

			if m.cursor < 0 {
				m.cursor = len(m.inputs) - 1
			}

			for i := 0; i < len(m.inputs); i++ {

				if i == m.cursor {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}

		default:

			var cmd tea.Cmd

			m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)

			return m, cmd

		}

	}

	return m, nil

}

func (m addProjectModel) View() string {

	s := "Add a new project\n\n"

	for i, input := range m.inputs {

		s += input.View()
		if i == m.cursor {
			s += " <--"
		}
		s += "\n"
	}

	s += "\nPress Enter to submit, Esc to quit"
	return s
}

func main() {

	if len(os.Args) == 1 {

		// Launch TUI
		p := tea.NewProgram(initialModel())
		m, err := p.Run()

		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		if addModel, ok := m.(addProjectModel); ok && addModel.submitted {
			name := addModel.inputs[0].Value()
			path := addModel.inputs[1].Value()
			language := addModel.inputs[2].Value()
			if language == "" {
				language = project.GuessLanguage(path)
			}

			if language == "" {
				fmt.Println("Could not guess language, please specify in the form")
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
			language = project.GuessLanguage(path)
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
