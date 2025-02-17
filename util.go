package main

// CalculatePrefixSum 前缀和算法，传入 counts 数组，返回累积和数组
func CalculatePrefixSum(counts []int64) []int64 {
	n := len(counts)
	prefix := make([]int64, n)
	if n == 0 {
		return prefix
	}
	prefix[0] = counts[0]
	for i := 1; i < n; i++ {
		prefix[i] = prefix[i-1] + counts[i]
	}
	return prefix
}
