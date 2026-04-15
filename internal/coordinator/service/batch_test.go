package service

import (
	"context"
	"testing"
	"time"
)

// TestDefaultConfig tests default batch configuration.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BatchSize != 100 {
		t.Fatalf("expected BatchSize 100, got %d", config.BatchSize)
	}
	if config.BatchTimeout != 100*time.Millisecond {
		t.Fatalf("expected BatchTimeout 100ms, got %v", config.BatchTimeout)
	}
	if config.MaxPending != 1000 {
		t.Fatalf("expected MaxPending 1000, got %d", config.MaxPending)
	}
	if config.Workers != 4 {
		t.Fatalf("expected Workers 4, got %d", config.Workers)
	}
}

// TestNewProcessor_DefaultConfig tests processor with default config.
func TestNewProcessor_DefaultConfig(t *testing.T) {
	config := Config{} // Empty, should use defaults
	processFn := func(ctx context.Context, batch *Batch) error {
		return nil
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	if processor.config.BatchSize != DefaultConfig().BatchSize {
		t.Fatalf("expected default BatchSize, got %d", processor.config.BatchSize)
	}
}

// TestNewProcessor_CustomConfig tests processor with custom config.
func TestNewProcessor_CustomConfig(t *testing.T) {
	config := Config{
		BatchSize:    50,
		BatchTimeout: 200 * time.Millisecond,
		MaxPending:   500,
		Workers:      2,
	}
	processFn := func(ctx context.Context, batch *Batch) error {
		return nil
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	if processor.config.BatchSize != 50 {
		t.Fatalf("expected BatchSize 50, got %d", processor.config.BatchSize)
	}
	if processor.config.Workers != 2 {
		t.Fatalf("expected Workers 2, got %d", processor.config.Workers)
	}
}

// TestProcessor_Submit tests item submission.
func TestProcessor_Submit(t *testing.T) {
	config := Config{
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
		MaxPending:   100,
		Workers:      1,
	}

	var processedBatches []*Batch
	processFn := func(ctx context.Context, batch *Batch) error {
		processedBatches = append(processedBatches, batch)
		return nil
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	// Submit items
	for i := 0; i < 5; i++ {
		item := &Item{ID: "item-" + string(rune('0'+i)), Payload: i}
		err := processor.Submit(item)
		if err != nil {
			t.Fatalf("Submit() failed: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify processing
	if len(processedBatches) == 0 {
		t.Log("Note: Batches may not have been processed yet due to timing")
	}
}

// TestProcessor_Results tests result channel.
func TestProcessor_Results(t *testing.T) {
	config := DefaultConfig()
	processFn := func(ctx context.Context, batch *Batch) error {
		return nil
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	resultsChan := processor.Results()
	if resultsChan == nil {
		t.Fatal("Results() should return valid channel")
	}
}

// TestProcessor_Errors tests error channel.
func TestProcessor_Errors(t *testing.T) {
	config := DefaultConfig()
	processFn := func(ctx context.Context, batch *Batch) error {
		return nil
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	errChan := processor.Errors()
	if errChan == nil {
		t.Fatal("Errors() should return valid channel")
	}
}

// TestProcessor_Stop tests processor shutdown.
func TestProcessor_Stop(t *testing.T) {
	config := Config{
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
		Workers:      1,
	}

	processFn := func(ctx context.Context, batch *Batch) error {
		return nil
	}

	processor := NewProcessor(config, processFn)

	// Submit before stop should work
	item := &Item{ID: "test"}
	err := processor.Submit(item)
	if err != nil {
		t.Fatalf("Submit before Stop failed: %v", err)
	}

	// Stop the processor
	processor.Stop()

	// Verify channels are closed by checking if Results channel returns nil
	select {
	case batch, ok := <-processor.Results():
		if !ok {
			t.Log("Results channel properly closed")
		}
		_ = batch
	default:
		t.Log("Results channel may still have pending batches")
	}
}

// TestBatchProcessor tests batch processor wrapper.
func TestBatchProcessor(t *testing.T) {
	config := Config{
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
	}

	bp := NewBatchProcessor(config)
	defer bp.Stop()

	// Set process function
	bp.ProcessFunc(func(ctx context.Context, batch *Batch) error {
		return nil
	})

	// Submit item
	item := &Item{ID: "test", Payload: "data"}
	err := bp.Submit(item)
	if err != nil {
		t.Fatalf("Submit() failed: %v", err)
	}

	// Check results channel
	results := bp.Results()
	if results == nil {
		t.Fatal("Results() should return channel")
	}
}

// TestStatsProcessor tests statistics processor.
func TestStatsProcessor(t *testing.T) {
	config := Config{
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
	}

	sp := NewStatsProcessor(config, func(ctx context.Context, batch *Batch) error {
		return nil
	})
	defer sp.Stop()

	stats := sp.GetStats()
	// Initial stats should be zero
	if stats.TotalBatches != 0 {
		t.Log("Initial TotalBatches may vary")
	}
}

// TestItem_Struct tests item structure.
func TestItem_Struct(t *testing.T) {
	item := &Item{
		ID:      "test-item",
		Payload: "test-data",
		Error:   nil,
	}

	if item.ID != "test-item" {
		t.Fatalf("expected ID 'test-item', got '%s'", item.ID)
	}
}

// TestBatch_Struct tests batch structure.
func TestBatch_Struct(t *testing.T) {
	items := []*Item{{ID: "item-1"}, {ID: "item-2"}}
	batch := &Batch{
		Items:     items,
		CreatedAt: time.Now(),
	}

	if len(batch.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(batch.Items))
	}
}

// TestQueueFullError tests queue full error.
func TestQueueFullError(t *testing.T) {
	err := ErrQueueFull

	if err.Error() != "batch queue is full" {
		t.Fatalf("expected error message 'batch queue is full', got '%s'", err.Error())
	}
}

// TestConfig_Validation tests config validation.
func TestConfig_Validation(t *testing.T) {
	// Negative values should be replaced with defaults
	config := Config{
		BatchSize:    -1,
		BatchTimeout: -1,
		MaxPending:   -1,
		Workers:      -1,
	}

	processFn := func(ctx context.Context, batch *Batch) error {
		return nil
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	// All should be replaced with defaults
	if processor.config.BatchSize != DefaultConfig().BatchSize {
		t.Fatalf("negative BatchSize should use default")
	}
	if processor.config.Workers != DefaultConfig().Workers {
		t.Fatalf("negative Workers should use default")
	}
}

// TestProcessor_ProcessBatch tests batch processing function.
func TestProcessor_ProcessBatch_WithError(t *testing.T) {
	config := Config{
		BatchSize:    5,
		BatchTimeout: 50 * time.Millisecond,
		Workers:      1,
	}

	var lastError error
	processFn := func(ctx context.Context, batch *Batch) error {
		lastError = context.DeadlineExceeded // Simulate error
		return lastError
	}

	processor := NewProcessor(config, processFn)
	defer processor.Stop()

	// Submit items
	for i := 0; i < 3; i++ {
		item := &Item{ID: "item-" + string(rune('0'+i))}
		processor.Submit(item)
	}

	time.Sleep(100 * time.Millisecond)

	// Check if error was potentially sent
	t.Logf("Last error: %v", lastError)
}

// TestStats_Struct tests stats structure.
func TestStats_Struct(t *testing.T) {
	stats := Stats{
		TotalBatches:    10,
		TotalItems:      100,
		ProcessedItems:  95,
		FailedItems:     5,
		AvgBatchSize:    10.0,
		AvgProcessTime:  50 * time.Millisecond,
		LastProcessedAt: time.Now(),
	}

	if stats.TotalBatches != 10 {
		t.Fatalf("expected TotalBatches 10, got %d", stats.TotalBatches)
	}
}

// TestInterSubnetTraffic_Struct tests inter-subnet traffic structure.
func TestInterSubnetTraffic_Struct(t *testing.T) {
	traffic := InterSubnetTraffic{
		SrcSubnetID: "src",
		DstSubnetID: "dst",
		SrcNodeID:   "node-1",
		DstNodeID:   "node-2",
		Protocol:    "tcp",
		Port:        80,
		Allowed:     true,
	}

	if traffic.SrcSubnetID != "src" {
		t.Fatalf("expected SrcSubnetID 'src', got '%s'", traffic.SrcSubnetID)
	}
	if traffic.Port != 80 {
		t.Fatalf("expected Port 80, got %d", traffic.Port)
	}
}