package continuation

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

const continuationKeyBytes = 32

// Manager は、process 固有鍵による継続トークンの発行と検証を行う。
type Manager struct {
	key [continuationKeyBytes]byte
	now func() time.Time
}

// NewManager は、CSPRNG から process 固有鍵を生成する。
func NewManager() (*Manager, error) {
	return newManager(rand.Reader, time.Now)
}

func newManager(random io.Reader, now func() time.Time) (*Manager, error) {
	if random == nil || now == nil {
		return nil, errInvalidManager
	}

	manager := &Manager{now: now}
	if _, err := io.ReadFull(random, manager.key[:]); err != nil {
		return nil, fmt.Errorf("継続トークン鍵を生成できません: %w", err)
	}
	return manager, nil
}

// String は、process 固有鍵を公開しない表現を返す。
func (*Manager) String() string {
	return "continuation.Manager"
}

// GoString は、process 固有鍵を公開しない Go 構文表現を返す。
func (*Manager) GoString() string {
	return "continuation.Manager"
}

func (m *Manager) valid() bool {
	return m != nil && m.now != nil
}
