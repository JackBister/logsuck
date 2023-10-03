package pipeline

type PipelinePipeType int

const (
	PipelinePipeTypeNone   PipelinePipeType = 0
	PipelinePipeTypeEvents PipelinePipeType = 1
	PipelinePipeTypeTable  PipelinePipeType = 2

	PipelinePipeTypePropagate PipelinePipeType = 999
)
