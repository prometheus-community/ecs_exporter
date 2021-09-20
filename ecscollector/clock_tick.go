// Copyright 2021 The Prometheus Authors
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

//go:build darwin || linux
// +build darwin linux

// Package ecscollector implements a Prometheus collector for Amazon ECS
// metrics available at the ECS metadata server.
package ecscollector

import (
	"log"

	"github.com/tklauser/go-sysconf"
)

func init() {
	tick, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err != nil {
		log.Printf("Can't get _SC_CLK_TCK; using 100 instead: %v\n", err)
		return
	}
	log.Printf("sysconf(_SC_CLK_TCK) = %d", tick)
	clockTick = tick
}
