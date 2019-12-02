// Copyright (c) 2019 Terre des hommes foundation - Claude Débieux
// Use of this source code is governed by a MIT License style
// license that can be found in the LICENSE file.
//
// check_backupexec is a icinga2/nagios plugin who return state of backupexec jobs
//
// Author : Claude Débieux - Terre des hommes foundation - Lausanne
// Version History:
//		26-nov-2019 - version #000 - Plugin Creation

package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"os"
)

var (
	arguments docopt.Opts
	err       error

	params struct {
		command          string
		host             string
		port             int
		username         string
		password         string
		identity         string
		version          bool
		backupDefinition bool
		jobName          string
	}

	icinga Icinga
)

func init() {
	usage := `check_backupexec
Check Backup Exec Jobs 
Usage: 
	check_backupexec (-h | --help | --version)
	check_backupexec get-job (-H <host> | --host=<host> -u <username> | --username=<username>) [-p <password> | --password=<password> | -i <pkey_file> | --identity=<pkey_file] [-P <port> | --port=<port>] [--backup-definition | -D] <job name> 
	check_backupexec get-setting (-H <host> | --host=<host> -u <username> | --username=<username>) [-p <password> | --password=<password> | -i <pkey_file> | --identity=<pkey_file] [-P <port> | --port=<port>] 
Options:
	--version  				Show check_cattools version.
	-h --help  				Show this screen.
	-H <host> --host=<host>  		Backup Exec server hostname or IP Address
	-u <username> --username=<username>  	Username
	-p <password> --password=<password>  	Password
	-i <pkey_file> --identity=<pkey_file>  	Private key file [default: ~/.ssh/id_rsa]
	-P <port> --port=<port>  		Port number [default: 22]
	-D --backup-definition  		Job name is a backup Definition
	<job name>				Name of backup exec Job
`

	arguments, err = docopt.ParseDoc(usage)
	if err != nil {
		fmt.Printf("%s Error parsing command line arguments: %v", UNK, err)
		os.Exit(UNK_CODE)
	}

	if c, _ := arguments.Bool("get-setting"); c {
		params.command = "get-setting"
	}

	if c, _ := arguments.Bool("get-job"); c {
		params.command = "get-job"
	}

	params.version, _ = arguments.Bool("--version")
	params.port, _ = arguments.Int("--port")
	params.host, _ = arguments.String("--host")
	params.username, _ = arguments.String("--username")
	params.password, _ = arguments.String("--password")
	params.identity, _ = arguments.String("--identity")
	params.backupDefinition, _ = arguments.Bool("--backup-definition")
	params.jobName, _ = arguments.String("<job name>")

	// Initialize icinga info
	icinga.Status = UNK
	icinga.StatusCode = UNK_CODE
	icinga.Message = ""
	icinga.Metric = ""
}

func main() {

	var bemcli *BEMCLI

	if params.version {
		fmt.Println("check_backupexec version 1.0.1")
		os.Exit(OK_CODE)
	}

	bemcli = new(BEMCLI)
	bemcli.Init(params.host, params.username, params.password, params.identity, params.port)

	switch params.command {
	case "get-job":
		if params.backupDefinition {
			jobs := bemcli.GetBEJobBackupDefinition(params.jobName)
			for jobName, job := range jobs {
				switch bemcli.Condition(job.JobStatus) {
				case OK_CODE:
					if icinga.StatusCode == UNK_CODE {
						icinga.StatusCode = OK_CODE
						icinga.Status = OK
					}
					if icinga.Message != "" {
						icinga.Message += "/"
					}
					icinga.Message += jobName + " " + job.JobStatus
				case WAR_CODE:
					if icinga.StatusCode == UNK_CODE || icinga.StatusCode < WAR_CODE {
						icinga.StatusCode = WAR_CODE
						icinga.Status = WAR
					}
					if icinga.Message != "" {
						icinga.Message += "/"
					}
					icinga.Message += jobName + " " + job.JobStatus
				case CRI_CODE:
					if icinga.StatusCode == UNK_CODE || icinga.StatusCode < CRI_CODE {
						icinga.StatusCode = CRI_CODE
						icinga.Status = CRI
					}
					if icinga.Message != "" {
						icinga.Message = jobName + " " + job.JobStatus + " [" + job.ErrorMessage + "]/" + icinga.Message
					} else {
						icinga.Message = jobName + " " + job.JobStatus + " [" + job.ErrorMessage + "]"
					}
				default:
					if job.IsActive {
						if icinga.StatusCode == UNK_CODE {
							icinga.StatusCode = OK_CODE
							icinga.Status = OK
						}
					} else {
						if icinga.StatusCode == UNK_CODE || icinga.StatusCode < WAR_CODE {
							icinga.StatusCode = WAR_CODE
							icinga.Status = WAR
						}
					}
					if icinga.Message != "" {
						icinga.Message += "/"
					}
					icinga.Message += jobName + " " + job.Status + "-" + job.SubStatus
				}
			}
			if icinga.Metric != "" {
				icinga.Metric = " | " + icinga.Metric
			}
			fmt.Printf("%s: %s%s\n", icinga.Status, icinga.Message, icinga.Metric)
			os.Exit(icinga.StatusCode)
		} else {
			bemcli.GetBEJob(params.jobName)
			os.Exit(OK_CODE)
		}
	case "get-setting":
		bemcli.GetBEBackupExecSetting()
		os.Exit(OK_CODE)
	}
}
