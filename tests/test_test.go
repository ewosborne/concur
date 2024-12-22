package test_concur

import (
	"testing"
	"time"

	"github.com/ewosborne/concur/infra"
)

func Test_BySleeping(t *testing.T) {

	// could use a table of test parameters here.

	type SleepNumber struct {
		Durations []string
		LowEst    time.Duration
		HighEst   time.Duration
	}

	tt := []SleepNumber{
		{
			Durations: []string{"1"},
			LowEst:    1 * time.Second,
			HighEst:   2 * time.Second,
		},
		{
			Durations: []string{"1", "2", "3"},
			LowEst:    3 * time.Second,
			HighEst:   4 * time.Second,
		},
	}

	// func Do(command string, substituteArgs []string, flags Flags) Results {

	// set up args, at for now

	command := "sleep {{1}}"

	flags := infra.Flags{
		ConcurrentLimit: "128",
		GoroutineLimit:  128,
		Timeout:         time.Duration(90 * time.Second),
		Token:           "{{1}}",
	}

	// call Do, returns Results

	for _, entry := range tt {
		argset := entry.Durations

		results := infra.Do(command, argset, flags)

		t.Log("so far", results.Info)

		// what to sanity check?
		srt := results.Info.SystemRunTime
		if srt < entry.LowEst || srt > entry.HighEst {
			t.Errorf("system runtime %v not withing range low:%v high:%v", results.Info.SystemRunTime, entry.LowEst, entry.HighEst)
		}
	}

}
