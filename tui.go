package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/charmbracelet/x/term"
)

type ThreadViewer struct {
	thread      Thread // contains .Posts
	cursor      int
	moveCount   int  // vim-like navigation (e.g. 5j)
	showComment bool // if false, show image (if available)
	catalog     bool // generally only affects View

	// TODO: ambiguous field names: thread / catalog

	searching bool
	input     string
	matches   []int

	height int
	width  int
	short  bool
}

// Write to $HOME/subject/time.ext
func (m *ThreadViewer) saveImage() error {
	post := m.currentPost()
	path, err := post.imagePath()
	if err != nil {
		panic(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	subj := m.thread.Posts[0].Subject
	dest := filepath.Join(
		home,
		strings.ToLower(subj),
		filepath.Base(path),
	)

	if strings.HasPrefix(path, home) {
		if _, err := os.Stat(dest); err == nil {
			return nil
		}
		log.Println(path, "->", dest)
		return os.Link(path, dest)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return os.WriteFile(dest, b, 0664)
}

// Render current image in background
func (m *ThreadViewer) display() {
	post := m.currentPost()

	fname, err := post.imagePath()
	if err == nil {
		post.download()
		log.Println("displaying:", fname, err)
	}

	hasImage := err == nil
	hasComment := post.Comment != ""

	sz := &Size{width: m.width, height: m.height}

	// ensure that going from text post -> img post displays the image
	m.showComment = !hasImage

	switch {
	case m.height < 50: // don't render
		// log.Println("too small")

	case m.showComment && !hasComment:
		m.showComment = false
		go Render(fname, sz)

	case !m.showComment && hasImage:
		go Render(fname, sz)

	case m.showComment && hasComment: // body will be displayed in View

	case !m.showComment && !hasImage:
		m.showComment = true

	default:
		panic("unhandled condition")
		// log.Println("unhandled", m.showComment, hasImage, hasComment)
	}
}

func (m *ThreadViewer) updateSearch() {
	m.matches = m.thread.filterPosts(m.input)
	m.cursor = 0
	log.Println("input:", m.input, len(m.thread.Posts), "posts", len(m.matches), "matches")
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
		m.cursor = 0
	case m.cursor < 0:
		m.cursor = len(m.thread.Posts) - 1
	}
	m.moveCount = 0
}

func (m *ThreadViewer) currentPost() *Post {
	if len(m.matches) > 0 { // get actual index via m.matches
		return m.thread.Posts[m.matches[m.cursor]]
	}
	return m.thread.Posts[m.cursor]
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

	// start thread view at last post. note that this is only triggered on
	// startup, and not on state transitions
	if !m.catalog {
		m.cursor = len(m.thread.Posts) - 1
	}

	_ = os.Mkdir(tmpDir, os.ModePerm)
	m.display() // doing this will render 1st image 2x on startup
	return nil
}

func (t *Thread) getIndex(id int) int {
	for i, p := range t.Posts {
		if p.Num == id {
			return i
		}
	}
	panic("not found")
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

	// log.Println("msg", msg, spew.Sdump(msg))

	switch msg := msg.(type) {

	// case tea.ClearScreenMsg: // no longer exported

	case tea.WindowSizeMsg:
		// log.Println(msg)
		m.width = msg.Width
		m.height = msg.Height
		m.short = m.height < 50
		m.showComment = m.short
		return m, cmd
		// log.Println("showText:", m.showText)

	case tea.KeyMsg:

		s := msg.String()

		// enter must be checked -before- possible state transitions!
		if m.searching {
			switch s {
			case "esc", "enter":
				m.searching = false
				return m, cmd
			case "backspace":
				if m.input == "" {
					m.searching = false
					break
				}
				m.input = m.input[:len(m.input)-1]
				m.updateSearch()
			default:
				m.input += string(msg.Runes[0])
				m.updateSearch()
			}
			return m, nil // do NOT redraw on search
		}

		// state transitions
		if m.catalog && s == "enter" {

			nt := getThread(m.thread.Board, m.currentPost().Num)
			m.thread = *nt
			m.cursor = 0 // TODO: could keep some kind of {thread_id: idx} history in a db
			m.catalog = false
			m.matches = nil
			m.input = ""
			return m, cmd

		} else if !m.catalog && s == "h" {

			go m.thread.cleanImages()
			id := m.thread.Posts[0].Num
			c := getCatalog(m.thread.Board)
			m.thread = Thread(c)
			m.cursor = m.thread.getIndex(id)
			m.catalog = true
			m.matches = nil
			m.input = ""
			return m, cmd

		}

		switch s {

		case "q", "esc":
			// TODO: cleanup tmp dir
			_ = os.RemoveAll(tmpDir)
			cmd = tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			m.updateMoveCount(s)
			cmd = nil

		case "/": // start search (not allowed in threads for now)
			if m.catalog {
				m.searching = true
			}
			cmd = nil

		case "ctrl+l": // redraw (like tty)

		// thread-only
		case "p": // play video urls (and/or webms)
		case "t": // toggle all posts / text posts only

		case "y": // copy current image url to clipboard
			url, err := m.currentPost().imageUrl()
			if err == nil {
				// https://github.com/rck/serve/blob/87b073e24bac82bd6f34434f2510b2a807d45982/main.go#L86
				// echo -n foo | xclip -sel c
				cmd := exec.Command("xclip", "-sel", "c", "-i")
				in, _ := cmd.StdinPipe()
				_ = cmd.Start()
				_, _ = in.Write([]byte(url))
				in.Close()
				_ = cmd.Wait()
			}
			cmd = nil

		case "r": // reload
			switch m.catalog {
			case true:
				m.thread.Posts = getCatalog(m.thread.Board).Posts
			case false:
				// oldLen := len(m.thread.Posts)
				m.thread.Posts = getThread(m.thread.Board, m.thread.Posts[0].Num).Posts
				// log.Println(oldLen, "->", len(m.thread.Posts))
			}
			m.matches = nil
			m.input = ""

		case "s": // save image (copy, rather)
			err := m.saveImage()
			if err != nil {
				panic(err)
			}
			m.move(1)

		case " ":
			// toggle img<>text
			// TODO: in short mode, this field is currently
			// irrelevant and thus does nothing
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

		default:
			log.Println("unhandled input:", s)

		}

	default: // if not nil, will spam redraws!
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

	posts := m.thread.Posts
	switch {
	case m.input != "" && len(m.matches) == 0:
		return "no matches"
	case m.input != "" && len(m.matches) > 0:
		var filtered []*Post
		for _, match := range m.matches {
			filtered = append(filtered, posts[match])
		}
		posts = filtered
	}

	start, end := getScrollWindow(m.cursor, &posts, scrolloff)

	postsList := list.New().Enumerator(blankEnum)

	curr := posts[m.cursor]

	// log.Println("view cursor at", m.cursor)
	// log.Println("model height", m.height, "/ posts", end-start)
	// log.Println(m.cursor, curr.Subject, curr.Comment)

	for _, p := range posts[start:end] {
		if p == nil { // window indices may exceed that of Posts
			panic("oob!")
		}
		selected := isSelected[curr.Num == p.Num]

		var item string
		switch {
		// TODO: imgCount
		case m.catalog && p.Subject != "":
			item = fmt.Sprintf("%s %s", selected, p.Subject)
		case m.catalog && p.Subject == "":
			item = fmt.Sprintf("%s %s", selected, p.lineComment())
		case !m.catalog:
			item = fmt.Sprintf("%s %s", selected, p.lineComment())
		}

		// truncate (-5 is somewhat arbitrary)
		// 6 chars of padding: border, space, cursor, space | space, border
		if len(item) > m.width-5 {
			item = item[:m.width-5]
		}
		postsList.Item(item)
	}

	header := m.header(curr.Num)

	var panes string
	switch m.short {
	case true: // replace border with underline, don't show images
		header = lipgloss.NewStyle().Underline(true).Render(header)
		panes = lipgloss.NewStyle().Width(m.width).Render(postsList.String())

	case false:
		var body string
		if m.showComment {
			body = curr.htmlComment()
		}

		panes = lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Width(m.width-3). // -1 for each border
				MaxHeight(m.height/2+2).
				Border(lipgloss.RoundedBorder()).
				Render(postsList.String()),
			lipgloss.NewStyle().Width(m.width).Render(body),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Right, header, panes)
}

func (m *ThreadViewer) header(currId int) (header string) {
	var total int
	switch len(m.matches) {
	case 0:
		total = len(m.thread.Posts)
	default:
		total = len(m.matches)
	}
	header = fmt.Sprintf("[%d/%d] %d ", m.cursor+1, total, currId)

	var title string
	switch m.catalog {
	case true:
		title = m.thread.Board
		header = fmt.Sprintf("https://boards.4chan.org/%s %s ", m.thread.Board, header)

	case false:
		title = m.thread.Posts[0].Subject
		header = fmt.Sprintf(
			"https://boards.4chan.org/%s/thread/%d %s ",
			m.thread.Board,
			m.thread.Posts[0].Num,
			header,
		)

	}

	switch {
	case m.searching && m.input == "":
		title = fmt.Sprintf("%s [type to start searching]", title)
	case m.searching && m.input != "":
		title = fmt.Sprintf("%s %s", title, m.input)
	case !m.searching && m.input != "":
		title = fmt.Sprintf("%s [%s]", title, m.input)
	}

	header = lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().PaddingRight(m.width-len(title)-len(header)).Render(" "+title),
		lipgloss.NewStyle().Render(header),
	)
	return header
}

func (t *Thread) cleanImages() {
	for _, p := range t.Posts {
		path, err := p.imagePath()
		if err != nil {
			continue
		}
		_ = os.Remove(path)
	}
}
