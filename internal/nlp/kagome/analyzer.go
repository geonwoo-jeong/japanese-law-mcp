// Package kagome は、共通検索語前処理用の Kagome adapter を提供する。
package kagome

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

const (
	maxDictionaryTermCount = 50000
	maxDictionaryTermBytes = 2048
	maxAnalyzerInputBytes  = 4096
)

// Analyzer は、Kagome user dictionary の登録語だけを抽出する。
type Analyzer struct {
	tokenizer *tokenizer.Tokenizer
	gate      chan struct{}
}

// TokenOccurrence は、Kagome token と原文上の UTF-8 byte span を保持する。
type TokenOccurrence struct {
	surface        string
	startByte      int
	endByte        int
	userDictionary bool
}

// Surface は、原文に現れた token 表記を返す。
func (o TokenOccurrence) Surface() string {
	return o.surface
}

// StartByte は、token の開始 byte offset を返す。
func (o TokenOccurrence) StartByte() int {
	return o.startByte
}

// EndByte は、token の終端 byte offset を返す。
func (o TokenOccurrence) EndByte() int {
	return o.endByte
}

// UserDictionary は、token が起動時の user dictionary に由来するかを返す。
func (o TokenOccurrence) UserDictionary() bool {
	return o.userDictionary
}

// NewAnalyzer は、登録語を複製して不変な tokenizer を構築する。
func NewAnalyzer(terms []string) (*Analyzer, error) {
	if len(terms) == 0 || len(terms) > maxDictionaryTermCount {
		return nil, fmt.Errorf(
			"形態素解析の登録語は 1 件以上 %d 件以下でなければなりません",
			maxDictionaryTermCount,
		)
	}
	uniqueTerms := make(map[string]struct{}, len(terms))
	for index, term := range terms {
		if !utf8.ValidString(term) ||
			strings.TrimSpace(term) == "" ||
			len(term) > maxDictionaryTermBytes {
			return nil, fmt.Errorf(
				"terms[%d] は有効な UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
				index,
				maxDictionaryTermBytes,
			)
		}
		uniqueTerms[term] = struct{}{}
	}
	sortedTerms := make([]string, 0, len(uniqueTerms))
	for term := range uniqueTerms {
		sortedTerms = append(sortedTerms, term)
	}
	slices.Sort(sortedTerms)

	records := make(dict.UserDictRecords, 0, len(sortedTerms))
	for _, term := range sortedTerms {
		records = append(records, dict.UserDicRecord{
			Text:   term,
			Tokens: []string{term},
			Yomi:   []string{"*"},
			Pos:    "検索登録語",
		})
	}
	userDictionary, err := records.NewUserDict()
	if err != nil {
		return nil, fmt.Errorf(
			"形態素解析の user dictionary を構築できません: %w",
			err,
		)
	}
	kagomeTokenizer, err := tokenizer.New(
		ipa.Dict(),
		tokenizer.UserDict(userDictionary),
		tokenizer.OmitBosEos(),
	)
	if err != nil {
		return nil, fmt.Errorf("形態素解析の tokenizer を構築できません: %w", err)
	}
	return &Analyzer{
		tokenizer: kagomeTokenizer,
		gate:      make(chan struct{}, 1),
	}, nil
}

// RegisteredTerms は、検索向け解析結果に現れた user dictionary 語を返す。
func (a *Analyzer) RegisteredTerms(
	ctx context.Context,
	input string,
) ([]string, error) {
	tokens, err := a.analyze(ctx, input)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	terms := make([]string, 0)
	for _, token := range tokens {
		if token.Class != tokenizer.USER || token.Surface == "" {
			continue
		}
		if _, duplicated := seen[token.Surface]; duplicated {
			continue
		}
		seen[token.Surface] = struct{}{}
		terms = append(terms, token.Surface)
	}
	return terms, nil
}

// AnalyzeTokenOccurrences は、BOS/EOS を除く token と原文上の byte span を返す。
func (a *Analyzer) AnalyzeTokenOccurrences(
	ctx context.Context,
	input string,
) ([]TokenOccurrence, error) {
	tokens, err := a.analyze(ctx, input)
	if err != nil {
		return nil, err
	}
	occurrences := make([]TokenOccurrence, 0, len(tokens))
	for _, token := range tokens {
		if token.Surface == "" {
			continue
		}
		startByte := token.Position
		occurrences = append(occurrences, TokenOccurrence{
			surface:        token.Surface,
			startByte:      startByte,
			endByte:        startByte + len(token.Surface),
			userDictionary: token.Class == tokenizer.USER,
		})
	}
	return occurrences, nil
}

func (a *Analyzer) analyze(
	ctx context.Context,
	input string,
) ([]tokenizer.Token, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.tokenizer == nil || a.gate == nil {
		return nil, fmt.Errorf("形態素解析器は初期化されていません")
	}
	if !utf8.ValidString(input) || len(input) > maxAnalyzerInputBytes {
		return nil, fmt.Errorf(
			"解析入力は有効な UTF-8 で %d byte 以下でなければなりません",
			maxAnalyzerInputBytes,
		)
	}

	select {
	case a.gate <- struct{}{}:
		defer func() {
			<-a.gate
		}()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	tokens := a.tokenizer.Analyze(input, tokenizer.Search)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return tokens, nil
}
