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
	"strconv"
)

var (
	arguments  docopt.Opts
	err        error
	buildcount string
	usage      string

	params struct {
		command          string
		host             string
		port             int
		username         string
		password         string
		identity         string
		version          bool
		verbose          bool
		backupDefinition bool
		jobName          string
	}
)

func init() {
	// Defining help and check_backupexec usage information
	usage = `check_backupexec
Check Backup Exec Jobs 
Usage: 
	check_backupexec (-h | --help | --version)
	check_backupexec get-job (-H <host> | --host=<host> -u <username> | --username=<username>) [-p <password> | --password=<password> | -i <pkey_file> | --identity=<pkey_file] [-P <port> | --port=<port>] [-v | --verbose] [--backup-definition | -D] <job name> 
	check_backupexec get-setting (-H <host> | --host=<host> -u <username> | --username=<username>) [-p <password> | --password=<password> | -i <pkey_file> | --identity=<pkey_file] [-P <port> | --port=<port>] 
Options:
	--version  				Show check_cattools version.
	-h --help  				Show this screen.
	-v --verbose  		Display verbose output
	-H <host> --host=<host>  		Backup Exec server hostname or IP Address
	-u <username> --username=<username>  	Username
	-p <password> --password=<password>  	Password
	-i <pkey_file> --identity=<pkey_file>  	Private key file [default: ~/.ssh/id_rsa]
	-P <port> --port=<port>  		Port number [default: 22]
	-D --backup-definition  		Job name is a backup Definition
	<job name>				Name of backup exec Job
`

	// Don't parse command line argument for testing argument must be passed with OS environment variable
	if os.Getenv("CHECK_MODE") == "TEST" {
		params.version, _ = strconv.ParseBool(os.Getenv("VERSION"))
		params.port, _ = strconv.Atoi(os.Getenv("PORT"))
		params.host = os.Getenv("HOST")
		params.username = os.Getenv("USERNAME")
		params.password = os.Getenv("PASSWORD")
		params.identity = os.Getenv("IDENTITY")
		if params.identity == "" && params.password == "" {
			params.identity = "~/.ssh/id_rsa"
		}
		params.backupDefinition, _ = strconv.ParseBool(os.Getenv("BACKUPDEFINITION"))
		params.jobName = os.Getenv("JOBNAME")
		params.verbose, _ = strconv.ParseBool(os.Getenv("VERBOSE"))
		params.command = os.Getenv("COMMAND")
	} else {
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
		params.verbose, _ = arguments.Bool("--verbose")
	}

}

func main() {

	var bemcli *BEMCLI

	// Display version and buildnumber of check_backupexec
	if params.version {
		fmt.Printf("check_backupexec version 1.0.2-build %s\n", buildcount)
		os.Exit(OK_CODE)
	}

	// Creating and initializing BackupExec Module session to BackupExec Server
	bemcli = new(BEMCLI)
	bemcli.Init(params.host, params.username, params.password, params.identity, params.port)

	// Executing defined command
	switch params.command {
	// Getting job status
	case "get-job":
		if params.backupDefinition {
			s, c := bemcli.BEJobsStatusToIcingaStatus(bemcli.GetBEJobBackupDefinition(params.jobName), params.verbose)
			fmt.Printf(s)
			os.Exit(c)
		} else {
			bemcli.GetBEJob(params.jobName)
			os.Exit(OK_CODE)
		}
	// Getting server setting if communication to server is Ok return OK_CODE
	case "get-setting":
		bemcli.GetBEBackupExecSetting()
		os.Exit(OK_CODE)
	// If command is not handled return CRI_CODE
	default:
		fmt.Printf("check_backupexec version 1.0.2-build %s\n", buildcount)
		fmt.Printf("Unknown command\n")
		fmt.Printf("Usage: %s", usage)
		os.Exit(CRI_CODE)
	}
}
