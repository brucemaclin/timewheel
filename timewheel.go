package timewheel

import (
	"container/list"
	"time"
)

//Index get nowtime index
func Index(n int, jiffies uint64) uint64 {
	return (jiffies >> (TVRBITS + uint(n)*TVNBITS)) & TVNMASK
}

func cascade(base *TimeWheel, tv *tvec, index uint64) uint64 {

	old := tv.vec[index]
	new := old
	old = list.New()
	node := new.Front()

	for node != nil {
		task := node.Value.(*Task)
		base.internalAddTimer(task)
		e := node
		node = node.Next()
		new.Remove(e)
	}
	return index
}
func (tw *TimeWheel) internalAddTimer(task *Task) {
	expires := task.Expires / tw.Interval
	idx := expires - tw.jiffies/tw.Interval
	var vec *list.List

	var tmp timerIndex
	if idx < TVRSIZE {
		index := expires & TVRMASK
		vec = tw.tv1.vec[index]
		tmp.listIndex = index
		tmp.tvIndex = 1
	} else if idx < 1<<(TVRBITS+TVNBITS) {
		i := (expires >> TVRBITS) & TVNMASK
		tmp.listIndex = i
		tmp.tvIndex = 2
		vec = tw.tv2.vec[i]
	} else if idx < 1<<(TVRBITS+2*TVNBITS) {
		i := (expires >> (TVNBITS + TVRBITS)) & TVNMASK
		tmp.listIndex = i
		tmp.tvIndex = 3
		vec = tw.tv3.vec[i]
	} else if idx < 1<<(TVRBITS+3*TVNBITS) {
		i := (expires >> (2*TVNBITS + TVRBITS)) & TVNMASK
		tmp.listIndex = i
		tmp.tvIndex = 4
		vec = tw.tv4.vec[i]
	} else if int64(idx) < 0 {
		vec = tw.tv1.vec[tw.jiffies/tw.Interval&TVRMASK]
		tmp.listIndex = tw.jiffies / tw.Interval & TVRMASK
		tmp.tvIndex = 1
	} else {
		if idx > MAXTVAL {
			idx = MAXTVAL
			expires = idx + tw.jiffies/tw.Interval
		}
		i := (expires >> (TVRBITS + 3*TVNBITS)) & TVNMASK
		tmp.listIndex = i
		tmp.tvIndex = 5
		vec = tw.tv5.vec[i]
	}
	vec.PushBack(task)
	//Debug(task.JobID, tmp.tvIndex, tmp.listIndex)
	tw.timerAddr[task.JobID] = tmp
}

//AddTimer add new task to timewheel
func (tw *TimeWheel) AddTimer(task *Task) {
	tw.lock.Lock()
	tw.internalAddTimer(task)
	tw.lock.Unlock()

}
func (tw *TimeWheel) runTimer() {
	jiffies := uint64(time.Now().UnixNano() / 1000 / 1000)
	day := weekDayMap[time.Now().Weekday().String()]
	for tw.jiffies <= jiffies {
		index := tw.jiffies / tw.Interval & TVRMASK

		if index == 0 &&
			cascade(tw, &tw.tv2, Index(0, tw.jiffies/tw.Interval)) == 0 &&
			cascade(tw, &tw.tv3, Index(1, tw.jiffies/tw.Interval)) == 0 &&
			cascade(tw, &tw.tv4, Index(2, tw.jiffies/tw.Interval)) == 0 {
			cascade(tw, &tw.tv5, Index(3, tw.jiffies/tw.Interval))
		}
		tw.jiffies += tw.Interval
		l := tw.tv1.vec[index]
		node := l.Front()
		for node != nil {
			task := node.Value.(*Task)

			task.exec()
			if task.NeedCycle {
				var addDay int
				for i := day + 1; i < 7; i++ {
					if task.RunWeekdays[i] {
						addDay = i - day
						break
					}
				}
				if addDay != 0 {
					Debug("addday:", addDay)
					task.Expires += dayMilliSeconds * uint64(addDay)
					tw.internalAddTimer(task)
				} else {
					for i := 0; i <= day; i++ {
						if task.RunWeekdays[i] {
							addDay = 7 - day + i
							break
						}
					}
					if addDay != 0 {
						task.Expires += dayMilliSeconds * uint64(addDay)
						tw.internalAddTimer(task)
					}
				}

			} else {
				delete(tw.timerAddr, task.JobID)
			}

			e := node
			node = node.Next()
			l.Remove(e)
		}
	}
}
func (tw *TimeWheel) internalDelete(index timerIndex, jobID string) {
	var l *list.List
	var listIndex = index.listIndex
	switch index.tvIndex {
	case 1:
		l = tw.tv1.vec[listIndex]
	case 2:
		l = tw.tv2.vec[listIndex]
	case 3:
		l = tw.tv3.vec[listIndex]
	case 4:
		l = tw.tv4.vec[listIndex]
	case 5:
		l = tw.tv4.vec[listIndex]

	}
	node := l.Front()
	for node != nil {
		task := node.Value.(*Task)
		if task.JobID == jobID {
			l.Remove(node)
			delete(tw.timerAddr, jobID)
			break
		}
		node = node.Next()
	}
}

//Delete delete the tw
func (tw *TimeWheel) Delete(jobID string) {
	tw.lock.Lock()
	index, ok := tw.timerAddr[jobID]
	if ok {
		tw.internalDelete(index, jobID)
	}
	tw.lock.Unlock()
}

//Modify modify the tw first delete and then add
func (tw *TimeWheel) Modify(task *Task) {
	tw.lock.Lock()
	index, ok := tw.timerAddr[task.JobID]
	if ok {
		tw.internalDelete(index, task.JobID)
	}
	tw.internalAddTimer(task)
	tw.lock.Unlock()
}

//Run start the tw
func (tw *TimeWheel) Run() {
	c := time.NewTicker(time.Millisecond * time.Duration(tw.Interval))
	for {
		select {
		case <-c.C:
			tw.lock.Lock()
			tw.runTimer()
			tw.lock.Unlock()
		case <-tw.stopChan: //TODO
		}
	}
}

//exec user's handle with user's data
func (task *Task) exec() {
	go task.Handle(task.Data)
}
func initTvecList(tv *tvec) {
	for i := 0; i < len(tv.vec); i++ {
		tv.vec[i] = list.New()
	}
}
func initTvecRootList(tv *tvecRoot) {
	for i := 0; i < len(tv.vec); i++ {
		tv.vec[i] = list.New()
	}
}

//InitTimeWheel init and set interval
func InitTimeWheel(interval uint64) *TimeWheel {
	if interval == 0 {
		interval = 1000
	}
	tw := new(TimeWheel)
	tw.Interval = interval
	tw.jiffies = uint64(time.Now().UnixNano() / 1000 / 1000)
	tw.timerAddr = make(map[string]timerIndex)
	initTvecRootList(&tw.tv1)
	initTvecList(&tw.tv2)
	initTvecList(&tw.tv3)
	initTvecList(&tw.tv4)
	initTvecList(&tw.tv5)

	return tw
}
