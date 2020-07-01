/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package autodump

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"
)

var (
	dumpPath                 = "./"
	maxDumpTimesPerDay       = 10
	cpuDumpingFlag     int64 = 0
)

func init() {
	Init("", 5, 0)
}

// Init the dumper, set dump path and dump tick interval
func Init(path string, interval int, maxDumpTimes int) {
	if len(path) > 0 {
		dumpPath = path
	}

	if maxDumpTimes > 0 {
		maxDumpTimesPerDay = maxDumpTimes
	}

	go startDumpLoop()
}

var threshold struct {
	cpuDumpInterval       time.Duration
	memDumpInterval       time.Duration
	goroutineDumpInterval time.Duration
}

var stats struct {
	latestCPUDumpTime time.Time
	latestCPUUsage    int
	cpuUsageLRU       []int

	latestMemDumpTime time.Time
	latestMemUsage    int
	memUsageLRU       []int

	latestGoroutineDumpTime time.Time
	latestGoroutineNum      int
	goroutineNumLRU         []int
}

func startDumpLoop() {
	lastDumpTime := time.Now()
	dumpTimes := 0
	for range time.Tick(time.Second * 5) {
		if atomic.LoadInt64(&cpuDumpingFlag) == 1 {
			// is cpu dumping now, skip this cycle
			continue
		}

		// last trigger time is the same day
		if time.Now().Format("20060102") != lastDumpTime.Format("20060102") {
			dumpTimes = 0
		} else if dumpTimes > maxDumpTimesPerDay {
			continue
		}

		if triggered := memProfile(); triggered {
			dumpTimes++
			lastDumpTime = time.Now()
			println("dump mem")
		}

		if triggered := cpuProfile(); triggered {
			dumpTimes++
			lastDumpTime = time.Now()
			println("dump cpu")
		}

		if triggered := goroutineProfile(); triggered {
			dumpTimes++
			lastDumpTime = time.Now()
			println("dump g")
		}
	}
}

func trim1elemIfMoreThan10(arr []int) []int {
	if len(arr) <= 10 {
		return arr
	}

	return arr[1:]
}

func memProfile() bool {
	curMemUsage := 0
	stats.memUsageLRU = append(stats.memUsageLRU, curMemUsage)
	stats.memUsageLRU = trim1elemIfMoreThan10(stats.memUsageLRU)

	if time.Since(stats.latestMemDumpTime) < threshold.memDumpInterval {
		return false
	}

	mStats, _ := getMemProfileStats()
	fmt.Println("memory stats", mStats)

	fileName := fmt.Sprintf("%v.dump_v%v", path.Join(dumpPath, "heap"), time.Now().Format("20060102150405"))
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer f.Close()

	pprof.Lookup("heap").WriteTo(f, 1)
	return true
}

func cpuProfile() bool {
	curCPUUsage, err := getCPUUsage()
	if err != nil {
		// log error
		return false
	}

	stats.cpuUsageLRU = append(stats.cpuUsageLRU, int(curCPUUsage))
	stats.cpuUsageLRU = trim1elemIfMoreThan10(stats.cpuUsageLRU)

	if time.Since(stats.latestCPUDumpTime) < threshold.cpuDumpInterval {
		return false
	}

	// TODO, if the cpu usage matches the rule
	fileName := fmt.Sprintf("%v.dump_%v", path.Join(dumpPath, "cpu"), time.Now().Format("20060102150405"))
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return false
	}
	defer f.Close()

	err = pprof.StartCPUProfile(f)
	if err != nil {
		return false
	}
	atomic.StoreInt64(&cpuDumpingFlag, 1)
	time.AfterFunc(time.Second*10, func() {
		pprof.StopCPUProfile()
		atomic.StoreInt64(&cpuDumpingFlag, 0)
	})

	return true
}

func goroutineProfile() bool {
	goroutineNum := runtime.NumGoroutine()
	stats.goroutineNumLRU = append(stats.goroutineNumLRU, goroutineNum)
	// trim to len == 10
	if len(stats.goroutineNumLRU) > 10 {
		stats.goroutineNumLRU = stats.goroutineNumLRU[1:]
	}

	if time.Since(stats.latestGoroutineDumpTime) < threshold.goroutineDumpInterval {
		return false
	}

	sum := 0
	for _, n := range stats.goroutineNumLRU {
		sum += n
	}
	avg := sum / len(stats.goroutineNumLRU)

	// if current goroutine num reaches 126% of the previous average num
	if float64(goroutineNum) > float64(avg)*1.25 {
		fileName := fmt.Sprintf("%v.dump_%v", path.Join(dumpPath, "goroutine"), time.Now().Format("20060102150405"))
		f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println(err)
			return false
		}
		defer f.Close()

		pprof.Lookup("goroutine").WriteTo(f, 1)
		return true
	}

	return false
}
