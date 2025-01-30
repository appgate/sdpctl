package appliance

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/appgate/sdpctl/pkg/configuration"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/hashicorp/go-multierror"
	"github.com/klauspost/compress/zstd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/appgate/journaldreader/journaldreader"
)

type logextractOpts struct {
	Path string
}

func NewExtractLogsCmd(f *factory.Factory) *cobra.Command {
	opts := logextractOpts{}
	cmd := &cobra.Command{
		Use:     "extract-logs",
		Short:   docs.ApplianceExtractLogsDoc.Short,
		Long:    docs.ApplianceExtractLogsDoc.Long,
		Example: docs.ApplianceExtractLogsDoc.ExampleString(),
		RunE: func(c *cobra.Command, args []string) error {
			return logsExtractRun(args, &opts)
		},
		Annotations: map[string]string{
			configuration.SkipAuthCheck: "true",
		},
	}
	cmd.Flags().StringVar(&opts.Path, "path", ".", "Optional path to write to")
	return cmd
}

func logsExtractRun(args []string, opts *logextractOpts) error {
	var errs *multierror.Error
	for i := 0; i < len(args); i++ {
		log.Infof("Starting processing %s", args[i])
		if err := processJournalFile(args[i], opts.Path); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}

func processJournalFile(file string, path string) error {
	const ZIPFILE_ZSTANDARD uint16 = 93 // Magic number for zstd in zip format
	r, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	defer r.Close()

	r.RegisterDecompressor(ZIPFILE_ZSTANDARD, func(in io.Reader) io.ReadCloser {
		dec, _ := zstd.NewReader(in)
		return io.NopCloser(dec)
	})

	var extractedFiles []string

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".journal") {
			log.Infof("Extracting journal file %s", f.Name)

			rc, err := f.Open()
			if err != nil {
				return err
			}

			extracted, err := os.CreateTemp("", "")
			if err != nil {
				return err
			}
			extractedFiles = append(extractedFiles, extracted.Name())

			_, err = io.Copy(extracted, rc)
			if err != nil {
				return err
			}
			rc.Close()
			extracted.Close()
		}
	}

	// Sort the extracted journald files
	extractedFiles = journaldreader.SortJournalFiles(extractedFiles)

	log.Infof("Extracting journal files complete. Processing...")

	textlogs := make(map[string]*os.File)

	// Parse the extracted files
	for _, journalfile := range extractedFiles {
		j := journaldreader.SdjournalReader{}

		log.Infof("Processing %s...", journalfile)

		err := j.Open(journalfile)
		if err != nil {
			continue
		}

		for {
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
					return err
				}
				defer logfile.Close()
				textlogs[identifier] = logfile
			}

			logfile.WriteString(formatEntry(entry))
		}

		j.Close()
		err = os.Remove(journalfile)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
 * Formats a log entry the default way like journalctl
 */
func formatEntry(entry map[string]string) string {
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
