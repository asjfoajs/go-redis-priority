package main

import (
	"fmt"
	"time"
)

type Updater struct {
	regularTime   time.Duration //timer 的间隔时间
	everyTime     time.Duration //在定时里多长时间检测一次是否超过阈值
	timer         *time.Timer   //定时器
	triggerUpdate chan struct{} // 触发立即更新的通道
	stopChan      chan struct{} // 停止信号
}

func NewUpdater(regular, every int64, trigger func() bool, update func() error) *Updater {
	regularTime := time.Duration(regular)
	u := &Updater{
		regularTime:   regularTime,
		everyTime:     time.Duration(every),
		timer:         time.NewTimer(regularTime),
		triggerUpdate: make(chan struct{}, 1), // 缓冲大小为1，避免阻塞
		stopChan:      make(chan struct{}),
	}
	go u.run(update)           // 启动事件循环
	go u.CheckTrigger(trigger) //启动指定方法方法触发更新
	return u
}

// CheckTrigger 启动一个goroutine，检查触发更新条件，并立即触发更新
func (u *Updater) CheckTrigger(trigger func() bool) {
	for {
		if trigger() {
			// 非阻塞发送信号，避免重复触发
			select {
			case u.triggerUpdate <- struct{}{}:
			case <-u.stopChan: // 停止信号
				return
			default:
			}
		}
		time.Sleep(u.everyTime)
	}
}

// run 处理定时器和立即触发的事件
func (u *Updater) run(update func() error) {
	for {
		select {
		case <-u.timer.C: // 定时器触发
			u.update(update)
			u.resetTimer()
		case <-u.triggerUpdate: // 立即触发信号
			u.update(update)
			u.resetTimer()
		case <-u.stopChan: // 停止信号
			return
		}
	}
}

// resetTimer 安全重置定时器
func (u *Updater) resetTimer() {
	// 停止并排空定时器通道
	if !u.timer.Stop() {
		select {
		case <-u.timer.C:
		default:
		}
	}
	u.timer.Reset(u.regularTime)
}

// update 执行更新操作
func (u *Updater) update(update func() error) {
	if err := update(); err != nil {
		fmt.Printf("Update failed %e, current time:%s \n", err, time.Now().Format("15:04:05"))
	} else {
		fmt.Printf("Update successful, current time:%s \n", time.Now().Format("15:04:05"))
	}
}

// Stop 停止Updater
func (u *Updater) Stop() {
	close(u.stopChan)
}
