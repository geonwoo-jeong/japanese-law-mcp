package legalquerycorpus

import "fmt"

// RequestKeyValues は、原 request の資源 key 値を保持する。
type RequestKeyValues struct {
	SourceID     string
	ResourceType string
	ResourceID   string
	VersionID    *string
}

// RequestKey は、境界違反を含む原 request key を型と存在情報を保って保持する。
type RequestKey struct {
	sourceID     string
	resourceType string
	resourceID   string
	versionID    *string
	initialized  bool
}

// NewRequestKey は、入力値を変更せず複製した corpus 専用 key を返す。
func NewRequestKey(values RequestKeyValues) (RequestKey, error) {
	key := RequestKey{
		sourceID:     values.SourceID,
		resourceType: values.ResourceType,
		resourceID:   values.ResourceID,
		versionID:    cloneOptionalString(values.VersionID),
		initialized:  true,
	}
	if err := key.Validate(); err != nil {
		return RequestKey{}, err
	}
	return key, nil
}

// SourceID は、原 sourceId を変更せず返す。
func (k RequestKey) SourceID() string {
	return k.sourceID
}

// ResourceType は、原 resourceType を変更せず返す。
func (k RequestKey) ResourceType() string {
	return k.resourceType
}

// ResourceID は、原 resourceId を変更せず返す。
func (k RequestKey) ResourceID() string {
	return k.resourceID
}

// VersionID は、原 versionId と項目の存在を返す。
func (k RequestKey) VersionID() (string, bool) {
	if k.versionID == nil {
		return "", false
	}
	return *k.versionID, true
}

// Validate は、constructor を介して作成された key であることを確認する。
func (k RequestKey) Validate() error {
	if !k.initialized {
		return fmt.Errorf("RequestKey は NewRequestKey で作成しなければなりません")
	}
	return nil
}

func (k RequestKey) clone() RequestKey {
	return RequestKey{
		sourceID:     k.sourceID,
		resourceType: k.resourceType,
		resourceID:   k.resourceID,
		versionID:    cloneOptionalString(k.versionID),
		initialized:  k.initialized,
	}
}

// RequestRefValues は、原 request の参照値を保持する。
type RequestRefValues struct {
	ProviderID string
	Key        RequestKey
}

// RequestRef は、境界違反を含む原参照を corpus 専用値として保持する。
type RequestRef struct {
	providerID  string
	key         RequestKey
	initialized bool
}

// NewRequestRef は、検証前の値を変更せず複製した corpus 専用参照を返す。
func NewRequestRef(values RequestRefValues) (RequestRef, error) {
	ref := RequestRef{
		providerID:  values.ProviderID,
		key:         values.Key.clone(),
		initialized: true,
	}
	if err := ref.Validate(); err != nil {
		return RequestRef{}, err
	}
	return ref, nil
}

// ProviderID は、原 providerId を変更せず返す。
func (r RequestRef) ProviderID() string {
	return r.providerID
}

// Key は、原資源 key の複製を返す。
func (r RequestRef) Key() RequestKey {
	return r.key.clone()
}

// Validate は、参照と必須 key が constructor を介して作成されたか確認する。
func (r RequestRef) Validate() error {
	if !r.initialized {
		return fmt.Errorf("RequestRef は NewRequestRef で作成しなければなりません")
	}
	if err := r.key.Validate(); err != nil {
		return fmt.Errorf("RequestRef の key が初期化されていません")
	}
	return nil
}

func (r RequestRef) clone() RequestRef {
	return RequestRef{
		providerID:  r.providerID,
		key:         r.key.clone(),
		initialized: r.initialized,
	}
}

// RequestValues は、semantic fixture の原 request 値を保持する。
type RequestValues struct {
	Query           string
	Ref             *RequestRef
	LimitPerAttempt *int
}

// Request は、製品 request へ矯正していない原 semantic request を保持する。
type Request struct {
	query           string
	ref             *RequestRef
	limitPerAttempt *int
	initialized     bool
}

// NewRequest は、境界違反と optional 項目の存在を保った不変 request を返す。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{
		query:           values.Query,
		limitPerAttempt: cloneOptionalInt(values.LimitPerAttempt),
		initialized:     true,
	}
	if values.Ref != nil {
		ref := values.Ref.clone()
		request.ref = &ref
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Query は、空文字、制御文字および上限超過を含む原 query を返す。
func (r Request) Query() string {
	return r.query
}

// Ref は、原 ref の複製と項目の存在を返す。
func (r Request) Ref() (RequestRef, bool) {
	if r.ref == nil {
		return RequestRef{}, false
	}
	return r.ref.clone(), true
}

// LimitPerAttempt は、原 limitPerAttempt と項目の存在を返す。
func (r Request) LimitPerAttempt() (int, bool) {
	if r.limitPerAttempt == nil {
		return 0, false
	}
	return *r.limitPerAttempt, true
}

// Validate は、request と存在する ref が constructor を介して作成されたか確認する。
func (r Request) Validate() error {
	if !r.initialized {
		return fmt.Errorf("Request は NewRequest で作成しなければなりません")
	}
	if r.ref != nil {
		if err := r.ref.Validate(); err != nil {
			return fmt.Errorf("Request の ref が初期化されていません")
		}
	}
	return nil
}

func (r Request) clone() Request {
	var ref *RequestRef
	if r.ref != nil {
		cloned := r.ref.clone()
		ref = &cloned
	}
	return Request{
		query:           r.query,
		ref:             ref,
		limitPerAttempt: cloneOptionalInt(r.limitPerAttempt),
		initialized:     r.initialized,
	}
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
