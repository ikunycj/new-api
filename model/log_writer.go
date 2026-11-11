package model

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// Relay request logs (consume and error) are written by a background writer so
// the INSERT is not on the request path. Under database pressure a synchronous
// INSERT used to hold every finished stream open until the row was committed:
// in production a 300 RPM run saw consume-log inserts take 65-113s, which kept
// clients waiting long after the upstream finished and eventually got the
// connections cut.
//
// The writer batches rows and uses a fixed number of workers, so log writes use
// at most requestLogWorkers database connections no matter how many
// requests finish at once, leaving the rest of the pool for auth and billing.
//
// The writer is only used while common.RequestLogAsyncEnabled is on (option
// "RequestLogAsyncEnabled", off by default). Turning it off at runtime is the
// kill switch: new rows go back to synchronous INSERTs at once, and rows already
// queued are still flushed by the workers. Until StartRequestLogWriter is called
// (tests, one-off tools) writes also stay synchronous.
const (
	requestLogQueueSize     = 20000
	requestLogBatchSize     = 200
	requestLogFlushInterval = 200 * time.Millisecond
	requestLogWorkers       = 4
)

type requestLogWriter struct {
	ch   chan *Log
	done chan struct{}
	wg   sync.WaitGroup
}

var (
	requestLogWriterMu     sync.RWMutex
	activeRequestLogWriter *requestLogWriter
)

// StartRequestLogWriter starts the background writer for relay request logs.
func StartRequestLogWriter() {
	requestLogWriterMu.Lock()
	defer requestLogWriterMu.Unlock()
	if activeRequestLogWriter != nil {
		return
	}
	w := &requestLogWriter{
		ch:   make(chan *Log, requestLogQueueSize),
		done: make(chan struct{}),
	}
	for i := 0; i < requestLogWorkers; i++ {
		w.wg.Add(1)
		go w.run()
	}
	activeRequestLogWriter = w
}

// StopRequestLogWriter drains queued logs to the database and stops the writer.
// Called during graceful shutdown so accepted rows are not lost.
func StopRequestLogWriter() {
	requestLogWriterMu.Lock()
	w := activeRequestLogWriter
	activeRequestLogWriter = nil
	requestLogWriterMu.Unlock()
	if w == nil {
		return
	}
	close(w.done)
	w.wg.Wait()
}

// submitRequestLog queues the log for the background writer. When async logging
// is switched off, the writer is not running or the queue is full it writes
// synchronously instead: a full queue means the database is far behind, and
// slowing the caller is preferable to dropping billing records.
func submitRequestLog(log *Log) error {
	ensureLogRequestId(log)
	if !common.RequestLogAsyncEnabled {
		return LOG_DB.Create(log).Error
	}
	requestLogWriterMu.RLock()
	w := activeRequestLogWriter
	requestLogWriterMu.RUnlock()
	if w != nil {
		select {
		case w.ch <- log:
			return nil
		default:
		}
	}
	return LOG_DB.Create(log).Error
}

func (w *requestLogWriter) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(requestLogFlushInterval)
	defer ticker.Stop()

	batch := make([]*Log, 0, requestLogBatchSize)
	for {
		select {
		case log := <-w.ch:
			batch = append(batch, log)
			if len(batch) >= requestLogBatchSize {
				batch = writeRequestLogBatch(batch)
			}
		case <-ticker.C:
			batch = writeRequestLogBatch(batch)
		case <-w.done:
			for {
				select {
				case log := <-w.ch:
					batch = append(batch, log)
					if len(batch) >= requestLogBatchSize {
						batch = writeRequestLogBatch(batch)
					}
				default:
					writeRequestLogBatch(batch)
					return
				}
			}
		}
	}
}

// writeRequestLogBatch inserts the batch and returns it emptied for reuse. A
// failed batch is retried row by row so one bad row does not lose the others.
func writeRequestLogBatch(batch []*Log) []*Log {
	if len(batch) == 0 {
		return batch
	}
	if err := LOG_DB.CreateInBatches(batch, requestLogBatchSize).Error; err != nil {
		common.SysError("failed to batch insert request logs, retrying one by one: " + err.Error())
		for _, log := range batch {
			if err := LOG_DB.Create(log).Error; err != nil {
				common.SysError("failed to record request log " + log.RequestId + ": " + err.Error())
			}
		}
	}
	return batch[:0]
}
