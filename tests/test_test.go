package test_concur

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ewosborne/concur/infra"
)

type SleepNumber struct {
	Durations []string
	LowEst    time.Duration
	HighEst   time.Duration
}

var tt = []SleepNumber{
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

var sleepCommand = "sleep {{1}}"

var sleepFlags = infra.Flags{
	ConcurrentJobLimit: "128",
	GoroutineLimit:     128,
	Timeout:            time.Duration(90 * time.Second),
	Token:              "{{1}}",
}

// TODO more tests.  tests concurrency params, --any and --first, etc.
func Test_DoBySleeping(t *testing.T) {

	// set up args, at for now

	// call Do, returns Results

	for _, entry := range tt {
		argset := entry.Durations

		// results is programmatic
		results := infra.Do(sleepCommand, argset, sleepFlags)

		t.Log("so far", results.Info)

		// what to sanity check?
		srt := results.Info.InternalSystemRunTime
		if (srt < entry.LowEst) || (srt > entry.HighEst) {
			//if srt < entry.LowEst || srt > entry.HighEst {
			t.Errorf("system runtime %v not withing range low:%v high:%v", results.Info.InternalSystemRunTime, entry.LowEst, entry.HighEst)
		}

		// do I even do anything with jsonresults?
		jsonResults, err := infra.GetJSONReport(results)
		if err != nil {
			t.Error(err)
		}

		if len(jsonResults) == 0 {
			t.Errorf("somehow got an empty json")
		}

		var holder = make(map[string]any)
		err = json.Unmarshal([]byte(jsonResults), &holder)
		if err != nil {
			t.Errorf("can't unmarshal json to map: %v\n", err)
		}

	}

}
