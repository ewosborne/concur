package infra_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ewosborne/concur/infra"
)

// test all exported functions
/*
	infra.go
	116:func Do(template string, targets []string, flags Flags) Results {
	169:func GetJSONReport(res Results) (string, error) {
	180:func ReportDone(res Results, flags Flags) {
	404:func PopulateFlags(cmd *cobra.Command) Flags {


	40:func (j JobStatus) MarshalJSON() ([]byte, error) {
	44:func (j JobStatus) String() string {
	92:func (c Command) String() string {

*/

func TestDo(t *testing.T) {
	// test with echoes

	results := infra.Do("echo {{1}}", []string{"booger", "nose"}, infra.Flags{
		ConcurrentJobLimit: "128",
		GoroutineLimit:     128,
		Timeout:            time.Duration(90 * time.Second),
		JobTimeout:         time.Duration(10 * time.Second),
		Token:              "{{1}}",
	})

	for _, cmd := range results.Commands {
		//t.Log("C", cmd)
		// TODO: figure out what to check here.
		if cmd.Status != infra.Finished {
			t.Errorf("test %v status %v expected %v", cmd.Substituted, cmd.Status, infra.Finished)
		}
	}

	j, _ := infra.GetJSONReport(results)
	if !json.Valid([]byte(j)) {
		t.Error("appears to not be valid json wtf")
	}

}

func TestGetJSONReport(t *testing.T) {
	t.Skip() // taken care of in TestDo()
}

func TestReportDone(t *testing.T) {
	t.Skip() // taken care of in TestDo()

}

// TODO I can't get out of this without mocking cmd.  ick.
func TestPopulateFlags(t *testing.T) {

}

/*

	I don't see much value in testing these, they're small and obvious

func Test_JobStatus_MarshalJSON(t *testing.T) {

}

func Test_JobStatus_String(t *testing.T) {

}

func Test_Command_String(t *testing.T) {

}

*/
