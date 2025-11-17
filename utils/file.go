package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ChunkConfig 文件分块配置
type ChunkConfig struct {
	// ChunkSize 分块大小（字节）
	ChunkSize int64
	// Concurrency 并发处理数量
	Concurrency int
	// OnProgress 进度回调函数
	OnProgress func(progress FileProgress)
}

// FileProgress 文件处理进度
type FileProgress struct {
	// Loaded 已处理字节数
	Loaded int64
	// Total 总字节数
	Total int64
	// Percentage 进度百分比（0-100）
	Percentage int
	// CurrentChunk 当前分块索引（从1开始）
	CurrentChunk int
	// TotalChunks 总分块数
	TotalChunks int
}

// DefaultChunkConfig 返回默认分块配置
func DefaultChunkConfig() *ChunkConfig {
	return &ChunkConfig{
		ChunkSize:   1024 * 1024, // 1MB
		Concurrency: 3,
		OnProgress: nil,
	}
}

// ChunkFile 将字节数组分块
func ChunkFile(data []byte, chunkSize int64) [][]byte {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkConfig().ChunkSize
	}

	chunks := make([][]byte, 0)
	for i := int64(0); i < int64(len(data)); i += chunkSize {
		end := i + chunkSize
		if end > int64(len(data)) {
			end = int64(len(data))
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

// ReadFileAsStream 流式读取文件（支持大文件）
//
// 示例：
//
//	progress := func(p FileProgress) {
//	    fmt.Printf("Progress: %d%%\n", p.Percentage)
//	}
//	data, err := ReadFileAsStream("large_file.bin", progress)
func ReadFileAsStream(filePath string, onProgress func(FileProgress)) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file failed: %w", err)
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("get file info failed: %w", err)
	}
	fileSize := fileInfo.Size()

	// 读取文件内容
	data := make([]byte, fileSize)
	var offset int64
	buffer := make([]byte, 64*1024) // 64KB 缓冲区

	for offset < fileSize {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read file failed: %w", err)
		}
		if n == 0 {
			break
		}

		copy(data[offset:], buffer[:n])
		offset += int64(n)

		// 调用进度回调
		if onProgress != nil {
			percentage := int((offset * 100) / fileSize)
			onProgress(FileProgress{
				Loaded:     offset,
				Total:      fileSize,
				Percentage: percentage,
			})
		}
	}

	return data, nil
}

// ProcessFileInChunks 分块处理文件
//
// 对文件数据进行分块处理，支持并发处理和进度回调
//
// 示例：
//
//	results, err := ProcessFileInChunks(ctx, data, func(chunk []byte, index int) (string, error) {
//	    // 处理分块
//	    return processChunk(chunk), nil
//	}, &ChunkConfig{
//	    ChunkSize: 5 * 1024 * 1024, // 5MB
//	    Concurrency: 3,
//	    OnProgress: func(p FileProgress) {
//	        fmt.Printf("Progress: %d%%\n", p.Percentage)
//	    },
//	})
func ProcessFileInChunks[T any](
	ctx context.Context,
	data []byte,
	processor func(chunk []byte, index int) (T, error),
	config *ChunkConfig,
) ([]T, error) {
	if config == nil {
		config = DefaultChunkConfig()
	}

	if config.ChunkSize <= 0 {
		config.ChunkSize = DefaultChunkConfig().ChunkSize
	}
	if config.Concurrency <= 0 {
		config.Concurrency = DefaultChunkConfig().Concurrency
	}

	// 如果文件小于分块大小，直接处理
	if int64(len(data)) <= config.ChunkSize {
		result, err := processor(data, 0)
		if err != nil {
			return nil, err
		}

		if config.OnProgress != nil {
			config.OnProgress(FileProgress{
				Loaded:     int64(len(data)),
				Total:      int64(len(data)),
				Percentage: 100,
				CurrentChunk: 1,
				TotalChunks: 1,
			})
		}

		return []T{result}, nil
	}

	// 分块处理
	chunks := ChunkFile(data, config.ChunkSize)
	results := make([]T, len(chunks))
	errors := make([]error, len(chunks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, config.Concurrency)
	var completed int64
	var completedMu sync.Mutex

	updateProgress := func(chunkIndex int) {
		completedMu.Lock()
		defer completedMu.Unlock()
		completed++
		percentage := int((completed * 100) / int64(len(chunks)))
		if config.OnProgress != nil {
			config.OnProgress(FileProgress{
				Loaded:      (completed * config.ChunkSize),
				Total:       int64(len(data)),
				Percentage:  percentage,
				CurrentChunk: int(completed),
				TotalChunks: len(chunks),
			})
		}
	}

	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, chunkData []byte) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				errors[index] = ctx.Err()
				return
			default:
			}

			// 处理分块
			result, err := processor(chunkData, index)
			if err != nil {
				errors[index] = err
			} else {
				results[index] = result
			}

			updateProgress(index)
		}(i, chunk)
	}

	wg.Wait()

	// 检查是否有错误
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("process file chunks failed: %w", err)
		}
	}

	return results, nil
}

// ProcessFileInChunksFromPath 从文件路径分块处理文件
//
// 示例：
//
//	results, err := ProcessFileInChunksFromPath(ctx, "large_file.bin", func(chunk []byte, index int) (string, error) {
//	    return processChunk(chunk), nil
//	}, &ChunkConfig{
//	    ChunkSize: 5 * 1024 * 1024,
//	    Concurrency: 3,
//	})
func ProcessFileInChunksFromPath[T any](
	ctx context.Context,
	filePath string,
	processor func(chunk []byte, index int) (T, error),
	config *ChunkConfig,
) ([]T, error) {
	// 读取文件
	data, err := ReadFileAsStream(filePath, nil)
	if err != nil {
		return nil, err
	}

	// 分块处理
	return ProcessFileInChunks(ctx, data, processor, config)
}

// EstimateProcessingTime 估算文件处理时间
//
// 参数：
//   - fileSize: 文件大小（字节）
//   - chunkSize: 分块大小（字节）
//   - processingSpeed: 处理速度（字节/秒），默认 1MB/s
//
// 返回：估算的处理时间（秒）
func EstimateProcessingTime(fileSize int64, chunkSize int64, processingSpeed int64) time.Duration {
	if processingSpeed <= 0 {
		processingSpeed = 1024 * 1024 // 1MB/s
	}

	if chunkSize <= 0 {
		chunkSize = DefaultChunkConfig().ChunkSize
	}

	// 计算分块数
	chunks := (fileSize + chunkSize - 1) / chunkSize // 向上取整

	// 估算时间（考虑并发）
	estimatedSeconds := float64(fileSize) / float64(processingSpeed)
	if chunks > 1 {
		// 并发处理可以减少时间
		concurrency := DefaultChunkConfig().Concurrency
		if concurrency > 0 {
			estimatedSeconds = estimatedSeconds / float64(concurrency)
		}
	}

	return time.Duration(estimatedSeconds) * time.Second
}

// ReadFileInChunks 分块读取文件（流式读取）
//
// 示例：
//
//	err := ReadFileInChunks("large_file.bin", func(chunk []byte, index int) error {
//	    // 处理每个分块
//	    return processChunk(chunk)
//	}, &ChunkConfig{
//	    ChunkSize: 5 * 1024 * 1024,
//	    OnProgress: func(p FileProgress) {
//	        fmt.Printf("Progress: %d%%\n", p.Percentage)
//	    },
//	})
func ReadFileInChunks(
	filePath string,
	processor func(chunk []byte, index int) error,
	config *ChunkConfig,
) error {
	if config == nil {
		config = DefaultChunkConfig()
	}

	if config.ChunkSize <= 0 {
		config.ChunkSize = DefaultChunkConfig().ChunkSize
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file failed: %w", err)
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("get file info failed: %w", err)
	}
	fileSize := fileInfo.Size()

	chunkIndex := 0
	var totalRead int64
	buffer := make([]byte, config.ChunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read file failed: %w", err)
		}
		if n == 0 {
			break
		}

		// 处理当前分块
		chunk := make([]byte, n)
		copy(chunk, buffer[:n])
		if err := processor(chunk, chunkIndex); err != nil {
			return fmt.Errorf("process chunk %d failed: %w", chunkIndex, err)
		}

		totalRead += int64(n)
		chunkIndex++

		// 调用进度回调
		if config.OnProgress != nil {
			percentage := int((totalRead * 100) / fileSize)
			config.OnProgress(FileProgress{
				Loaded:      totalRead,
				Total:       fileSize,
				Percentage:  percentage,
				CurrentChunk: chunkIndex,
				TotalChunks: int((fileSize + config.ChunkSize - 1) / config.ChunkSize),
			})
		}
	}

	return nil
}

