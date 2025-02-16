/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pusher_test

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openGemini/openGemini/lib/fileops"
	"github.com/openGemini/openGemini/lib/statisticsPusher/pusher"
	"github.com/stretchr/testify/assert"
)

var testPushData []string

func initTestData() {
	tmp := `executor,app=ts-store,hostname=127.0.0.1:8401 column_length_count=1,column_width_last=0,source_length_sum=0,sink_rows_sum=0,trans_abort_last=0,memory_sum=0,goroutine_last=0,exec_wait_time_sum=0,exec_run_time_count=0,exec_abort_sum=0,trans_error_sum=0,trans_abort_count=0,column_length_sum=0,sink_length_sum=0,sink_length_count=0,limit_rows_last=0,fill_rows_sum=0,fill_rows_last=0,trans_error_last=0,trans_abort_sum=0,trans_error_abort_count=0,source_width_count=0,fill_rows_count=0,sink_rows_count=0,sink_rows_last=0,dag_edge_last=0,exec_timeout_last=0,exec_abort_count=0,exec_abort_last=0,source_length_last=0,limit_rows_count=0,trans_wait_time_last=0,trans_error_count=0,trans_error_abort_last=0,source_rows_last=0,filter_rows_count=0,memory_last=0,exec_wait_time_count=0,dag_vertex_count=0,source_length_count=0,source_width_last=0,sink_length_last=0,trans_wait_time_count=0,exec_scheduled_last=0,exec_timeout_sum=0,column_width_count=0,source_rows_sum=0,agg_rows_count=0,materialized_rows_last=0,dag_vertex_sum=0,dag_vertex_last=0,limit_rows_sum=0,exec_error_sum=0,memory_count=0,exec_wait_time_last=0,exec_run_time_last=0,trans_wait_time_sum=0,trans_run_time_sum=0,exec_scheduled_count=0,exec_timeout_count=0,sink_width_last=0,agg_rows_sum=0,merge_rows_sum=0,source_width_sum=0,dag_edge_sum=0,exec_scheduled_sum=0,exec_error_last=0,agg_rows_last=0,materialized_rows_count=0,goroutine_count=0,trans_run_time_last=0,exec_error_count=0,filter_rows_last=0,merge_rows_count=0,materialized_rows_sum=0,goroutine_sum=0,exec_run_time_sum=0,trans_run_time_count=0,dag_edge_count=0,trans_error_abort_sum=0,column_length_last=0,sink_width_sum=0,column_width_sum=0,sink_width_count=0,source_rows_count=0,filter_rows_sum=0,merge_rows_last=0 1652342817034026796
executor,app=ts-store,hostname=21.138.23.131:8402 column_length_count=2,column_width_last=0,source_length_sum=0,sink_rows_sum=0,trans_abort_last=0,memory_sum=0,goroutine_last=0,exec_wait_time_sum=0,exec_run_time_count=0,exec_abort_sum=0,trans_error_sum=0,trans_abort_count=0,column_length_sum=0,sink_length_sum=0,sink_length_count=0,limit_rows_last=0,fill_rows_sum=0,fill_rows_last=0,trans_error_last=0,trans_abort_sum=0,trans_error_abort_count=0,source_width_count=0,fill_rows_count=0,sink_rows_count=0,sink_rows_last=0,dag_edge_last=0,exec_timeout_last=0,exec_abort_count=0,exec_abort_last=0,source_length_last=0,limit_rows_count=0,trans_wait_time_last=0,trans_error_count=0,trans_error_abort_last=0,source_rows_last=0,filter_rows_count=0,memory_last=0,exec_wait_time_count=0,dag_vertex_count=0,source_length_count=0,source_width_last=0,sink_length_last=0,trans_wait_time_count=0,exec_scheduled_last=0,exec_timeout_sum=0,column_width_count=0,source_rows_sum=0,agg_rows_count=0,materialized_rows_last=0,dag_vertex_sum=0,dag_vertex_last=0,limit_rows_sum=0,exec_error_sum=0,memory_count=0,exec_wait_time_last=0,exec_run_time_last=0,trans_wait_time_sum=0,trans_run_time_sum=0,exec_scheduled_count=0,exec_timeout_count=0,sink_width_last=0,agg_rows_sum=0,merge_rows_sum=0,source_width_sum=0,dag_edge_sum=0,exec_scheduled_sum=0,exec_error_last=0,agg_rows_last=0,materialized_rows_count=0,goroutine_count=0,trans_run_time_last=0,exec_error_count=0,filter_rows_last=0,merge_rows_count=0,materialized_rows_sum=0,goroutine_sum=0,exec_run_time_sum=0,trans_run_time_count=0,dag_edge_count=0,trans_error_abort_sum=0,column_length_last=0,sink_width_sum=0,column_width_sum=0,sink_width_count=0,source_rows_count=0,filter_rows_sum=0,merge_rows_last=0 1652342817034026796
executor,app=ts-store,hostname=21.138.23.131:8403 column_length_count=3,column_width_last=0,source_length_sum=0,sink_rows_sum=0,trans_abort_last=0,memory_sum=0,goroutine_last=0,exec_wait_time_sum=0,exec_run_time_count=0,exec_abort_sum=0,trans_error_sum=0,trans_abort_count=0,column_length_sum=0,sink_length_sum=0,sink_length_count=0,limit_rows_last=0,fill_rows_sum=0,fill_rows_last=0,trans_error_last=0,trans_abort_sum=0,trans_error_abort_count=0,source_width_count=0,fill_rows_count=0,sink_rows_count=0,sink_rows_last=0,dag_edge_last=0,exec_timeout_last=0,exec_abort_count=0,exec_abort_last=0,source_length_last=0,limit_rows_count=0,trans_wait_time_last=0,trans_error_count=0,trans_error_abort_last=0,source_rows_last=0,filter_rows_count=0,memory_last=0,exec_wait_time_count=0,dag_vertex_count=0,source_length_count=0,source_width_last=0,sink_length_last=0,trans_wait_time_count=0,exec_scheduled_last=0,exec_timeout_sum=0,column_width_count=0,source_rows_sum=0,agg_rows_count=0,materialized_rows_last=0,dag_vertex_sum=0,dag_vertex_last=0,limit_rows_sum=0,exec_error_sum=0,memory_count=0,exec_wait_time_last=0,exec_run_time_last=0,trans_wait_time_sum=0,trans_run_time_sum=0,exec_scheduled_count=0,exec_timeout_count=0,sink_width_last=0,agg_rows_sum=0,merge_rows_sum=0,source_width_sum=0,dag_edge_sum=0,exec_scheduled_sum=0,exec_error_last=0,agg_rows_last=0,materialized_rows_count=0,goroutine_count=0,trans_run_time_last=0,exec_error_count=0,filter_rows_last=0,merge_rows_count=0,materialized_rows_sum=0,goroutine_sum=0,exec_run_time_sum=0,trans_run_time_count=0,dag_edge_count=0,trans_error_abort_sum=0,column_length_last=0,sink_width_sum=0,column_width_sum=0,sink_width_count=0,source_rows_count=0,filter_rows_sum=0,merge_rows_last=0 1652342817034026796`

	testPushData = strings.Split(tmp, "\n")
}

func TestSnappy(t *testing.T) {
	initTestData()

	file := os.TempDir() + "/stat_file_push.data"
	writeTestData(t, file, 0)

	r := pusher.NewSnappyReader()
	if !assert.NoError(t, r.OpenFile(file)) {
		return
	}
	defer r.Close()

	idx := 0
	for {
		line, err := r.ReadBlock()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, testPushData[idx], string(line))
		idx++
	}

	stat, err := r.Stat()
	assert.NoError(t, err)
	assert.Equal(t, stat.Size(), r.Location())
	assert.NoError(t, r.SeekStart(r.Location()))
}

func TestSnappyReader_Tail(t *testing.T) {
	initTestData()
	file := os.TempDir() + "/stat_file_tail.data"
	go func() {
		writeTestData(t, file, time.Second)
	}()
	time.Sleep(time.Second / 10)

	tail := pusher.NewSnappyTail(0, false)
	n := 0
	err := tail.Tail(file, func(block []byte) {
		fmt.Println(testPushData[n])
		assert.Equal(t, testPushData[n], string(block))
		n++
	})
	if err != io.EOF {
		assert.NoError(t, err)
	}
	assert.Equal(t, len(testPushData), n)

	tail.Close()
}

func writeTestData(t *testing.T, file string, interval time.Duration) {
	_ = fileops.Remove(file)

	w := pusher.NewSnappyWriter()
	if !assert.NoError(t, w.OpenFile(file)) {
		return
	}

	for i := range testPushData {
		if interval > 0 {
			time.Sleep(interval)
		}
		err := w.WriteBlock([]byte(testPushData[i]))
		if !assert.NoError(t, err) {
			return
		}
	}

	assert.NoError(t, w.WriteBlock(nil))

	if !assert.NoError(t, w.Close()) {
		return
	}
}

func TestSnappy_ReadEOF(t *testing.T) {
	initTestData()

	file := t.TempDir() + "/stat_file_push_eof.data"
	writer := pusher.NewSnappyWriter()
	if !assert.NoError(t, writer.OpenFile(file)) {
		return
	}

	_, err := writer.File().Write([]byte{0, 0, 0, 10})
	if !assert.NoError(t, err) {
		return
	}

	r := pusher.NewSnappyReader()
	if !assert.NoError(t, r.OpenFile(file)) {
		return
	}
	defer r.Close()

	line, err := r.ReadBlock()
	assert.NoError(t, err)
	assert.Empty(t, line)

	data := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	_, err = writer.File().Write(data)
	if !assert.NoError(t, err) {
		return
	}

	line, err = r.ReadBlock()
	assert.NoError(t, err)
	assert.Equal(t, data, line)
}
