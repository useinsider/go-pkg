package insssm

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	awsssm "github.com/Jamil-Najafov/go-aws-ssm"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type stubSSM struct {
	mu     sync.Mutex
	calls  map[string]int
	values map[string]string
	err    error
}

func (s *stubSSM) GetParameter(input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[aws.StringValue(input.Name)]++

	if s.err != nil {
		return nil, s.err
	}

	v, ok := s.values[aws.StringValue(input.Name)]
	if !ok {
		return nil, errors.New("stub: parameter not found")
	}

	return &ssm.GetParameterOutput{Parameter: &ssm.Parameter{Value: aws.String(v)}}, nil
}

func (s *stubSSM) GetParametersByPathPages(_ *ssm.GetParametersByPathInput, _ func(*ssm.GetParametersByPathOutput, bool) bool) error {
	return errors.New("stub: not implemented")
}

func (s *stubSSM) PutParameter(_ *ssm.PutParameterInput) (*ssm.PutParameterOutput, error) {
	return nil, errors.New("stub: not implemented")
}

func (s *stubSSM) callCount(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[key]
}

func TestInit(t *testing.T) {
	t.Run("it_should_skip_initialization_when_env_is_local", func(t *testing.T) {
		prev := ParameterStore
		t.Cleanup(func() { ParameterStore = prev })
		ParameterStore = nil

		t.Setenv("ENV", "LOCAL")
		Init()

		if ParameterStore != nil {
			t.Error("Init() initialized ParameterStore despite ENV=LOCAL")
		}
	})

	t.Run("it_should_initialize_parameter_store_otherwise", func(t *testing.T) {
		prev := ParameterStore
		t.Cleanup(func() { ParameterStore = prev })
		ParameterStore = nil

		t.Setenv("ENV", "TEST")
		Init()

		if ParameterStore == nil {
			t.Error("Init() left ParameterStore nil")
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("it_should_return_value_from_parameter_store", func(t *testing.T) {
		prev := ParameterStore
		t.Cleanup(func() { ParameterStore = prev })

		stub := &stubSSM{values: map[string]string{"insssm-test-key-1": "value-1"}}
		ParameterStore = awsssm.NewParameterStoreWithClient(stub)

		if got := Get("insssm-test-key-1"); got != "value-1" {
			t.Errorf("Get() = %q, want %q", got, "value-1")
		}
	})

	t.Run("it_should_serve_repeated_reads_from_cache", func(t *testing.T) {
		prev := ParameterStore
		t.Cleanup(func() { ParameterStore = prev })

		stub := &stubSSM{values: map[string]string{"insssm-test-key-2": "value-2"}}
		ParameterStore = awsssm.NewParameterStoreWithClient(stub)

		for i := 0; i < 3; i++ {
			if got := Get("insssm-test-key-2"); got != "value-2" {
				t.Fatalf("Get() call %d = %q, want %q", i+1, got, "value-2")
			}
		}

		if n := stub.callCount("insssm-test-key-2"); n != 1 {
			t.Errorf("underlying SSM was called %d times, want 1 (cached)", n)
		}
	})

	t.Run("it_should_panic_when_parameter_store_fails", func(t *testing.T) {
		prev := ParameterStore
		t.Cleanup(func() { ParameterStore = prev })

		stub := &stubSSM{err: errors.New("aws is down")}
		ParameterStore = awsssm.NewParameterStoreWithClient(stub)

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Get() did not panic on SSM error")
			}
			if msg := fmt.Sprint(r); !strings.Contains(msg, "aws is down") {
				t.Errorf("panic message %q does not mention the SSM error", msg)
			}
		}()

		Get("insssm-test-key-err")
	})
}
