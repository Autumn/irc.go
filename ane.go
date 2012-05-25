package main

import (
    "./bot"
    "net/http"
    "net/url"
    "encoding/json"
    "io/ioutil"
    "strings"
    "fmt"
    "bytes"
    "time"
    
)

func main() {
    reg := bot.NewRegUser("oneesan", "kurugaya", "0", "*", "oneesan")
    b := bot.NewBot(reg, "irc.pantsuland.net:6667")


    ch2 := bot.NewMessageChan()

    perform := func() {
        for {
            msg := <- ch2
            if msg.Command == "001" {
                b.Send("PRIVMSG NICKSERV :IDENTIFY *")
// send password here
                time.Sleep(time.Second)
                b.Send("JOIN #oneechan")
            }
        }
    }

    b.Add(ch2, perform)

    hc := bot.NewMessageChan()
    ping := func() {
        for {
            msg := <- hc
            if msg.Command == "PING" {
                b.Send("PONG " + msg.Hostname)
            }
        }
    }
    b.Add(hc, ping)

    ch := bot.NewMessageChan()
    printer := func() {
        for {
            msg := <- ch
            fmt.Println(msg.Servername + " " + msg.Username + " " + msg.Hostname + " " + msg.Command + " " + msg.Target + " " + msg.Message)
        }
    }

    b.Add(ch, printer)

    ch1 := bot.NewMessageChan()
    b.Add(ch1, func() { 
        for {
            msg := <- ch1
            ss := strings.SplitN(msg.Message, " ", 2)
            if len(ss) == 2 {
                if ss[0] == ".mal" {
                    resp, err := http.Get("http://mal-api.com/anime/search?q=" + url.QueryEscape(ss[1]))
                    if err == nil {
                        body, _ := ioutil.ReadAll(resp.Body)
                        fmt.Println(bytes.NewBuffer(body).String())
                        var f interface{}
                        _ = json.Unmarshal(body, &f)
                        /*m := f.([]interface{})
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
                        }*/
                    } else {
                        fmt.Println("error occured")
                    }
                    resp.Body.Close()
                }
            }
        }
    })
    b.Connect()
}
