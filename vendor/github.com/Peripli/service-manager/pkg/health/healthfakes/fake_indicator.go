// Code generated by counterfeiter. DO NOT EDIT.
package healthfakes

import (
	"sync"

	"github.com/Peripli/service-manager/pkg/health"
)

type FakeIndicator struct {
	NameStub        func() string
	nameMutex       sync.RWMutex
	nameArgsForCall []struct{}
	nameReturns     struct {
		result1 string
	}
	nameReturnsOnCall map[int]struct {
		result1 string
	}
	HealthStub        func() *health.Health
	healthMutex       sync.RWMutex
	healthArgsForCall []struct{}
	healthReturns     struct {
		result1 *health.Health
	}
	healthReturnsOnCall map[int]struct {
		result1 *health.Health
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIndicator) Name() string {
	fake.nameMutex.Lock()
	ret, specificReturn := fake.nameReturnsOnCall[len(fake.nameArgsForCall)]
	fake.nameArgsForCall = append(fake.nameArgsForCall, struct{}{})
	fake.recordInvocation("Name", []interface{}{})
	fake.nameMutex.Unlock()
	if fake.NameStub != nil {
		return fake.NameStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.nameReturns.result1
}

func (fake *FakeIndicator) NameCallCount() int {
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	return len(fake.nameArgsForCall)
}

func (fake *FakeIndicator) NameReturns(result1 string) {
	fake.NameStub = nil
	fake.nameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeIndicator) NameReturnsOnCall(i int, result1 string) {
	fake.NameStub = nil
	if fake.nameReturnsOnCall == nil {
		fake.nameReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.nameReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeIndicator) Health() *health.Health {
	fake.healthMutex.Lock()
	ret, specificReturn := fake.healthReturnsOnCall[len(fake.healthArgsForCall)]
	fake.healthArgsForCall = append(fake.healthArgsForCall, struct{}{})
	fake.recordInvocation("Health", []interface{}{})
	fake.healthMutex.Unlock()
	if fake.HealthStub != nil {
		return fake.HealthStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.healthReturns.result1
}

func (fake *FakeIndicator) HealthCallCount() int {
	fake.healthMutex.RLock()
	defer fake.healthMutex.RUnlock()
	return len(fake.healthArgsForCall)
}

func (fake *FakeIndicator) HealthReturns(result1 *health.Health) {
	fake.HealthStub = nil
	fake.healthReturns = struct {
		result1 *health.Health
	}{result1}
}

func (fake *FakeIndicator) HealthReturnsOnCall(i int, result1 *health.Health) {
	fake.HealthStub = nil
	if fake.healthReturnsOnCall == nil {
		fake.healthReturnsOnCall = make(map[int]struct {
			result1 *health.Health
		})
	}
	fake.healthReturnsOnCall[i] = struct {
		result1 *health.Health
	}{result1}
}

func (fake *FakeIndicator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.nameMutex.RLock()
	defer fake.nameMutex.RUnlock()
	fake.healthMutex.RLock()
	defer fake.healthMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeIndicator) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ health.Indicator = new(FakeIndicator)
