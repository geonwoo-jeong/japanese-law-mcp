package hanrei

type searchResponse struct {
	totalCount int
	rows       []searchResultRow
}

type searchResultRow struct {
	detailHref          string
	sourceCategoryLabel string
	caseNumber          string
	caseName            string
	decisionDate        string
	courtName           string
	branchName          string
	divisionName        string
	decisionType        string
	outcome             string
	documents           []searchDocumentLink
	location            string
}

type searchDocumentLink struct {
	label string
	href  string
}
