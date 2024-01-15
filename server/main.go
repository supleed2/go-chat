package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	c "go-chat/common"

	"github.com/alexflint/go-arg"
	ws "nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type user struct {
	room string
	nick string
}

type conns struct {
	sm sync.Mutex
	cm map[*ws.Conn]user
}

type server struct {
	admin string
	logFn func(string, ...interface{})
	conns *conns
	rooms map[string]struct{}
	rhist map[string][]c.SMsg
	rhlen int
	nickm map[string]string
}

type args struct {
	Admin   string  `arg:"-a" default:"8bit" help:"admin user nick, allows access to /sudo" placeholder:"NICK"`
	HistLen uint    `arg:"-l" default:"10" help:"set message history size" placeholder:"N"`
	Port    uint    `arg:"positional" default:"0" help:"port to listen on, random available port if not set"`
	NickMap *string `arg:"-n" help:"path to nick:pass JSON file" placeholder:"FILE"`
}

func (a *args) Version() string {
	return "v0.1.2"
}

func (a *args) Description() string {
	return "Go, chat! Server\nA basic irc-style chat server, written in Go using websockets"
}

func main() {
	log := log.New(os.Stderr, "ws server 🚀 ", log.LstdFlags|log.Lshortfile|log.Lmsgprefix)

	var args args
	arg.MustParse(&args)

	nickMap, err := loadNickMap(args.NickMap)
	if err != nil {
		log.Fatal(err)
	}

	err = run("localhost:"+fmt.Sprint(args.Port), nickMap, args.Admin, int(args.HistLen), log)
	if err != nil {
		log.Fatal(err)
	}
}

func run(addr string, nickMap map[string]string, admin string, rhlen int, log *log.Logger) error {
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	log.Printf("listening on ws://%v", listener.Addr())

	server := &http.Server{
		Handler: server{
			admin: admin,
			logFn: log.Printf,
			conns: &conns{cm: make(map[*ws.Conn]user)},
			rooms: map[string]struct{}{"general": {}, "test1": {}, "test2": {}},
			rhist: make(map[string][]c.SMsg),
			rhlen: rhlen,
			nickm: nickMap,
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	errch := make(chan error, 1)
	go func() {
		errch <- server.Serve(listener)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	select {
	case err := <-errch:
		log.Printf("failed to serve: %v", err)
	case signal := <-signals:
		log.Printf("quitting: %v", signal)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoAtLeast(1, 1) && !hasUpgradeHeader(r.Header) {
		http.Redirect(w, r, "https://github.com/supleed2/go-chat", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	conn, err := ws.Accept(w, r, nil)
	if err != nil {
		s.logFn("%v", err)
		return
	}
	defer conn.CloseNow()

	if conn.Subprotocol() != "" {
		return
	}

	port := strings.Split(r.RemoteAddr, ":")[1]
	s.conns.sm.Lock()
	s.conns.cm[conn] = user{room: "general", nick: port}
	s.conns.sm.Unlock()
	defer func() {
		s.conns.sm.Lock()
		delete(s.conns.cm, conn)
		s.logFn("Remaining connections: %v", len(s.conns.cm))
		s.conns.sm.Unlock()
	}()

	s.logFn("connected: %v", r.RemoteAddr)
	for i := range s.rhist["general"] {
		wsjson.Write(ctx, conn, s.rhist["general"][i])
	}
	cmsg := c.CMsg{}
	smsg := c.SMsg{Id: port}
	for {
		err := func(ctx context.Context, conn *ws.Conn) error {
			err := wsjson.Read(ctx, conn, &cmsg)
			if err != nil {
				return err
			}

			switch cmsg.Typ {
			case c.Sudo:
				s.logFn("(%v) sudo: %v", smsg.Id, cmsg.Msg)
				if smsg.Id == s.admin {
					cmd := strings.Split(cmsg.Msg, " ")
					if len(cmd) == 2 {
						if cmd[0] == "mk" {
							if _, ok := s.rooms[cmd[1]]; ok {
								wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Room exists: %v", cmd)})
							} else {
								s.rooms[cmd[1]] = struct{}{}
								wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Created room: %v", cmd)})
							}
						} else if cmd[0] == "rm" {
							if _, ok := s.rooms[cmd[1]]; ok && cmd[1] != "general" {
								delete(s.rooms, cmd[1])
								s.rhist[cmd[1]] = []c.SMsg{}
								tim := time.Now()
								s.conns.sm.Lock()
								for cn, r := range s.conns.cm {
									if r.room == cmd[1] {
										r.room = "general"
										s.conns.cm[cn] = r
										wsjson.Write(ctx, cn, c.SMsg{Tim: tim, Id: "system", Msg: "room deleted, reconnected to general"})
									}
								}
								s.conns.sm.Unlock()
								wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Deleted room: %v", cmd)})
							} else {
								wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Room does not exist: %v", cmd)})
							}
						} else if cmd[0] == "yeet" {
							found := false
							s.conns.sm.Lock()
							for c, r := range s.conns.cm {
								if r.nick == cmd[1] {
									c.Close(ws.StatusNormalClosure, "Kicked")
									found = true
									break
								}
							}
							s.conns.sm.Unlock()
							if found {
								wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Yeet: %v", cmd[1])})
							} else {
								wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Not found: %v", cmd[1])})
							}
						} else {
							wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Invalid command: %v", cmd)})
						}
					} else if cmd[0] == "wc" {
						s.conns.sm.Lock()
						wc := len(s.conns.cm)
						s.conns.sm.Unlock()
						wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Online: %v", wc)})
					} else if cmd[0] == "man" {
						wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: "Available commands: man, mk, rm, wc, yeet"})
					} else {
						wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("Invalid command: %v", cmd)})
					}
				} else {
					wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: "Unrecognised command, use /man for more info"})
				}
			case c.Echo:
				s.logFn("(%v) echo: %v", smsg.Id, cmsg.Msg)
				s.conns.sm.Lock()
				room := s.conns.cm[conn].room
				s.conns.sm.Unlock()
				smsg.Tim = time.Now()
				smsg.Msg = cmsg.Msg
				if len(s.rhist[room]) < s.rhlen {
					s.rhist[room] = append(s.rhist[room], smsg)
				} else {
					s.rhist[room] = append(s.rhist[room][1:], smsg)
				}
				s.conns.sm.Lock()
				for c, r := range s.conns.cm {
					if r.room == room {
						wsjson.Write(ctx, c, &smsg)
					}
				}
				s.conns.sm.Unlock()
			case c.Mv:
				switch nick, valid := verifyNick(&s, cmsg.Msg); valid {
				case nickOk:
					s.logFn("(%v) mv: %v", smsg.Id, cmsg.Msg)
					smsg.Id = nick
					s.conns.sm.Lock()
					u := s.conns.cm[conn]
					u.nick = smsg.Id
					s.conns.cm[conn] = u
					s.conns.sm.Unlock()
					wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("nick set: %v", nick)})
				case nickUsed:
					s.logFn("(%v) mv used: %v", smsg.Id, cmsg.Msg)
					wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("nick in use: %v", cmsg.Msg)})
				case nickInvalid:
					s.logFn("(%v) mv invalid: %v", smsg.Id, cmsg.Msg)
					wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("invalid nick: %v", cmsg.Msg)})
				}
			case c.Ls:
				s.logFn("(%v) ls", smsg.Id)
				s.conns.sm.Lock()
				room := s.conns.cm[conn].room
				s.conns.sm.Unlock()
				avRooms := ""
				for r := range s.rooms {
					avRooms += r + ", "
				}
				wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("connected to: %v, available: %v", room, avRooms[:len(avRooms)-2])})
			case c.Cd:
				if _, ok := s.rooms[cmsg.Msg]; ok {
					s.logFn("(%v) cd: %v", smsg.Id, cmsg.Msg)
					s.conns.sm.Lock()
					u := s.conns.cm[conn]
					u.room = cmsg.Msg
					s.conns.cm[conn] = u
					s.conns.sm.Unlock()
					wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("connected to: %v", u.room)})
					rhistSize := max(len(s.rhist[u.room])-10, 0)
					recentHistory := s.rhist[u.room][rhistSize:]
					for i := range recentHistory {
						wsjson.Write(ctx, conn, recentHistory[i])
					}
				} else {
					s.logFn("(%v) cd invalid: %v", smsg.Id, cmsg.Msg)
					wsjson.Write(ctx, conn, c.SMsg{Tim: time.Now(), Id: "system", Msg: fmt.Sprintf("unchanged, invalid room: %v", cmsg.Msg)})
				}
			case c.Who:
				s.conns.sm.Lock()
				room := s.conns.cm[conn].room
				s.logFn("(%v) who: %v", smsg.Id, room)
				users := fmt.Sprintf("users in %v: ", room)
				for _, r := range s.conns.cm {
					if r.room == room {
						users += fmt.Sprintf("%v, ", r.nick)
					}
				}
				s.conns.sm.Unlock()
				wsjson.Write(ctx, conn, &c.SMsg{Tim: time.Now(), Id: "system", Msg: users[:len(users)-2]})
			}
			return nil
		}(ctx, conn)

		if ws.CloseStatus(err) == ws.StatusNormalClosure {
			s.logFn("disconnected: %v", r.RemoteAddr)
			return
		}
		if err != nil {
			s.logFn("failed, addr %v: %v", r.RemoteAddr, err)
			return
		}
	}
}

type nickErr int

const (
	nickOk nickErr = iota
	nickUsed
	nickInvalid
)

func verifyNick(s *server, n string) (string, nickErr) {
	nick, pass, _ := strings.Cut(n, ":")

	s.conns.sm.Lock()
	defer s.conns.sm.Unlock()
	for _, u := range s.conns.cm {
		if u.nick == nick {
			return "", nickUsed
		}
	}

	expPass, needAuth := s.nickm[nick]

	if (!needAuth || pass == expPass) && alphanumeric(nick) {
		return nick, nickOk
	} else {
		return "", nickInvalid
	}
}

func loadNickMap(m *string) (map[string]string, error) {
	nm := make(map[string]string)
	if m == nil {
		return nm, nil
	}

	path, err := filepath.Abs(*m)
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(file, &nm)
	if err != nil {
		return nil, err
	}

	return nm, nil
}

func alphanumeric(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func hasUpgradeHeader(h http.Header) bool {
	for _, v := range h["Connection"] {
		v = strings.TrimSpace(v)
		for _, t := range strings.Split(v, ",") {
			t = strings.TrimSpace(t)
			if strings.EqualFold(t, "Upgrade") {
				return true
			}
		}
	}
	return false
}
