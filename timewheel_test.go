package timewheel

import (
	"runtime"
	"strconv"
	"testing"
	"time"
)

func getTestTask(interval int64) *Task {
	task := &Task{
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
func TestRun(t *testing.T) {

	task0 := getTestTask(0)
	task1 := getTestTask(5000)
	task2 := getTestTask(6000)
	task3 := getTestTask(7000)
	task4 := getTestTask(124000)
	tw.AddTimer(task0)
	tw.AddTimer(task1)
	tw.AddTimer(task2)
	tw.AddTimer(task3)
	tw.AddTimer(task4)
	succ := <-task0.Data.(*result).succ
	if succ != "succ" {
		t.Error("fail to run timewheel")
	}
	Debug("succ:", succ)
	//time.Sleep(time.Minute * 10)
}

func test(v interface{}) {
	d := v.(*result)
	Debugf("my first timewheel %v", d.data)
	d.succ <- "succ"
}

type result struct {
	succ chan string
	data string
}

var tw *TimeWheel

func init() {
	tw = InitTimeWheel(1000)
	go tw.Run()
}
func BenchmarkAddTimer(t *testing.B) {
	for i := 0; i < t.N; i++ {
		task := getTestTask(int64((i + 1) * 1000))
		tw.AddTimer(task)
	}
}

func BenchmarkMultiAddTimer(t *testing.B) {
	t.StopTimer()
	tws := make([]*TimeWheel, runtime.NumCPU())
	for i := 0; i < len(tws); i++ {
		tws[i] = InitTimeWheel(1000)
		go tws[i].Run()
	}
	t.StartTimer()
	var index int
	for i := 0; i < t.N; i++ {
		if index == len(tws) {
			index = 0
		}
		task := getTestTask(int64((i + 1) * 1000))
		tws[index].AddTimer(task)
		index++
	}
}
