/*
Copyright 2020 Netflix, Inc.

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
package v1

import "math"

// Data structure representing compute resources. We use uint64 type as aggregated resources
// may amount to very large values.
type ComputeResource struct {
	CPU int64 `json:"cpu"`

	GPU int64 `json:"gpu"`

	MemoryMB int64 `json:"memoryMB"`

	DiskMB int64 `json:"diskMB"`

	NetworkMBPS int64 `json:"networkMBPS"`
}

var (
	Zero = ComputeResource{}
)

func (r ComputeResource) Add(second ComputeResource) ComputeResource {
	return ComputeResource{
		CPU:         r.CPU + second.CPU,
		GPU:         r.GPU + second.GPU,
		MemoryMB:    r.MemoryMB + second.MemoryMB,
		DiskMB:      r.DiskMB + second.DiskMB,
		NetworkMBPS: r.NetworkMBPS + second.NetworkMBPS,
	}
}

func (r ComputeResource) Sub(second ComputeResource) ComputeResource {
	return ComputeResource{
		CPU:         r.CPU - second.CPU,
		GPU:         r.GPU - second.GPU,
		MemoryMB:    r.MemoryMB - second.MemoryMB,
		DiskMB:      r.DiskMB - second.DiskMB,
		NetworkMBPS: r.NetworkMBPS - second.NetworkMBPS,
	}
}

// Subtraction with resource minimum value set to the provided lower bound.
func (r ComputeResource) SubWithLimit(second ComputeResource, lowerBound int64) ComputeResource {
	sub := func(v1 int64, v2 int64) int64 {
		r := v1 - v2
		if r > lowerBound {
			return r
		}
		return lowerBound
	}

	return ComputeResource{
		CPU:         sub(r.CPU, second.CPU),
		GPU:         sub(r.GPU, second.GPU),
		MemoryMB:    sub(r.MemoryMB, second.MemoryMB),
		DiskMB:      sub(r.DiskMB, second.DiskMB),
		NetworkMBPS: sub(r.NetworkMBPS, second.NetworkMBPS),
	}
}

func (r ComputeResource) Multiply(multiplier int64) ComputeResource {
	return ComputeResource{
		CPU:         r.CPU * multiplier,
		GPU:         r.GPU * multiplier,
		MemoryMB:    r.MemoryMB * multiplier,
		DiskMB:      r.DiskMB * multiplier,
		NetworkMBPS: r.NetworkMBPS * multiplier,
	}
}

func (r ComputeResource) Divide(divider int64) ComputeResource {
	return ComputeResource{
		CPU:         r.CPU / divider,
		GPU:         r.GPU / divider,
		MemoryMB:    r.MemoryMB / divider,
		DiskMB:      r.DiskMB / divider,
		NetworkMBPS: r.NetworkMBPS / divider,
	}
}

// Align resource ratios to be the same as in the reference. The resulting ComputeResource is the smallest value that
// is >= the original, with the resource ratios identical to the reference.
func (r ComputeResource) AlignResourceRatios(reference ComputeResource) ComputeResource {
	rdiv := func(currentMax float64, v1 int64, v2 int64) float64 {
		if v2 == 0 {
			return 0
		}
		return math.Max(currentMax, float64(v1)/float64(v2))
	}
	maxRatio := rdiv(0, r.CPU, reference.CPU)
	maxRatio = rdiv(maxRatio, r.MemoryMB, reference.MemoryMB)
	maxRatio = rdiv(maxRatio, r.DiskMB, reference.DiskMB)
	maxRatio = rdiv(maxRatio, r.NetworkMBPS, reference.NetworkMBPS)

	if maxRatio == 0 {
		return r
	}

	return ComputeResource{
		CPU:         int64(float64(reference.CPU) * maxRatio),
		GPU:         int64(float64(reference.GPU) * maxRatio),
		MemoryMB:    int64(float64(reference.MemoryMB) * maxRatio),
		DiskMB:      int64(float64(reference.DiskMB) * maxRatio),
		NetworkMBPS: int64(float64(reference.NetworkMBPS) * maxRatio),
	}
}

// Check if can computes how many times the argument fully fits into the give resource.
func (r ComputeResource) CanSplitBy(second ComputeResource) bool {
	return (r.CPU == 0 || second.CPU != 0) &&
		(r.GPU == 0 || second.GPU != 0) &&
		(r.MemoryMB == 0 || second.MemoryMB != 0) &&
		(r.DiskMB == 0 || second.DiskMB != 0) &&
		(r.NetworkMBPS == 0 || second.NetworkMBPS != 0)
}

// Computes how many times the argument fully fits into the give resource.
func (r ComputeResource) SplitBy(second ComputeResource) int64 {
	if !r.CanSplitBy(second) {
		return 0
	}

	rdiv := func(currentMin int64, v1 int64, v2 int64) int64 {
		if v2 == 0 {
			return currentMin
		}
		ratio := v1 / v2
		if currentMin < 0 || ratio < currentMin {
			return ratio
		}
		return currentMin
	}

	min := rdiv(-1, r.CPU, second.CPU)
	min = rdiv(min, r.GPU, second.GPU)
	min = rdiv(min, r.MemoryMB, second.MemoryMB)
	min = rdiv(min, r.DiskMB, second.DiskMB)
	min = rdiv(min, r.NetworkMBPS, second.NetworkMBPS)

	if min == -1 {
		return 0
	}
	return min
}

// Similar to SplitBy, but with rounding up.
func (r ComputeResource) SplitByWithCeil(second ComputeResource) int64 {
	if !r.CanSplitBy(second) {
		return 0
	}

	result := r.SplitBy(second)
	if second.Multiply(result).LessThan(r) {
		return result + 1
	}
	return result
}

// For a compute resource to be strictly less than the other one, all individual resources (CPU, memory, etc) must be smaller.
func (r ComputeResource) StrictlyLessThan(second ComputeResource) bool {
	return r.CPU < second.CPU &&
		r.GPU < second.GPU &&
		r.MemoryMB < second.MemoryMB &&
		r.DiskMB < second.DiskMB &&
		r.NetworkMBPS < second.NetworkMBPS
}

// For a compute resource to be less than the other one, all individual resources (CPU, memory, etc) must not greater than
// their counterparts, and at least one of them must be smaller.
func (r ComputeResource) LessThan(second ComputeResource) bool {
	allNotBigger := r.CPU <= second.CPU &&
		r.GPU <= second.GPU &&
		r.MemoryMB <= second.MemoryMB &&
		r.DiskMB <= second.DiskMB &&
		r.NetworkMBPS <= second.NetworkMBPS
	return allNotBigger && r != second
}

// For a compute resource to be strictly greater than the other one, all individual resources (CPU, memory, etc) must be greater.
func (r ComputeResource) StrictlyGreaterThan(second ComputeResource) bool {
	return r.CPU > second.CPU &&
		r.GPU > second.GPU &&
		r.MemoryMB > second.MemoryMB &&
		r.DiskMB > second.DiskMB &&
		r.NetworkMBPS > second.NetworkMBPS
}

// For a compute resource to be greater than the other one, all individual resources (CPU, memory, etc) must not be smaller than
// their counterparts, and at least one of them must be bigger.
func (r ComputeResource) GreaterThan(second ComputeResource) bool {
	allNotSmaller := r.CPU >= second.CPU &&
		r.GPU >= second.GPU &&
		r.MemoryMB >= second.MemoryMB &&
		r.DiskMB >= second.DiskMB &&
		r.NetworkMBPS >= second.NetworkMBPS
	return allNotSmaller && r != second
}

func (r ComputeResource) IsAboveZero() bool {
	return r.CPU > 0 || r.GPU > 0 || r.MemoryMB > 0 || r.DiskMB > 0 || r.NetworkMBPS > 0
}
