package utils

import (
	"context"
	"testing"
)

func BenchmarkBatchQuery(b *testing.B) {
	ctx := context.Background()
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
			return item * 2, nil
		}, &BatchConfig{
			BatchSize:  50,
			Concurrency: 5,
		})
	}
}

func BenchmarkBatchQuery_NoConcurrency(b *testing.B) {
	ctx := context.Background()
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
			return item * 2, nil
		}, &BatchConfig{
			BatchSize:  50,
			Concurrency: 1,
		})
	}
}

func BenchmarkBatchQuery_HighConcurrency(b *testing.B) {
	ctx := context.Background()
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
			return item * 2, nil
		}, &BatchConfig{
			BatchSize:  50,
			Concurrency: 20,
		})
	}
}

func BenchmarkParallelExecute(b *testing.B) {
	ctx := context.Background()
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParallelExecute(ctx, items, func(ctx context.Context, item int) (int, error) {
			return item * 2, nil
		}, 5)
	}
}

func BenchmarkChunkFile(b *testing.B) {
	data := make([]byte, 10*1024*1024) // 10MB
	chunkSize := int64(1024 * 1024)    // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ChunkFile(data, chunkSize)
	}
}

func BenchmarkBatchArray(b *testing.B) {
	array := make([]int, 10000)
	batchSize := 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BatchArray(array, batchSize)
	}
}

