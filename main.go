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

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/mitchellh/go-wordwrap"
	"github.com/zackproser/bubbletea-ssh-aws-quiz/questions"
	"golang.org/x/term"
)

var IntroBannerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#13EC1F")).
	Align(lipgloss.Center).
	Width(24).
	Margin(2)

var HeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4"))

var (
	FailureEmoji = "❌"
	SuccessEmoji = "✅"
)

type doneMsg int

type model struct {
	done              bool
	categorySelection bool
	playingIntro      bool
	displayingResults bool
	cursor            int
	current           int
	categories        []string
	QuestionBank      []questions.Question
	answers           map[int]int
	results           string
	viewport          viewport.Model
}

func initialModel() model {
	return model{
		cursor:            0,
		current:           0,
		categories:        questions.ListCategories(),
		QuestionBank:      make([]questions.Question, 0),
		answers:           make(map[int]int),
		done:              false,
		playingIntro:      true,
		categorySelection: false,
		displayingResults: false,
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

	sb.WriteString("# Your quiz results: \n\n")

	for questionNum, responseNum := range m.answers {
		sb.WriteString(fmt.Sprintf("**Question** %d \n\n", questionNum))
		sb.WriteString(fmt.Sprintf("%s\n\n", wordwrap.WrapString(m.QuestionBank[questionNum].Prompt, 65)))
		sb.WriteString(fmt.Sprintf("**Your answer**:\n\n"))
		sb.WriteString(
			fmt.Sprintf("%s %s\n\n",
				renderCorrectColumn(m.QuestionBank[questionNum].CorrectAnswerIdx, responseNum),
				wordwrap.WrapString(m.QuestionBank[questionNum].Choices[responseNum], 65)))
	}

	sb.WriteString(fmt.Sprintf("# Your score: %s", m.RenderScore()))

	return sb.String()
}

type initMsg int

type displayResultsMsg int

type stopIntroMsg int

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return initMsg(1) }
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
	if m.categorySelection {
		if m.cursor >= len(m.categories) {
			m.cursor = 0
		}
	} else {
		if m.cursor >= len(m.QuestionBank[m.current].Choices) {
			m.cursor = 0
		}
	}
	return m
}

func (m model) SelectionCursorUp() model {
	m.cursor--
	if m.cursor < 0 {
		if m.categorySelection {
			m.cursor = len(m.categories)
		} else {
			m.cursor = len(m.QuestionBank[m.current].Choices)
		}
	}
	return m
}

func signalDisplayResults() tea.Msg {
	return displayResultsMsg(1)
}

func signalDone() tea.Msg {
	return doneMsg(1)
}

func stopIntro() tea.Msg {
	return stopIntroMsg(1)
}

func triggerDisableIntro() tea.Msg {
	<-time.After(3 * time.Second)
	return stopIntro()
}

func sendWindowSizeMsg() tea.Msg {
	width, height, _ := term.GetSize(0)
	return tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {

	case initMsg:
		return m, triggerDisableIntro

	case stopIntroMsg:
		m.playingIntro = false
		m.categorySelection = true
		return m, nil

	case displayResultsMsg:
		m.displayingResults = true
		m.results = printResults(m)
		return m, sendWindowSizeMsg

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if m.displayingResults {

			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = false
			out, _ := glamour.Render(m.results, "dark")
			m.viewport.SetContent(out)

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// If we're in category selection mode, and the user presses enter, they've selected a category
			// of questions to practice, so use it to filter questions

			if m.categorySelection {
				selectedCategory := m.categories[m.cursor]
				m.QuestionBank = questions.GetQuestionsByCategory(selectedCategory)
				m.categorySelection = false
				return m, nil
			}

			// Record user's submission
			m.recordAnswer(m.current, m.cursor)

			m.current++
			if m.current >= len(m.QuestionBank) {
				m.current = len(m.QuestionBank) - 1
				return m, signalDisplayResults
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

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) RenderIntroView() string {
	return IntroBannerStyle.Render("Welcome to\nAWS QUIZ OVER SSH\nA Zachary Proser joint\n")
}

func (m model) RenderCategorySelectionView() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("# Choose a category to practice\n\n"))

	for i := 0; i < len(m.categories); i++ {
		if m.cursor == i {
			s.WriteString(fmt.Sprintf("[%s] ", SuccessEmoji))
		} else {
			s.WriteString("[  ] ")
		}
		s.WriteString(wordwrap.WrapString(m.categories[i], 65))
		s.WriteString("\n\n")
	}
	s.WriteString("\n # (q quit - {up, k} up - {down, j} down - enter select)\n")

	return s.String()
}

func (m model) RenderQuizView() string {
	if m.current >= len(m.QuestionBank) {
		m.current = len(m.QuestionBank) - 1
	}
	currentQ := m.QuestionBank[m.current]

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("# Question #%d\n\n", m.current))
	s.WriteString(fmt.Sprintf("%s\n\n", wordwrap.WrapString(currentQ.Prompt, 65)))

	for i := 0; i < len(currentQ.Choices); i++ {
		if m.cursor == i {
			s.WriteString(fmt.Sprintf("[%s] ", SuccessEmoji))
		} else {
			s.WriteString("[  ] ")
		}
		s.WriteString(wordwrap.WrapString(currentQ.Choices[i], 65))
		s.WriteString("\n\n")
	}
	s.WriteString("\n # (press q to quit - {h, <-} for prev - {l, ->} for next)\n")
	return s.String()
}

func (m model) RenderResultsView() string {
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m model) View() string {
	s := strings.Builder{}

	if m.displayingResults {
		s.WriteString(m.RenderResultsView())
	} else if m.playingIntro {
		s.WriteString(m.RenderIntroView())
	} else if m.categorySelection {
		s.WriteString(m.RenderCategorySelectionView())
	} else {
		s.WriteString(m.RenderQuizView())
	}

	var final string
	// These two views are current styled indepdendently of glamour, so just return their string value
	if m.playingIntro || m.displayingResults {
		final = s.String()
	} else {
		// Everything else is intended to be a markdown view, so render it as such
		final, _ = glamour.Render(s.String(), "dark")
	}
	return final
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
	return m, []tea.ProgramOption{}
}

func (m model) headerView() string {
	title := HeaderStyle.Render("Your AWS SSH Quiz Results!")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := HeaderStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
