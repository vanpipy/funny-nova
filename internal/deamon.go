package internal

import (
	"context"
	"fmt"
	"time"
)

type Deamon struct {
	storage *Storage
	scheduler *Scheduler
	queue *Queue
}

func NewDeamon(storage *Storage, scheduler *Scheduler, queue *Queue) *Deamon {
	return &Deamon{
		storage: storage,
		scheduler: scheduler,
		queue: queue,
	}
}

func (d *Deamon) Run(ctx context.Context)  {
	d.loadExistingProcs()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		  d.Tick()
		case <-ctx.Done():
		  return
		}
	}
}

func (d *Deamon) Tick()  {
	tasks := d.queue.PopAll()

	for _, task := range tasks {
		switch task.Type {
		case TaskSetNode:
			d.handleSetNode(task.Proc)
		case TaskStart:
			d.handleStart(task.Proc)
		case TaskRetry:
			d.handleRetry(task.Proc)
		case TaskRecycle:
			d.handleRecycle(task.Proc)
		}
	}

	d.checkHealth()
}

func (d *Deamon) Recycle(uuid string) error {
	proc := d.storage.QueryByUUID(uuid)

	if proc == nil {
		return fmt.Errorf("proc not found: %s", uuid)
	}

	d.queue.Push(Task{Type: TaskRecycle, Proc: proc})

	return nil
}

func (d *Deamon) checkHealth() {
	executor := NewExecutor()

	for _, proc := range d.storage.QueryByPhase(ProcRunning) {
		status, _ := executor.Inspect(proc.UUID)
		if status == "stopped" || status == "not_found" {
			proc.Phase = ProcFailed
			proc.Message = "container stopped unexpectedly"
			d.storage.Update(proc)
			d.queue.Push(Task{Type: TaskRetry, Proc: proc})
		}
	}
}

func (d *Deamon) handleSetNode(proc *Proc) {
	node := d.scheduler.SelectNode(proc)

	if node.Name == "" {
		proc.Message = "no available node"
		d.queue.Push(Task{Type: TaskSetNode, Proc: proc})
		return
	}

	proc.Node = node.Name
	proc.Phase = ProcScheduled

	if err := d.storage.Update(proc); err != nil {
		return
	}

	d.queue.Push(Task{Type: TaskStart, Proc: proc})
}

func (d *Deamon) handleStart(proc *Proc) {
	if proc.Node == "" {
		proc.Message = "no node assigned"
		proc.Phase = ProcFailed

		if err := d.storage.Update(proc); err != nil {
			return
		}

		d.queue.Push(Task{Type: TaskRetry, Proc: proc})

		return
	}

	executor := NewExecutor()

	if err := executor.Run(proc); err != nil {
		proc.Message = err.Error()
		proc.Phase = ProcFailed
		
		if err := d.storage.Update(proc); err != nil {
			return
		}

		d.queue.Push(Task{Type: TaskRetry, Proc: proc})

		return
	}

	proc.Phase = ProcRunning
	proc.Message = ""
	proc.RestartCount = 0
	d.storage.Update(proc)
}

func (d *Deamon) handleRetry(proc *Proc) {
	if proc.RestartPolicy == "Never" {
		return
	}

	if proc.RestartCount > 3 {
		proc.Message = "max retry exceeded"
		d.storage.Update(proc)
		return
	}

	if proc.Node == "" {
		proc.Phase = ProcPending
		proc.Message = "retry scheduling"
		d.storage.Update(proc)
		d.queue.Push(Task{Type: TaskSetNode, Proc: proc})
		return
	}

	executor := NewExecutor()

	if err := executor.Run(proc); err != nil {
		proc.RestartCount++
		proc.Message = err.Error()
		d.storage.Update(proc)
		d.queue.Push(Task{Type: TaskRetry, Proc: proc})
		return
	}

	proc.Phase = ProcRunning
	proc.Message = ""
	d.storage.Update(proc)
}

func (d *Deamon) handleRecycle(proc *Proc) {
	executor := NewExecutor()

	if err := executor.Kill(proc.UUID); err != nil {
		proc.Message = err.Error()
		d.storage.Update(proc)
		return
	}

	if err := d.storage.Delete(proc.UUID); err != nil {
		return
	}
}

func (d *Deamon) loadExistingProcs() {
	for _, proc := range d.storage.QueryByPhase(ProcPending) {
		d.queue.Push(Task{Type: TaskSetNode, Proc: proc})
	}

	for _, proc := range d.storage.QueryByPhase(ProcScheduled) {
		d.queue.Push(Task{Type: TaskStart, Proc: proc})
	}

	for _, proc := range d.storage.QueryByPhase(ProcFailed) {
		d.queue.Push(Task{Type: TaskRetry, Proc: proc})
	}
}
