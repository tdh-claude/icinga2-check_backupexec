package main

import "testing"

func TestBEMCLI_BEJobsStatusToIcingaStatus(t *testing.T) {

	bemcli := new(BEMCLI)

	data := ` Name : tdhmmd01 - Full Weekend JobType : Backup TaskType : Full TaskName : Full Weekend IsActive : False Status : Scheduled SubStatus : Ok SelectionSummary : Fully selected Storage : Deduplication Storage 001 Schedule : Saturday every 1 week(s) at 11:00 PM effective on 3/28/2019. IsBackupDefinitionJob : True JobHistory : @{JobStatus=Succeeded; StartTime=11/30/2019 11:00:02 PM; EndTime=12/1/2019 7:32:37 PM; PercentComplete=100; TotalDataSizeBytes=6722145064097; JobRateMBPerMinute=5366.038; ErrorCategory=0; ErrorCode=0; ErrorMessage=} Name : tdhmmd01 - Duplicate Monthly JobType : Duplicate TaskType : Duplicate TaskName : Duplicate Monthly IsActive : False Status : Unknown SubStatus : Ok SelectionSummary : Fully selected Storage : Any tape cartridge storage Schedule : IsBackupDefinitionJob : True JobHistory : Name : tdhmmd01 - Full Monthly JobType : Backup TaskType : Full TaskName : Full Monthly IsActive : False Status : Scheduled SubStatus : Ok SelectionSummary : Fully selected Storage : Deduplication Storage 001 Schedule : First Saturday of every 1 month(s) at 11:00 PM effective on 3/28/2019. IsBackupDefinitionJob : True JobHistory : @{JobStatus=Canceled; StartTime=11/2/2019 11:00:03 PM; EndTime=11/3/2019 2:09:52 AM; PercentComplete=-1; TotalDataSizeBytes=751432335139; JobRateMBPerMinute=3890; ErrorCategory=1; ErrorCode=0; ErrorMessage=} Name : TDHMMD01-Duplicate Weekend JobType : Duplicate TaskType : Duplicate TaskName : Duplicate Weekend IsActive : False Status : Linked SubStatus : Ok SelectionSummary : Fully selected Storage : Any tape cartridge storage Schedule : IsBackupDefinitionJob : True JobHistory : @{JobStatus=Succeeded; StartTime=12/1/2019 7:32:47 PM; EndTime=12/2/2019 4:18:45 AM; PercentComplete=100; TotalDataSizeBytes=6722145064097; JobRateMBPerMinute=12612.89; ErrorCategory=0; ErrorCode=0; ErrorMessage=}`
	result, cond := bemcli.BEJobsStatusToIcingaStatus(data, false)
	if cond != OK_CODE {
		t.Errorf("Error testing BEJobsStatusToIcingaStatus incorrect code %d, want 0\nResult string is %s", cond, result)
	} else {
		t.Logf("Return code is Ok")
	}
	if len(bemcli.beJobStatus) != 3 {
		t.Errorf("Number of jobs incorrect %d, want 3", len(bemcli.beJobStatus))
	}

}
