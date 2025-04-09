package generator

import (
	"math/rand"
	"time"
)

// RandomGenerator генератор случайных строк.
type RandomGenerator struct {
	r *rand.Rand
}

// NewRandomGenerator создает и возвращает новый экземпляр RandomGenerator
func NewRandomGenerator() *RandomGenerator {
	return &RandomGenerator{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewRandomString генерирует случайную строку заданной длины, состоящую из букв и цифр.
func (g *RandomGenerator) NewRandomString(size int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[g.r.Intn(len(chars))]
	}
	return string(result)
}
