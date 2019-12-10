// Copyright (c) 2019 Terre des hommes foundation - Claude Débieux
// Use of this source code is governed by a MIT License style
// license that can be found in the LICENSE file.
//
// check_backupexec is a icinga2/nagios plugin who return state of backupexec jobs
//
// Author : Claude Débieux - Terre des hommes foundation - Lausanne
// Version History:
//		26-nov-2019 - version #000 - Plugin Creationpackage main
package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// icinga2 Constant
	OK       = "OK"
	WAR      = "WARNING"
	CRI      = "CRITICAL"
	UNK      = "UNKNOWN"
	OK_CODE  = 0
	WAR_CODE = 1
	CRI_CODE = 2
	UNK_CODE = 3

	// BackupExec constant
	BE_US_TIME_FORMAT = "1/2/2006 3:04:05 PM"

	// BackupExec JobStatus
	BE_JS_UNK  = "Unknown"
	BE_JS_CAN  = "Canceled"
	BE_JS_COM  = "Completed"
	BE_JS_SUE  = "SucceededWithExceptions"
	BE_JS_ONH  = "OnHold"
	BE_JS_ERR  = "Error"
	BE_JS_MIS  = "Missed"
	BE_JS_REC  = "Recovered"
	BE_JS_RES  = "Resumed"
	BE_JS_SUC  = "Succeeded"
	BE_JS_THA  = "ThresholdAbort"
	BE_JS_DISP = "Dispatched"
	BE_JS_DIF  = "DispatchFailed"
	BE_JS_INS  = "InvalidSchedule"
	BE_JS_INT  = "InvalidTimeWindow"
	BE_JS_NOI  = "NotInTimeWindow"
	BE_JS_QUE  = "Queued"
	BE_JS_DISA = "Disabled"
	BE_JS_ACT  = "Active"
	BE_JS_RDY  = "Ready"
	BE_JS_SCH  = "Scheduled"
	BE_JS_SUP  = "Superseded"
	BE_JS_TBS  = "ToBeScheduled"
	BE_JS_LIN  = "Linked"
	BE_JS_RUB  = "RuleBlocked"

	// BackupExec JobType
)

// Backup Exec JobStatus - Level of return text status
var (
	OkCondition       = []string{BE_JS_COM, BE_JS_SUE, BE_JS_SUC, BE_JS_ACT, BE_JS_RDY, BE_JS_SCH, BE_JS_LIN}
	WarningCondition  = []string{BE_JS_ONH, BE_JS_REC, BE_JS_RES, BE_JS_DISA, BE_JS_SUP, BE_JS_RUB, BE_JS_UNK, BE_JS_DISP, BE_JS_QUE, BE_JS_TBS}
	CriticalCondition = []string{BE_JS_CAN, BE_JS_ERR, BE_JS_MIS, BE_JS_THA, BE_JS_DIF, BE_JS_INS, BE_JS_INT, BE_JS_NOI}
)

// Job Status Record
type BEJobStatus struct {
	Name                  string
	JobType               string
	TaskType              string
	TaskName              string
	IsActive              bool
	Status                string
	SubStatus             string
	SelectionSummary      string
	Storage               string
	Schedule              string
	IsBackupDefinitionJob bool
	JobStatus             string
	StartTime             time.Time
	EndTime               time.Time
	PercentComplete       int
	TotalDataSizeBytes    int64
	JobRateMBPerMinute    float64
	ErrorCategory         int
	ErrorCode             int
	ErrorMessage          string
}

// icinga state
type Icinga struct {
	StatusCode int
	Status     string
	Message    string
	Metric     string
	NewestLog  time.Time
}

// BEMCLI Class
type BEMCLI struct {
	sshClient   *ssh.Client
	beJobStatus []BEJobStatus
}

// Helper function who return true if occurrence of value exist in array
func valInArray(val string, array []string) bool {
	for _, value := range array {
		if val == value {
			return true
		}
	}
	return false
}

// Return level of BackupExec status
func (bemcli *BEMCLI) Condition(val string) int {
	if valInArray(val, OkCondition) {
		return OK_CODE
	}
	if valInArray(val, WarningCondition) {
		return WAR_CODE
	}
	if valInArray(val, CriticalCondition) {
		return CRI_CODE
	}

	return UNK_CODE
}

// func sendCommand (Private)
// Send command to remote SSH server and return result as string
func (bemcli *BEMCLI) sendCommand(command string) string {
	var b bytes.Buffer

	session, err := bemcli.sshClient.NewSession()
	if err != nil {
		fmt.Printf("%s Error creating session on SSH server: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}
	defer session.Close()

	_ = session.WindowChange(1000, 1000)
	session.Stdout = &b
	if err := session.Run(command); err != nil {
		fmt.Printf("%s Failed to run command: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}
	return b.String()
}

// func Init
// Initialize connection to SSH server
func (bemcli *BEMCLI) Init(host string, username string, password string, identity string, port int) {
	var signer ssh.Signer

	// replacing tilde char by real home directory
	home, _ := user.Current()
	re := regexp.MustCompile(`^~(.*)$`)
	identity = re.ReplaceAllString(identity, home.HomeDir+"${1}")

	// Reading and parsing identity file (private key)
	key, err := ioutil.ReadFile(identity)
	if err == nil {
		// Create the Signer for this private key.
		signer, err = ssh.ParsePrivateKey(key)
		if err != nil {
			signer = nil
		}
	} else {
		signer = nil
	}

	// Configure authentication methods
	var auths []ssh.AuthMethod
	if signer != nil {
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if password != "" {
		auths = append(auths, ssh.Password(password))
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // No key validation in known host
	}

	// Connecting SSH Server (Backup Exec Server)
	bemcli.sshClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		fmt.Printf("%s Error connecting SSH server: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}

}

// func GetBEBackupExecSetting
// Get Server settings
func (bemcli *BEMCLI) GetBEBackupExecSetting() {
	beCommand := fmt.Sprintf("Import-Module BEMCLI; %s", "Get-BEBackupExecSetting")
	fmt.Printf("OK %v\n", bemcli.sendCommand(beCommand))
}

// func GetBEJob
// Get information about job
func (bemcli *BEMCLI) GetBEJob(jobName string) {

	beCommand := fmt.Sprintf("Import-Module BEMCLI; Get-BEJob -Name \"%s\"", jobName)
	fmt.Printf("OK %v\n", bemcli.sendCommand(beCommand))

}

// func GetBEJobBackupDefinition
// Get Detailed information for all job corresponding to a BackupDefinition and return a map (by jobname) of status
func (bemcli *BEMCLI) GetBEJobBackupDefinition(backupDefinition string) string {
	// Regex to clean received from ssh session
	reBlank := regexp.MustCompile(`(?m)[\s]{2,}`)
	reNewLine := regexp.MustCompile(`(?m)[\r|\n]+`)

	// Building PowerShell Command
	selectObject := `Name, JobType, TaskType, TaskName, IsActive, Status, SubStatus, SelectionSummary, Storage, Schedule, IsBackupDefinitionJob,
 		@{name="JobHistory"; expression={Get-BEJobHistory -FromLastJobRun -Job $_.Name |
			Select-Object JobStatus, startTime, EndTime, PercentComplete, TotalDataSizeBytes, JobRateMBPerMinute, ErrorCategory, ErrorCode, ErrorMessage}}`
	beCommand := fmt.Sprintf("Import-Module BEMCLI; Get-BEJob -BackupDefinition \"%s\" | Select-Object %s", backupDefinition, selectObject)

	// sending PowerShell Command and get result
	data := bemcli.sendCommand(beCommand)
	data = reBlank.ReplaceAllString(data, " ")
	data = reNewLine.ReplaceAllString(data, "")
	return data
}

// fuc BEJobStatusToIcingaStatus
// Geneation of a Icinga/Nagios status string
func (bemcli *BEMCLI) BEJobsStatusToIcingaStatus(data string, verbose bool) (string, int) {
	// Definition of Regex to parse Data
	reJob := regexp.MustCompile(`(?m)\s?Name\s:\s(.*?)\s?JobType\s:\s(.*?)\s?TaskType\s:\s(.*?)\s?TaskName\s:\s(.*?)\s?IsActive\s:\s(.*?)\s?Status\s:\s(.*?)\s?SubStatus\s:\s(.*?)\s?SelectionSummary\s:\s(.*?)\s?Storage\s:\s(.*?)\s?Schedule\s:\s(.*?)\s?IsBackupDefinitionJob\s:\s(.*?)\s?JobHistory\s:\s(?:@{JobStatus=(.*?);\s?StartTime=(.*?);\s?EndTime=(.*?);\s?PercentComplete=(.*?);\s?TotalDataSizeBytes=(.*?);\s?JobRateMBPerMinute=(.*?);\s?ErrorCategory=(.*?);\s?ErrorCode=(.*?);\s?ErrorMessage=(.*?)})?`)

	var icinga Icinga

	// Initialize icinga info
	icinga.Status = UNK
	icinga.StatusCode = UNK_CODE
	icinga.Message = ""
	icinga.Metric = ""

	// Initialize empty slice for return BEJobStatus
	var beJobStatus []BEJobStatus

	// If verbose mode is enable displaying RAW Data
	if verbose {
		fmt.Println("-------------------------------------")
		fmt.Printf("%s\n", data)
		fmt.Println("-------------------------------------")
	}

	// Building structure from RAW Data
	match := reJob.FindAllStringSubmatch(data, -1)

	for _, m := range match {
		js := BEJobStatus{}
		js.Name = m[1]
		js.JobType = m[2]
		js.TaskType = m[3]
		js.TaskName = m[4]
		js.IsActive, _ = strconv.ParseBool(m[5])
		js.Status = m[6]
		js.SubStatus = m[7]
		js.SelectionSummary = m[8]
		js.Storage = m[9]
		js.Schedule = m[10]
		js.IsBackupDefinitionJob, _ = strconv.ParseBool(m[11])
		js.JobStatus = m[12]
		js.StartTime, _ = time.Parse(BE_US_TIME_FORMAT, m[13])
		js.EndTime, _ = time.Parse(BE_US_TIME_FORMAT, m[14])
		js.PercentComplete, _ = strconv.Atoi(m[15])
		js.TotalDataSizeBytes, _ = strconv.ParseInt(m[16], 10, 64)
		js.JobRateMBPerMinute, _ = strconv.ParseFloat(m[17], 64)
		js.ErrorCategory, _ = strconv.Atoi(m[18])
		js.ErrorCode, _ = strconv.Atoi(m[19])
		js.ErrorMessage = m[20]
		beJobStatus = append(beJobStatus, js)
	}

	// Sorting slice by descending job StartTime (most recent job is first)
	sort.Slice(beJobStatus, func(i, j int) bool {
		return beJobStatus[i].StartTime.Unix() > beJobStatus[j].StartTime.Unix()
	})
	bemcli.beJobStatus = beJobStatus

	// Checking jobs status to build Icinga response
	for idx, job := range bemcli.beJobStatus {
		msgAdd := ""
		// First jobStatus give general ICINGA status
		if idx == 0 {
			icinga.NewestLog = job.StartTime
			switch bemcli.Condition(job.JobStatus) {
			case OK_CODE:
				icinga.StatusCode = OK_CODE
				icinga.Status = OK
			case WAR_CODE:
				icinga.StatusCode = WAR_CODE
				icinga.Status = WAR
			case CRI_CODE:
				icinga.StatusCode = CRI_CODE
				icinga.Status = CRI
				msgAdd = " [" + job.ErrorMessage + "]"
			default:
				if job.IsActive {
					icinga.StatusCode = OK_CODE
					icinga.Status = OK
					msgAdd = " [Job is Running]"
				} else {
					icinga.StatusCode = UNK_CODE
					icinga.Status = UNK
				}
			}
		}
		switch bemcli.Condition(job.JobStatus) {
		case CRI_CODE:
			msgAdd = "[" + job.ErrorMessage + "]"
		default:
			if job.IsActive {
				icinga.StatusCode = OK_CODE
				icinga.Status = OK
				msgAdd = "[Job is Running]"
			}
		}

		if icinga.Message != "" {
			icinga.Message += "/"
		}
		icinga.Message += job.Name + " " + job.Status
		if strings.ToLower(job.SubStatus) != "ok" {
			icinga.Message += "-" + job.SubStatus
		}
		if job.JobStatus != "" {
			icinga.Message += "-" + job.JobStatus
		}
		if msgAdd != "" {
			icinga.Message += " " + msgAdd
		}

	}
	if icinga.Metric != "" {
		icinga.Metric = " | " + icinga.Metric
	}

	// Returning Icinga monitoring formatted status
	return fmt.Sprintf("%s: Last Run '%v' %s%s\n", icinga.Status, icinga.NewestLog.Format("02/01/2006 15:04:05"), icinga.Message, icinga.Metric), icinga.StatusCode

}
