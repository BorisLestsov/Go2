package main

import (
    //"fmt"
    "math/rand"
    "time"
    "./graph"
    "os"
    "strconv"
)


func main() { 
    t := int64(time.Now().Unix())
    //t := int64(1510681144)
    //fmt.Println("Seed:", t)
    rand.Seed(t)

    NProc, _     := strconv.Atoi(os.Args[1])
    BasePort, _  := strconv.Atoi(os.Args[2])
    MinDegree, _ := strconv.Atoi(os.Args[3])
    MaxDegree, _ := strconv.Atoi(os.Args[4])
    BaseTtl, _   := strconv.Atoi(os.Args[5])

    quitCh       := make(chan struct{})
    killCh       := make(chan struct{}, NProc)

    gr := graph.Generate(NProc, MinDegree, MaxDegree, BasePort)
    //fmt.Println(gr)

    go proc(0, gr, quitCh, killCh, BaseTtl, true)
    for i := 1; i < NProc; i++ {
        go proc(i, gr, quitCh, killCh, BaseTtl, false)
    }

    <-quitCh
    for i := 0; i < NProc; i++ {
    	killCh <- struct{}{}
	}
    time.Sleep(time.Millisecond * 100)
}