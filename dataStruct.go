package timewheel

import (
	"container/list"
	"sync"
	"time"
)

const (
	TVNBITS uint = 6
	TVRBITS uint = 8
	TVNSIZE      = 1 << 6
	TVRSIZE      = 1 << 8
	TVNMASK      = TVNSIZE - 1
	TVRMASK      = TVRSIZE - 1
	MAXTVAL      = 1<<32 - 1
)

type tvec struct {
	vec          [TVNSIZE]*list.List
	currentIndex int
}

type tvecRoot struct {
	vec          [TVRSIZE]*list.List
	currentIndex int
}

var d = time.Second * 3

//TimeWheel make 5 floors to save
type TimeWheel struct {
	Interval  uint64
	stopChan  chan int
	timerAddr map[string]timerIndex
	tv1       tvecRoot
	tv2       tvec
	tv3       tvec
	tv4       tvec
	tv5       tvec
	jiffies   uint64

	lock sync.RWMutex
}
type timerIndex struct {
	tvIndex   uint64
	listIndex uint64
}

//Task save job to run user's Handle
type Task struct {
	JobID       string
	Expires     uint64 //time left to run
	Deleted     bool
	Handle      func(interface{}) //user's func to run
	Data        interface{}       //user's data for Handle to use
	NeedCycle   bool              //if needcycle if will check runweekdays
	RunWeekdays [7]bool           //which day need to run
	RunTime     string            //save the first time to  run
	AddDay      string
}

var weekDayMap = map[string]int{
	"Sunday":    0,
	"Monday":    1,
	"Tuesday":   2,
	"Wednesday": 3,
	"Thursday":  4,
	"Friday":    5,
	"Saturday":  6,
}

const dayMilliSeconds = 24 * 60 * 60 * 1000
