package main

// A simple example that shows how to retrieve a value from a Bubble Tea
// program after the Bubble Tea has exited.

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/go-wordwrap"
	"github.com/pterm/pterm"
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

type doneMsg int

type model struct {
	done         bool
	cursor       int
	current      int
	QuestionBank []Question
	answers      map[int]int
}

func initialModel() model {
	return model{
		cursor:       0,
		current:      0,
		QuestionBank: questions,
		answers:      make(map[int]int),
		done:         false,
	}
}

func getTableHeaders() []string {
	return []string{"Question", "Your response", "Correct"}
}

func (m model) recordAnswer(questionNumber, responseNumber int) {
	m.answers[questionNumber] = responseNumber
}

func renderCorrectColumn(a, b int) string {
	if a == b {
		return "true"
	}
	return "false"
}

func printResults(m model) string {
	td := pterm.TableData{
		getTableHeaders(),
	}

	for questionNum, responseNum := range m.answers {
		td = append(td, []string{
			wordwrap.WrapString(m.QuestionBank[questionNum].Prompt, 65),
			wordwrap.WrapString(m.QuestionBank[questionNum].Choices[responseNum], 65),
			renderCorrectColumn(m.QuestionBank[questionNum].CorrectAnswerIdx, responseNum),
		})
	}

	tblStr, _ := pterm.DefaultTable.
		WithHasHeader().
		WithData(td).
		Srender()

	return tblStr
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) NextQuestion() model {
	m.current++
	if m.current >= len(m.QuestionBank) {
		m.current = len(m.QuestionBank) - 1
	}
	m.cursor = 0
	return m
}

func (m model) PreviousQuestion() model {
	m.current--
	if m.current <= len(m.QuestionBank) {
		m.current = 0
	}
	m.cursor = 0
	return m
}

func (m model) SelectionCursorDown() model {
	m.cursor++
	if m.cursor >= len(m.QuestionBank[m.current].Choices) {
		m.cursor = 0
	}
	return m
}

func (m model) SelectionCursorUp() model {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.QuestionBank[m.current].Choices)
	}
	return m
}

func signalDone() tea.Msg {
	return doneMsg(1)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case doneMsg:
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Record user's submission
			m.recordAnswer(m.current, m.cursor)

			m.current++
			if m.current >= len(m.QuestionBank) {
				m.current = len(m.QuestionBank) - 1
				return m, signalDone
			}

		case "down", "j":
			m = m.SelectionCursorDown()

		case "up", "k":
			m = m.SelectionCursorUp()

		case "left", "h":
			m = m.PreviousQuestion()

		case "right", "l":
			m = m.NextQuestion()
		}
	}

	return m, nil
}

func (m model) View() string {
	s := strings.Builder{}

	if m.current >= len(m.QuestionBank) {
		m.current = len(m.QuestionBank) - 1
	}
	currentQ := m.QuestionBank[m.current]

	s.WriteString(fmt.Sprintf("Question #%d\n", m.current))
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
	s.WriteString("\n(press q to quit - {h, <-} for prev - {l, ->} for next)\n")

	return s.String()
}

func main() {
	p := tea.NewProgram(initialModel())

	finalModel, err := p.StartReturningModel()

	// Cast finalModel to our own model
	m, _ := finalModel.(model)

	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}
	fmt.Print(printResults(m))
	os.Exit(0)
}
