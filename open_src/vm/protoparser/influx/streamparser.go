package influx

/*
Copyright 2019-2021 VictoriaMetrics, Inc.
This code is originally from: https://github.com/VictoriaMetrics/VictoriaMetrics/tree/v1.67.0/lib/protoparser/influx/streamparser.go

2022.01.23 Add error code to influx client error etc.
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.
*/

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/cgroup"
	"github.com/openGemini/openGemini/lib/statisticsPusher/statistics"
)

// The maximum size of a single line returned by ReadLinesBlock.
const maxLineSize = 256 * 1024

// Default size in bytes of a single block returned by ReadLinesBlock.
const defaultBlockSize = 64 * 1024

func ReadLinesBlockExt(r io.Reader, dstBuf, tailBuf []byte, maxLineLen, blockSize int) ([]byte, []byte, error) {
	startTime := time.Now()
	if cap(dstBuf) < blockSize {
		dstBuf = bytesutil.Resize(dstBuf, blockSize)
	}

	dstBuf = append(dstBuf[:0], tailBuf...)
	tailBuf = tailBuf[:0]
	originLen := len(dstBuf)
again:
	for {
		n, err := r.Read(dstBuf[len(dstBuf):cap(dstBuf)])
		// Check for error only if zero bytes read from r, i.e. no forward progress made.
		// Otherwise process the read data.
		if n == 0 {
			if err == nil {
				return dstBuf, tailBuf, fmt.Errorf("no forward progress made")
			}
			if err == io.EOF && len(dstBuf) > 0 {
				// Missing newline in the end of stream. This is OK,
				// so suppress io.EOF for now. It will be returned during the next
				// call to ReadLinesBlock.
				// This fixes https://github.com/VictoriaMetrics/VictoriaMetrics/issues/60 .
				return dstBuf, tailBuf, nil
			}
			if err != io.EOF {
				err = fmt.Errorf("cannot read a block of data in %.3fs: %w", time.Since(startTime).Seconds(), err)
			}
			return dstBuf, tailBuf, err
		}
		dstBuf = dstBuf[:len(dstBuf)+n]
		if len(dstBuf) == cap(dstBuf) {
			break
		}
	}

	// Search for the last newline in dstBuf and put the rest into tailBuf.
	nn := bytes.LastIndexByte(dstBuf[originLen:], '\n')
	if nn < 0 {
		// Didn't found at least a single line.
		if len(dstBuf) > maxLineLen {
			return dstBuf, tailBuf, fmt.Errorf("too long line: more than %d bytes", maxLineLen)
		}
		if cap(dstBuf) < 2*len(dstBuf) {
			// Increase dsbBuf capacity, so more data could be read into it.
			dstBufLen := len(dstBuf)
			dstBuf = bytesutil.Resize(dstBuf, 2*cap(dstBuf))
			dstBuf = dstBuf[:dstBufLen]
		}
		goto again
	}
	// Found at least a single line. Return it.
	nn += originLen
	tailBuf = append(tailBuf[:0], dstBuf[nn+1:]...)
	dstBuf = dstBuf[:nn]

	return dstBuf, tailBuf, nil
}

func (ctx *streamContext) Read(blockSize int) bool {
	if ctx.err != nil {
		return false
	}
	ctx.ReqBuf, ctx.tailBuf, ctx.err = ReadLinesBlockExt(ctx.br, ctx.ReqBuf, ctx.tailBuf, maxLineSize, blockSize)
	if ctx.err != nil {
		if ctx.err != io.EOF {
			ctx.err = fmt.Errorf("cannot read influx line protocol data: %w", ctx.err)
		}
		return false
	}
	return true
}

type streamContext struct {
	br      *bufio.Reader
	ReqBuf  []byte
	tailBuf []byte
	err     error

	Wg              sync.WaitGroup
	CallbackErrLock sync.Mutex
	CallbackErr     error
}

func (ctx *streamContext) Error() error {
	if ctx.err == io.EOF {
		return nil
	}
	return ctx.err
}

func (ctx *streamContext) reset() {
	ctx.br.Reset(nil)
	ctx.ReqBuf = ctx.ReqBuf[:0]
	ctx.tailBuf = ctx.tailBuf[:0]
	ctx.err = nil
	ctx.CallbackErr = nil
}

func GetStreamContext(r io.Reader) *streamContext {
	select {
	case ctx := <-streamContextPoolCh:
		ctx.br.Reset(r)
		return ctx
	default:
		if v := streamContextPool.Get(); v != nil {
			ctx := v.(*streamContext)
			ctx.br.Reset(r)
			return ctx
		}
		return &streamContext{
			br: bufio.NewReaderSize(r, 64*1024),
		}
	}
}

func PutStreamContext(ctx *streamContext) {
	ctx.reset()
	select {
	case streamContextPoolCh <- ctx:
	default:
		streamContextPool.Put(ctx)
	}
}

var streamContextPool sync.Pool
var streamContextPoolCh = make(chan *streamContext, cgroup.AvailableCPUs())

type unmarshalWork struct {
	rows         PointRows
	Callback     func(db string, rows []Row, err error)
	Db           string
	TsMultiplier int64
	ReqBuf       []byte
}

func (uw *unmarshalWork) reset() {
	uw.rows.Reset()
	uw.Callback = nil
	uw.Db = ""
	uw.TsMultiplier = 0
	uw.ReqBuf = uw.ReqBuf[:0]
}

// Unmarshal implements common.UnmarshalWork
func (uw *unmarshalWork) Unmarshal() {
	start := time.Now()
	err := uw.rows.Unmarshal(bytesutil.ToUnsafeString(uw.ReqBuf))
	rows := uw.rows.Rows
	if err != nil {
		uw.Callback(uw.Db, rows, err)
		putUnmarshalWork(uw)
		return
	}
	atomic.AddInt64(&statistics.HandlerStat.WriteRequestParseDuration, time.Since(start).Nanoseconds())
	currentTs := time.Now().UnixNano()
	tsMultiplier := uw.TsMultiplier
	if tsMultiplier >= 1 {
		for i := range rows {
			row := &rows[i]
			err = row.CheckValid()
			if err != nil {
				break
			}
			if row.Timestamp == NoTimestamp {
				row.Timestamp = currentTs
			} else {
				row.Timestamp *= tsMultiplier
			}
		}
	} else if tsMultiplier < 0 {
		tsMultiplier = -tsMultiplier
		currentTs -= currentTs % tsMultiplier
		for i := range rows {
			row := &rows[i]
			err = row.CheckValid()
			if err != nil {
				break
			}
			if row.Timestamp == NoTimestamp {
				row.Timestamp = currentTs
			} else {
				row.Timestamp *= tsMultiplier
			}
		}
	}

	uw.Callback(uw.Db, rows, err)
	putUnmarshalWork(uw)
}

func GetUnmarshalWork() *unmarshalWork {
	v := unmarshalWorkPool.Get()
	if v == nil {
		return &unmarshalWork{}
	}
	return v.(*unmarshalWork)
}

func putUnmarshalWork(uw *unmarshalWork) {
	uw.reset()
	unmarshalWorkPool.Put(uw)
}

var unmarshalWorkPool sync.Pool

// ScheduleUnmarshalWork schedules uw to run in the worker pool.
//
// It is expected that StartUnmarshalWorkers is already called.
func ScheduleUnmarshalWork(uw UnmarshalWork) {
	unmarshalWorkCh <- uw
}

// UnmarshalWork is a unit of unmarshal work.
type UnmarshalWork interface {
	// Unmarshal must implement CPU-bound unmarshal work.
	Unmarshal()
}

// StartUnmarshalWorkers starts unmarshal workers.
func StartUnmarshalWorkers() {
	if unmarshalWorkCh != nil {
		panic("BUG: it looks like startUnmarshalWorkers() has been alread called without stopUnmarshalWorkers()")
	}
	gomaxprocs := cgroup.AvailableCPUs()
	unmarshalWorkCh = make(chan UnmarshalWork, 2*gomaxprocs)
	unmarshalWorkersWG.Add(gomaxprocs)
	for i := 0; i < gomaxprocs; i++ {
		go func() {
			defer unmarshalWorkersWG.Done()
			for uw := range unmarshalWorkCh {
				uw.Unmarshal()
			}
		}()
	}
}

// StopUnmarshalWorkers stops unmarshal workers.
//
// No more calles to ScheduleUnmarshalWork are allowed after callsing stopUnmarshalWorkers
func StopUnmarshalWorkers() {
	close(unmarshalWorkCh)
	unmarshalWorkersWG.Wait()
	unmarshalWorkCh = nil
}

var (
	unmarshalWorkCh    chan UnmarshalWork
	unmarshalWorkersWG sync.WaitGroup
)
