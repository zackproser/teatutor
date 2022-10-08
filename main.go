package main

// A simple example that shows how to retrieve a value from a Bubble Tea
// program after the Bubble Tea has exited.

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/go-wordwrap"
)

type Question struct {
	Prompt           string
	Choices          []string
	CorrectAnswerIdx int
}

var questions = []Question{
	{
		Prompt: "You need to provide AWS credentials to an EC2 instance so that an application running on the instance can contact the S3 and DynamoDB services. How should you provide AWS credentials to the instance?",
		Choices: []string{
			"Create an IAM role",
			"Create an IAM user. Generate security credentials for the IAM user, then write them to ~/.aws/credentials on the EC2 instance",
			"SSH into the EC2 instance. Export the ${AWS_ACCESS_KEY_ID} and ${AWS_SECRET_ACCESS_KEY} environment variables so that the application running on the instance can contact the other AWS services",
		},
		CorrectAnswerIdx: 0,
	},
	{
		Prompt:           "Is it a good idea to learn AWS?",
		Choices:          []string{"Yes", "No"},
		CorrectAnswerIdx: 0,
	},
}

var choices = []string{"Taro", "Coffee", "Lychee"}

type model struct {
	cursor       int
	choice       string
	current      int
	QuestionBank []Question
}

func initialModel() model {
	return model{
		cursor:       0,
		choice:       "",
		current:      0,
		QuestionBank: questions,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Send the choice on the channel and exit.
			m.choice = m.QuestionBank[m.current].Choices[m.cursor]
			return m, tea.Quit

		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.QuestionBank[m.current].Choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.QuestionBank[m.current].Choices) - 1
			}

		case "left", "h":
			m.current--
			if m.current < 0 {
				m.current = len(m.QuestionBank) - 1
			}

		case "right", "l":
			m.current++
			if m.current >= len(m.QuestionBank) {
				m.current = 0
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := strings.Builder{}

	currentQ := m.QuestionBank[m.current]

	s.WriteString(fmt.Sprintf("%s\n\n", wordwrap.WrapString(currentQ.Prompt, 65)))

	for i := 0; i < len(currentQ.Choices); i++ {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(wordwrap.WrapString(currentQ.Choices[i], 65))
		s.WriteString("\n")
	}
	s.WriteString("\n(press q to quit)\n")

	return s.String()
}

func main() {
	p := tea.NewProgram(initialModel())

	// Run returns the model as a tea.Model.
	err := p.Start()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}
}
