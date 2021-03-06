package pserver_test

import (
	"io/ioutil"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/PaddlePaddle/Paddle/go/pserver"
)

const (
	OptimizerConfig = "./client/c/test/testdata/optimizer.pb"
)

func TestServiceFull(t *testing.T) {
	var cp pserver.Checkpoint
	s, err := pserver.NewService(0, 1, "", nil, cp)
	if err != nil {
		t.Error(err)
	}
	var p pserver.Parameter
	p.Name = "param_a"
	p.Content = []byte{1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}
	p.ElementType = pserver.Int32
	config, err := ioutil.ReadFile(OptimizerConfig)
	if err != nil {
		t.Fatalf("read optimizer proto failed")
	}

	err = s.InitParam(pserver.ParameterWithConfig{Param: p, Config: config}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var p1 pserver.Parameter
	p1.Name = "param_b"
	p1.Content = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	p1.ElementType = pserver.Float32
	err = s.InitParam(pserver.ParameterWithConfig{Param: p1, Config: config}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = s.FinishInitParams(0, nil)
	if err != nil {
		t.Fatal(err)
	}

	var param pserver.Parameter
	err = s.GetParam("param_b", &param)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(param, p1) {
		t.Fatal("not equal:", param, p1)
	}

	g1, g2 := pserver.Gradient(p1), pserver.Gradient(p)

	err = s.SendGrad(g1, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.SendGrad(g2, nil)

	if err != nil {
		t.Fatal(err)
	}

	var param1 pserver.Parameter
	err = s.GetParam("param_a", &param1)
	if err != nil {
		t.Fatal(err)
	}

	// don't compare content, since it's already changed by
	// gradient update.
	param1.Content = nil
	p.Content = nil

	if !reflect.DeepEqual(param1, p) {
		t.Fatal("not equal:", param1, p)
	}
}

func TestMultipleInit(t *testing.T) {
	var cp pserver.Checkpoint
	s, err := pserver.NewService(0, 1, "", nil, cp)
	if err != nil {
		t.Fatal(err)
	}
	err = s.FinishInitParams(0, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = s.FinishInitParams(0, nil)
	if err.Error() != pserver.AlreadyInitialized {
		t.Fatal(err)
	}
}

func TestUninitialized(t *testing.T) {
	var cp pserver.Checkpoint
	s, err := pserver.NewService(0, 1, "", nil, cp)
	err = s.SendGrad(pserver.Gradient{}, nil)
	if err.Error() != pserver.Uninitialized {
		t.Fatal(err)
	}
}

func TestBlockUntilInitialized(t *testing.T) {
	var cp pserver.Checkpoint
	s, err := pserver.NewService(0, 1, "", nil, cp)
	if err != nil {
		t.Error(err)
	}
	ch := make(chan struct{}, 2)
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		var param pserver.Parameter
		err := s.GetParam("param_a", &param)
		if err != nil {
			errCh <- err
		}
		wg.Done()
		ch <- struct{}{}
	}()

	time.Sleep(50 * time.Millisecond)

	select {
	case <-ch:
		// some function returned before initialization is completed.
		t.FailNow()
	case <-errCh:
		t.FailNow()
	default:
	}

	var p pserver.Parameter
	p.Name = "param_a"
	p.Content = []byte{1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}
	p.ElementType = pserver.Int32
	config, err := ioutil.ReadFile(OptimizerConfig)
	if err != nil {
		t.Fatalf("read optimizer proto failed")
	}
	err = s.InitParam(pserver.ParameterWithConfig{Param: p, Config: config}, nil)

	if err != nil {
		t.Fatal(err)
	}

	err = s.FinishInitParams(0, nil)
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()
}

func TestCheckpointSpeed(t *testing.T) {
	//TODO(zhihong): test speed
}
