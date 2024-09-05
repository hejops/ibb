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
	thread      Thread
	cursor      int
	moveCount   int  // vim-like navigation (e.g. 5j)
	showComment bool // if false, show image (if available)

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

	case m.showComment && hasComment: // body will be displayed in View

	case m.showComment && !hasComment:
		m.showComment = false
		go Render(fname, sz)

	case !m.showComment && hasImage:
		go Render(fname, sz)

	case !m.showComment && !hasImage:
		m.showComment = true

	default:
		log.Println("unhandled", m.showComment, hasImage, hasComment)
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
	log.Println("moveCount:", m.moveCount)
}

func (m *ThreadViewer) move(n int) {
	if m.moveCount > 0 {
		n *= m.moveCount
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
	m.cursor = len(m.thread.Posts) - 1 // start at last post

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
		m.showComment = m.short
		// log.Println("showText:", m.showText)

	case tea.KeyMsg:
		s := msg.String()
		switch s {

		case "q", "esc":
			cmd = tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			m.updateMoveCount(s)
			cmd = nil

		case "y": // copy image url to clipboard
		case "s": // save image (move, rather)
		case "p": // play video urls (and/or webms)
		case "t": // toggle

		case " ": // toggle img<>text; in short mode, this field is irrelevant
			m.showComment = !m.showComment

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

var (
	blankEnum  = func(items list.Items, index int) string { return "" }
	isSelected = map[bool]string{true: ">", false: " "}
)

// View renders the program's UI, which is just a string. The view is
// rendered after every Update.
func (m *ThreadViewer) View() string {
	var scrolloff int
	switch m.short {
	case true:
		// scrolloff = (m.height - 3) / 2 // with border
		scrolloff = (m.height - 1) / 2
	case false:
		scrolloff = m.height / 4
	}

	start, end := getScrollWindow(m.cursor, &m.thread.Posts, scrolloff)

	posts := list.New().Enumerator(blankEnum)

	curr := m.thread.Posts[m.cursor]

	// log.Println("model height", m.height, "/ posts", end-start)

	for _, p := range m.thread.Posts[start:end] {
		if p == nil { // window indices may exceed that of Posts
			panic("oob!")
		}
		selected := isSelected[curr.Num == p.Num]

		var comment string
		switch {
		case m.short && p.Comment == "":
			comment, _ = p.imageUrl()
		default:
			comment = p.lineComment()
		}

		item := fmt.Sprintf("%s %s", selected, comment)

		// truncate (-4 is somewhat arbitrary)
		if len(item) > m.width-4 {
			item = item[:m.width-4]
		}
		posts.Item(item)
	}

	// imgCount

	title := " " + m.thread.Posts[0].Subject
	status := fmt.Sprintf(
		"https://boards.4chan.org/%s/thread/%d [%d/%d] %d ",
		m.thread.Board, m.thread.Posts[0].Num,
		m.cursor+1, len(m.thread.Posts),
		curr.Num,
	)
	status = lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().PaddingRight(m.width-len(title)-len(status)).Render(title),
		lipgloss.NewStyle().Render(status),
	)

	var panes string
	switch m.short {
	case true: // don't show images
		status = lipgloss.NewStyle().Underline(true).Render(status)
		panes = lipgloss.NewStyle().Width(m.width).Render(posts.String())

	case false:
		var body string
		if m.showComment {
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
