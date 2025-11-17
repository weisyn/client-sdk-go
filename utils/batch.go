package utils

import (
	"context"
	"fmt"
	"sync"
)

// BatchConfig 批量操作配置
type BatchConfig struct {
	// BatchSize 批量大小
	BatchSize int
	// Concurrency 并发数量
	Concurrency int
	// OnProgress 进度回调函数
	OnProgress func(progress BatchProgress)
}

// BatchProgress 批量操作进度
type BatchProgress struct {
	// Completed 已完成数量
	Completed int
	// Total 总数量
	Total int
	// Percentage 进度百分比（0-100）
	Percentage int
	// Success 成功数量
	Success int
	// Failed 失败数量
	Failed int
}

// DefaultBatchConfig 返回默认批量配置
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		BatchSize:  50,
		Concurrency: 5,
		OnProgress: nil,
	}
}

// BatchQueryResult 批量查询结果
type BatchQueryResult[T any] struct {
	// Results 成功的结果
	Results []T
	// Errors 失败的项目
	Errors []BatchError
	// Total 总数量
	Total int
	// Success 成功数量
	Success int
	// Failed 失败数量
	Failed int
}

// BatchError 批量操作错误
type BatchError struct {
	// Index 项目索引
	Index int
	// Error 错误信息
	Error error
}

// BatchQuery 批量查询
//
// 对一组输入并发调用查询函数，返回成功和失败的结果列表
//
// 示例：
//
//	addresses := [][]byte{addr1, addr2, addr3}
//	results, err := BatchQuery(ctx, addresses, func(ctx context.Context, addr []byte, index int) (uint64, error) {
//	    return tokenService.GetBalance(ctx, addr, nil)
//	}, DefaultBatchConfig())
func BatchQuery[T any, R any](
	ctx context.Context,
	items []T,
	queryFn func(ctx context.Context, item T, index int) (R, error),
	config *BatchConfig,
) (*BatchQueryResult[R], error) {
	if config == nil {
		config = DefaultBatchConfig()
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}

	results := make([]R, 0, len(items))
	errors := make([]BatchError, 0)
	var resultsMu sync.Mutex

	completed := 0
	success := 0
	failed := 0
	var progressMu sync.Mutex

	updateProgress := func() {
		progressMu.Lock()
		defer progressMu.Unlock()
		completed++
		percentage := (completed * 100) / len(items)
		if config.OnProgress != nil {
			config.OnProgress(BatchProgress{
				Completed:  completed,
				Total:      len(items),
				Percentage: percentage,
				Success:    success,
				Failed:     failed,
			})
		}
	}

	// 分批处理
	batches := batchArray(items, config.BatchSize)

	for batchIdx, batch := range batches {
		// 并发处理当前批次
		var wg sync.WaitGroup
		sem := make(chan struct{}, config.Concurrency)

		for i, item := range batch {
			wg.Add(1)
			globalIndex := batchIdx*config.BatchSize + i
			go func(idx int, batchItem T) {
				defer wg.Done()

				// 获取信号量
				sem <- struct{}{}
				defer func() { <-sem }()

				// 执行查询
				result, err := queryFn(ctx, batchItem, idx)
				if err != nil {
					resultsMu.Lock()
					errors = append(errors, BatchError{
						Index: idx,
						Error: err,
					})
					failed++
					resultsMu.Unlock()
				} else {
					resultsMu.Lock()
					results = append(results, result)
					success++
					resultsMu.Unlock()
				}

				updateProgress()
			}(globalIndex, item)
		}

		wg.Wait()
	}

	return &BatchQueryResult[R]{
		Results: results,
		Errors:  errors,
		Total:   len(items),
		Success: success,
		Failed:  failed,
	}, nil
}

// batchArray 将数组分批次处理
func batchArray[T any](array []T, batchSize int) [][]T {
	batches := make([][]T, 0)
	for i := 0; i < len(array); i += batchSize {
		end := i + batchSize
		if end > len(array) {
			end = len(array)
		}
		batches = append(batches, array[i:end])
	}
	return batches
}

// ParallelExecute 并行执行多个操作
//
// 对一组输入并发执行操作函数，限制并发数量
//
// 示例：
//
//	items := []string{"item1", "item2", "item3"}
//	results, err := ParallelExecute(ctx, items, func(ctx context.Context, item string) (string, error) {
//	    return processItem(item)
//	}, 5) // 并发5个
func ParallelExecute[T any, R any](
	ctx context.Context,
	items []T,
	executeFn func(ctx context.Context, item T) (R, error),
	concurrency int,
) ([]R, error) {
	if concurrency <= 0 {
		concurrency = 5
	}

	results := make([]R, len(items))
	errors := make([]error, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i, item := range items {
		wg.Add(1)
		go func(index int, batchItem T) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 执行操作
			result, err := executeFn(ctx, batchItem)
			if err != nil {
				errors[index] = err
			} else {
				results[index] = result
			}
		}(i, item)
	}

	wg.Wait()

	// 检查是否有错误
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("parallel execute failed: %w", err)
		}
	}

	return results, nil
}

// BatchArray 将数组分批次处理（导出函数）
func BatchArray[T any](array []T, batchSize int) [][]T {
	return batchArray(array, batchSize)
}

