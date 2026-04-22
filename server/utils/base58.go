package utils

import (
	"math/big"
	"strings"
)

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var (
	base58Radix = big.NewInt(58)
	base58Index = func() map[rune]int {
		m := make(map[rune]int, len(base58Alphabet))
		for i, ch := range base58Alphabet {
			m[ch] = i
		}
		return m
	}()
)

// Base58Encode 对二进制数据做 base58 编码。
// 主要用于会话 ID 在前端/WS 信封中的安全传输。
func Base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	leadingZeroes := 0
	for leadingZeroes < len(input) && input[leadingZeroes] == 0 {
		leadingZeroes++
	}

	n := new(big.Int).SetBytes(input)
	if n.Sign() == 0 {
		return strings.Repeat("1", leadingZeroes)
	}

	var encoded []byte
	mod := new(big.Int)
	for n.Sign() > 0 {
		n.DivMod(n, base58Radix, mod)
		encoded = append(encoded, base58Alphabet[mod.Int64()])
	}
	for i := 0; i < leadingZeroes; i++ {
		encoded = append(encoded, '1')
	}

	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}
	return string(encoded)
}

// Base58Decode 将 base58 字符串解码为原始二进制数据。
// 输入非法时返回 nil。
func Base58Decode(input string) []byte {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil
	}

	leadingOnes := 0
	for leadingOnes < len(text) && text[leadingOnes] == '1' {
		leadingOnes++
	}

	n := big.NewInt(0)
	for _, ch := range text {
		idx, ok := base58Index[ch]
		if !ok {
			return nil
		}
		n.Mul(n, base58Radix)
		n.Add(n, big.NewInt(int64(idx)))
	}

	decoded := n.Bytes()
	if leadingOnes > 0 {
		out := make([]byte, leadingOnes+len(decoded))
		copy(out[leadingOnes:], decoded)
		return out
	}
	return decoded
}
