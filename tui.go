package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/charmbracelet/x/term"
)

type ThreadViewer struct {
	thread    Thread
	cursor    int
	moveCount int // vim-like navigation (e.g. 5j)
	showText  bool

	height int
	width  int
	short  bool
}

// Render current image in background
func (m *ThreadViewer) display() {
	post := m.thread.Posts[m.cursor]
	fname, err := post.download()

	hasImage := err == nil
	hasComment := post.Comment != ""

	sz := &Size{width: m.width, height: m.height}

	switch {
	case m.height < 50: // don't render
		// log.Println("too small")

	case m.showText && hasComment: // body will be displayed in View

	case m.showText && !hasComment:
		m.showText = false
		go Render(fname, sz)

	case !m.showText && hasImage:
		go Render(fname, sz)

	case !m.showText && !hasImage:
		m.showText = true

	default:
		log.Println("unhandled", m.showText, hasImage, hasComment)
	}
}

func (m *ThreadViewer) updateMoveCount(s string) {
	currDigits := strconv.Itoa(m.moveCount)

	var mergedDigits string
	switch m.moveCount {
	case 0:
		mergedDigits = s
	default:
		mergedDigits = currDigits + s
	}

	newCount, err := strconv.Atoi(mergedDigits)
	if err != nil {
		m.moveCount = 0
	}

	m.moveCount = newCount
	// log.Println(m.moveCount)
}

func (m *ThreadViewer) move(n int) {
	if m.moveCount > 0 {
		n = m.moveCount
	}
	m.cursor += n
	switch {
	case m.cursor > len(m.thread.Posts)-1:
		// m.cursor = len(m.thread.Posts) - 1
		m.cursor = 0
	case m.cursor < 0:
		// m.cursor = 0
		m.cursor = len(m.thread.Posts) - 1
	}
	m.moveCount = 0
}

// Init is the first function that will be called. It returns an optional
// initial command. To not perform an initial command return nil.
func (m *ThreadViewer) Init() tea.Cmd {
	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}
	m.width = w
	m.height = h
	m.short = m.height < 50

	// m.display() // doing this will render 1st image 2x on startup
	return nil
}

// Update is called when a message is received. Use it to inspect messages
// and, in response, update the model and/or send a command.
func (m *ThreadViewer) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	// TODO: race condition -- ClearScreen is not async, so multiple images
	// may be rendered
	cmd = tea.Sequence(
		tea.ClearScreen,
		func() tea.Msg { m.display(); return nil },
	)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// log.Println(msg)
		m.width = msg.Width
		m.height = msg.Height
		m.short = m.height < 50

	case tea.KeyMsg:
		s := msg.String()
		switch s {

		case "q", "esc":
			cmd = tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			m.updateMoveCount(s)
			cmd = nil

		case "s": // save image (move, rather)
		case "p": // play video urls (and/or webms)

		case " ": // toggle img<>text
			m.showText = !m.showText

		case "j":
			m.move(1)
		case "k":
			m.move(-1)

		case "pgup":
			m.move(-m.height / 4)
		case "pgdown":
			m.move(m.height / 4)

		case "g":
			switch m.moveCount {
			case 0:
				m.cursor = 0
			default:
				m.cursor = m.moveCount - 1
			}
			m.moveCount = 0

		case "G":
			m.cursor = len(m.thread.Posts) - 1
			m.moveCount = 0

		}

	default:
		cmd = nil
	}
	return m, cmd
}

// View renders the program's UI, which is just a string. The view is
// rendered after every Update.
func (m *ThreadViewer) View() string {
	var scrolloff int
	switch m.short {
	case true:
		scrolloff = (m.height - 3) / 2
	case false:
		scrolloff = m.height / 4
	}

	start, end := getScrollWindow(m.cursor, &m.thread.Posts, scrolloff)

	blankEnum := func(items list.Items, index int) string { return "" }
	posts := list.New().Enumerator(blankEnum)

	isSelected := map[bool]string{true: ">", false: " "}
	curr := m.thread.Posts[m.cursor]

	for _, p := range m.thread.Posts[start:end] {
		if p == nil { // window indices may exceed that of Posts
			panic("oob!")
		}

		// log.Println("choosing", i, end, p)
		selected := isSelected[curr.Num == p.Num]
		// TODO: if m.short, empty comment should show imageUrl
		item := fmt.Sprintf("%s %s", selected, p.lineComment())

		// truncate (-4 is somewhat arbitrary)
		if len(item) > m.width-4 {
			item = item[:m.width-4]
		}
		posts.Item(item)
	}

	status := fmt.Sprintf(
		"https://boards.4chan.org/%s/thread/%d [%d/%d] %d",
		m.thread.Board, m.thread.Posts[0].Num,
		m.cursor+1, len(m.thread.Posts)+1,
		curr.Num,
	)

	var panes string
	switch m.short {
	case true: // don't show images
		panes = lipgloss.NewStyle().
			Width(m.width - 2).   // -1 for each border
			Height(m.height - 3). // -1 for each border, and status
			Border(lipgloss.RoundedBorder()).
			Render(posts.String())

	case false:
		var body string
		if m.showText {
			// TODO: parse html? in particular, turn <br> into \n
			body = curr.Comment
			// io.ReadAll([]byte(curr.Comment))
		}

		panes = lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Width(m.width-2). // -1 for each border
				MaxHeight(m.height/2+2).
				Border(lipgloss.RoundedBorder()).
				Render(posts.String()),
			lipgloss.NewStyle().Width(m.width).Render(body),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Right, status, panes)
}
