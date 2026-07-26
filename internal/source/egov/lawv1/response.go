package lawv1

type updateListResponse struct {
	code    string
	message string
	date    string
	items   []updateListItem
}

type updateListItem struct {
	lawTypeName           string
	lawNumber             string
	lawName               string
	lawNameKana           string
	oldLawName            string
	promulgationDate      string
	amendmentName         string
	amendmentNumber       string
	amendmentPromulgation string
	enforcementDate       string
	enforcementComment    string
	lawID                 string
	lawURL                string
	enforcementFlag       string
	authorityReviewFlag   string
}
