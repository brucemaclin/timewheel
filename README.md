# timewheel
provide a golang library of timewheel refer to the linux kernel timer.

[![Build Status](https://travis-ci.org/brucemaclin/timewheel.svg?branch=master)](https://travis-ci.org/brucemaclin/timewheel)

# Install
go get github.com/brucemaclin/timewheel

# Use
    package main
    import (
     tw "github.com/brucemaclin/timewheel"
     "time"
     "strconv"
     "fmt"
    )
      
    func getTestTask(interval int64) *tw.Task {
      task := &tw.Task{
        Expires: uint64(time.Now().UnixNano()/1000/1000 + interval),
      }
      task.Handle = test
      task.Data = &result{
        succ: make(chan string),
        data: time.Now().String(),
      }
      task.NeedCycle = true
      task.JobID = strconv.Itoa(int(interval))
      task.RunWeekdays[2] = true
      task.RunWeekdays[3] = true
      return task
    }

    func test(v interface{}) {
      d := v.(*result)
      fmt.Printf("my first timewheel %v\n", d.data)
      d.succ <- "succ"
    }

    type result struct {
     succ chan string
     data string
    }

    var mytw *tw.TimeWheel

    func init() {
      mytw = tw.InitTimeWheel(1000)
      go mytw.Run()
    }
    func main() {
      task0 := getTestTask(0)
      mytw.AddTimer(task0)
      succ := <-task0.Data.(*result).succ
      fmt.Println(succ)
     }

