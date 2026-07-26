package providerconformance

// NotApplicableCase は、適用しない標準 case とその理由を表す。
type NotApplicableCase struct {
	Case   string `json:"case"`
	Reason string `json:"reason"`
}

// Row は、一つの provider capability の適合性情報を表す。
type Row struct {
	ProviderID            string              `json:"providerId"`
	CapabilityID          string              `json:"capabilityId"`
	MajorVersion          int                 `json:"majorVersion"`
	Operation             string              `json:"operation"`
	InterfaceSOTIDs       []string            `json:"interfaceSotIds"`
	BudgetSOTID           string              `json:"budgetSotId"`
	BudgetKey             string              `json:"budgetKey"`
	ConcurrencyGroup      string              `json:"concurrencyGroup"`
	ArtifactType          string              `json:"artifactType"`
	FixtureSet            string              `json:"fixtureSet"`
	RequiredCases         []string            `json:"requiredCases"`
	NotApplicableCases    []NotApplicableCase `json:"notApplicableCases"`
	SupportsContinuation  bool                `json:"supportsContinuation"`
	SupportsAuth          bool                `json:"supportsAuth"`
	PublicErrorSet        []string            `json:"publicErrorSet"`
	ParserContractVersion string              `json:"parserContractVersion"`
	ImplementedBy         string              `json:"implementedBy"`
	ConformanceTarget     string              `json:"conformanceTarget"`
	Status                string              `json:"status"`
}

// ProviderMatrix は、一つの provider matrix を表す。
type ProviderMatrix struct {
	ProviderID    string
	SchemaVersion int
	rows          []Row
}

// Rows は、外部から変更しても ProviderMatrix に影響しない row の複製を返す。
func (m ProviderMatrix) Rows() []Row {
	return cloneRows(m.rows)
}

// Catalog は、repository 内の provider matrix を保持する。
type Catalog struct {
	providers []ProviderMatrix
}

// Providers は、外部から変更しても Catalog に影響しない provider matrix の複製を返す。
func (c Catalog) Providers() []ProviderMatrix {
	providers := make([]ProviderMatrix, len(c.providers))
	for i, provider := range c.providers {
		providers[i] = cloneProviderMatrix(provider)
	}
	return providers
}

// Rows は、provider 順と matrix 内の順序を保った全 row の複製を返す。
func (c Catalog) Rows() []Row {
	count := 0
	for _, provider := range c.providers {
		count += len(provider.rows)
	}
	rows := make([]Row, 0, count)
	for _, provider := range c.providers {
		rows = append(rows, cloneRows(provider.rows)...)
	}
	return rows
}

func cloneProviderMatrix(source ProviderMatrix) ProviderMatrix {
	return ProviderMatrix{
		ProviderID:    source.ProviderID,
		SchemaVersion: source.SchemaVersion,
		rows:          cloneRows(source.rows),
	}
}

func cloneRows(source []Row) []Row {
	rows := make([]Row, len(source))
	for i, row := range source {
		rows[i] = cloneRow(row)
	}
	return rows
}

func cloneRow(source Row) Row {
	row := source
	row.InterfaceSOTIDs = append([]string(nil), source.InterfaceSOTIDs...)
	row.RequiredCases = append([]string(nil), source.RequiredCases...)
	row.NotApplicableCases = append([]NotApplicableCase(nil), source.NotApplicableCases...)
	row.PublicErrorSet = append([]string(nil), source.PublicErrorSet...)
	return row
}
