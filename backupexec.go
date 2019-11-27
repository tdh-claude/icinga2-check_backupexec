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
	"log"
	"os"
	"os/user"
	"regexp"
	"time"
)

// icinga2 const for return code and message
const (
	OK       = "OK"
	WAR      = "WARNING"
	CRI      = "CRITICAL"
	UNK      = "UNKNOWN"
	OK_CODE  = 0
	WAR_CODE = 1
	CRI_CODE = 2
	UNK_CODE = 3
)

// Job Status Record
type BEJobStatus struct {
	Name                  string
	JobType               string
	TaskType              string
	TaskName              string
	IsActive              string
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

// BEMCLI Class
type BEMCLI struct {
	message   string
	status    int
	sshClient *ssh.Client
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

	session.Stdout = &b
	if err := session.Run(command); err != nil {
		log.Fatal("Failed to run: " + err.Error())
	}
	return b.String()
}

// func NewBEMCLI
// Initialize connection to SSH server
func (bemcli *BEMCLI) NewBEMCLI(host string, username string, password string, identity string, port int) {
	var signer ssh.Signer

	// Initialize BEMCLI properties
	bemcli.message = ""
	bemcli.status = UNK_CODE

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
	fmt.Println(bemcli.sendCommand(beCommand))

}

// func GetBEJobBackupDefinition
// Get Detailed information for all job corresponding to a BackupDefinition and return a map (by jobname) of status
func (bemcli *BEMCLI) GetBEJobBackupDefinition(backupDefinition string) map[string]BEJobStatus {
	// Definition of Regex to parse Data
	reParser := regexp.MustCompile(`Name.*: ($`)

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

	fmt.Println(data)
	return beJobStatus
}

// func GetBEBackupExecSetting
// Get Server settings
func (bemcli *BEMCLI) GetBEBackupExecSetting() {
	beCommand := fmt.Sprintf("Import-Module BEMCLI; %s", "Get-BEBackupExecSetting")
	fmt.Println(bemcli.sendCommand(beCommand))
}
