package main

import (
    "fmt"
    "time"
    "./graph"
)


func main() { 
    quitCh := make(chan struct{})
    NProc := 5
    BasePort := 30000
    MinDegree := 2
    MaxDegree := 3

    gr := graph.Generate(NProc, MinDegree, MaxDegree, BasePort)
    fmt.Println(gr)

    for i := 0; i < NProc; i++ {
        go proc(i, gr, quitCh)
    }

    for i := 0; i < NProc; i++ {
	    <-quitCh
	}
    time.Sleep(time.Millisecond * 100)
}