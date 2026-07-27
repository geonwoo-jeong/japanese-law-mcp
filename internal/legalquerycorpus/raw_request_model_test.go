package legalquerycorpus

import (
	"strings"
	"sync"
	"testing"
)

func TestRawRequestV1DTOは不正境界値とoptionalの存在を保持する(t *testing.T) {
	t.Parallel()

	empty := ""
	query := strings.Repeat("a", 2049) + "\x00"
	limit := -1
	request, err := convertRawRequestV1(rawRequestV1DTO{
		Query: &query,
		Ref: &rawRequestRefV1DTO{
			ProviderID: &empty,
			Key: &rawRequestKeyV1DTO{
				SourceID:     &empty,
				ResourceType: &empty,
				ResourceID:   &empty,
				VersionID:    &empty,
			},
		},
		LimitPerAttempt: &limit,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: raw request の変換 error = %v", err)
	}
	if request.Query() != strings.Repeat("a", 2049)+"\x00" {
		t.Fatal("SOT-ENG-026: query の原値が失われた")
	}
	if got, exists := request.LimitPerAttempt(); !exists || got != -1 {
		t.Fatalf("SOT-ENG-026: limitPerAttempt = (%d, %t)", got, exists)
	}
	ref, exists := request.Ref()
	if !exists || ref.ProviderID() != "" {
		t.Fatalf("SOT-ENG-026: ref = (%#v, %t)", ref, exists)
	}
	key := ref.Key()
	if key.SourceID() != "" || key.ResourceType() != "" || key.ResourceID() != "" {
		t.Fatalf("SOT-ENG-026: raw key = %#v", key)
	}
	if got, exists := key.VersionID(); !exists || got != "" {
		t.Fatalf("SOT-ENG-026: versionId = (%q, %t)", got, exists)
	}
}

func TestRawRequestV1DTOは必須項目の欠落を拒否する(t *testing.T) {
	t.Parallel()

	tests := []rawRequestV1DTO{
		{},
		{
			Query: stringPointer("query"),
			Ref:   &rawRequestRefV1DTO{},
		},
		{
			Query: stringPointer("query"),
			Ref: &rawRequestRefV1DTO{
				ProviderID: stringPointer(""),
			},
		},
		{
			Query: stringPointer("query"),
			Ref: &rawRequestRefV1DTO{
				ProviderID: stringPointer(""),
				Key:        &rawRequestKeyV1DTO{},
			},
		},
	}
	for _, dto := range tests {
		if _, err := convertRawRequestV1(dto); err == nil {
			t.Fatal("SOT-ENG-026: raw request DTO の必須項目欠落を受理した")
		}
	}
}

func TestRequestはoptionalの欠落と空値を区別する(t *testing.T) {
	t.Parallel()

	withoutOptionals, err := NewRequest(RequestValues{Query: ""})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequest() error = %v", err)
	}
	if _, exists := withoutOptionals.Ref(); exists {
		t.Fatal("SOT-ENG-026: 欠落した ref が存在する")
	}
	if _, exists := withoutOptionals.LimitPerAttempt(); exists {
		t.Fatal("SOT-ENG-026: 欠落した limitPerAttempt が存在する")
	}

	empty := ""
	key, err := NewRequestKey(RequestKeyValues{
		SourceID:     "",
		ResourceType: "",
		ResourceID:   "",
		VersionID:    &empty,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequestKey() error = %v", err)
	}
	ref, err := NewRequestRef(RequestRefValues{ProviderID: "", Key: key})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequestRef() error = %v", err)
	}
	zero := 0
	withEmptyValues, err := NewRequest(RequestValues{
		Query:           "",
		Ref:             &ref,
		LimitPerAttempt: &zero,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequest() error = %v", err)
	}
	if _, exists := withEmptyValues.Ref(); !exists {
		t.Fatal("SOT-ENG-026: 空値を含む ref の存在が失われた")
	}
	if got, exists := withEmptyValues.LimitPerAttempt(); !exists || got != 0 {
		t.Fatalf("SOT-ENG-026: limitPerAttempt = (%d, %t)", got, exists)
	}
}

func TestRequestConstructorとgetterは深く複製する(t *testing.T) {
	t.Parallel()

	version := "v1"
	key, err := NewRequestKey(RequestKeyValues{
		SourceID:     "source",
		ResourceType: "law",
		ResourceID:   "resource",
		VersionID:    &version,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequestKey() error = %v", err)
	}
	ref, err := NewRequestRef(RequestRefValues{ProviderID: "provider", Key: key})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequestRef() error = %v", err)
	}
	limit := 20
	request, err := NewRequest(RequestValues{
		Query:           "query",
		Ref:             &ref,
		LimitPerAttempt: &limit,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequest() error = %v", err)
	}
	version = "changed"
	limit = -1
	gotRef, _ := request.Ref()
	if got, _ := gotRef.Key().VersionID(); got != "v1" {
		t.Fatal("SOT-ENG-026: constructor の入力 pointer が共有された")
	}
	if got, _ := request.LimitPerAttempt(); got != 20 {
		t.Fatal("SOT-ENG-026: limit pointer が共有された")
	}

	gotRef.key = RequestKey{}
	if err := gotRef.Validate(); err == nil {
		t.Fatal("SOT-ENG-026: getter が返した複製を変更できなかった")
	}
	again, _ := request.Ref()
	if again.ProviderID() != "provider" {
		t.Fatal("SOT-ENG-026: getter から request が変更された")
	}
}

func TestRequestはzero値と未初期化の構成要素を拒否する(t *testing.T) {
	t.Parallel()

	if err := (Request{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: Request の zero value を受理した")
	}
	if err := (RequestRef{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: RequestRef の zero value を受理した")
	}
	if err := (RequestKey{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: RequestKey の zero value を受理した")
	}
	if _, err := NewRequestRef(RequestRefValues{Key: RequestKey{}}); err == nil {
		t.Fatal("SOT-ENG-026: 未初期化 key を受理した")
	}
	ref := RequestRef{}
	if _, err := NewRequest(RequestValues{Query: "query", Ref: &ref}); err == nil {
		t.Fatal("SOT-ENG-026: 未初期化 ref を受理した")
	}
}

func TestRequestGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	version := "v1"
	key, _ := NewRequestKey(RequestKeyValues{
		SourceID: "source", ResourceType: "law", ResourceID: "resource",
		VersionID: &version,
	})
	ref, _ := NewRequestRef(RequestRefValues{ProviderID: "provider", Key: key})
	limit := 20
	request, err := NewRequest(RequestValues{
		Query: "query", Ref: &ref, LimitPerAttempt: &limit,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewRequest() error = %v", err)
	}

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				gotRef, _ := request.Ref()
				gotRef.key = RequestKey{}
				_ = gotRef.Validate()
				_, _ = request.LimitPerAttempt()
			}
		}()
	}
	wait.Wait()
	gotRef, _ := request.Ref()
	if gotRef.ProviderID() != "provider" {
		t.Fatal("SOT-ENG-026: 並行 getter が request を変更した")
	}
}

func stringPointer(value string) *string {
	return &value
}
