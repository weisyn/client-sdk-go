package utils

import (
	"context"
	"errors"
	"testing"
)

// TestBatchQuery_EdgeCases 测试批量查询的边界情况
func TestBatchQuery_EdgeCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		items   []int
		config  *BatchConfig
		wantErr bool
	}{
		{
			name:    "nil items",
			items:   nil,
			config:  DefaultBatchConfig(),
			wantErr: false,
		},
		{
			name:    "very large batch",
			items:   make([]int, 10000),
			config:  DefaultBatchConfig(),
			wantErr: false,
		},
		{
			name:    "batch size larger than items",
			items:   []int{1, 2, 3},
			config:  &BatchConfig{BatchSize: 100, Concurrency: 5},
			wantErr: false,
		},
		{
			name:    "concurrency zero (should use default)",
			items:   []int{1, 2, 3},
			config:  &BatchConfig{BatchSize: 50, Concurrency: 0},
			wantErr: false,
		},
		{
			name:    "batch size zero (should use default)",
			items:   []int{1, 2, 3},
			config:  &BatchConfig{BatchSize: 0, Concurrency: 5},
			wantErr: false,
		},
		{
			name:    "very high concurrency",
			items:   make([]int, 100),
			config:  &BatchConfig{BatchSize: 10, Concurrency: 100},
			wantErr: false,
		},
		{
			name:    "single concurrency",
			items:   []int{1, 2, 3, 4, 5},
			config:  &BatchConfig{BatchSize: 2, Concurrency: 1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.items == nil {
				tt.items = []int{}
			}
			for i := range tt.items {
				tt.items[i] = i
			}

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
			}
		})
	}
}

// TestBatchQuery_AllErrors 测试所有项目都失败的情况
func TestBatchQuery_AllErrors(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	result, err := BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
		return 0, errors.New("test error")
	}, DefaultBatchConfig())

	if err != nil {
		t.Errorf("BatchQuery() error = %v, want nil", err)
	}

	if result.Success != 0 {
		t.Errorf("BatchQuery() Success = %d, want 0", result.Success)
	}
	if result.Failed != len(items) {
		t.Errorf("BatchQuery() Failed = %d, want %d", result.Failed, len(items))
	}
	if len(result.Errors) != len(items) {
		t.Errorf("BatchQuery() Errors = %d, want %d", len(result.Errors), len(items))
	}
}

// TestBatchQuery_PartialErrors 测试部分失败的情况
func TestBatchQuery_PartialErrors(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	result, err := BatchQuery(ctx, items, func(ctx context.Context, item int, index int) (int, error) {
		if item%2 == 0 {
			return 0, errors.New("even number error")
		}
		return item * 2, nil
	}, DefaultBatchConfig())

	if err != nil {
		t.Errorf("BatchQuery() error = %v, want nil", err)
	}

	if result.Success != 3 {
		t.Errorf("BatchQuery() Success = %d, want 3", result.Success)
	}
	if result.Failed != 2 {
		t.Errorf("BatchQuery() Failed = %d, want 2", result.Failed)
	}
}

// TestParallelExecute_EdgeCases 测试并行执行的边界情况
func TestParallelExecute_EdgeCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		items      []int
		concurrency int
		wantErr    bool
	}{
		{
			name:       "nil items",
			items:      nil,
			concurrency: 5,
			wantErr:    false,
		},
		{
			name:       "concurrency zero",
			items:      []int{1, 2, 3},
			concurrency: 0,
			wantErr:    true, // 应该失败或使用默认值
		},
		{
			name:       "concurrency negative",
			items:      []int{1, 2, 3},
			concurrency: -1,
			wantErr:    true,
		},
		{
			name:       "concurrency larger than items",
			items:      []int{1, 2, 3},
			concurrency: 100,
			wantErr:    false,
		},
		{
			name:       "very large items array",
			items:      make([]int, 1000),
			concurrency: 10,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.items == nil {
				tt.items = []int{}
			}
			for i := range tt.items {
				tt.items[i] = i
			}

			_, err := ParallelExecute(ctx, tt.items, func(ctx context.Context, item int) (int, error) {
				return item * 2, nil
			}, tt.concurrency)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParallelExecute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBatchArray_EdgeCases 测试数组分批的边界情况
func TestBatchArray_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		array     []int
		batchSize int
		wantBatches int
	}{
		{
			name:       "nil array",
			array:      nil,
			batchSize:  5,
			wantBatches: 0,
		},
		{
			name:       "batch size zero",
			array:      []int{1, 2, 3},
			batchSize:  0,
			wantBatches: 0, // 应该返回空数组或处理错误
		},
		{
			name:       "batch size negative",
			array:      []int{1, 2, 3},
			batchSize:  -1,
			wantBatches: 0,
		},
		{
			name:       "batch size larger than array",
			array:      []int{1, 2, 3},
			batchSize:  100,
			wantBatches: 1,
		},
		{
			name:       "very large array",
			array:      make([]int, 10000),
			batchSize:  50,
			wantBatches: 200,
		},
		{
			name:       "single element batches",
			array:      []int{1, 2, 3, 4, 5},
			batchSize:  1,
			wantBatches: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.array == nil {
				tt.array = []int{}
			}
			for i := range tt.array {
				tt.array[i] = i
			}

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

