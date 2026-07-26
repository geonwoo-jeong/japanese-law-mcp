package lawarticleread

import "errors"

var (
	// ErrNotFound は、指定した法令、リビジョンまたは条文位置が存在しないことを表す。
	ErrNotFound = errors.New("指定した条件に該当する条文が見つかりません")
	// ErrAmbiguousLocation は、指定した条文位置を一意に決定できないことを表す。
	ErrAmbiguousLocation = errors.New("指定した条文位置を一意に決定できません")
)
