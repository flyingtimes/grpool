package main

import (
	"fmt"
	"github.com/flyingtimes/grpool"
	"runtime"
	"testing"
	"time"
)

func printit(ino []interface{}) interface{} {
	i := ino[0].(int)
	var j = (i*i + 2*i + i) * i
	return fmt.Sprintf("from callback %d", j)
}
func TestNewPool(t *testing.T) {
	start := time.Now()
	runtime.GOMAXPROCS(4)
	pool := grpool.NewPool(10, 30)
	defer pool.Release()
	for i := 0; i < 10000; i++ {
		var jb grpool.Job
		jb.SetCallback(printit, i)
		pool.JobQueue <- jb
	}
	elapsed := time.Since(start)
	fmt.Println(elapsed)
}
