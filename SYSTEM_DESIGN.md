# Funny Nova

> Nova.

```mermaid
graph TB
  proc[Proc]
  sto[Storage]
  dea[Deamon]
  sched[Node Scheduler]
  exec[Executor]
  cont[Container spawned by runc]
  queue[ProcQueue]

  proc -- create and save --> sto
  dea -- trigger --> sched
  sched -. sched result .-> dea
  exec -- proc --> cont

  dea -- read --> sto
  sto -. return proc .-> dea
  dea -- push set node --> queue
  dea -- push start node --> queue
  dea -- push retry node --> queue
  dea -- pop --> queue
  queue -. task .-> dea
  dea -- task --> exec

```

## Proc

```
type ProcPhase string

const (
  ProcPending ProcPhase = "Pending"
  ProcScheduled ProcPhase = "Scheduled"
  ProcRunning ProcPhase = "Running"
  ProcFailed ProcPhase = "Failed"
)

type Proc struct {
  UUID string
  Name string
  Command []string
  Env map[string]string
  CPU int64
  Memory int64

  Phase ProcPhase
  Image string
  Node string
  Message string
  RestartPolicy string
  RestartCount int

  CreatedAt time.Time
  UpdatedAt time.Time
}
```

## Storage

> Use Proc Data-Model

```
Insert(proc *Proc)

Delete(proc *Proc)

QueryByPhase(phase ProcPhase) []*Proc

QueryByNode(node string) []*Proc

Update(proc *Proc)
```

## Executor

```
Run(proc *Proc)

Kill(proc *Proc)
```

## Scheduler

```
SelectNode(proc *Proc) string
```

## ProcQueue

```
type TaskType string

const (
  TaskSetNode  TaskType = "SetNode"
  TaskStart    TaskType = "Start"
  TaskRetry    TaskType = "Retry"
  TaskRecycle  TaskType = "Recycle"
)

type Task struct {
  Type TaskType
  Proc *Proc
}

Push(task Task)

Pop() Task

Len() int
```
