package profileevidence

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

// ClusterMember は、一つの step に対応する主題順と根拠 span である。
type ClusterMember struct {
	topicOrdinal int
	evidenceSpan legalquery.QuerySpan
}

// TopicOrdinal は、原文順で一から始まる主題番号を返す。
func (m ClusterMember) TopicOrdinal() int {
	return m.topicOrdinal
}

// EvidenceSpan は、この step の cluster 根拠 span を返す。
func (m ClusterMember) EvidenceSpan() legalquery.QuerySpan {
	return m.evidenceSpan
}

// ClusterKey は、候補 draft の step ごとの cluster member を原文順に保持する。
type ClusterKey struct {
	members []ClusterMember
}

// Members は、step ごとの member の複製を返す。
func (k ClusterKey) Members() []ClusterMember {
	return append([]ClusterMember(nil), k.members...)
}

// Canonical は、mapping の寿命内で比較する決定的な key を返す。
func (k ClusterKey) Canonical() string {
	var builder strings.Builder
	for index, member := range k.members {
		if index > 0 {
			builder.WriteByte('|')
		}
		builder.WriteString(strconv.Itoa(member.topicOrdinal))
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(member.evidenceSpan.StartByte()))
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(member.evidenceSpan.EndByte()))
	}
	return builder.String()
}

// ClusterKey は、一つの draft の cluster key と追加分岐適格性を返す。
func (m Mapping) ClusterKey(draftID string) (ClusterKey, bool, error) {
	value, exists := m.drafts[draftID]
	if !exists {
		return ClusterKey{}, false, fmt.Errorf(
			"draftId %q は mapping に存在しません",
			draftID,
		)
	}

	members := make([]ClusterMember, 0, len(value.steps))
	positiveTopics := make(map[int]bool, len(value.steps))
	requiredTopics := make(map[int]struct{}, len(value.steps))
	for _, current := range value.steps {
		requiredTopics[current.topicOrdinal] = struct{}{}
		var selected *legalquery.QuerySpan
		for _, evidence := range current.normalizedEvidence {
			if evidence.independentPositive {
				positiveTopics[current.topicOrdinal] = true
			}
			if selected == nil && evidence.clusterSpan {
				selected = cloneSpan(evidence.span)
			}
		}
		if selected == nil {
			return ClusterKey{}, false, nil
		}
		members = append(members, ClusterMember{
			topicOrdinal: current.topicOrdinal,
			evidenceSpan: *selected,
		})
	}
	for topicOrdinal := range requiredTopics {
		if !positiveTopics[topicOrdinal] {
			return ClusterKey{}, false, nil
		}
	}
	return ClusterKey{members: members}, true, nil
}
