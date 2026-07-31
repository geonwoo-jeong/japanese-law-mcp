package legalquerycandidateprepare

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

type prepareRepository struct {
	root *legalqueryartifact.Repository
}

func openPrepareRepository(path string) (*prepareRepository, error) {
	root, err := legalqueryartifact.OpenRepository(path)
	if err != nil {
		return nil, fmt.Errorf("候補準備 repository を開けません: %w", err)
	}
	return &prepareRepository{root: root}, nil
}

func (r *prepareRepository) Close() error {
	if r == nil || r.root == nil {
		return nil
	}
	err := r.root.Close()
	r.root = nil
	return err
}

func (r *prepareRepository) Read(path string, maximumBytes int64) ([]byte, error) {
	if r == nil || r.root == nil || !fs.ValidPath(path) || path == "." ||
		strings.Contains(path, "\\") {
		return nil, fmt.Errorf("候補準備 path が不正です")
	}
	segments := strings.Split(path, "/")
	current := r.root
	opened := make([]*legalqueryartifact.Repository, 0, len(segments)-1)
	defer func() {
		for index := len(opened) - 1; index >= 0; index-- {
			_ = opened[index].Close()
		}
	}()
	for _, segment := range segments[:len(segments)-1] {
		next, err := current.OpenChild(segment)
		if err != nil {
			return nil, fmt.Errorf("候補準備 path を開けません: %w", err)
		}
		opened = append(opened, next)
		current = next
	}
	raw, err := current.ReadRegular(segments[len(segments)-1], maximumBytes)
	if err != nil {
		return nil, fmt.Errorf("候補準備 file を読めません: %w", err)
	}
	return raw, nil
}

func checkPrepareContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("候補準備 context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("候補準備が中止されました: %w", err)
	}
	return nil
}
