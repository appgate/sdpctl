package files

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	appliancepkg "github.com/appgate/sdpctl/pkg/appliance"
	"github.com/appgate/sdpctl/pkg/tui"
	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
)

type FilesAPI struct {
	API      *appliancepkg.Appliance
	Progress *tui.Progress
}

func (f *FilesAPI) Upload(ctx context.Context, file *os.File, rename string) error {
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	size := fileInfo.Size()
	name := fileInfo.Name()
	uploadName := name
	if len(rename) > 0 {
		uploadName = rename
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", uploadName)
		if err != nil {
			log.Warnf("multipart form err %s", err)
			return
		}

		_, err = io.Copy(part, file)
		if err != nil {
			log.Warnf("copy err %s", err)
		}
	}()

	headers := map[string]string{
		"Content-Type":        writer.FormDataContentType(),
		"Content-Disposition": fmt.Sprintf("attachment; filename=%s", uploadName),
	}

	var reader io.ReadCloser
	reader = pr
	defer reader.Close()
	var t *tui.Tracker
	endMsg := "uploaded"
	if f.Progress != nil {
		progressString := name
		if len(rename) > 0 {
			progressString = fmt.Sprintf("%s -> %s", progressString, rename)
		}
		reader, t = f.Progress.FileUploadProgress(progressString, endMsg, size, pr)
		go t.Watch([]string{endMsg}, []string{appliancepkg.FileFailed})
	}
	if err := f.API.UploadFile(ctx, reader, headers); err != nil {
		if t != nil {
			t.Fail(err.Error())
		}
		return err
	}
	return func(ctx context.Context, api *appliancepkg.Appliance) error {
		return backoff.Retry(func() error {
			v, err := api.FileStatus(ctx, name)
			if err != nil {
				if t != nil {
					t.Fail(err.Error())
				}
				return backoff.Permanent(err)
			}
			status := v.GetStatus()
			if t != nil {
				t.Update(status)
			}
			if status == appliancepkg.FileReady {
				if t != nil {
					t.Update(endMsg)
				}
				return nil
			}
			if status == appliancepkg.FileFailed {
				return backoff.Permanent(fmt.Errorf("%s failed: %q", name, err))
			}
			return fmt.Errorf("file not ready")
		}, backoff.NewExponentialBackOff())
	}(ctx, f.API)
}
