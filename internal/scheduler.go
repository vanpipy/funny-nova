package internal

import "github.com/google/uuid"

type NodeStatus string

const (
	NodeValid NodeStatus = "valid"
	NodeInvalid NodeStatus = "invalid"
)

type Node struct {
	ID string
	Name string
	Status NodeStatus
}

type Scheduler struct {
	nodes []Node
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		nodes: []Node{},
	}
}


func (scheduler *Scheduler) RegisterNodes() {
	scheduler.nodes = append(scheduler.nodes, Node{
		ID: uuid.NewString(),
		Name: "node-1",
		Status: NodeValid,
	})
}

func (scheduler *Scheduler) SelectNode(proc *Proc) Node {
	for _, node := range scheduler.nodes {
		if node.Status == NodeValid {
			return node
		}
	}

	return Node{}
}
