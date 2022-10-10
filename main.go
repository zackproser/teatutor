package main

// A simple example that shows how to retrieve a value from a Bubble Tea
// program after the Bubble Tea has exited.

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/mitchellh/go-wordwrap"
)

type Question struct {
	Prompt           string
	Choices          []string
	CorrectAnswerIdx int
}

var (
	FailureEmoji = "❌"
	SuccessEmoji = "✅"

	questions = []Question{
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
)

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
		return SuccessEmoji
	}
	return FailureEmoji
}

func (m model) RenderScore() string {
	totalQuestions := len(m.QuestionBank)
	numCorrect := 0
	for questionNum, responseNum := range m.answers {
		if m.QuestionBank[questionNum].CorrectAnswerIdx == responseNum {
			numCorrect++
		}
	}
	floatVal := (float64(numCorrect) * float64(100)) / float64(totalQuestions)
	return fmt.Sprintf("%.0f%%", floatVal)
}

func printResults(m model) string {
	sb := strings.Builder{}

	sb.WriteString("Your quiz results: \n\n")

	for questionNum, responseNum := range m.answers {
		sb.WriteString(fmt.Sprintf("Question #%d\n\n", questionNum))
		sb.WriteString(wordwrap.WrapString(m.QuestionBank[questionNum].Prompt, 65))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Your answer: \n\n"))
		sb.WriteString(
			fmt.Sprintf("%s %s",
				renderCorrectColumn(m.QuestionBank[questionNum].CorrectAnswerIdx, responseNum),
				wordwrap.WrapString(m.QuestionBank[questionNum].Choices[responseNum], 65)))
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("Your score: %s\n\n", m.RenderScore()))

	return sb.String()
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
		m.done = true
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

	if m.done {
		out := printResults(m)
		s.WriteString(out)
	} else {

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
	}

	return s.String()
}

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	_, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "no active terminal, skipping")
		return nil, nil
	}
	m := initialModel()
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func main() {
	// There are two main "modes" to run this bubbletea program in:
	// 1. Local mode, where you'd prefer to run this program if you're a developer working on it
	// 2. Server mode, where you're running this service so users can connect and run the bubbletea program over ssh

	if os.Getenv("QUIZ_SERVER") == "true" {
		host := "0.0.0.0"
		port := 23234

		s, err := wish.NewServer(
			wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
			wish.WithHostKeyPath(".ssh/term_info_ed25519"),
			wish.WithMiddleware(
				bm.Middleware(teaHandler),
				lm.Middleware(),
			),
		)
		if err != nil {
			log.Fatalln(err)
		}

		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("Starting SSH server on %s:%d", host, port)
		go func() {
			if err = s.ListenAndServe(); err != nil {
				log.Fatalln(err)
			}
		}()

		<-done
		log.Println("Stopping SSH server")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer func() { cancel() }()
		if err := s.Shutdown(ctx); err != nil {
			log.Fatalln(err)
		}
	} else {

		p := tea.NewProgram(initialModel())

		finalModel, err := p.StartReturningModel()

		// Cast finalModel to our own model
		m, _ := finalModel.(model)

		_ = m

		if err != nil {
			fmt.Println("Oh no:", err)
			os.Exit(1)
		}
		fmt.Println()
		//		fmt.Print(printResults(m))
		os.Exit(0)
	}
}
