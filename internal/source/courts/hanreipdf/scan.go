package hanreipdf

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/judicialcasenumber"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var (
	strictCourtDateReferencePattern = regexp.MustCompile(
		`(?:最高裁判所|最高裁|知的財産高等裁判所|知財高裁|(?:札幌|仙台|東京|名古屋|大阪|広島|高松|福岡)高等裁判所|(?:札幌|仙台|東京|名古屋|大阪|広島|高松|福岡)高裁|[ぁ-んァ-ヶ一-龯々ー]{1,12}(?:地方裁判所|家庭裁判所|簡易裁判所|地裁|家裁|簡裁))[ 　]*(?:(?:大法廷|第?[一二三1-3]小法廷)[ 　]*)?(?:明治|大正|昭和|平成|令和)(?:元|[1-9][0-9]{0,2})年(?:[1-9]|1[0-2])月(?:[1-9]|[12][0-9]|3[01])日[ 　]*(?:(?:大法廷|第?[一二三1-3]小法廷)[ 　]*)?(?:判決|決定)`,
	)
	strictReporterReferencePattern = regexp.MustCompile(
		`(?:民集|刑集|最高裁判所(?:民事|刑事)判例集)(?:第)?[1-9][0-9]{0,3}巻(?:第)?[1-9][0-9]{0,3}号(?:[1-9][0-9]{0,6})頁`,
	)
	strictJapaneseDecisionDatePattern = regexp.MustCompile(
		`^(明治|大正|昭和|平成|令和)(元|[1-9][0-9]{0,2})年([1-9]|1[0-2])月([1-9]|[12][0-9]|3[01])日`,
	)
)

const (
	maximumExtractedTextBytes = 2 * 1024 * 1024
	maximumDecompressedBytes  = 24 * 1024 * 1024
	maximumPageCount          = 300
	maximumObjectCount        = 50_000
	maximumReferenceDepth     = 32
	maximumPDFResponseBytes   = 16 * 1024 * 1024
	maximumOccurrences        = 256
	maximumExcerptBytes       = 256
	maximumReferenceBytes     = 256
	maximumWorkerStdoutBytes  = 512 * 1024
)

type workerMention struct {
	Page             int                                    `json:"page"`
	ReferenceText    string                                 `json:"referenceText"`
	DecisionIdentity string                                 `json:"decisionIdentity,omitempty"`
	Excerpt          string                                 `json:"excerpt"`
	Reason           model.JudicialCitationUnresolvedReason `json:"reason,omitempty"`
}

func (m workerMention) confirmed() bool {
	return m.DecisionIdentity != ""
}

type workerOutput struct {
	PageCount         int             `json:"pageCount"`
	ObjectCount       int             `json:"objectCount"`
	DecompressedBytes int             `json:"decompressedBytes"`
	Occurrences       []workerMention `json:"occurrences"`
	TextUnavailable   bool            `json:"textUnavailable"`
	Truncated         bool            `json:"truncated"`
}

func buildExtractResult(
	request judicialcasecitationextract.Request,
	retrievedAt time.Time,
	pdfBytes []byte,
	output workerOutput,
) (judicialcasecitationextract.Result, error) {
	if err := validateWorkerOutput(output); err != nil {
		return judicialcasecitationextract.Result{}, err
	}
	if output.TextUnavailable {
		return judicialcasecitationextract.NewResult(judicialcasecitationextract.ResultValues{
			ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
			UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
			DocumentTextStatus:        judicialcasecitationextract.DocumentTextStatusDocumentTextUnavailable,
			ExaminedPageCount:         output.PageCount,
			OccurrenceCount:           0,
			Truncated:                 false,
		})
	}

	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(pdfBytes))
	rootIdentities := rootDecisionIdentities(request)
	confirmed := make([]model.JudicialCitationDecisionMention, 0, len(output.Occurrences))
	unresolved := make([]model.JudicialCitationUnresolvedMention, 0, len(output.Occurrences))
	for _, occurrence := range output.Occurrences {
		if occurrence.confirmed() {
			if isRootDecisionIdentity(request, rootIdentities, occurrence.DecisionIdentity) {
				continue
			}
		}
		provenance, err := newPDFProvenance(request, retrievedAt, digest, occurrence.Page)
		if err != nil {
			return judicialcasecitationextract.Result{}, newSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				operationParse,
				"",
			)
		}
		if occurrence.confirmed() {
			evidence, evidenceErr := model.NewJudicialCitationEvidence(
				model.JudicialCitationEvidenceValues{
					EvidenceLevel: model.JudicialCitationEvidenceLevelExactTextMatch,
					Provenance:    provenance,
					Excerpt:       &occurrence.Excerpt,
				},
			)
			if evidenceErr != nil {
				return judicialcasecitationextract.Result{}, newSourceError(
					model.SourceErrorCodeInvalidSourceResponse,
					operationParse,
					"",
				)
			}
			mention, mentionErr := model.NewJudicialCitationDecisionMention(
				model.JudicialCitationDecisionMentionValues{
					ReferenceText:        occurrence.ReferenceText,
					DecisionIdentityText: occurrence.DecisionIdentity,
					Evidence:             evidence,
				},
			)
			if mentionErr != nil {
				return judicialcasecitationextract.Result{}, newSourceError(
					model.SourceErrorCodeInvalidSourceResponse,
					operationParse,
					"",
				)
			}
			confirmed = append(confirmed, mention)
			continue
		}
		mention, mentionErr := model.NewJudicialCitationUnresolvedMention(
			model.JudicialCitationUnresolvedMentionValues{
				MentionType: model.JudicialCitationMentionTypeDecision,
				MentionText: occurrence.ReferenceText,
				Reason:      occurrence.Reason,
				Provenance:  provenance,
			},
		)
		if mentionErr != nil {
			return judicialcasecitationextract.Result{}, newSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				operationParse,
				"",
			)
		}
		unresolved = append(unresolved, mention)
	}
	if output.Truncated && len(confirmed)+len(unresolved) != maximumOccurrences {
		return judicialcasecitationextract.Result{}, newSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			operationParse,
			"",
		)
	}
	return judicialcasecitationextract.NewResult(judicialcasecitationextract.ResultValues{
		ConfirmedDecisionMentions: confirmed,
		UnresolvedMentions:        unresolved,
		DocumentTextStatus:        judicialcasecitationextract.DocumentTextStatusAvailable,
		ExaminedPageCount:         output.PageCount,
		OccurrenceCount:           len(confirmed) + len(unresolved),
		Truncated:                 output.Truncated,
	})
}

func validateWorkerOutput(output workerOutput) error {
	if output.PageCount < 1 || output.PageCount > maximumPageCount ||
		output.ObjectCount < 1 || output.ObjectCount > maximumObjectCount ||
		output.DecompressedBytes < 0 || output.DecompressedBytes > maximumDecompressedBytes ||
		len(output.Occurrences) > maximumOccurrences {
		return newSourceError(model.SourceErrorCodeSourceProcessingLimit, operationParse, "")
	}
	if output.Occurrences == nil {
		return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
	}
	if output.TextUnavailable && (len(output.Occurrences) != 0 || output.Truncated) {
		return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
	}
	if output.Truncated && len(output.Occurrences) != maximumOccurrences {
		return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
	}
	previousPage := 0
	for _, occurrence := range output.Occurrences {
		if occurrence.Page < 1 || occurrence.Page > output.PageCount ||
			occurrence.Page < previousPage ||
			!validBoundedText(occurrence.ReferenceText, maximumReferenceBytes) ||
			!validBoundedText(occurrence.Excerpt, maximumExcerptBytes) {
			return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
		}
		previousPage = occurrence.Page
		if occurrence.confirmed() {
			if occurrence.Reason != "" ||
				normalizedStrictDecisionIdentity(occurrence.DecisionIdentity) != occurrence.DecisionIdentity {
				return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
			}
			continue
		}
		if occurrence.Reason != model.JudicialCitationUnresolvedReasonInsufficientIdentity &&
			occurrence.Reason != model.JudicialCitationUnresolvedReasonUnsupportedReference {
			return newSourceError(model.SourceErrorCodeInvalidSourceResponse, operationParse, "")
		}
	}
	return nil
}

func normalizedCompleteCaseNumber(value string) string {
	trimmed := strings.TrimSpace(value)
	parsed, err := judicialcasenumber.ParsePrefix(trimmed)
	if err != nil || parsed.EndByte() != len(trimmed) {
		return ""
	}
	return parsed.SearchText()
}

func normalizedStrictDecisionIdentity(value string) string {
	if identity := normalizedCompleteCaseNumber(value); identity != "" {
		return identity
	}
	if identity := normalizedCourtDateIdentity(value); identity != "" {
		return identity
	}
	return normalizedReporterIdentity(value)
}

func normalizedCourtDateIdentity(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	compact := removeAllUnicodeSpace(trimmed)
	if matched := strictCourtDateReferencePattern.FindString(compact); matched == compact {
		if _, exists := parseStrictCourtDateIdentity(compact); !exists {
			return ""
		}
		return compact
	}
	return ""
}

func normalizedReporterIdentity(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	compact := removeAllUnicodeSpace(trimmed)
	if matched := strictReporterReferencePattern.FindString(compact); matched == compact {
		compact = strings.ReplaceAll(compact, "第", "")
		compact = strings.Replace(
			compact,
			"最高裁判所民事判例集",
			"民集",
			1,
		)
		return strings.Replace(
			compact,
			"最高裁判所刑事判例集",
			"刑集",
			1,
		)
	}
	return ""
}

func removeAllUnicodeSpace(value string) string {
	return strings.Map(func(current rune) rune {
		if unicode.IsSpace(current) {
			return -1
		}
		return current
	}, value)
}

func rootDecisionIdentities(request judicialcasecitationextract.Request) map[string]struct{} {
	identities := make(map[string]struct{}, 2)
	if identity := normalizedCompleteCaseNumber(request.Decision().Data().Summary().CaseNumber()); identity != "" {
		identities[identity] = struct{}{}
	}
	if reporterCitation, exists := request.Decision().Data().ReporterCitation(); exists {
		if identity := normalizedReporterIdentity(reporterCitation); identity != "" {
			identities[identity] = struct{}{}
		}
	}
	return identities
}

type strictCourtDateIdentity struct {
	courtName string
	date      string
}

func parseStrictCourtDateIdentity(value string) (strictCourtDateIdentity, bool) {
	compact := removeAllUnicodeSpace(strings.TrimSpace(value))
	eraOffset := -1
	for _, era := range []string{"明治", "大正", "昭和", "平成", "令和"} {
		if offset := strings.Index(compact, era); offset >= 0 &&
			(eraOffset < 0 || offset < eraOffset) {
			eraOffset = offset
		}
	}
	if eraOffset < 1 {
		return strictCourtDateIdentity{}, false
	}
	courtName := canonicalCourtName(trimAnySuffix(
		compact[:eraOffset],
		"大法廷",
		"第一小法廷",
		"第二小法廷",
		"第三小法廷",
		"第1小法廷",
		"第2小法廷",
		"第3小法廷",
		"一小法廷",
		"二小法廷",
		"三小法廷",
		"1小法廷",
		"2小法廷",
		"3小法廷",
	))
	date, exists := parseStrictJapaneseDecisionDate(compact[eraOffset:])
	if courtName == "" || !exists {
		return strictCourtDateIdentity{}, false
	}
	return strictCourtDateIdentity{courtName: courtName, date: date}, true
}

func trimAnySuffix(value string, suffixes ...string) string {
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			return strings.TrimSuffix(value, suffix)
		}
	}
	return value
}

func parseStrictJapaneseDecisionDate(value string) (string, bool) {
	match := strictJapaneseDecisionDatePattern.FindStringSubmatch(value)
	if len(match) != 5 {
		return "", false
	}
	year := 1
	var err error
	if match[2] != "元" {
		year, err = strconv.Atoi(match[2])
		if err != nil {
			return "", false
		}
	}
	month, monthErr := strconv.Atoi(match[3])
	day, dayErr := strconv.Atoi(match[4])
	if monthErr != nil || dayErr != nil {
		return "", false
	}
	type eraRange struct {
		baseYear int
		start    time.Time
		end      time.Time
	}
	ranges := map[string]eraRange{
		"明治": {1867, time.Date(1868, 1, 25, 0, 0, 0, 0, time.UTC), time.Date(1912, 7, 29, 0, 0, 0, 0, time.UTC)},
		"大正": {1911, time.Date(1912, 7, 30, 0, 0, 0, 0, time.UTC), time.Date(1926, 12, 24, 0, 0, 0, 0, time.UTC)},
		"昭和": {1925, time.Date(1926, 12, 25, 0, 0, 0, 0, time.UTC), time.Date(1989, 1, 7, 0, 0, 0, 0, time.UTC)},
		"平成": {1988, time.Date(1989, 1, 8, 0, 0, 0, 0, time.UTC), time.Date(2019, 4, 30, 0, 0, 0, 0, time.UTC)},
		"令和": {2018, time.Date(2019, 5, 1, 0, 0, 0, 0, time.UTC), time.Time{}},
	}
	currentRange, exists := ranges[match[1]]
	if !exists {
		return "", false
	}
	parsed := time.Date(currentRange.baseYear+year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if parsed.Month() != time.Month(month) || parsed.Day() != day ||
		parsed.Before(currentRange.start) ||
		(!currentRange.end.IsZero() && parsed.After(currentRange.end)) {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

func canonicalCourtName(value string) string {
	compact := removeAllUnicodeSpace(strings.TrimSpace(value))
	aliases := []struct {
		alias     string
		canonical string
	}{
		{"知的財産高等裁判所", "知的財産高等裁判所"},
		{"知財高裁", "知的財産高等裁判所"},
		{"最高裁判所", "最高裁判所"},
		{"最高裁", "最高裁判所"},
	}
	for _, city := range []string{"札幌", "仙台", "東京", "名古屋", "大阪", "広島", "高松", "福岡"} {
		aliases = append(aliases,
			struct {
				alias     string
				canonical string
			}{city + "高等裁判所", city + "高等裁判所"},
			struct {
				alias     string
				canonical string
			}{city + "高裁", city + "高等裁判所"},
		)
	}
	for _, alias := range aliases {
		if strings.HasPrefix(compact, alias.alias) {
			return alias.canonical
		}
	}
	for _, suffix := range []struct {
		short string
		full  string
	}{
		{"地方裁判所", "地方裁判所"},
		{"家庭裁判所", "家庭裁判所"},
		{"簡易裁判所", "簡易裁判所"},
		{"地裁", "地方裁判所"},
		{"家裁", "家庭裁判所"},
		{"簡裁", "簡易裁判所"},
	} {
		if index := strings.Index(compact, suffix.short); index > 0 {
			return compact[:index] + suffix.full
		}
	}
	return ""
}

func isRootDecisionIdentity(
	request judicialcasecitationextract.Request,
	rootIdentities map[string]struct{},
	identity string,
) bool {
	if _, exists := rootIdentities[identity]; exists {
		return true
	}
	courtDate, exists := parseStrictCourtDateIdentity(identity)
	if !exists {
		return false
	}
	summary := request.Decision().Data().Summary()
	rootCourtName := canonicalCourtName(summary.CourtName())
	if rootCourtName == "" && summary.PublicationCategory() == model.JudicialPublicationCategorySupremeCourt {
		rootCourtName = "最高裁判所"
	}
	return courtDate.courtName == rootCourtName && courtDate.date == summary.DecisionDate().String()
}

func newPDFProvenance(
	request judicialcasecitationextract.Request,
	retrievedAt time.Time,
	digest string,
	page int,
) (model.Provenance, error) {
	return model.NewProvenance(model.ProvenanceValues{
		Source:         informationSource(),
		ResourceKey:    request.Decision().Ref().Key(),
		URL:            request.Document().URL(),
		RetrievedAt:    retrievedAt,
		MediaType:      model.JudicialDocumentMediaTypePDF,
		Location:       fmt.Sprintf("page=%d", page),
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       "SOT-IF-071",
		ContentDigest:  digest,
	})
}

func scanExtractedText(text string, page int, remaining int) ([]workerMention, bool) {
	if remaining < 1 {
		mentions, truncated := scanExtractedText(text, page, 1)
		return []workerMention{}, len(mentions) > 0 || truncated
	}
	mentions := make([]workerMention, 0)
	for offset := 0; offset < len(text); {
		candidate, found := nextReferenceCandidate(text, offset)
		if !found {
			break
		}
		start := candidate.start
		if candidate.kind == referenceCandidateCourtDate || candidate.kind == referenceCandidateReporter {
			referenceText := text[start:candidate.end]
			identity := normalizedCourtDateIdentity(referenceText)
			if candidate.kind == referenceCandidateReporter {
				identity = normalizedReporterIdentity(referenceText)
			}
			if identity != "" {
				mentions = append(mentions, workerMention{
					Page:             page,
					ReferenceText:    referenceText,
					DecisionIdentity: identity,
					Excerpt:          excerptAround(text, start, candidate.end),
				})
				offset = candidate.end
				if len(mentions) > remaining {
					return mentions[:remaining], true
				}
				continue
			}
		}
		if candidate.kind != referenceCandidateCaseNumber {
			mentions = append(mentions, workerMention{
				Page:          page,
				ReferenceText: text[start:candidate.end],
				Excerpt:       excerptAround(text, start, candidate.end),
				Reason:        model.JudicialCitationUnresolvedReasonUnsupportedReference,
			})
			offset = candidate.end
			if len(mentions) > remaining {
				return mentions[:remaining], true
			}
			continue
		}
		substring := text[start:]
		parsed, err := judicialcasenumber.ParsePrefix(substring)
		if err == nil && validCaseNumberBoundary(substring, parsed.EndByte()) {
			end := start + parsed.EndByte()
			mentions = append(mentions, workerMention{
				Page:             page,
				ReferenceText:    text[start:end],
				DecisionIdentity: parsed.SearchText(),
				Excerpt:          excerptAround(text, start, end),
			})
			offset = end
		} else {
			token, end := decisionLikeToken(text, start)
			if token == "" {
				offset = start + utf8.RuneLen(firstRune(substring))
				continue
			}
			mentions = append(mentions, workerMention{
				Page:          page,
				ReferenceText: token,
				Excerpt:       excerptAround(text, start, end),
				Reason:        model.JudicialCitationUnresolvedReasonInsufficientIdentity,
			})
			offset = end
		}
		if len(mentions) > remaining {
			return mentions[:remaining], true
		}
	}
	return mentions, false
}

type referenceCandidateKind int

const (
	referenceCandidateCaseNumber referenceCandidateKind = iota
	referenceCandidateCourtDate
	referenceCandidateReporter
)

type referenceCandidate struct {
	kind  referenceCandidateKind
	start int
	end   int
}

func nextReferenceCandidate(value string, offset int) (referenceCandidate, bool) {
	if offset < 0 || offset >= len(value) {
		return referenceCandidate{}, false
	}
	candidates := make([]referenceCandidate, 0, 3)
	if start := nextEraOffset(value, offset); start >= 0 {
		candidates = append(candidates, referenceCandidate{
			kind: referenceCandidateCaseNumber, start: start, end: start,
		})
	}
	if indexes := strictCourtDateReferencePattern.FindStringIndex(value[offset:]); indexes != nil {
		candidates = append(candidates, referenceCandidate{
			kind:  referenceCandidateCourtDate,
			start: offset + indexes[0],
			end:   offset + indexes[1],
		})
	}
	if indexes := strictReporterReferencePattern.FindStringIndex(value[offset:]); indexes != nil {
		candidates = append(candidates, referenceCandidate{
			kind:  referenceCandidateReporter,
			start: offset + indexes[0],
			end:   offset + indexes[1],
		})
	}
	if len(candidates) == 0 {
		return referenceCandidate{}, false
	}
	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.start < selected.start {
			selected = candidate
		}
	}
	return selected, true
}

func nextEraOffset(value string, offset int) int {
	for index, current := range value[offset:] {
		if current != '令' && current != '平' && current != '昭' {
			continue
		}
		candidate := offset + index
		if strings.HasPrefix(value[candidate:], "令和") ||
			strings.HasPrefix(value[candidate:], "平成") ||
			strings.HasPrefix(value[candidate:], "昭和") {
			return candidate
		}
	}
	return -1
}

func validCaseNumberBoundary(value string, end int) bool {
	if end >= len(value) {
		return true
	}
	next, _ := utf8.DecodeRuneInString(value[end:])
	return !unicode.IsLetter(next) && !unicode.IsNumber(next) && next != '_' && next != '第' && next != '号'
}

func decisionLikeToken(value string, start int) (string, int) {
	end := start
	for end < len(value) && end-start < maximumReferenceBytes {
		current, size := utf8.DecodeRuneInString(value[end:])
		if end > start && isReferenceDelimiter(current) {
			break
		}
		end += size
	}
	token := strings.TrimSpace(value[start:end])
	if token == "" || (!strings.Contains(token, "(") && !strings.Contains(token, "（")) {
		return "", start
	}
	return token, end
}

func isReferenceDelimiter(value rune) bool {
	return unicode.IsSpace(value) || strings.ContainsRune("。、，,；;：:「」『』【】", value)
}

func excerptAround(value string, start int, end int) string {
	if end-start >= maximumExcerptBytes {
		return utf8Prefix(value[start:end], maximumExcerptBytes)
	}
	remaining := maximumExcerptBytes - (end - start)
	leftBudget := remaining / 2
	rightBudget := remaining - leftBudget
	left := utf8Suffix(value[:start], leftBudget)
	right := utf8Prefix(value[end:], rightBudget)
	return left + value[start:end] + right
}

func utf8Prefix(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	end := limit
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func utf8Suffix(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	start := len(value) - limit
	for start < len(value) && !utf8.RuneStart(value[start]) {
		start++
	}
	return value[start:]
}

func validBoundedText(value string, limit int) bool {
	return value != "" && utf8.ValidString(value) && len(value) <= limit
}

func firstRune(value string) rune {
	current, _ := utf8.DecodeRuneInString(value)
	return current
}
