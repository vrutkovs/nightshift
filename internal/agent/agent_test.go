package agent

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	obj := New()
	for i := 0; i < 2; i++ {
		_obj := New()
		if _obj != obj {
			t.Errorf("New failed %d - got different instance", i)
		}
	}
}

func TestSetResyncInterval(t *testing.T) {
	obj := &worker{}
	for i := 0; i < 2; i++ {
		dur := time.Duration(i) * time.Minute
		obj.SetResyncInterval(dur)
		if obj.interval != dur {
			t.Errorf("SetResyncInterval failed %d - got %v, expected %v", i, obj.interval, dur)
		}
	}
}

func TestAddGetScanner(t *testing.T) {
	obj := New()
	for i := 0; i < 10; i++ {
		scnr := &mockScanner{id: i}
		obj.AddScanner(scnr)
	}
	scnrs := obj.GetScanners()
	if len(scnrs) != 10 {
		t.Errorf("Invalid number of scanners in GetScanners; got %d, expected 10", len(scnrs))
	}
	for i, scnr := range scnrs {
		mock, ok := scnr.(*mockScanner)
		if !ok {
			t.Errorf("New failed %d - invalid scanner, expected a mockScanner", i)
		}
		if mock.id != i {
			t.Errorf("Invalid scanner in GetScanners; got %d, expected %d", mock.id, i)
		}
	}
}

func TestAddGetTrigger(t *testing.T) {
	tests := []string{"foo", "bar"}
	obj := New()
	for _, id := range tests {
		trgr := &mockTrigger{id: id}
		obj.AddTrigger(id, trgr)
	}
	trgrs := obj.GetTriggers()
	if len(trgrs) != len(tests) {
		t.Errorf("Invalid number of scanners in GetScanners; got %d, expected %d", len(trgrs), len(tests))
	}
	for id, trgr := range trgrs {
		mock, ok := trgr.(*mockTrigger)
		if !ok {
			t.Errorf("Invalid trigger, expected a mockTrigger, got %#v", trgr)
		}
		if mock.id != id {
			t.Errorf("Invalid trigger in GetTriggers; got %s, expected %s", mock.id, id)
		}
	}
}
