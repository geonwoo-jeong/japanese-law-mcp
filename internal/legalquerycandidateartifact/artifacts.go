// Package legalquerycandidateartifact は、評価候補だけが使う校正済み成果物を保持する。
package legalquerycandidateartifact

import _ "embed"

var (
	//go:embed data/core/profile.json
	coreProfileJSON []byte

	//go:embed data/core/cues.json
	coreCuesJSON []byte

	//go:embed data/judicial-cases/profile.json
	judicialCasesProfileJSON []byte

	//go:embed data/judicial-cases/cues.json
	judicialCasesCuesJSON []byte
)

// ProfileBytes は、一 profile の metadata と cue の原 byte 複製を保持する。
type ProfileBytes struct {
	metadata []byte
	cues     []byte
}

// Metadata は metadata 成果物の原 byte 複製を返す。
func (b ProfileBytes) Metadata() []byte { return append([]byte(nil), b.metadata...) }

// Cues は cue 成果物の原 byte 複製を返す。
func (b ProfileBytes) Cues() []byte { return append([]byte(nil), b.cues...) }

// Core は校正済み core 成果物を返す。
func Core() ProfileBytes {
	return ProfileBytes{metadata: coreProfileJSON, cues: coreCuesJSON}
}

// JudicialCases は校正済み judicial-cases 成果物を返す。
func JudicialCases() ProfileBytes {
	return ProfileBytes{
		metadata: judicialCasesProfileJSON,
		cues:     judicialCasesCuesJSON,
	}
}
