package sotvet

import (
	"go/ast"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
)

const suppressionPolicySOT = "SOT-ENG-019"

var (
	sotIDPattern       = regexp.MustCompile(`(?:^|[^A-Za-z0-9-])(SOT-[A-Z]+-[0-9]{3})(?:$|[^A-Za-z0-9-])`)
	staticcheckPattern = regexp.MustCompile(`^[A-Z]+[0-9]{4}$`)
	gosecPattern       = regexp.MustCompile(`^G[0-9]{3}$`)
)

// SuppressionPolicyAnalyzer は、SOT-ENG-019 に反する解析抑制を検査する。
var SuppressionPolicyAnalyzer = &analysis.Analyzer{
	Name: "sotsuppression",
	Doc:  "SOT-ENG-019 に反する静的解析の抑制記述を検出する",
	Run:  runSuppressionPolicy,
}

func runSuppressionPolicy(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				checkSuppressionComment(pass, file, comment)
			}
		}
	}
	return nil, nil
}

func checkSuppressionComment(
	pass *analysis.Pass,
	file *ast.File,
	comment *ast.Comment,
) {
	if _, ok := lineDirectiveRest(comment.Text, "//lint:file-ignore"); ok {
		pass.Reportf(
			comment.Pos(),
			"%s: //lint:file-ignore によるファイル全体の抑制は禁止されています",
			suppressionPolicySOT,
		)
		return
	}
	if rest, ok := lineDirectiveRest(comment.Text, "//lint:ignore"); ok {
		checkStaticcheckSuppression(pass, comment, rest)
		return
	}
	if reason, ok := nolintReason(comment.Text); ok {
		if comment.Pos() < file.Package {
			pass.Reportf(
				comment.Pos(),
				"%s: //nolint によるファイル全体の抑制は禁止されています",
				suppressionPolicySOT,
			)
			return
		}
		checkSuppressionReason(pass, comment, reason)
		return
	}
	if _, ok := lineDirectiveRest(comment.Text, "//gosec:disable"); ok {
		pass.Reportf(
			comment.Pos(),
			"%s: //gosec:disable による範囲抑制は禁止されています",
			suppressionPolicySOT,
		)
		return
	}
	if rest, ok := nosecRest(comment.Text); ok {
		checkGosecSuppression(pass, comment, rest)
	}
}

func checkStaticcheckSuppression(
	pass *analysis.Pass,
	comment *ast.Comment,
	rest string,
) {
	checks, reason := firstFieldAndRest(rest)
	if !validStaticcheckIDs(checks) {
		pass.Reportf(
			comment.Pos(),
			"%s: //lint:ignore は検査 ID を個別に指定してください",
			suppressionPolicySOT,
		)
		return
	}
	checkSuppressionReason(pass, comment, reason)
}

func checkGosecSuppression(
	pass *analysis.Pass,
	comment *ast.Comment,
	rest string,
) {
	rules, reason, hasReason := strings.Cut(rest, "--")
	if !validGosecIDs(rules) {
		pass.Reportf(
			comment.Pos(),
			"%s: gosec の抑制は G101 のように検査 ID を個別に指定してください",
			suppressionPolicySOT,
		)
		return
	}
	if !hasReason {
		reason = ""
	}
	checkSuppressionReason(pass, comment, reason)
}

func checkSuppressionReason(
	pass *analysis.Pass,
	comment *ast.Comment,
	reason string,
) {
	if hasSOTAndJapaneseReason(reason) {
		return
	}
	pass.Reportf(
		comment.Pos(),
		"%s: 抑制記述には SOT ID と日本語の理由を同じコメントに記載してください",
		suppressionPolicySOT,
	)
}

func nolintReason(text string) (string, bool) {
	if !strings.HasPrefix(text, "//") {
		return "", false
	}
	body := strings.TrimLeftFunc(text[2:], unicode.IsSpace)
	if !strings.HasPrefix(body, "nolint:") {
		return "", false
	}
	_, reason, hasReason := strings.Cut(strings.TrimPrefix(body, "nolint:"), "//")
	if !hasReason {
		return "", true
	}
	return reason, true
}

func lineDirectiveRest(text, directive string) (string, bool) {
	if !strings.HasPrefix(text, directive) {
		return "", false
	}
	return tokenRest(text[len(directive):], "")
}

func tokenRest(text, token string) (string, bool) {
	if token != "" {
		if !strings.HasPrefix(text, token) {
			return "", false
		}
		text = text[len(token):]
	}
	if text == "" {
		return "", true
	}
	first, _ := utf8.DecodeRuneInString(text)
	if !unicode.IsSpace(first) {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func firstFieldAndRest(text string) (string, string) {
	index := strings.IndexFunc(text, unicode.IsSpace)
	if index < 0 {
		return text, ""
	}
	return text[:index], strings.TrimSpace(text[index:])
}

func validStaticcheckIDs(checks string) bool {
	items := strings.Split(checks, ",")
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !staticcheckPattern.MatchString(item) {
			return false
		}
	}
	return true
}

func hasSOTAndJapaneseReason(reason string) bool {
	if !sotIDPattern.MatchString(reason) {
		return false
	}
	for _, character := range reason {
		if unicode.In(character, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}

func validGosecIDs(rules string) bool {
	items := strings.Fields(rules)
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !gosecPattern.MatchString(item) {
			return false
		}
	}
	return true
}

func normalizedCommentLines(text string) []string {
	switch {
	case strings.HasPrefix(text, "//"):
		return []string{strings.TrimSpace(text[2:])}
	case strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/"):
		body := strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
		lines := strings.Split(body, "\n")
		for index, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
			lines[index] = line
		}
		return lines
	default:
		return nil
	}
}

func nosecRest(text string) (string, bool) {
	lines := normalizedCommentLines(text)
	for index, line := range lines {
		rest, ok := tokenRest(line, "#nosec")
		if !ok {
			continue
		}
		parts := append([]string{rest}, lines[index+1:]...)
		return strings.TrimSpace(strings.Join(parts, "\n")), true
	}
	return "", false
}
