package appliance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/appgate/sdpctl/pkg/api"
	"github.com/appgate/sdpctl/pkg/cmdappliance"
	"github.com/appgate/sdpctl/pkg/docs"
	"github.com/appgate/sdpctl/pkg/factory"
	"github.com/appgate/sdpctl/pkg/terminal"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/appgate/sdpctl/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type logOpts struct {
	cmdappliance.AppliancCmdOpts
	Out        io.Writer
	BaseURL    string
	HTTPClient func() (*http.Client, error)
	SpinnerOut func() io.Writer
	Version    int
	Path       string
	json       bool
}

func NewLogsCmd(f *factory.Factory) *cobra.Command {
	aopts := cmdappliance.AppliancCmdOpts{
		Appliance: f.Appliance,
		Config:    f.Config,
		CanPrompt: f.CanPrompt(),
	}

	opts := logOpts{
		aopts,
		f.IOOutWriter,
		f.BaseURL(),
		f.CustomHTTPClient,
		f.GetSpinnerOutput(),
		f.Config.Version,
		"",
		false,
	}
	cmd := &cobra.Command{
		Use:     "logs",
		Short:   docs.ApplianceLogsDoc.Short,
		Long:    docs.ApplianceLogsDoc.Short,
		Example: docs.ApplianceLogsDoc.ExampleString(),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return cmdappliance.ArgsSelectAppliance(cmd, args, &opts.AppliancCmdOpts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			return logsRun(c, args, &opts)
		},
	}
	cmd.Flags().StringVarP(&opts.Path, "path", "", "", "Optional path to write to")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Display in JSON format")
	return cmd
}

func logsRun(cmd *cobra.Command, args []string, opts *logOpts) error {
	terminal.Lock()
	defer terminal.Unlock()
	client, err := opts.HTTPClient()
	if err != nil {
		return fmt.Errorf("Could not resolve a HTTP client based on your current config %s", err)
	}
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(opts.Path) > 0 {
		path = opts.Path
	}

	requestURL := fmt.Sprintf("%s/appliances/%s/logs", opts.BaseURL, opts.ApplianceID)
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	request = request.WithContext(context.WithValue(context.Background(), factory.ContextAcceptValue, fmt.Sprintf("application/vnd.appgate.peer-v%d+zip", opts.Version)))
	log.Infof("Starting downloading log zip bundle or %s", opts.ApplianceID)
	response, err := client.Do(request)
	if response == nil || err != nil {
		return api.HTTPErrorResponse(response, err)
	}
	if response.StatusCode != http.StatusOK {
		return api.HTTPErrorResponse(response, err)
	}
	defer response.Body.Close()

	name := fmt.Sprintf("%s_logs.zip", opts.ApplianceID)
	_, params, err := mime.ParseMediaType(response.Header.Get("Content-Disposition"))
	if err == nil {
		if v, ok := params["filename"]; ok {
			name = v
		}
	}

	file, err := os.Create(filepath.Join(path, name))
	if err != nil {
		return err
	}

	p := mpb.New(mpb.WithWidth(64), mpb.WithOutput(opts.SpinnerOut()))

	bar := p.New(0,
		mpb.SpinnerStyle(tui.SpinnerStyle...),
		mpb.BarFillerOnComplete(tui.Check),
		mpb.BarWidth(1),
		mpb.AppendDecorators(decor.Name(name+" "), decor.CurrentKiloByte("% .1f")),
	)

	size, err := copy(file, response.Body, bar)
	if err != nil {
		return err
	}
	p.Wait()
	log.Infof("Downloaded %d bytes zip bundle for %s", size, opts.ApplianceID)
	if opts.json {
		return util.PrintJSON(opts.Out, map[string]string{"path": file.Name()})
	}
	fmt.Fprintf(opts.Out, "saved to %s\n", file.Name())
	return nil
}

func copy(dst io.Writer, src io.Reader, bar *mpb.Bar) (written int64, err error) {
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			// increment methods won't trigger complete event because bar was constructed with total = 0
			bar.IncrBy(nr)
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("Invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er == io.EOF {
				// triggering complete event now
				bar.SetTotal(-1, true)
			} else if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
