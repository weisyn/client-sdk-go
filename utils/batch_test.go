package utils

import (
	"context"
	"errors"
	"testing"
)

// TestChunkFile 已移至 file_test.go（ChunkFile 在 file.go 中定义）

func TestBatchQuery(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		items   []int
		config  *BatchConfig
		wantErr bool
	}{
		{
			name:  "empty items",
			items: []int{},
			config: DefaultBatchConfig(),
		},
		{
			name:  "single item",
			items: []int{1},
			config: DefaultBatchConfig(),
		},
		{
			name:  "multiple items",
			items: []int{1, 2, 3, 4, 5},
			config: &BatchConfig{
				BatchSize:  2,
				Concurrency: 2,
			},
		},
		{
			name:  "with progress callback",
			items: []int{1, 2, 3, 4, 5},
			config: &BatchConfig{
				BatchSize:  2,
				Concurrency: 2,
				OnProgress: func(progress BatchProgress) {
					// 验证进度信息
					if progress.Total != 5 {
						t.Errorf("OnProgress: Total = %d, want 5", progress.Total)
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BatchQuery(ctx, tt.items, func(ctx context.Context, item int, index int) (int, error) {
				return item * 2, nil
			}, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("BatchQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(result.Results) != len(tt.items) {
					t.Errorf("BatchQuery() got %d results, want %d", len(result.Results), len(tt.items))
				}
				if result.Total != len(tt.items) {
					t.Errorf("BatchQuery() Total = %d, want %d", result.Total, len(tt.items))
				}
				if result.Success != len(tt.items) {
					t.Errorf("BatchQuery() Success = %d, want %d", result.Success, len(tt.items))
				}
			}
		})
	}
}

func TestBatchQuery_WithErrors(t *testing.T) {
	ctx := context.Background()

	items := []int{1, 2, 3, 4, 5}
	result, err := BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
		if item == 3 {
			return 0, errors.New("test error")
		}
		return item * 2, nil
	}, DefaultBatchConfig())

	if err != nil {
		t.Errorf("BatchQuery() error = %v, want nil", err)
	}

	if result.Success != 4 {
		t.Errorf("BatchQuery() Success = %d, want 4", result.Success)
	}
	if result.Failed != 1 {
		t.Errorf("BatchQuery() Failed = %d, want 1", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Errorf("BatchQuery() Errors = %d, want 1", len(result.Errors))
	}
}

func TestBatchQuery_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	items := []int{1, 2, 3, 4, 5}
	_, err := BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
		return item * 2, nil
	}, DefaultBatchConfig())

	if err == nil {
		t.Error("BatchQuery() error = nil, want context canceled error")
	}
}

func TestParallelExecute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		items      []int
		concurrency int
		wantErr    bool
	}{
		{
			name:       "empty items",
			items:      []int{},
			concurrency: 5,
		},
		{
			name:       "single item",
			items:      []int{1},
			concurrency: 5,
		},
		{
			name:       "multiple items",
			items:      []int{1, 2, 3, 4, 5},
			concurrency: 3,
		},
		{
			name:       "high concurrency",
			items:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			concurrency: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParallelExecute(ctx, tt.items, func(ctx context.Context, item int) (int, error) {
				return item * 2, nil
			}, tt.concurrency)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParallelExecute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(results) != len(tt.items) {
					t.Errorf("ParallelExecute() got %d results, want %d", len(results), len(tt.items))
				}
				for i, result := range results {
					if result != tt.items[i]*2 {
						t.Errorf("ParallelExecute() results[%d] = %d, want %d", i, result, tt.items[i]*2)
					}
				}
			}
		})
	}
}

func TestParallelExecute_WithErrors(t *testing.T) {
	ctx := context.Background()

	items := []int{1, 2, 3, 4, 5}
	_, err := ParallelExecute(ctx, items, func(ctx context.Context, item int) (int, error) {
		if item == 3 {
			return 0, errors.New("test error")
		}
		return item * 2, nil
	}, 5)

	if err == nil {
		t.Error("ParallelExecute() error = nil, want error")
	}
}

func TestBatchArray(t *testing.T) {
	tests := []struct {
		name      string
		array     []int
		batchSize int
		wantBatches int
	}{
		{
			name:       "empty array",
			array:      []int{},
			batchSize:  5,
			wantBatches: 0,
		},
		{
			name:       "single batch",
			array:      []int{1, 2, 3},
			batchSize:  5,
			wantBatches: 1,
		},
		{
			name:       "multiple batches",
			array:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			batchSize:  3,
			wantBatches: 4,
		},
		{
			name:       "exact batch size",
			array:      []int{1, 2, 3, 4, 5},
			batchSize:  5,
			wantBatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := BatchArray(tt.array, tt.batchSize)
			if len(batches) != tt.wantBatches {
				t.Errorf("BatchArray() got %d batches, want %d", len(batches), tt.wantBatches)
			}

			// 验证所有元素都被包含
			totalLen := 0
			for _, batch := range batches {
				totalLen += len(batch)
			}
			if totalLen != len(tt.array) {
				t.Errorf("BatchArray() total length = %d, want %d", totalLen, len(tt.array))
			}
		})
	}
}

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()
	if config.BatchSize != 50 {
		t.Errorf("DefaultBatchConfig() BatchSize = %d, want 50", config.BatchSize)
	}
	if config.Concurrency != 5 {
		t.Errorf("DefaultBatchConfig() Concurrency = %d, want 5", config.Concurrency)
	}
}

