package appliance

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/appgate/sdpctl/pkg/cmdappliance"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/appgate/journaldreader/journaldreader"
)

type logextractOpts struct {
	cmdappliance.AppliancCmdOpts
	Path string
}

func NewExtractLogsCmd(f *factory.Factory) *cobra.Command {
	aopts := cmdappliance.AppliancCmdOpts{
		Appliance: f.Appliance,
		Config:    f.Config,
		CanPrompt: f.CanPrompt(),
	}

	opts := logextractOpts{
		aopts,
		"",
	}
	cmd := &cobra.Command{
		Use:     "extract-logs",
		Short:   docs.ApplianceLogsDoc.Short,
		Long:    docs.ApplianceLogsDoc.Short,
		Example: docs.ApplianceLogsDoc.ExampleString(),
		// PreRunE: func(cmd *cobra.Command, args []string) error {
		// 	return cmdappliance.ArgsSelectAppliance(cmd, args, &opts.AppliancCmdOpts)
		// },
		RunE: func(c *cobra.Command, args []string) error {
			return logsExtractRun(c, args, &opts)
		},
	}
	cmd.Flags().StringVarP(&opts.Path, "path", "", "", "Optional path to write to")
	return cmd
}

func logsExtractRun(cmd *cobra.Command, args []string, opts *logextractOpts) error {
	for i := 0; i < len(args); i++ {
		fmt.Println("Starting processing %s", args[i])
		err := process_journal_file(args[i], opts.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func process_journal_file(file string, path string) error {
	r, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	defer r.Close()

	var extracted_files []string

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".journal") {
			fmt.Println("Extracting journal file %s", f.Name)

			rc, err := f.Open()
			if err != nil {
				return err
			}

			extracted, err := os.CreateTemp("", "")
			if err != nil {
				return err
			}
			extracted_files = append(extracted_files, extracted.Name())

			written, err := io.Copy(extracted, rc)
			if written+1 == written-1 { // Completely useless statement to get go to compile
				log.Fatal(written)
			}
			if err != nil {
				log.Fatal(err)
			}
			rc.Close()
			extracted.Close()
		}
	}

	// Sort the extracted journald files
	sort.Strings(extracted_files)

	fmt.Println("Extracting journal files complete. Processing...")

	textlogs := make(map[string]*os.File)

	// Parse the extracted files
	for _, journalfile := range extracted_files {
		j := journaldreader.SdjournalReader{}

		fmt.Println("Processing %s...", journalfile)

		err := j.Open(journalfile)
		if err != nil {
			log.Fatal(err)
		}

		for true {
			entry, hasnext, err := j.Next()
			if err != nil {
				// We just move to the next log file in case of error
				break
			}
			if !hasnext {
				break
			}

			identifier, exists := entry["SYSLOG_IDENTIFIER"]
			if !exists {
				identifier = "uncategorised_entries"
			}

			logfile, exists := textlogs[identifier]
			if !exists {
				logfile, err := os.Create(filepath.Join(path, identifier+".log"))
				if err != nil {
					return nil
				}
				defer logfile.Close()
				textlogs[identifier] = logfile
			}

			logfile.WriteString(format_entry(entry))
		}

		j.Close()
		if os.Remove(journalfile) != nil {
			return err
		}
	}

	return nil
}

/*
 * Formats a log entry the default way like journalctl
 */
func format_entry(entry map[string]string) string {
	timestamp, exists := entry["SYSLOG_TIMESTAMP"]
	if !exists {
		timestamp = ""
	}

	hostname, exists := entry["_HOSTNAME"]
	if !exists {
		hostname = ""
	}

	identifier, exists := entry["SYSLOG_IDENTIFIER"]
	if !exists {
		identifier = ""
	}

	pid, exists := entry["_PID"]
	if !exists {
		pid = ""
	}

	message, exists := entry["MESSAGE"]
	if !exists {
		message = ""
	}
	return fmt.Sprintf("%s %s %s[%s]: %s\n", timestamp, hostname, identifier, pid, message)
}
