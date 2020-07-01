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
	"runtime"

	"github.com/shirou/gopsutil/process"
)

type memProfileStats struct {
	totalAllocSize, totalAllocObject int
	totalFreedSize, totalFreedObject int
	totalInuseSize, totalInuseObject int
}

// = rss / max memory
func getMemUsage() (float64, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, err
	}

	fmt.Println(p.MemoryPercent())

	percent, err := p.MemoryPercent()
	return float64(percent), err
}

func getMemProfileStats() (memProfileStats, error) {
	var (
		totalAllocSize, totalAllocObject = 0, 0
		totalFreedSize, totalFreedObject = 0, 0
		totalInuseSize, totalInuseObject = 0, 0
	)

	profiles := make([]runtime.MemProfileRecord, 2)
	n, ok := runtime.MemProfile(profiles, false)
	if ok {
		profiles = profiles[0:n]
	} else {
		// TODO
		return memProfileStats{}, nil
	}

	for _, profile := range profiles {
		totalAllocSize += int(profile.AllocBytes)
		totalFreedSize += int(profile.FreeBytes)
		totalInuseSize += int(profile.InUseBytes())

		totalAllocObject += int(profile.AllocObjects)
		totalFreedObject += int(profile.FreeObjects)
		totalInuseObject += int(profile.InUseObjects())
	}

	return memProfileStats{
		totalAllocSize:   totalAllocSize,
		totalAllocObject: totalAllocObject,
		totalFreedSize:   totalFreedSize,
		totalFreedObject: totalFreedObject,
		totalInuseSize:   totalInuseSize,
		totalInuseObject: totalInuseObject,
	}, nil
}
