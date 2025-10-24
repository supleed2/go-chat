package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	c "go-chat/common"

	"github.com/alexflint/go-arg"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const manText string = `The current available commands are:
  man
    prints this message
  mv <string>
    set your nick
  ls
    get the available rooms
  cd <string>
    connect to a room
  who
    list users in the current room
  moo
    :)`

const mooText string = `
                 (__)
                 (oo)
           /------\/
          / |    ||
         *  /\---/\
            ~~   ~~
..."Have you mooed today?"...`

type showTim int

const (
	off showTim = iota
	short
	full
)

type model struct {
	kpHist  bool
	history viewport.Model
	msgs    []c.SMsg
	showTim showTim
	tz      time.Location
	input   textinput.Model
	idStyle lipgloss.Style
	pStyle  lipgloss.Style
	help    help.Model
	recvCh  chan c.SMsg
	sendCh  chan c.CMsg
	conn    *ws.Conn
	exitCh  chan exit
}

type args struct {
	Address     string  `arg:"positional" default:"gochat.8bit.lol" help:"address to connect to, without ws://" placeholder:"HOST[:PORT]"`
	KeepHistory bool    `arg:"-k" help:"append chat history when changing rooms, instead of clearing"`
	Timestamps  showTim `arg:"-t" default:"off" help:"display timestamps of messages, ctrl+t to cycle after startup [off, short, full]" placeholder:"CHOICE"`
	Nick        *string `arg:"-n" help:"attempt to automatically set nick after connecting"`
	Password    *string `arg:"-p" help:"password, if required"`
}

func (a *args) Version() string {
	return "v0.2.3"
}

func (a *args) Description() string {
	return "Go, chat!\nA basic irc-style chat client, written in Go using bubbletea and websockets"
}

func main() {
	ctx := context.Background()

	var a args
	arg.MustParse(&a)

	conn, _, err := ws.Dial(ctx, "ws://"+a.Address, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ws.StatusNormalClosure, "")

	local, err := time.LoadLocation("Local")
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(initModel(ctx, conn, a, *local), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func initModel(ctx context.Context, conn *ws.Conn, a args, tz time.Location) model {
	discCtx, disc := context.WithCancel(ctx)
	exitCh := make(chan exit)

	recvCh := make(chan c.SMsg)
	go func() {
		smsg := c.SMsg{}
		for {
			err := wsjson.Read(ctx, conn, &smsg)
			if err != nil {
				if ws.CloseStatus(err) != ws.StatusNormalClosure {
					log.Println(err)
				}
				disc()
				return
			}
			recvCh <- smsg
		}
	}()

	sendCh := make(chan c.CMsg)
	go func() {
		for {
			select {
			case cmsg := <-sendCh:
				err := wsjson.Write(ctx, conn, cmsg)
				if err != nil {
					recvCh <- c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("wsjson error when sending message: %v", err)}
				}
			case <-discCtx.Done():
				exitCh <- exit{}
				return
			}
		}
	}()

	ta := textinput.New()
	ta.Placeholder = "Send a message (or a command with /)"
	ta.Focus()
	ta.CharLimit = 128
	ta.Width = 60

	vp := viewport.New(60, 5)
	vp.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
	}

	if a.Nick != nil {
		login := *a.Nick
		if a.Password != nil {
			login += ":" + *a.Password
		}
		sendCh <- c.CMsg{Typ: c.Mv, Msg: login}
	}

	messages := []c.SMsg{{Tim: time.Now(), Id: "system", Msg: "Welcome to the chat room! Press Enter to send, /man for more info :)"}}

	return model{
		input:   ta,
		msgs:    messages,
		showTim: a.Timestamps,
		tz:      tz,
		kpHist:  a.KeepHistory,
		history: vp,
		idStyle: lipgloss.NewStyle().Width(60),
		pStyle:  lipgloss.NewStyle().Bold(true),
		help:    help.New(),
		recvCh:  recvCh,
		sendCh:  sendCh,
		conn:    conn,
		exitCh:  exitCh,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("go-chat by 8bit"),
		textinput.Blink,
		getNextSMsg(m.recvCh),
		getExitMsg(m.exitCh),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var tiCmd, vpCmd, smCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)
	m.history, vpCmd = m.history.Update(msg)

	switch msg := msg.(type) {
	case exit:
		return m, tea.Quit
	case c.SMsg:
		m.msgs = append(m.msgs, msg)
		m.history.SetContent(m.viewMessages())
		m.history.GotoBottom()
		smCmd = getNextSMsg(m.recvCh)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlT:
			m.showTim = (m.showTim + 1) % 3
			m.history.SetContent(m.viewMessages())
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			if text, ok := strings.CutPrefix(text, "/"); ok {
				if text == "man" {
					m.recvCh <- c.SMsg{Tim: time.Now(), Id: "system", Msg: manText}
				} else if text, ok := strings.CutPrefix(text, "mv "); ok {
					m.sendCh <- c.CMsg{Typ: c.Mv, Msg: text}
				} else if text == "ls" {
					m.sendCh <- c.CMsg{Typ: c.Ls, Msg: ""}
				} else if text, ok := strings.CutPrefix(text, "cd "); ok {
					if !m.kpHist {
						m.msgs = []c.SMsg{}
					}
					m.sendCh <- c.CMsg{Typ: c.Cd, Msg: text}
				} else if text == "who" {
					m.sendCh <- c.CMsg{Typ: c.Who, Msg: ""}
				} else if text, ok := strings.CutPrefix(text, "sudo "); ok {
					m.sendCh <- c.CMsg{Typ: c.Sudo, Msg: text}
				} else if text == "moo" {
					m.recvCh <- c.SMsg{Tim: time.Now(), Id: "cow", Msg: mooText}
				} else {
					m.recvCh <- c.SMsg{Tim: time.Now(), Id: "system", Msg: "Unrecognised command, use /man for more info"}
				}
			} else if text != "" {
				m.sendCh <- c.CMsg{Typ: c.Echo, Msg: text}
			}
			m.input.Reset()
		}
	case tea.WindowSizeMsg:
		m.history.Height = msg.Height - 2
		m.history.Width = msg.Width
		m.history.GotoBottom()
		m.input.Width = msg.Width - 3
		m.idStyle = m.idStyle.Width(msg.Width)
		m.help.Width = msg.Width
		m.history.SetContent(m.viewMessages())
	}

	return m, tea.Batch(tiCmd, vpCmd, smCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.history.View(),
		m.input.View(),
		m.help.View(m),
	)
}

func (m model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.history.KeyMap.PageDown,
		m.history.KeyMap.PageUp,
		m.history.KeyMap.Down,
		m.history.KeyMap.Up,
		key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle timestamps"),
		),
	}
}

func (m model) FullHelp() [][]key.Binding {
	return nil
}

func (m model) viewMessages() string {
	s := ""
	for i := range m.msgs {
		prefix := ""
		if m.showTim == short {
			prefix += m.msgs[i].Tim.In(&m.tz).Format(time.TimeOnly) + " "
		} else if m.showTim == full {
			prefix += m.msgs[i].Tim.In(&m.tz).Format(time.DateTime) + " "
		}
		if m.msgs[i].Id == "system" {
			prefix += m.pStyle.Foreground(lipgloss.Color("201")).Render("system:")
		} else {
			prefix += m.pStyle.Foreground(lipgloss.Color(prefixColor(m.msgs[i].Id))).Render(m.msgs[i].Id + ":")
		}
		s += m.idStyle.SetString(prefix).Render(m.msgs[i].Msg) + "\n"
	}
	return s[:len(s)-1]
}

func prefixColor(s string) string {
	if len(s) == 0 {
		s = "missing"
	}
	return fmt.Sprint(uint(s[0]+s[len(s)-1]) % 8)
}

func getNextSMsg(c <-chan c.SMsg) tea.Cmd {
	return func() tea.Msg {
		return <-c
	}
}

type exit struct{}

func getExitMsg(c <-chan exit) tea.Cmd {
	return func() tea.Msg {
		return <-c
	}
}

func (st *showTim) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "off":
		*st = off
	case "short":
		*st = short
	case "full":
		*st = full
	default:
		return fmt.Errorf("invalid choice: %s [off, short, full]", s)
	}
	return nil
}
