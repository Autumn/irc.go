// TO-DO

// write a core module, with necessary functionality like PING, CTCP, XDCC?
// fix parse()
// add buffering to the split/parse to handle the case where server sends incomplete messages
// finalise interface

package main

import (
    "fmt"
    "net"
    "bytes"
    "regexp"
    "time"
    "container/list"
)

func NewMessageChan() chan *Message {
    return make(chan *Message)
}

type Bot struct {
    nick string
    pass string
    user string
    host string
    serv string
    real string

    server string
    port string
    
    ssl bool

    plugins *list.List
    chans *list.List
    
    send chan *SendMessage
    recv chan *ServerResponse
    bot chan *Message
    con net.Conn
}

func New() *Bot {
    return &Bot{ nick:"oneechan",
                 pass:"",
                 user:"kurugaya",
                 host:"0",
                 serv:"*",
                 real:"umu umu",
                 server:"",
                 port:"6667",
                 ssl:false,
                 plugins:list.New(),
                 chans:list.New(),
                 send:make(chan *SendMessage),
                 recv:make(chan *ServerResponse),
                 bot:make(chan *Message),
                 con:nil           }
}

func (b *Bot) Nick(nick string) *Bot {
    b.nick = nick
    return b
}

func (b *Bot) Pass(pass string) *Bot {
    b.pass = pass
    return b
}

func (b *Bot) User(user string) *Bot {
    b.user = user
    return b
}

func (b *Bot) Host(host string) *Bot {
    b.host = host
    return b
}

func (b *Bot) Serv(serv string) *Bot {
    b.serv = serv
    return b
}

func (b *Bot) Real(real string) *Bot {
    b.real = real
    return b
}

func (b *Bot) Server(server string) *Bot {
    b.server = server
    return b
}

func (b *Bot) Port(port string) *Bot {
    b.port = port
    return b
}

func (b *Bot) Ssl(ssl bool) *Bot {
    b.ssl = ssl
    return b
}

func (b *Bot) userString() string {
    return b.user + " " + b.host + " " + b.serv + " " + b.real
}

func (b *Bot) nickString() string {
    return b.nick
}

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
    if len(b.pass) != 0 {
        b.Send("PASS " + b.pass)
    }
    b.Send("NICK " + b.nickString())
    b.Send("USER " + b.userString())
}

type ServerResponse struct {
    nBytes int
    err error
    msg []byte
}

func (b *Bot) Add(ch chan *Message, f func()) {
   b.chans.PushBack(ch)
   b.plugins.PushBack(f)
}

func (b *Bot) Send(s string) {
   b.con.Write(bytes.NewBufferString(s + "\r\n").Bytes())
}

func (b *Bot) Connect() {
    con, err := net.Dial("tcp",b.server + ":" + b.port)
    if err == nil {
       b.con = con

        // goroutine for sending messages to server

        go func() {
            for {
                msg := <- b.send
               b.Send(msg.command + " " + msg.target + " :" + msg.message)
            }
        }()
        
        // goroutine to read messages from server and send it to the received channel

        go func() {
            for {
                var resp [512]byte
                nbytes, ok := b.con.Read(resp[0:512])
               b.recv <- &ServerResponse{nBytes:nbytes, err: ok, msg: resp[0:512]}
            }
        }()

        // initiate all the module goroutines

        for e := b.plugins.Front(); e != nil; e = e.Next() {
            f := e.Value.(func())
            go f()
        }

        // goroutine to send messages to each module, when received from server

        go func() {
            for {
                msg := <- b.bot
                // send message to a list of channels, which implement features
                for e := b.chans.Front(); e != nil; e = e.Next() {
                    ch := e.Value.(chan *Message)
                    ch <- msg
                }
            }
        }()

       b.register()
 
        // - read server response from reader goroutine
        // - split the message into correctly delimited irc messages

        // TO DO: handle the case where the server sends an incomplete message
        //        required to buffer incomplete message and prepend it to the next message

        // - send each split string into the goroutine

        for {
            resp := <- b.recv 
            _, err, msg := resp.nBytes, resp.err, resp.msg
            if err == nil {
                msgs := bytes.Split(msg[0:512], bytes.NewBufferString("\r\n").Bytes())
                for _, val := range msgs {
                    // match and switch on the message
                    b.bot <- parse(val)
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
// better: rewrite emulating the irc EBNF

// need to handle case where server sends an incomplete message, 
// buffering the message to send to the next parse round

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

func main() {
    b := New().Server("irc.rizon.net")

    ch2 := NewMessageChan()

    perform := func() {
        for {
            msg := <- ch2
            if msg.Command == "001" {
//                b.Send("PRIVMSG NICKSERV :IDENTIFY *")
// send password here
                time.Sleep(time.Second)
                b.Send("JOIN #oneechan")
            }
        }
    }

    b.Add(ch2, perform)

    hc := NewMessageChan()
    ping := func() {
        for {
            msg := <- hc
            if msg.Command == "PING" {
                b.Send("PONG " + msg.Hostname)
            }
        }
    }
    b.Add(hc, ping)

    ch := NewMessageChan()
    printer := func() {
        for {
            msg := <- ch
            fmt.Println(msg.Servername + " " + msg.Username + " " + msg.Hostname + " " + msg.Command + " " + msg.Target + " " + msg.Message)
        }
    }

    b.Add(ch, printer)
    b.Connect()
}
