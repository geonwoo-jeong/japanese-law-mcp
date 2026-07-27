package judicialdecisionread

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// PortResolver は、裁判例参照の providerId に対応する詳細取得 port を返す。
type PortResolver interface {
	Resolve(Request) (Port, error)
}

// Service は、参照元 provider の選択と一リクエスト単位の期限を適用する。
type Service struct {
	resolver       PortResolver
	requestTimeout time.Duration
}

var _ Port = (*Service)(nil)

// NewService は、providerId resolver と request timeout を結び付ける。
func NewService(
	resolver PortResolver,
	requestTimeout time.Duration,
) (*Service, error) {
	if isNilResolver(resolver) {
		return nil, fmt.Errorf("judicial-decision.read resolver は必須です")
	}
	if requestTimeout <= 0 {
		return nil, fmt.Errorf("requestTimeout は 0 秒より長くなければなりません")
	}
	return &Service{
		resolver:       resolver,
		requestTimeout: requestTimeout,
	}, nil
}

// Read は、参照元 provider を選択し、期限付き context で詳細を取得する。
func (s *Service) Read(
	ctx context.Context,
	request Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	if ctx == nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	provider, err := s.resolver.Resolve(request)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	if isNilPort(provider) {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("judicial-decision.read resolver が有効な port を返しませんでした")
	}

	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()
	result, err := provider.Read(requestContext, request)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	if err := result.Validate(); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("judicial-decision.read の結果が有効ではありません: %w", err)
	}
	if !sameResourceRef(result.Ref(), request.Ref()) {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("judicial-decision.read の出力 ref が入力 ref と一致しません")
	}
	if result.Data().Summary().Source().ID() != request.Ref().Key().SourceID() {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf(
				"judicial-decision.read の data.summary.source.id が入力 ref と一致しません",
			)
	}
	return result, nil
}

func sameResourceRef(
	left model.SourceResourceRef,
	right model.SourceResourceRef,
) bool {
	if left.ProviderID() != right.ProviderID() {
		return false
	}
	leftKey := left.Key()
	rightKey := right.Key()
	leftVersion, leftHasVersion := leftKey.VersionID()
	rightVersion, rightHasVersion := rightKey.VersionID()
	return leftKey.SourceID() == rightKey.SourceID() &&
		leftKey.ResourceType() == rightKey.ResourceType() &&
		leftKey.ResourceID() == rightKey.ResourceID() &&
		leftHasVersion == rightHasVersion &&
		leftVersion == rightVersion
}

func isNilResolver(resolver PortResolver) bool {
	if resolver == nil {
		return true
	}
	value := reflect.ValueOf(resolver)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
