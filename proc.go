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

var DEBUG_OUTPUT = false


func ManageConn(Conn *net.UDPConn,
                dataCh chan msg.Message,
                timeoutCh chan time.Duration,
                quitCh chan struct{}) {
    var buffer = make([]byte, 4096)
    var m msg.Message
    //var data string
    var timeout time.Duration
    timeout = time.Duration(0)

    for {
        select {
            case timeout = <-timeoutCh: {}
            case _ = <-quitCh: {return}
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
    
    
    for {
        select {
            case _ = <-quitCh: return
            case t = <- taskCh: {
                time.Sleep(SendInterval)
                buffer = t.Msg.ToJsonMsg()
                _, err := MyConn.WriteToUDP(buffer, t.Dst)
                msg.CheckError(err)
            }
        }
    }
}

func InitGossip(MyID int, NeigAmt int, NeigAddrArr []*net.UDPAddr, taskCh chan Task, ttl int){
    for i := 0; i < ttl; i++ {
        Dst := rand.Intn(NeigAmt)
        SendAddr := NeigAddrArr[Dst]
        m := Task{msg.Message{ID_: MyID, Type_: "msg", Sender_: 0, Origin_: 0, Data_: "kek"}, SendAddr}
        if DEBUG_OUTPUT {
            fmt.Println(MyID, "sending", m, "to", SendAddr)
        }
        taskCh <- m
    }
}


func UpdateTtl(TtlMap map[int]int, BaseTtl int, TtlVal int){
    _, ok := TtlMap[TtlVal]
    if !ok {
        TtlMap[TtlVal] = BaseTtl
    }
}


func ProcessMsg(MyID int,
                m msg.Message, 
                NeigAddrArr []*net.UDPAddr, 
                NeigAmt int, 
                BaseTtl int, 
                TtlMap map[int]int, 
                BaseConfID int,
                taskCh chan Task){

    UpdateTtl(TtlMap, BaseTtl, m.ID_)
    
    if TtlMap[m.ID_] > 0 {
        for ; TtlMap[m.ID_] > 0; {
            Dst := rand.Intn(NeigAmt)
            //Dst := TtlMap[m.ID_] % NeigAmt
                //fmt.Println(Dst)
            SendAddr := NeigAddrArr[Dst]
            m.Sender_ = MyID

            // pass further
            if DEBUG_OUTPUT {
                fmt.Println(MyID, "sending", m, "to", SendAddr)
            }
            taskCh <- Task{m, SendAddr}
            TtlMap[m.ID_] -= 1
        }

        // ID of confirmation message
        confID := m.ID_+BaseConfID+MyID
        UpdateTtl(TtlMap, BaseTtl, confID)

        // confirmation msg
        m := msg.Message{ID_: confID, Type_: "conf", Sender_: MyID, Origin_: m.Origin_, Data_: strconv.Itoa(MyID)}
        for ; TtlMap[confID] > 0; {
            Dst := rand.Intn(NeigAmt)
            //Dst := TtlMap[confID] % NeigAmt
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


func ProcessConfirmationMsg(m msg.Message,
                            MyID int,
                            NeigAddrArr []*net.UDPAddr, 
                            NeigAmt int, 
                            AllGot []bool, 
                            TtlMap map[int]int,
                            BaseTtl int,
                            taskCh chan Task) bool{

    if m.Origin_ == MyID {
        // confurmed message
        confID, err := strconv.Atoi(m.Data_)
        msg.CheckError(err)
        AllGot[confID] = true
        if DEBUG_OUTPUT{
            fmt.Println(AllGot)
        }
        all := true
        for i := range AllGot {
            if AllGot[i] == false {
                all = false
                break
            }
        }
        if all {
            return true
        }
    } else {
        //pass further
        UpdateTtl(TtlMap, BaseTtl, m.ID_)

        if TtlMap[m.ID_] > 0 {
            for ; TtlMap[m.ID_] > 0; {
                Dst := rand.Intn(NeigAmt)
                //Dst := TtlMap[m.ID_] % NeigAmt
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
    return false
}


func proc(MyID int, 
          gr graph.Graph,
          quitCh chan struct{},
          killCh chan struct{},
          BaseTtl int,
          needInit bool) {
    
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

    timeoutCh <- time.Duration(0) //time.Millisecond * 50)

    AllGot := make([]bool, len(gr))
    TtlMap := make(map[int]int)
    BaseConfID := 5000

    qCh1 := make(chan struct{}, 1)
    qCh2 := make(chan struct{}, 1)

    go ManageConn(MyConn, dataCh, timeoutCh, qCh1)
    go ManageSendQueue(MyConn, taskCh, time.Microsecond * 10, qCh2)

    time.Sleep(time.Millisecond*10)

    // init gossip
    if needInit {
        InitGossip(MyID, NeigAmt, NeigAddrArr, taskCh, BaseTtl)
        for index, _ := range AllGot {
            AllGot[index] = false
        }
        AllGot[MyID] = true
    }

    var m msg.Message
    ticks := 0

    MainLoop:
    for ;; ticks++ {
        select {
            case m = <- dataCh: {
                select {
                    case <-killCh:{}
                    default: {
                        if m.Type_ != "timeout" && DEBUG_OUTPUT {
                            fmt.Println(MyID, "got msg:", m)
                        }
                        if m.Type_ == "msg" {
                            ProcessMsg(MyID, m, NeigAddrArr, NeigAmt, BaseTtl, TtlMap, BaseConfID, taskCh)
                        } else if m.Type_ == "conf" {
                            isFinished := ProcessConfirmationMsg(m, MyID, NeigAddrArr, NeigAmt, AllGot, TtlMap, BaseTtl, taskCh)

                            if (isFinished) {
                                fmt.Println("Finished in", ticks, "ticks")
                                break MainLoop
                            }
                        } else if m.Type_ == "timeout" {
                            //break MainLoop
                        } else {
                            panic("Unknown message type"+m.Type_)
                        }
                    }
                }
            }
        }
    }

    qCh1 <- struct{}{}
    qCh2 <- struct{}{}

    //fmt.Println(MyID, "exited")

    quitCh <- struct{}{}
}
