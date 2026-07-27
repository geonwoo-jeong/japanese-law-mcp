package hanrei

type readResponse struct {
	sourceCategoryLabel string
	fields              []readField
	documents           []searchDocumentLink
}

type readField struct {
	label string
	value string
}
