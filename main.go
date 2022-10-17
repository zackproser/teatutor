package main

// A simple example that shows how to retrieve a value from a Bubble Tea
// program after the Bubble Tea has exited.

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/mitchellh/go-wordwrap"
	"github.com/muesli/termenv"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/zackproser/bubbletea-ssh-aws-quiz/questions"
	"golang.org/x/term"
)

const (
	padding  = 2
	maxWidth = 80
)

var IntroBannerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#13EC1F")).
	Align(lipgloss.Center).
	Margin(2).
	Padding(2)

var AppTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#29A4D6")).
	Inherit(IntroBannerStyle)

var BlinkingStyle = IntroBannerStyle.Copy().
	Blink(true).
	Foreground(lipgloss.Color("#FFA600"))

var HeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4"))

var QuestionStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#17E9A7"))

var (
	FailureEmoji  = "❌"
	SuccessEmoji  = "✅"
	QuestionEmoji = "⁉️"
)

type doneMsg int

type tickMsg time.Time

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
	spinner           spinner.Model
	progress          progress.Model
	percent           float64
	debugMsg          string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner.Frames = spinner.Monkey.Frames
	s.Spinner.FPS = 1 * time.Second

	p := progress.New()
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
		spinner:           s,
		progress:          p,
		percent:           0.0,
		debugMsg:          "",
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

// Return 85% of the supplied value - used to render horizontal lines that don't go
// all the way to the end of the line
func getHorizontalLineLength(total int) int {
	return int((float64(total) * float64(85)) / float64(100))
}

type Answer struct {
	QuestionNum int
	ResponseNum int
}

func sortUserResponses(m map[int]int) []Answer {
	answers := []Answer{}
	for k, v := range m {
		answers = append(answers, Answer{QuestionNum: k, ResponseNum: v})
	}
	sort.Slice(answers, func(i, j int) bool {
		return answers[i].QuestionNum < answers[j].QuestionNum
	})
	return answers
}

func printResults(m model) string {
	sb := strings.Builder{}

	tWidth, _, _ := term.GetSize(0)
	horizontalDelimiter := "."

	sortedUserResponses := sortUserResponses(m.answers)

	for _, userResponse := range sortedUserResponses {
		sb.WriteString("\n\n")

		sb.WriteString(fmt.Sprintf("# %s\n\n", wordwrap.WrapString(m.QuestionBank[userResponse.QuestionNum].Prompt, 65)))
		sb.WriteString(fmt.Sprintf("**Your answer**:\n\n"))
		sb.WriteString(
			fmt.Sprintf("%s %s\n\n",
				renderCorrectColumn(m.QuestionBank[userResponse.QuestionNum].CorrectAnswerIdx, userResponse.ResponseNum),
				wordwrap.WrapString(m.QuestionBank[userResponse.QuestionNum].Choices[userResponse.ResponseNum], 65)))
		// Render a visual border
		for i := 0; i < getHorizontalLineLength(tWidth); i++ {
			sb.WriteString(horizontalDelimiter)
		}
		sb.WriteString("\n\n")

		sb.WriteString("\n\n\n\n")
	}

	sb.WriteString(fmt.Sprintf("# Your score: %s", m.RenderScore()))

	return sb.String()
}

type initMsg int

type displayResultsMsg int

type stopIntroMsg int

func sendInitMsg() tea.Msg {
	return initMsg(1)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(sendInitMsg, m.spinner.Tick, m.progress.SetPercent(0), tickCmd())
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
	if m.current <= 0 {
		m.current = 0
	}
	m.cursor = 0
	return m
}

func (m model) SelectionCursorDown() model {
	if m.playingIntro {
		return m
	}

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
	if m.playingIntro {
		return m
	}

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

func sendWindowSizeMsg() tea.Msg {
	time.Sleep(500 * time.Millisecond)
	width, height, _ := term.GetSize(0)
	return tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		viewportCmd tea.Cmd
		spinnerCmd  tea.Cmd
		progressCmd tea.Cmd
		cmds        []tea.Cmd
	)
	switch msg := msg.(type) {

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

	// Sent when it's time to update the position of the progress bar relative to quiz completion
	case tickMsg:
		total := len(m.QuestionBank)
		percent := float64(m.current) / float64(total)
		percent = float64(int(percent*100)) / 100
		m.percent = percent
		cmds = append(cmds, tickCmd(), m.progress.SetPercent(percent))

		// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		// m.debugMsg = fmt.Sprintf("Received FrameMsg: %+v\n", msg)
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

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

		// Handle progress bar updates
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// If we're playing the intro and the user presses enter, it means they're impatient to get started
			if m.playingIntro {
				return m, stopIntro
			}

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
	m.viewport, viewportCmd = m.viewport.Update(msg)
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	progressModel, progCmd := m.progress.Update(msg)
	m.progress = progressModel.(progress.Model)
	progressCmd = progCmd

	cmds = append(cmds, viewportCmd, spinnerCmd, progressCmd)

	return m, tea.Batch(cmds...)
}

func (m model) RenderIntroView() string {
	sb := strings.Builder{}

	welcomeTo, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithRGB("WELCOME TO", pterm.NewRGB(255, 215, 0))).
		Srender()

	awsQuiz, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithRGB("AWS QUIZ", pterm.NewRGB(255, 215, 0))).
		Srender()

	overSSH, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithRGB("OVER SSH", pterm.NewRGB(255, 215, 0))).
		Srender()

	introBannerText := fmt.Sprintf("A Zachary Proser \n\n %s joint %s \n\n", m.spinner.View(), m.spinner.View())

	getStartedPrompt := BlinkingStyle.Render("[ Press ENTER to get started ]")

	fullBannerText := welcomeTo + "\n\n" + awsQuiz + "\n\n" + overSSH + "\n\n" + IntroBannerStyle.Render(introBannerText) + getStartedPrompt

	tWidth, _, _ := term.GetSize(0)

	sb.WriteString(lipgloss.PlaceHorizontal(tWidth, lipgloss.Center, fullBannerText))

	return sb.String()
}

// RenderCategorySelectionView is a markdown view that renders the picker allowing the user to select their topic of study
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

	rendered, _ := glamour.Render(s.String(), "dark")

	return rendered
}

func (m model) RenderQuizProgressView() string {
	pad := strings.Repeat(" ", padding)
	return "\n" + pad + m.progress.View() + "\n\n"
}

// RenderQuizView is a markdown view that represents the question prompt and multi-select interface
func (m model) RenderQuizView() string {
	if m.current >= len(m.QuestionBank) {
		m.current = len(m.QuestionBank) - 1
	}
	currentQ := m.QuestionBank[m.current]

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("# Question #%d\n\n", m.current+1))
	s.WriteString((fmt.Sprintf("**%s  %s %s**\n\n", QuestionEmoji, wordwrap.WrapString(currentQ.Prompt, 65), QuestionEmoji)))

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

	rendered, _ := glamour.Render(s.String(), "dark")
	return rendered
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
		s.WriteString(m.RenderQuizProgressView())
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
	return m, []tea.ProgramOption{tea.WithInput(s), tea.WithOutput(s)}
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

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
				bm.MiddlewareWithColorProfile(teaHandler, termenv.TrueColor),
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

		err := p.Start()
		if err != nil {
			fmt.Println("Error running Bubbletea program:", err)
			os.Exit(1)
		}
	}
}
