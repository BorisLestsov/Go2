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

    quitCh       := make(chan struct{})
    NProc, _     := strconv.Atoi(os.Args[1])
    BasePort, _  := strconv.Atoi(os.Args[2])
    MinDegree, _ := strconv.Atoi(os.Args[3])
    MaxDegree, _ := strconv.Atoi(os.Args[4])
    BaseTtl, _   := strconv.Atoi(os.Args[5])

    gr := graph.Generate(NProc, MinDegree, MaxDegree, BasePort)
    //fmt.Println(gr)

    for i := 0; i < NProc; i++ {
        go proc(i, gr, quitCh, BaseTtl)
    }

    for i := 0; i < NProc; i++ {
	    <-quitCh
	}
    time.Sleep(time.Millisecond * 100)
}