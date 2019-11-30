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
	BE_JS_CAN  = "Cancel"
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

var (
	OkCondition       = []string{BE_JS_COM, BE_JS_SUC, BE_JS_ACT, BE_JS_RDY, BE_JS_SCH, BE_JS_LIN}
	WarningCondition  = []string{BE_JS_SUE, BE_JS_ONH, BE_JS_REC, BE_JS_RES, BE_JS_DISA, BE_JS_SUP, BE_JS_RUB, BE_JS_UNK, BE_JS_DISP, BE_JS_QUE, BE_JS_TBS}
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
}

// BEMCLI Class
type BEMCLI struct {
	sshClient *ssh.Client
}

func valInArray(val string, array []string) bool {
	for _, value := range array {
		if val == value {
			return true
		}
	}
	return false
}

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

	session.WindowChange(1000, 1000)
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	bemcli.sshClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		fmt.Printf("%s Error connecting SSH server: %s\n", CRI, err)
		os.Exit(CRI_CODE)
	}

}

// func GetBEJob
// Get information about job
func (bemcli *BEMCLI) GetBEJob(jobName string) {

	beCommand := fmt.Sprintf("Import-Module BEMCLI; Get-BEJob -Name \"%s\"", jobName)
	fmt.Printf("OK %v\n", bemcli.sendCommand(beCommand))

}

// func GetBEJobBackupDefinition
// Get Detailed information for all job corresponding to a BackupDefinition and return a map (by jobname) of status
func (bemcli *BEMCLI) GetBEJobBackupDefinition(backupDefinition string) map[string]BEJobStatus {

	// Definition of Regex to parse Data
	reJob := regexp.MustCompile(`(?m)^Name[\s]*:\s(.*)\r\n(?:(?:JobType[\s]*:\s(.*))\r\n|(?:TaskType[\s]*:\s(.*))\r\n|(?:TaskName[\s]*:\s(.*))\r\n|(?:IsActive[\s]*:\s(.*))\r\n|(?:Status[\s]*:\s(.*))\r\n|(?:SubStatus[\s]*:\s(.*))\r\n|(?:SelectionSummary[\s]*:\s(.*))\r\n|(?:Schedule[\s]*:\s([^I]*))\r\n|(?:Storage[\s]*:\s(.*))\r\n|(?:IsBackupDefinitionJob[\s]*:\s(.*))\r\n|(?:JobHistory[\s]*:\s@{([^}]*)))*`)
	reHistory := regexp.MustCompile(`(?m)([\w]+)=([^;]*)?`)
	reBlank := regexp.MustCompile(`(?m)[\s]{2,}`)
	reNewLine := regexp.MustCompile(`(?m)[\r|\n]+`)

	// Initialize empty maps for return BEJobStatus
	beJobStatus := make(map[string]BEJobStatus)

	// Building PowerShell Command
	selectObject := `Name, JobType, TaskType, TaskName, IsActive, Status, SubStatus, SelectionSummary, Storage, Schedule, IsBackupDefinitionJob,
 		@{name="JobHistory"; expression={Get-BEJobHistory -FromLastJobRun -Job $_.Name |
			Select-Object JobStatus, startTime, EndTime, PercentComplete, TotalDataSizeBytes, JobRateMBPerMinute, ErrorCategory, ErrorCode, ErrorMessage}}`
	beCommand := fmt.Sprintf("Import-Module BEMCLI; Get-BEJob -BackupDefinition \"%s\" | Select-Object %s", backupDefinition, selectObject)

	// sending PowerShell Command and get result
	data := bemcli.sendCommand(beCommand)

	// Parsing content of returned data
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

		schedule := reBlank.ReplaceAllString(m[9], " ")
		schedule = reNewLine.ReplaceAllString(schedule, "")
		js.Schedule = schedule

		js.Storage = m[10]
		js.IsBackupDefinitionJob, _ = strconv.ParseBool(m[11])
		history := reBlank.ReplaceAllString(m[12], " ")
		history = reNewLine.ReplaceAllString(history, "")

		for _, h := range reHistory.FindAllStringSubmatch(history, -1) {
			switch strings.ToLower(h[1]) {
			case "jobstatus":
				js.JobStatus = h[2]
			case "starttime":
				js.StartTime, _ = time.Parse(BE_US_TIME_FORMAT, h[2])
			case "endtime":
				js.EndTime, _ = time.Parse(BE_US_TIME_FORMAT, h[2])
			case "percentcomplete":
				js.PercentComplete, _ = strconv.Atoi(h[2])
			case "totaldatasizebytes":
				js.TotalDataSizeBytes, _ = strconv.ParseInt(h[2], 10, 64)
			case "jobratembperminute":
				js.JobRateMBPerMinute, _ = strconv.ParseFloat(h[2], 64)
			case "errorcategory":
				js.ErrorCategory, _ = strconv.Atoi(h[2])
			case "errorcode":
				js.ErrorCode, _ = strconv.Atoi(h[2])
			case "errormessage":
				js.ErrorMessage = h[2]
			}
		}
		beJobStatus[m[1]] = js
	}

	return beJobStatus
}

// func GetBEBackupExecSetting
// Get Server settings
func (bemcli *BEMCLI) GetBEBackupExecSetting() {
	beCommand := fmt.Sprintf("Import-Module BEMCLI; %s", "Get-BEBackupExecSetting")
	fmt.Printf("OK %v\n", bemcli.sendCommand(beCommand))
}
