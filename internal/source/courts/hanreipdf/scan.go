package hanreipdf

import (
	"crypto/sha256"
	"fmt"
	"regexp"
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
	rootIdentity := normalizedCompleteCaseNumber(
		request.Decision().Data().Summary().CaseNumber(),
	)
	confirmed := make([]model.JudicialCitationDecisionMention, 0, len(output.Occurrences))
	unresolved := make([]model.JudicialCitationUnresolvedMention, 0, len(output.Occurrences))
	for _, occurrence := range output.Occurrences {
		if occurrence.confirmed() && occurrence.DecisionIdentity == rootIdentity {
			continue
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
				normalizedCompleteCaseNumber(occurrence.DecisionIdentity) != occurrence.DecisionIdentity {
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
