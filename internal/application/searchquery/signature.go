package searchquery

import "math/bits"

// SOT-ARCH-021: 辞書の文字集合だけを固定幅に写し、入力由来の状態は呼出し内に置く。
func fuzzyRuneSignature(value string) (signature [2]uint64) {
	for _, character := range value {
		bit := uint32(character) * 0x9e3779b1 >> 25 //nolint:gosec // SOT-ARCH-021: string の range が返す rune は非負であり、uint32 に収まる。
		signature[bit>>6] |= uint64(1) << (bit & 63)
	}
	return signature
}

// SOT-ARCH-030: 一編集で各方向の集合差は高々一、転置では変わらない。
// hash の衝突は候補を余分に残すだけであり、距離と一意性は既存の照合で確定する。
func fuzzySignatureMatches(left, right [2]uint64, maximum int) bool {
	return bits.OnesCount64(left[0]&^right[0])+bits.OnesCount64(left[1]&^right[1]) <= maximum &&
		bits.OnesCount64(right[0]&^left[0])+bits.OnesCount64(right[1]&^left[1]) <= maximum
}
