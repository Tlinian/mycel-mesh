// Package batch provides batch processing for Coordinator services.
package batch

import (
	"context"
	"sync"
	"time"
)

// Config holds configuration for batch processing.
type Config struct {
	// BatchSize is the maximum number of items to process in a batch.
	BatchSize int
	// BatchTimeout is the maximum time to wait before processing a partial batch.
	BatchTimeout time.Duration
	// MaxPending is the maximum number of pending items before blocking.
	MaxPending int
	// Workers is the number of concurrent workers processing batches.
	Workers int
}

// DefaultConfig returns a default batch configuration.
func DefaultConfig() Config {
	return Config{
		BatchSize:    100,
		BatchTimeout: 100 * time.Millisecond,
		MaxPending:   1000,
		Workers:      4,
	}
}

// Item represents an item to be batch processed.
type Item struct {
	ID      string
	Payload interface{}
	Error   error
}

// Batch represents a collection of items to be processed together.
type Batch struct {
	Items     []*Item
	CreatedAt time.Time
	Processed time.Time
}

// Processor handles batch processing of items.
type Processor struct {
	config     Config
	inputChan  chan *Item
	outputChan chan *Batch
	errChan    chan error
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	processorFn ProcessFunc
}

// ProcessFunc is the function type for processing a batch.
type ProcessFunc func(ctx context.Context, batch *Batch) error

// NewProcessor creates a new batch processor.
func NewProcessor(config Config, processFn ProcessFunc) *Processor {
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultConfig().BatchSize
	}
	if config.BatchTimeout <= 0 {
		config.BatchTimeout = DefaultConfig().BatchTimeout
	}
	if config.MaxPending <= 0 {
		config.MaxPending = DefaultConfig().MaxPending
	}
	if config.Workers <= 0 {
		config.Workers = DefaultConfig().Workers
	}

	ctx, cancel := context.WithCancel(context.Background())

	processor := &Processor{
		config:      config,
		inputChan:   make(chan *Item, config.MaxPending),
		outputChan:  make(chan *Batch, config.Workers),
		errChan:     make(chan error, config.Workers),
		ctx:         ctx,
		cancel:      cancel,
		processorFn: processFn,
	}

	// Start workers
	for i := 0; i < config.Workers; i++ {
		processor.wg.Add(1)
		go processor.worker(i)
	}

	// Start batch collector
	processor.wg.Add(1)
	go processor.collector()

	return processor
}

// Submit submits an item for batch processing.
func (p *Processor) Submit(item *Item) error {
	select {
	case p.inputChan <- item:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
		return ErrQueueFull
	}
}

// SubmitSync submits an item and waits for it to be processed.
func (p *Processor) SubmitSync(ctx context.Context, item *Item) error {
	done := make(chan error, 1)

	wrappedItem := &Item{
		ID:      item.ID,
		Payload: item.Payload,
	}

	// Submit with callback
	select {
	case p.inputChan <- wrappedItem:
		// Wait for result
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
		return ErrQueueFull
	}
}

// Results returns the output channel for processed batches.
func (p *Processor) Results() <-chan *Batch {
	return p.outputChan
}

// Errors returns the error channel.
func (p *Processor) Errors() <-chan error {
	return p.errChan
}

// Stop stops the processor and waits for all workers to finish.
func (p *Processor) Stop() {
	p.cancel()
	close(p.inputChan)
	p.wg.Wait()
	close(p.outputChan)
	close(p.errChan)
}

// worker processes batches from the input channel.
func (p *Processor) worker(id int) {
	defer p.wg.Done()

	batch := make([]*Item, 0, p.config.BatchSize)
	timer := time.NewTimer(p.config.BatchTimeout)
	defer timer.Stop()

	for {
		select {
		case item, ok := <-p.inputChan:
			if !ok {
				// Input channel closed, process remaining items
				if len(batch) > 0 {
					p.processBatch(batch)
				}
				return
			}

			batch = append(batch, item)

			// Check if batch is full
			if len(batch) >= p.config.BatchSize {
				if !timer.Stop() {
					<-timer.C
				}
				p.processBatch(batch)
				batch = make([]*Item, 0, p.config.BatchSize)
				timer.Reset(p.config.BatchTimeout)
			}

		case <-timer.C:
			// Timeout, process partial batch
			if len(batch) > 0 {
				p.processBatch(batch)
				batch = make([]*Item, 0, p.config.BatchSize)
			}
			timer.Reset(p.config.BatchTimeout)

		case <-p.ctx.Done():
			// Context cancelled, drain and exit
			if len(batch) > 0 {
				p.processBatch(batch)
			}
			// Drain remaining items
			for item := range p.inputChan {
				batch = append(batch, item)
				if len(batch) >= p.config.BatchSize {
					p.processBatch(batch)
					batch = make([]*Item, 0, p.config.BatchSize)
				}
			}
			return
		}
	}
}

// collector collects processed batches and sends to output channel.
func (p *Processor) collector() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		}
	}
}

// processBatch processes a single batch.
func (p *Processor) processBatch(items []*Item) {
	if len(items) == 0 {
		return
	}

	batch := &Batch{
		Items:     items,
		CreatedAt: time.Now(),
	}

	// Process the batch
	if p.processorFn != nil {
		err := p.processorFn(p.ctx, batch)
		if err != nil {
			select {
			case p.errChan <- err:
			default:
			}

			// Mark items with error
			for _, item := range items {
				item.Error = err
			}
		}
	}

	batch.Processed = time.Now()

	// Send to output channel
	select {
	case p.outputChan <- batch:
	case <-p.ctx.Done():
	default:
	}
}

// Stats holds statistics about batch processing.
type Stats struct {
	TotalBatches    int64
	TotalItems      int64
	ProcessedItems  int64
	FailedItems     int64
	AvgBatchSize    float64
	AvgProcessTime  time.Duration
	LastProcessedAt time.Time
	mu              sync.RWMutex
}

// StatsProcessor wraps Processor with statistics tracking.
type StatsProcessor struct {
	*Processor
	stats Stats
}

// NewStatsProcessor creates a batch processor with statistics.
func NewStatsProcessor(config Config, processFn ProcessFunc) *StatsProcessor {
	return &StatsProcessor{
		Processor: NewProcessor(config, processFn),
	}
}

// GetStats returns current processing statistics.
func (sp *StatsProcessor) GetStats() Stats {
	sp.stats.mu.RLock()
	defer sp.stats.mu.RUnlock()
	return sp.stats
}

// ErrQueueFull is returned when the input queue is full.
var ErrQueueFull = &QueueFullError{}

// QueueFullError indicates the queue is full.
type QueueFullError struct{}

func (e *QueueFullError) Error() string {
	return "batch queue is full"
}

// BatchProcessor provides a simple interface for batch operations.
type BatchProcessor struct {
	processor *Processor
}

// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(config Config) *BatchProcessor {
	return &BatchProcessor{
		processor: NewProcessor(config, nil),
	}
}

// ProcessFunc sets the processing function.
func (bp *BatchProcessor) ProcessFunc(fn ProcessFunc) {
	bp.processor.processorFn = fn
}

// Submit submits an item for processing.
func (bp *BatchProcessor) Submit(item *Item) error {
	return bp.processor.Submit(item)
}

// Stop stops the processor.
func (bp *BatchProcessor) Stop() {
	bp.processor.Stop()
}

// Results returns processed batches.
func (bp *BatchProcessor) Results() <-chan *Batch {
	return bp.processor.Results()
}
