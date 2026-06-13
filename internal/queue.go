package internal

import (
	"sync"
)

type TaskType string

const (
	TaskSetNode TaskType = "TaskSetNode"
	TaskStart TaskType = "TaskStart"
	TaskRetry TaskType = "TaskRetry"
	TaskRecycle TaskType = "TaskRecycle"
)

type Task struct {
	Type TaskType
	Proc *Proc
}

type Queue struct {
	mu sync.Mutex
	items []Task
}

func NewQueue() *Queue {
	return &Queue{}
}

func (queue *Queue) Push(task Task) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.items = append(queue.items, task)
}

func (queue *Queue) PopAll() []Task {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	items := queue.items
	queue.items = nil
	return items
}
