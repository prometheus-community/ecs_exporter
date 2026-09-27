// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	markerPath        = "/tmp/fixture-workload-complete"
	cachePath         = "/tmp/fixture-cache"
	sharedMemoryPath  = "/dev/shm/fixture-shmem"
	cacheSize         = 256 << 20
	sharedMemorySize  = 8 << 20
	lazyFreeSize      = 8 << 20
	lockedThreadCount = 32
	cacheReadPasses   = 3
	madvFree          = 8
	madvCollapse      = 25
)

var (
	retainedMappings       [][]byte
	retainedHugePageMemory []byte
	retainedFiles          []*os.File
	retainedConns          []net.Conn
	retainedListener       net.Listener
	lockedThreads          = make(chan struct{})
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "healthcheck":
			if _, err := os.Stat(markerPath); err != nil {
				os.Exit(1)
			}
			return
		case "stats":
			if err := printMemoryStats(); err != nil {
				log.Fatal(err)
			}
			return
		}
	}
	if len(os.Args) != 1 {
		log.Fatalf("usage: %s [healthcheck|stats]", os.Args[0])
	}
	if err := runWorkload(); err != nil {
		recordFailure(err)
	} else {
		for {
			time.Sleep(time.Second)
			if err := ensureHugePages(retainedHugePageMemory); err != nil {
				recordFailure(err)
				break
			}
		}
	}
	for {
		time.Sleep(time.Hour)
	}
}

func printMemoryStats() error {
	paths := []string{
		"/sys/fs/cgroup/memory.stat",
		"/sys/fs/cgroup/memory.events",
		"/sys/fs/cgroup/memory/memory.stat",
		"/sys/fs/cgroup/memory/memory.failcnt",
	}
	found := false
	for _, path := range paths {
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		found = true
		fmt.Printf("==> %s <==\n", path)
		if _, err := io.Copy(os.Stdout, file); err != nil {
			file.Close()
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close %s: %w", path, err)
		}
	}
	if !found {
		return errors.New("no memory cgroup statistics found")
	}
	return nil
}

func recordFailure(err error) {
	if removeErr := os.Remove(markerPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		log.Fatalf("remove health marker after workload failure %q: %v", err, removeErr)
	}
	log.Print(err)
}

func runWorkload() error {
	if err := os.MkdirAll("/tmp", 0o755); err != nil {
		return fmt.Errorf("create /tmp: %w", err)
	}
	if err := os.Remove(markerPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale health marker: %w", err)
	}

	// Request huge pages before creating memory pressure so this fixture tests
	// THP accounting rather than the host's ability to compact fragmented RAM.
	if err := retainHugePages(); err != nil {
		return err
	}
	if err := retainLazyFreeMemory(); err != nil {
		return err
	}
	if err := churnFileCache(); err != nil {
		return err
	}
	if err := retainSharedMemory(); err != nil {
		return err
	}
	if err := retainSocketBuffers(); err != nil {
		return err
	}
	retainLockedThreads()
	// Cache churn can split a THP while reclaiming memory from this cgroup.
	// Restore and verify it after all pressure-producing work has finished so
	// the task stats endpoint observes a live anonymous huge page.
	if err := ensureHugePages(retainedHugePageMemory); err != nil {
		return err
	}

	if err := os.WriteFile(markerPath, []byte("ready\n"), 0o644); err != nil {
		return fmt.Errorf("write health marker: %w", err)
	}
	log.Print("fixture workload ready")
	return nil
}

func retainHugePages() error {
	hugePageSize, err := transparentHugePageSize()
	if err != nil {
		return err
	}
	// Over-allocate so that an aligned region remains available regardless of
	// the address selected by mmap.
	memory, err := anonymousMapping(hugePageSize * 2)
	if err != nil {
		return fmt.Errorf("map huge-page memory: %w", err)
	}
	address := uintptr(unsafe.Pointer(&memory[0]))
	hugePageAddress := uintptr(hugePageSize)
	offset := int((hugePageAddress - address%hugePageAddress) % hugePageAddress)
	aligned := memory[offset : offset+hugePageSize]
	if err := syscall.Madvise(aligned, syscall.MADV_HUGEPAGE); err != nil {
		return fmt.Errorf("mark memory MADV_HUGEPAGE: %w", err)
	}
	touchPages(aligned)
	if err := ensureHugePages(aligned); err != nil {
		return err
	}
	retainedMappings = append(retainedMappings, memory)
	retainedHugePageMemory = aligned
	return nil
}

func transparentHugePageSize() (int, error) {
	const path = "/sys/kernel/mm/transparent_hugepage/hpage_pmd_size"
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read transparent huge-page size: %w", err)
	}
	size, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	if err != nil || size <= 0 {
		return 0, fmt.Errorf("parse transparent huge-page size %q", strings.TrimSpace(string(contents)))
	}
	return size, nil
}

func anonymousMapping(size int) ([]byte, error) {
	mapping, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

func touchPages(memory []byte) {
	for offset := 0; offset < len(memory); offset += os.Getpagesize() {
		memory[offset] = 1
	}
}

func retainLazyFreeMemory() error {
	memory, err := anonymousMapping(lazyFreeSize)
	if err != nil {
		return fmt.Errorf("map lazy-free memory: %w", err)
	}
	touchPages(memory)
	if err := syscall.Madvise(memory, madvFree); err != nil {
		return fmt.Errorf("mark memory MADV_FREE: %w", err)
	}
	retainedMappings = append(retainedMappings, memory)
	return nil
}

func churnFileCache() error {
	file, err := os.OpenFile(cachePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}
	buffer := make([]byte, 1<<20)
	for written := 0; written < cacheSize; written += len(buffer) {
		if _, err := file.Write(buffer); err != nil {
			return fmt.Errorf("populate cache file: %w", err)
		}
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync cache file: %w", err)
	}
	for pass := 0; pass < cacheReadPasses; pass++ {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("rewind cache file: %w", err)
		}
		if _, err := io.CopyBuffer(io.Discard, file, buffer); err != nil {
			return fmt.Errorf("read cache file: %w", err)
		}
	}
	retainedFiles = append(retainedFiles, file)
	return nil
}

func retainSharedMemory() error {
	file, err := os.OpenFile(sharedMemoryPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("create shared-memory file: %w", err)
	}
	if err := file.Truncate(sharedMemorySize); err != nil {
		return fmt.Errorf("size shared-memory file: %w", err)
	}
	memory, err := syscall.Mmap(int(file.Fd()), 0, sharedMemorySize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("map shared-memory file: %w", err)
	}
	touchPages(memory)
	if err := syscall.Mlock(memory[:os.Getpagesize()]); err != nil {
		return fmt.Errorf("lock shared memory: %w", err)
	}
	retainedFiles = append(retainedFiles, file)
	retainedMappings = append(retainedMappings, memory)
	return nil
}

func retainSocketBuffers() error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback socket: %w", err)
	}
	type acceptResult struct {
		connection net.Conn
		err        error
	}
	accepted := make(chan acceptResult, 1)
	go func() {
		connection, err := listener.Accept()
		accepted <- acceptResult{connection: connection, err: err}
	}()
	client, err := net.Dial("tcp4", listener.Addr().String())
	if err != nil {
		return fmt.Errorf("connect loopback socket: %w", err)
	}
	result := <-accepted
	if result.err != nil {
		return fmt.Errorf("accept loopback socket: %w", result.err)
	}
	if tcp, ok := client.(*net.TCPConn); ok {
		if err := tcp.SetWriteBuffer(4 << 20); err != nil {
			return fmt.Errorf("size socket write buffer: %w", err)
		}
	}
	if tcp, ok := result.connection.(*net.TCPConn); ok {
		if err := tcp.SetReadBuffer(4 << 20); err != nil {
			return fmt.Errorf("size socket read buffer: %w", err)
		}
	}
	if err := client.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		return fmt.Errorf("set socket write deadline: %w", err)
	}
	payload := make([]byte, 64<<10)
	written := 0
	for {
		count, err := client.Write(payload)
		written += count
		if err != nil {
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				break
			}
			return fmt.Errorf("fill socket buffers: %w", err)
		}
	}
	if written == 0 {
		return errors.New("socket buffers accepted no data")
	}
	retainedListener = listener
	retainedConns = append(retainedConns, client, result.connection)
	log.Printf("retained %d bytes in socket buffers", written)
	return nil
}

func retainLockedThreads() {
	ready := make(chan struct{}, lockedThreadCount)
	for range lockedThreadCount {
		go func() {
			runtime.LockOSThread()
			ready <- struct{}{}
			<-lockedThreads
		}()
	}
	for range lockedThreadCount {
		<-ready
	}
}

func ensureHugePages(memory []byte) error {
	address := uintptr(unsafe.Pointer(&memory[0]))
	hugeBytes, err := anonymousHugePageBytes(address)
	if err != nil {
		return err
	}
	if hugeBytes < len(memory) {
		// Re-fault the advised mapping first. This works on kernels whose THP
		// policy is "madvise", including ones that reject MADV_COLLAPSE.
		if err := syscall.Madvise(memory, syscall.MADV_DONTNEED); err != nil {
			return fmt.Errorf("discard split huge-page mapping: %w", err)
		}
		touchPages(memory)
		hugeBytes, err = anonymousHugePageBytes(address)
		if err != nil {
			return err
		}
	}
	if hugeBytes < len(memory) {
		if err := syscall.Madvise(memory, madvCollapse); err != nil {
			return fmt.Errorf("collapse transparent huge pages: %w", err)
		}
		hugeBytes, err = anonymousHugePageBytes(address)
		if err != nil {
			return err
		}
	}
	if hugeBytes < len(memory) {
		return fmt.Errorf("transparent huge-page mapping has %d bytes, want at least %d", hugeBytes, len(memory))
	}
	return nil
}

func anonymousHugePageBytes(address uintptr) (int, error) {
	file, err := os.Open("/proc/self/smaps")
	if err != nil {
		return 0, fmt.Errorf("open process memory mappings: %w", err)
	}
	defer file.Close()

	inMapping := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var start, end uintptr
		if _, err := fmt.Sscanf(line, "%x-%x", &start, &end); err == nil {
			inMapping = address >= start && address < end
			continue
		}
		if !inMapping || !strings.HasPrefix(line, "AnonHugePages:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[2] != "kB" {
			return 0, fmt.Errorf("parse AnonHugePages line %q", line)
		}
		kilobytes, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0, fmt.Errorf("parse AnonHugePages value %q: %w", fields[1], err)
		}
		return kilobytes << 10, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan process memory mappings: %w", err)
	}
	return 0, fmt.Errorf("find mapping containing %#x", address)
}
