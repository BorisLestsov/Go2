package main

import (
    "fmt"
    "time"
    "net"
    "strconv"
    "math/rand"
    msg "./message"
    "./graph"
)


func ManageConn(Conn *net.UDPConn,
                dataCh chan msg.Message,
                timeoutCh chan time.Duration) {
    var buffer = make([]byte, 4096)
    var m msg.Message
    //var data string
    var timeout time.Duration
    timeout = time.Duration(0)

    for {
        select {
            case timeout = <-timeoutCh: {}
            default: {
                if timeout != 0 {
                    Conn.SetReadDeadline(time.Now().Add(timeout))
                }
                n,addr,err := Conn.ReadFromUDP(buffer)
                _ = addr
                if err != nil {
                    if e, ok := err.(net.Error); !ok || !e.Timeout() {
                        // not a timeout
                        panic(err)
                    } else {
                        m = msg.Message{ID_: 0, Type_: "timeout", Sender_: 0, Origin_: 0, Data_: ""}
                    }
                } else {
                    //data := string(buffer[0:n])
                    //fmt.Println(data)
                    m = msg.FromJsonMsg(buffer[0:n])
                }
                dataCh <- m   
            }
        }  
    }
}


type Task struct { 
    Msg msg.Message 
    Dst *net.UDPAddr
}

func ManageSendQueue(MyConn *net.UDPConn, taskCh chan Task, SendInterval time.Duration) {
    var t Task
    var buffer = make([]byte, 4096)
    
    for {
        select {
            case t = <- taskCh: {
                time.Sleep(SendInterval)
                buffer = t.Msg.ToJsonMsg()
                _, err := MyConn.WriteToUDP(buffer, t.Dst)
                msg.CheckError(err)
            }
        }
    }
}


func proc(MyID int, 
          gr graph.Graph,
          quitCh chan struct{}) {

    Neig, _ := gr.Neighbors(MyID)
    NeigAmt := len(Neig)
    MyNode, _ := gr.GetNode(MyID)
    MyPort := MyNode.Port()

    NeigPortArr  := make([]int, NeigAmt)
    NeigIdArr    := make([]int, NeigAmt)
    NeigAddrArr  := make([]*net.UDPAddr, NeigAmt)
    for i := 0; i < NeigAmt; i++ {
        NeigPortArr[i] = Neig[i].Port() 
        var err error
        NeigIdArr[i], err = strconv.Atoi(Neig[i].String())
        msg.CheckError(err)

        NeigAddrArr[i], err = net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(NeigPortArr[i]))
        msg.CheckError(err)
    }

    MyAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+strconv.Itoa(MyPort))
    msg.CheckError(err)

    MyConn, err := net.ListenUDP("udp", MyAddr)
    msg.CheckError(err)

    dataCh    := make(chan msg.Message)
    taskCh    := make(chan Task, 4096)
    timeoutCh := make(chan time.Duration, 1)

    timeoutCh <- time.Duration(time.Second * 2)

    if MyID == 0 {
        Dst := rand.Intn(NeigAmt)
        SendAddr := NeigAddrArr[Dst]
        m := Task{msg.Message{ID_: 0, Type_: "msg", Sender_: 0, Origin_: 0, Data_: "kek"}, SendAddr}
        taskCh <- m
    }

    go ManageConn(MyConn, dataCh, timeoutCh)
    go ManageSendQueue(MyConn, taskCh, time.Second * 1)

    var m msg.Message

    for {
        select {
            case m = <- dataCh: {
                if m.Type_ != "timeout" {
                    fmt.Println(MyID, "got msg:", m)
                
                    if !(NeigAmt <= 1) {
                        Dst := rand.Intn(NeigAmt)
                        for ; NeigIdArr[Dst] == m.Sender_ ; {
                            Dst = rand.Intn(NeigAmt)
                            //fmt.Println(Dst)
                        }
                        SendAddr := NeigAddrArr[Dst]
                        m.Sender_ = MyID

                        // send
                        taskCh <- Task{m, SendAddr}
                    }
                }
            }
        }
    }

    quitCh <- struct{}{}
}
