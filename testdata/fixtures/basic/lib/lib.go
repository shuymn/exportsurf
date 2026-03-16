package lib

type Candidate struct{}

type UsedExternally struct{}

const ExportedConst = "value"

var ExportedVar = Candidate{}

func NewCandidate() Candidate {
	return Candidate{}
}

func internalUse() {
	_ = Candidate{}
	_ = ExportedConst
	_ = ExportedVar
	_ = NewCandidate()
}
