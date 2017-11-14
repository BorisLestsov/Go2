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
                timeoutCh chan time.Duration,
                quitCh chan struct{}) {
    var buffer = make([]byte, 4096)
    var m msg.Message
    //var data string
    var timeout time.Duration
    timeout = time.Duration(0)

    Loop:
    for {
        select {
            case timeout = <-timeoutCh: {}
            case _ = <-quitCh: {break Loop}
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

func ManageSendQueue(MyConn *net.UDPConn, 
                     taskCh chan Task, 
                     SendInterval time.Duration,
                     quitCh chan struct{}) {
    var t Task
    var buffer = make([]byte, 4096)
    
    Loop:
    for {
        select {
            case _ = <-quitCh: {break Loop}
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
          quitCh chan struct{},
          BaseTtl int) {

    DEBUG_OUTPUT := false
    
    rand.Seed(int64(time.Now().Unix()) + int64(MyID))

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

    if DEBUG_OUTPUT {
        fmt.Println(MyID, MyAddr, NeigAddrArr)
    }

    MyConn, err := net.ListenUDP("udp", MyAddr)
    msg.CheckError(err)

    dataCh    := make(chan msg.Message)
    taskCh    := make(chan Task, 4096)
    timeoutCh := make(chan time.Duration, 1)

    timeoutCh <- time.Duration(time.Millisecond * 50)

    AllGot := make([]bool, len(gr))
    TtlMap := make(map[int]int)
    BaseConfID := 5000

    qCh1 := make(chan struct{}, 1)
    qCh2 := make(chan struct{}, 1)

    go ManageConn(MyConn, dataCh, timeoutCh, qCh1)
    go ManageSendQueue(MyConn, taskCh, time.Microsecond * 10, qCh2)

    time.Sleep(time.Millisecond*10)

    // init gossip
    if MyID == 0 {
        Dst := rand.Intn(NeigAmt)
        SendAddr := NeigAddrArr[Dst]
        m := Task{msg.Message{ID_: 0, Type_: "msg", Sender_: 0, Origin_: 0, Data_: "kek"}, SendAddr}
        if DEBUG_OUTPUT {
            fmt.Println(MyID, "sending", m, "to", SendAddr)
        }
        taskCh <- m

        for index, _ := range AllGot {
            AllGot[index] = false
        }

    }

    var m msg.Message
    ticks := 0

    MainLoop:
    for {
        ticks += 1
        select {
            case m = <- dataCh: {
                if m.Type_ != "timeout" && DEBUG_OUTPUT {
                    //fmt.Println(MyID, "got msg:", m)
                }
                if m.Type_ == "msg" {
                    if !(NeigAmt <= 1) {
                        _, ok := TtlMap[m.ID_]
                        if !ok {
                            TtlMap[m.ID_] = BaseTtl
                        }

                        SenderTmp :=  m.Sender_
                        
                        if TtlMap[m.ID_] > 0 {
                            for ; TtlMap[m.ID_] > 0; {
                                Dst := rand.Intn(NeigAmt)
                                for ; NeigIdArr[Dst] == SenderTmp ; {
                                    Dst = rand.Intn(NeigAmt)
                                    //fmt.Println(Dst)
                                }
                                SendAddr := NeigAddrArr[Dst]
                                m.Sender_ = MyID

                                // pass further
                                if DEBUG_OUTPUT {
                                    fmt.Println(MyID, "sending", m, "to", SendAddr)
                                }
                                taskCh <- Task{m, SendAddr}
                                TtlMap[m.ID_] -= 1
                            }

                            // ID of message with conf
                            confID := m.ID_+BaseConfID+MyID
                            _, ok = TtlMap[confID]
                            if !ok {
                                TtlMap[confID] = BaseTtl
                            }

                            // conf msg
                            m := msg.Message{ID_: confID, Type_: "conf", Sender_: MyID, Origin_: m.Origin_, Data_: strconv.Itoa(MyID)}
                            for ; TtlMap[confID] > 0; {
                                Dst := rand.Intn(NeigAmt)
                                for ; NeigIdArr[Dst] == SenderTmp ; {
                                    Dst = rand.Intn(NeigAmt)
                                    //fmt.Println(Dst)
                                }
                                SendAddr := NeigAddrArr[Dst]
                                // send conf
                                if DEBUG_OUTPUT {
                                    fmt.Println(MyID, "sending", m, "to", SendAddr)
                                }
                                taskCh <- Task{m, SendAddr}
                                TtlMap[confID] -= 1
                            }
                        } else if DEBUG_OUTPUT {
                                fmt.Println(MyID, "discarded", m)
                        }
                    }
                } else if m.Type_ == "conf" {
                    if m.Origin_ == MyID {
                        // confurmed message
                        confID, err := strconv.Atoi(m.Data_)
                        msg.CheckError(err)
                        AllGot[confID] = true
                        if DEBUG_OUTPUT{
                            fmt.Println(AllGot)
                        }
                        all := true
                        LocalLoop:
                        for i := range AllGot {
                            if AllGot[i] == false {
                                all = false
                                break LocalLoop
                            }
                        }
                        if all {
                            fmt.Println("Finished in", ticks, "ticks")
                            qCh1 <- struct{}{}
                            qCh2 <- struct{}{}
                            break MainLoop
                        }
                    } else {
                        //pass further
                        if !(NeigAmt <= 1) {
                            _, ok := TtlMap[m.ID_]
                            if !ok {
                                TtlMap[m.ID_] = BaseTtl
                            }

                            SenderTmp :=  m.Sender_
                            
                            if TtlMap[m.ID_] > 0 {
                                for ; TtlMap[m.ID_] > 0; {
                                    Dst := rand.Intn(NeigAmt)
                                    for ; NeigIdArr[Dst] == SenderTmp ; {
                                        Dst = rand.Intn(NeigAmt)
                                        //fmt.Println(Dst)
                                    }
                                    SendAddr := NeigAddrArr[Dst]
                                    m.Sender_ = MyID

                                    // pass further
                                    if DEBUG_OUTPUT {
                                        fmt.Println(MyID, "sending", m, "to", SendAddr)
                                    }
                                    taskCh <- Task{m, SendAddr}
                                    TtlMap[m.ID_] -= 1
                                }
                            } else if DEBUG_OUTPUT {
                                fmt.Println(MyID, "discarded", m)
                            }
                        }
                    }
                } else if m.Type_ == "timeout" {
                    qCh1 <- struct{}{}
                    qCh2 <- struct{}{}
                    break MainLoop
                } else {
                    panic("Unknown message type"+m.Type_)
                }
            }
        }
    }

    //fmt.Println(MyID, "exited")
    quitCh <- struct{}{}
}
