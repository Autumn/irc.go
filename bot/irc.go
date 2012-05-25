package bot

import (
    "fmt"
    "net"
    "bytes"
    "regexp"
    "container/list"
)

type RegUser struct {
    pass string
    nick string
    user string
    host string
    server string
    real string
}

func NewRegUser(nick, user, host, server, real string) *RegUser {
    return &RegUser{pass:"", nick:nick, user:user, host:host, server:server, real:real}
}

func NewMessageChan() chan *Message {
    return make(chan *Message)
}

func (r *RegUser) userString() string {
    return r.user + " " + r.host+ " " + r.server+ " " + r.real
}

func (r *RegUser) nickString() string {
    return r.nick
}

// TO DO
// rename structs 

type Message struct {
    Servername string
    Nickname string
    Username string
    Hostname string
    Command string
    Target string
    Message string
}

type SendMessage struct {
    command string
    target string
    message string
}

func (b *Bot) register() {
    if len(b.user.pass) != 0 {
        b.Send("PASS " + b.user.pass)
    }
    b.Send("NICK " + b.user.nickString())
    b.Send("USER " + b.user.userString())
}

type ServerResponse struct {
    nBytes int
    err error
    msg []byte
}

type Bot struct {
    user *RegUser
    server string
    f *list.List
    g *list.List
    send chan *SendMessage
    recv chan *ServerResponse
    bot chan *Message
    con net.Conn
}

func NewBot(user *RegUser, server string) *Bot {
    send := make(chan *SendMessage);
    recv := make(chan *ServerResponse)
    bot := make(chan *Message)
    return &Bot{user, server, list.New(), list.New(), send, recv, bot, nil}
}

/* TO DO
   - connect active and connect passive
    - active will remain in the loop and print to stdout
    - passive will run in background, allow user to handle output by listening to channels
*/

func (bot *Bot) Add(ch chan *Message, f func()) {
    bot.f.PushBack(ch)
    bot.g.PushBack(f)
}

func (bot *Bot) Send(s string) {
    bot.con.Write(bytes.NewBufferString(s + "\r\n").Bytes())
}

func (bot *Bot) Connect() {
    con, err := net.Dial("tcp", bot.server)
    if err == nil {
        bot.con = con

        go func() {
            for {
                msg := <- bot.send
                bot.Send(msg.command + " " + msg.target + " :" + msg.message)
            }
        }()

        go func() {
            for {
                var b [512]byte
                nbytes, ok := bot.con.Read(b[0:512])
                bot.recv <- &ServerResponse{nBytes:nbytes, err: ok, msg: b[0:512]}
            }
        }()

        for e := bot.g.Front(); e != nil; e = e.Next() {
            f := e.Value.(func())
            go f()
        }

        go func() {
            for {
                msg := <- bot.bot
                // send message to a list of channels, which implement bot features
                for e := bot.f.Front(); e != nil; e = e.Next() {
                    ch := e.Value.(chan *Message)
                    ch <- msg
                }
            }
        }()

        bot.register()
 

        for {
            resp := <- bot.recv 
            _, err, msg := resp.nBytes, resp.err, resp.msg
            if err == nil {
                msgs := bytes.Split(msg[0:512], bytes.NewBufferString("\r\n").Bytes())
                for _, val := range msgs {
                    // match and switch on the message
                    bot.bot <- parse(val)
                }
            } else {
                fmt.Printf("error: connection terminated by server.\n")
                break
                // probably should terminate program or some sort of error
            }
        }
    } else {
        fmt.Println(err)
    } 
}


// TO DO
// rewrite using the new regexp library

func parse(b []byte) *Message {
    var servername, nick, user, host string
    var command, target, msg string
    words := bytes.Split(b, bytes.NewBufferString(" ").Bytes())

    if len(words) >= 4 {
        if match, _ := regexp.Match("^:", words[0]); match {
            if match, _ := regexp.Match("!|@", words[0]); match {
                i := 1
                for words[0][i] != '!' { i++ }
                    nick = bytes.NewBuffer(words[0][1:i]).String()
                    j := i+1
                    for words[0][j] != '@' { j++ }
                    var wordstart int = i + 1
                    if words[0][i+1] == '~' {
                        wordstart = i+2
                    }

                    user = bytes.NewBuffer(words[0][wordstart:j]).String()
                    k := j+1
                    host = bytes.NewBuffer(words[0][k:len(words[0])]).String()
            } else {
                servername = bytes.NewBuffer(words[0][1:len(words[0])]).String()
            }
        }
        command = bytes.NewBuffer(words[1]).String()
        target = bytes.NewBuffer(words[2]).String()
        str := bytes.Join(words[3:len(words)], bytes.NewBufferString(" ").Bytes())
        msg = bytes.NewBuffer(str[1:len(str)]).String()
    } else {
        if match, _ := regexp.Match("PING", words[0]); match {
            command = "PING"
            host= bytes.NewBuffer(words[1][1:len(words[1])]).String()
            fmt.Println(host)
        }
    }

    return &Message{
        Servername: servername,
        Nickname: nick,
        Username: user,
        Hostname: host,
        Command: command,
        Target: target,
        Message: msg,
    }
}
/*
func main() {
    reg := &RegUser{
        nick: "oneechan",
        user: "g",
        host: "0", 
        server: "*", 
        real:"~oneechan~",
    }

    bot := NewBot(reg, "irc.pantsuland.net:6667")
    ch := make(chan *Message)

    // join channels
    f :=  func() {
        for {
            msg := <- ch
            if msg.command == "376" {
                bot.Send("JOIN #oneechan")
            }
        }
    }

    bot.Add(ch, f)
    ch1 := make(chan *Message)
    bot.Add(ch1, func() { 
        for {
            msg := <- ch1
            ss := strings.SplitN(msg.message, " ", 2)
            if len(ss) == 2 {
                if ss[0] == ".mal" {
                    resp, err := http.Get("http://mal-api.com/anime/search?q=" + url.QueryEscape(ss[1]))
                    if err == nil {
                        body, _ := ioutil.ReadAll(resp.Body)
                        fmt.Println(bytes.NewBuffer(body).String())
                        var f interface{}
                        _ = json.Unmarshal(body, &f)
                        m := f.([]interface{})
                        for k, v := range m {
                             switch vv := v.(type) {
                                 case string:
                                     fmt.Println(k, "is string", vv)
                                 case int:
                                     fmt.Println(k, "is int", vv)
                                 case []interface{}:
                                     fmt.Println(k, "is an array:")
                                 for i, u := range vv {
                                     fmt.Println(i, u)
                                 }
                                 default:
                                     fmt.Println(k, "is of a type I don't know how to handle")
                             }
                            
                    } else {
                        fmt.Println("error occured")
                    }
                    resp.Body.Close()
                }
            }
        }
    })
    bot.Connect()
}
*/
