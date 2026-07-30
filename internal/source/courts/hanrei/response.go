package hanrei

type searchResponse struct {
	totalCount int
	rows       []searchResultRow
}

type searchResultRow struct {
	detailLinks  []searchDetailLink
	caseNumber   string
	caseName     string
	decisionDate string
	courtName    string
	branchName   string
	divisionName string
	decisionType string
	outcome      string
	documents    []searchDocumentLink
	location     string
}

type searchDetailLink struct {
	href                string
	sourceCategoryLabel string
}

type searchDocumentLink struct {
	label string
	href  string
}
